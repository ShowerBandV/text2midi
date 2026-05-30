// Package composer — Motif Engine (Midra-style: simple, musical).
// Generates lead melody from scale degrees with stepwise bias and random velocities.
package composer

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ─── Motif Plan ────────────────────────────────────────────────────

type MotifPlan struct {
	UseRate         float64 // 0-1: how often motif appears
	VariationLevel  float64 // 0-1: how aggressive variations are
	CallResponse    bool
	OctaveStrategy  string // "flat", "chorus_up", "gradual"
	BarsPerPhrase   int
	TotalBars       int
}

// ─── Motif Variator ────────────────────────────────────────────────

// Transpose shifts all notes by an interval.
func Transpose(motif []int, interval int) []int {
	out := make([]int, len(motif))
	for i, n := range motif {
		out[i] = n + interval
	}
	return out
}

// Invert mirrors intervals: +2 +3 → -2 -3
func Invert(motif []int) []int {
	if len(motif) == 0 {
		return motif
	}
	out := make([]int, len(motif))
	out[0] = motif[0] // keep first note
	for i := 1; i < len(motif); i++ {
		interval := motif[i] - motif[i-1]
		out[i] = out[i-1] - interval
	}
	return out
}

// Retrograde reverses the note order.
func Retrograde(motif []int) []int {
	out := make([]int, len(motif))
	for i, n := range motif {
		out[len(motif)-1-i] = n
	}
	return out
}

// Fragment takes the first n notes.
func Fragment(motif []int, n int) []int {
	if n > len(motif) {
		n = len(motif)
	}
	out := make([]int, n)
	copy(out, motif[:n])
	return out
}

// Extend appends passing notes to lengthen the motif.
func Extend(motif []int, extra int) []int {
	if len(motif) < 2 {
		return motif
	}
	out := make([]int, len(motif)+extra)
	copy(out, motif)
	last := motif[len(motif)-1]
	avgStep := 0
	for i := 1; i < len(motif); i++ {
		avgStep += motif[i] - motif[i-1]
	}
	avgStep /= len(motif) - 1
	for i := 0; i < extra; i++ {
		out[len(motif)+i] = last + avgStep*(i+1)
	}
	return out
}

// ─── Phrase Builder ────────────────────────────────────────────────

// Phrase = 4 bars.
type Phrase struct {
	Bars [4][]int // each bar = []int of relative pitches
}

// BuildPhrase creates a 4-bar phrase from a motif.
// Structure: A (motif), A' (variation), B (contrast), A (return).
func BuildPhrase(motif []int, plan MotifPlan, rng *rand.Rand) Phrase {
	p := Phrase{}

	// Bar 0: Motif A (original or slightly varied).
	if rng.Float64() < 0.3 && plan.VariationLevel > 0.3 {
		p.Bars[0] = Transpose(motif, []int{2, 3, 5}[rng.Intn(3)])
	} else {
		p.Bars[0] = copySlice(motif)
	}

	// Bar 1: Motif A' (variation).
	switch rng.Intn(3) {
	case 0:
		p.Bars[1] = Invert(motif)
	case 1:
		p.Bars[1] = Transpose(motif, []int{-5, -3, 3, 5}[rng.Intn(4)])
	case 2:
		p.Bars[1] = Fragment(Retrograde(motif), len(motif)-1)
	}

	// Bar 2: Motif B (contrast — different rhythm/interval direction).
	if rng.Float64() < 0.5 {
		p.Bars[2] = Extend(motif, 2)
	} else {
		p.Bars[2] = Transpose(Invert(motif), 7)
	}

	// Bar 3: Motif A (return).
	if rng.Float64() < 0.4 {
		p.Bars[3] = copySlice(motif) // exact return
	} else {
		p.Bars[3] = Transpose(motif, -2) // return with slight shift
	}

	return p
}

// ─── Section Composer ──────────────────────────────────────────────

// BuildSection generates phrases for a given section type.
// BuildSection moved to phrase.go (style-aware)

// GenerateLeadMidra is a Go port of Midra's generate_lead().
// Generates a random scale-degree motif with stepwise bias, anchors, and random velocities.
// secDensity: per-bar density [0-1]. secRegister: per-bar octave shift (nil = auto from density).
// usePentatonic: if true, uses pentatonic scale (1-2-3-5-6) with Chinese ornamentation.
func GenerateLeadMidra(keyRoot, keyMode string, totalBars int, stepProb float64, velMin, velMax int, secDensity []float64, secRegister []int, usePentatonic bool) []schema.NoteEvent {
	scale := getScaleDegrees(keyRoot, keyMode)
	if len(scale) == 0 {
		scale = []int{0, 2, 3, 5, 7, 8, 10} // fallback: C minor
	}

	// Pentatonic override: use 1-2-3-5-6 scale (5 notes per octave).
	scaleSize := 7
	if usePentatonic {
		scale = getPentatonicDegrees(keyRoot)
		scaleSize = 5
	}

	rng := rand.New(rand.NewSource(globalSeed))
	motifLen := 8
	motif := make([]int, motifLen)

	// Generate random motif from scale degrees.
	for i := range motif {
		motif[i] = rng.Intn(scaleSize)
	}
	// Anchor: first note = root (0), last = root/third (0, 2, 4).
	motif[0] = 0
	lastChoices := []int{0, 2}
	if scaleSize == 5 {
		lastChoices = []int{0, 2}
	}
	motif[motifLen-1] = lastChoices[rng.Intn(len(lastChoices))]

	// Stepwise bias.
	for i := 1; i < motifLen-1; i++ {
		if rng.Float64() < stepProb {
			step := []int{-2, -1, 1, 2}[rng.Intn(4)]
			motif[i] = motif[i-1] + step
			if motif[i] < 0 {
				motif[i] = 0
			}
			if motif[i] >= scaleSize {
				motif[i] = scaleSize - 1
			}
		}
	}

	// Generate events with per-section density and register.
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		density := 0.5 // default
		if bar < len(secDensity) && secDensity[bar] > 0 {
			density = secDensity[bar]
		}
		// Low density: only play first few notes of motif.
		// High density: full motif.
		noteCount := int(float64(motifLen) * density)
		if noteCount < 2 { noteCount = 2 }
		if noteCount > motifLen { noteCount = motifLen }
		// Register: use explicit section register if provided, else auto from density.
		octave := 5
		if bar < len(secRegister) && secRegister[bar] != 0 {
			octave = secRegister[bar]
		} else {
			if density > 0.7 {
				octave = 6
			}
			if density < 0.3 {
				octave = 4
			}
		}

		for i := 0; i < noteCount; i++ {
			step := motif[i]
			scaleIdx := step % len(scale)
			if scaleIdx < 0 {
				scaleIdx += len(scale)
			}
			pitch := scale[scaleIdx] + 12*(octave-1)
			velocity := velMin + rng.Intn(velMax-velMin)

			// Articulation: mix legato (long, connected) and staccato (short, detached).
			duration := []float64{0.25, 0.4, 0.5, 0.75}[rng.Intn(4)]
			articRoll := rng.Float64()
			if articRoll < 0.15 {
				duration = 0.08 // staccato pop
			} else if articRoll < 0.35 {
				duration = 1.2 // legato sustain
			} else if articRoll < 0.50 {
				duration = 0.15 // short accent
			}
			// Clamp to note spacing.
			noteSpacing := 0.5
			if i+1 < noteCount {
				noteSpacing = float64(i+1)*0.5 - float64(i)*0.5
			}
			if duration > noteSpacing*0.9 {
				duration = noteSpacing * 0.9
			}

			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat:    base + float64(i)*0.5,
				DurationBeat: duration,
				Velocity:     velocity,
			})
		}
	}

	// Pentatonic ornamentation: grace notes and slides for Chinese feel.
	if usePentatonic {
		events = addPentatonicOrnaments(events, totalBars)
	}

	// Arch shape: reshape pitch contour so melody has direction.
	events = shapeMelodyArch(events, totalBars, scale[rng.Intn(len(scale))])

	fmt.Printf("[MidraLead] %d-note motif from %d-note scale, %d events, %d bars (pentatonic=%t)\n",
		motifLen, len(scale), len(events), totalBars, usePentatonic)
	return events
}

// getPentatonicDegrees returns MIDI pitches for a pentatonic (1-2-3-5-6) scale.
func getPentatonicDegrees(root string) []int {
	rootSemi := map[string]int{
		"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5,
		"F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11,
	}
	rs := rootSemi[root]
	intervals := []int{0, 2, 4, 7, 9} // 1-2-3-5-6
	result := make([]int, 0, len(intervals)*4)
	for oct := 3; oct <= 6; oct++ {
		base := (oct + 1) * 12
		for _, iv := range intervals {
			p := base + rs + iv
			if p >= 21 && p <= 108 {
				result = append(result, p)
			}
		}
	}
	return result
}

// addPentatonicOrnaments adds grace notes and slides for Chinese pentatonic feel.
// Grace note: a quick note a minor 3rd below, sliding up to the target.
func addPentatonicOrnaments(events []schema.NoteEvent, totalBars int) []schema.NoteEvent {
	if len(events) < 4 {
		return events
	}
	rng := rand.New(rand.NewSource(globalSeed))
	var result []schema.NoteEvent
	for _, ev := range events {
		// ~25% of notes get a grace note ornament.
		if rng.Float64() < 0.25 && ev.DurationBeat > 0.2 && ev.Pitch > 24 {
			// Grace note: quick slide from a minor 3rd below.
			result = append(result, schema.NoteEvent{
				Type: "note", Pitch: ev.Pitch - 3,
				StartBeat: ev.StartBeat - 0.04, DurationBeat: 0.04,
				Velocity: ev.Velocity - 25,
			})
		}
		result = append(result, ev)
		// ~10% of notes get a vibrato-like trill (quick upper neighbor).
		if rng.Float64() < 0.10 && ev.DurationBeat > 0.5 {
			result = append(result, schema.NoteEvent{
				Type: "note", Pitch: ev.Pitch + 2,
				StartBeat: ev.StartBeat + ev.DurationBeat*0.5, DurationBeat: 0.06,
				Velocity: ev.Velocity - 20,
			})
		}
	}
	return result
}

// GenerateLeadMetal generates metal lead guitar: harmonic minor, fast runs, sweep fragments.
// songSection returns (name, startBar, endBar) for a position in a standard metal song.
// Structure: Intro(8) → Verse(16) → Chorus(16) → Bridge(8) → Chorus(16) → Outro(8).
func songSection(bar, totalBars int) string {
	introEnd := 8
	verse1End := introEnd + 16
	chorus1End := verse1End + 16
	bridgeEnd := chorus1End + 8
	chorus2End := bridgeEnd + 16

	if totalBars <= 32 {
		// Short song: scale down.
		introEnd = 4
		verse1End = introEnd + 8
		chorus1End = verse1End + 8
		bridgeEnd = chorus1End + 4
		chorus2End = bridgeEnd + 8
	}

	switch {
	case bar < introEnd:
		return "intro"
	case bar < verse1End:
		return "verse"
	case bar < chorus1End:
		return "chorus"
	case bar < bridgeEnd:
		return "bridge"
	case bar < chorus2End:
		return "chorus2"
	default:
		return "outro"
	}
}

func GenerateLeadMetal(keyRoot string, totalBars int, energy float64) []schema.NoteEvent {
	scale := getHarmonicMinorDegrees(keyRoot)
	var events []schema.NoteEvent

	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		sec := songSection(bar, totalBars)

		switch sec {
		// ── INTRO: Riff theme — establish the motif ──────────
		case "intro":
			hook := []int{scale[0] + 12*3, scale[2] + 12*3, scale[4] + 12*3, scale[1] + 12*4}
			if bar%2 == 0 {
				for i, p := range hook {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p,
						StartBeat: base + float64(i)*0.5, DurationBeat: 0.15,
						Velocity: 110,
					})
				}
			} else {
				for i, p := range hook {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p + 5,
						StartBeat: base + float64(i)*0.5, DurationBeat: 0.15,
						Velocity: 105,
					})
				}
			}

		// ── VERSE: Arpeggio or rest — support, don't lead ────
		case "verse":
			root := scale[0] + 12*3
			// Sparse: only play on bars 0 and 2 of each 4-bar phrase.
			if bar%4 == 0 || bar%4 == 2 {
				arp := []int{root, root + 7, root + 12}
				for i, p := range arp {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p,
						StartBeat: base + float64(i)*0.8, DurationBeat: 0.5,
						Velocity: 65,
					})
				}
			}

		// ── CHORUS: Silent — rhythm guitar + twin harmony carry ──
		case "chorus", "chorus2":
			// Lead rests. Twin harmony (separate track) handles the melody.

		// ── BRIDGE: SOLO — sweep + tremolo + whammy + neo-classical ──
		case "bridge":
			root := scale[0] + 12*3
			dim := getDiminishedDegrees(keyRoot)
			// First half of bridge: sweep+tap. Second half: neo-classical diminished runs.
			if bar%4 < 2 && bar%2 == 0 {
				// Sweep + tap.
				up := []int{root, root + 4, root + 7, root + 12, root + 16, root + 19}
				for i, p := range up {
					vel := 110
					if i == 5 {
						vel = 80
					}
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p,
						StartBeat: base + float64(i)*0.10, DurationBeat: 0.08,
						Velocity: vel,
					})
				}
				top := root + 19
				for rep := 0; rep < 8; rep++ {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: top,
						StartBeat: base + 0.6 + float64(rep)*0.06, DurationBeat: 0.04,
						Velocity: 100,
					})
				}
			} else if bar%4 >= 2 {
				// Neo-classical: diminished arpeggio with chromatic passing tones.
				start := dim[0] + 12*3
				for step := 0; step < 6; step++ {
					p := start + step*3
					if p < 48 || p > 96 { continue }
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p,
						StartBeat: base + float64(step)*0.15, DurationBeat: 0.08,
						Velocity: 105,
					})
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p - 1,
						StartBeat: base + float64(step)*0.15 + 0.06, DurationBeat: 0.04,
						Velocity: 70,
					})
				}
			} else {
				// Whammy dive + descending run.
				top := root + 19
				for step := 0; step < 6; step++ {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: top - step*3,
						StartBeat: base + float64(step)*0.08, DurationBeat: 0.06,
						Velocity: 90,
					})
				}
				down := []int{root + 16, root + 12, root + 9, root + 7, root + 4, root}
				for i, p := range down {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p,
						StartBeat: base + 1.0 + float64(i)*0.14, DurationBeat: 0.08,
						Velocity: 100,
					})
				}
			}

		// ── OUTRO: Riff recap — fade out ─────────────────────
		case "outro":
			hook := []int{scale[0] + 12*3, scale[2] + 12*3, scale[4] + 12*3, scale[1] + 12*4}
			for i, p := range hook {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base + float64(i)*0.5, DurationBeat: 0.2,
					Velocity: 90,
				})
			}
		}
	}

	fmt.Printf("[Lead-Metal] %d events, %d bars (intro→verse→chorus→bridge→outro)\n", len(events), totalBars)
	return events
}

// getDiminishedDegrees returns MIDI pitches for a diminished 7th arpeggio (1-b3-b5-bb7).
// GeneratePianoLegend creates a John Legend-style piano part: left hand bass + right hand chords + melody.
// Section-aware: sparse intro → building verse → full chorus → featured bridge → sparse outro.
func GeneratePianoLegend(keyRoot, keyMode string, totalBars int, chords []string) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent

	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		sec := songSection(bar, totalBars)
		chord := chords[bar%len(chords)]
		root := chordRootMIDIPiano(chord, 2) // C2-C3 for left hand

		switch sec {
		// ── INTRO: Solo piano, sparse, establishing the mood ──
		case "intro":
			// Left hand: single bass note, held.
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: root,
				StartBeat: base, DurationBeat: 3.8,
				Velocity: 40,
			})
			// Right hand: one or two chord tones, floating.
			rhNotes := chordTones(chord, 3, 2)
			for i, p := range rhNotes {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base + float64(i)*2.0, DurationBeat: 1.8,
					Velocity: 50,
				})
			}

		// ── VERSE: Left hand movement + right hand sparse chords ──
		case "verse":
			// Left hand: root → 5th → octave pattern.
			lhPattern := []struct {
				beat float64
				pitch int
				dur   float64
			}{
				{0.0, root, 1.5},
				{2.0, root + 7, 1.5},
			}
			if bar%2 == 1 {
				lhPattern[1].pitch = root + 12
			}
			for _, lh := range lhPattern {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: lh.pitch,
					StartBeat: base + lh.beat, DurationBeat: lh.dur,
					Velocity: 55,
				})
			}
			// Right hand: sparse chord stabs on 2 and 4.
			if bar%2 == 0 {
				rh := chordTones(chord, 3, 3)
				for _, beat := range []float64{1.0, 3.0} {
					for _, p := range rh {
						events = append(events, schema.NoteEvent{
							Type: "note", Pitch: p,
							StartBeat: base + beat, DurationBeat: 0.5,
							Velocity: 60,
						})
					}
				}
			}

		// ── CHORUS: Full piano — left hand octaves, right hand rich chords ──
		case "chorus", "chorus2":
			// Left hand: octaves on 1 and 3.
			for _, beat := range []float64{0.0, 2.0} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: root,
					StartBeat: base + beat, DurationBeat: 1.5,
					Velocity: 75,
				})
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: root + 12,
					StartBeat: base + beat, DurationBeat: 1.5,
					Velocity: 70,
				})
			}
			// Right hand: full chord voicing on every beat.
			rh := chordTones(chord, 3, 4)
			for _, beat := range []float64{0.0, 1.0, 2.0, 3.0} {
				for _, p := range rh {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p,
						StartBeat: base + beat, DurationBeat: 0.7,
						Velocity: 80,
					})
				}
			}
			// Melody fragment: a quick pentatonic run on beat 3&.
			if bar%2 == 0 {
				mel := []int{root + 12, root + 16, root + 19, root + 24}
				for i, p := range mel {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p,
						StartBeat: base + 2.5 + float64(i)*0.15, DurationBeat: 0.12,
						Velocity: 85,
					})
				}
			}

		// ── BRIDGE: Piano feature — expressive, melodic ───────
		case "bridge":
			// Left hand: arpeggiated pattern.
			lhArp := []int{root, root + 7, root + 12, root + 7}
			for i, p := range lhArp {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base + float64(i)*0.8, DurationBeat: 0.6,
					Velocity: 65,
				})
			}
			// Right hand: melodic phrase — improvisatory.
			rhMel := []int{
				root + 12, root + 16, root + 19,
				root + 16, root + 12, root + 7 + 12,
			}
			for i, p := range rhMel {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base + 1.5 + float64(i)*0.3, DurationBeat: 0.25,
					Velocity: 70 + rng.Intn(20),
				})
			}

		// ── OUTRO: Sparse, returning to intro feel ──────────
		case "outro":
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: root,
				StartBeat: base, DurationBeat: 3.8,
				Velocity: 35,
			})
			rh := chordTones(chord, 3, 1)
			if len(rh) > 0 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: rh[0],
					StartBeat: base + 1.0, DurationBeat: 2.5,
					Velocity: 40,
				})
			}
		}
	}

	fmt.Printf("[Piano-Legend] %d events, %d bars\n", len(events), totalBars)
	return events
}

// chordTones returns n MIDI pitches representing chord tones at a given octave.
func chordTones(chord string, octave int, n int) []int {
	root := chordRootMIDIPiano(chord, octave)
	isMinor := strings.Contains(chord, "m")
	third := root + 4
	if isMinor {
		third = root + 3
	}
	fifth := root + 7
	tones := []int{root, third, fifth, root + 12, fifth + 12}
	if n > len(tones) {
		n = len(tones)
	}
	return tones[:n]
}

// chordRootMIDIPiano strips chord suffixes to get the root MIDI pitch.
func chordRootMIDIPiano(chord string, octave int) int {
	root := chord
	for {
		trimmed := false
		for _, suf := range []string{"maj7", "min7", "m7", "sus4", "sus2", "dim", "aug", "7", "m", "M"} {
			if strings.HasSuffix(root, suf) && len(root) > len(suf) {
				root = root[:len(root)-len(suf)]
				trimmed = true
				break
			}
		}
		if !trimmed {
			break
		}
	}
	rootSemi := map[string]int{
		"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5,
		"F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11,
	}
	if rs, ok := rootSemi[root]; ok {
		return (octave+1)*12 + rs
	}
	return 36
}

// GenerateLeadRock creates a blues-scale rock lead: double-stops, repeating licks, sparse.
func GenerateLeadRock(keyRoot string, totalBars int, energy float64) []schema.NoteEvent {
	scale := getBluesScale(keyRoot)
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent

	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		sec := songSection(bar, totalBars)
		root := scale[0] + 12*3

		switch sec {
		case "intro":
			// Intro: repeating blues lick — 4 notes, call and response.
			lick := []int{root + 3, root, root + 2, root - 2}
			for i, p := range lick {
				if p < 48 {
					p += 12
				}
				// Double-stop: play the note + a 4th above.
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base + float64(i)*0.5, DurationBeat: 0.2,
					Velocity: 100,
				})
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p + 5,
					StartBeat: base + float64(i)*0.5, DurationBeat: 0.2,
					Velocity: 90,
				})
			}

		case "verse":
			// Verse: sparse, only every other bar. One short lick, then rest.
			if bar%4 < 2 {
				lick := []int{root + 3, root, root + 2}
				for i, p := range lick {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p,
						StartBeat: base + 1.0 + float64(i)*0.75, DurationBeat: 0.3,
						Velocity: 85,
					})
				}
			}

		case "chorus", "chorus2":
			// Chorus: double-stop lick every bar, more aggressive.
			lick := []int{root + 3, root, root + 2, root - 2, root + 7}
			for i, p := range lick {
				if p < 48 {
					p += 12
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base + float64(i)*0.4, DurationBeat: 0.15,
					Velocity: 105,
				})
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p + 5,
					StartBeat: base + float64(i)*0.4, DurationBeat: 0.15,
					Velocity: 95,
				})
			}

		case "bridge":
			// Bridge: longer solo-like phrase with bends (simulated by quick neighbor notes).
			solo := []int{root, root + 3, root + 7, root + 10, root + 12, root + 10, root + 7, root + 3}
			for i, p := range solo {
				// Bend: quick neighbor before target.
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p - 1,
					StartBeat: base + float64(i)*0.35 - 0.03, DurationBeat: 0.04,
					Velocity: 70,
				})
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base + float64(i)*0.35, DurationBeat: 0.2,
					Velocity: 95 + rng.Intn(15),
				})
			}

		case "outro":
			// Outro: just the first lick, fading.
			lick := []int{root + 3, root}
			for i, p := range lick {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base + float64(i)*1.0, DurationBeat: 0.5,
					Velocity: 70,
				})
			}
		}
	}

	fmt.Printf("[Lead-Rock] %d events, %d bars (blues double-stops)\n", len(events), totalBars)
	return events
}

// GenerateLeadSectioned creates a lead that varies by song section.
// Intro=hook, Verse=sparse, Chorus=full, Bridge=solo/rest, Outro=hook recap.
func GenerateLeadSectioned(scale []int, totalBars int, _ float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent
	hook := buildHook(scale, rng)

	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		sec := songSection(bar, totalBars)

		switch sec {
		case "intro":
			// Hook statement — play the motif with breathing.
			if bar%2 == 0 {
				for i, p := range hook {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p + 12*4,
						StartBeat: base + float64(i)*0.5, DurationBeat: 0.2, Velocity: 100,
					})
				}
			}
		case "verse":
			// Sparse — only every 4th bar, single notes.
			if bar%4 == 0 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: hook[0] + 12*4,
					StartBeat: base + 1.0, DurationBeat: 1.5, Velocity: 100,
				})
			}
		case "chorus", "chorus2":
			// Full — hook with double-stops.
			for i, p := range hook {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p + 12*5,
					StartBeat: base + float64(i)*0.4, DurationBeat: 0.15, Velocity: 100,
				})
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p + 12*5 + 5,
					StartBeat: base + float64(i)*0.4, DurationBeat: 0.15, Velocity: 100,
				})
			}
		case "bridge":
			// Rest — let other instruments shine.
		case "outro":
			// Hook recap — same as intro but softer.
			if bar%2 == 0 {
				for i, p := range hook {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p + 12*4,
						StartBeat: base + float64(i)*0.5, DurationBeat: 0.25, Velocity: 100,
					})
				}
			}
		}
	}
	return events
}

// GenerateLeadPunk creates punk lead: short 3-4 note licks, octave unison, mostly rests.
func GenerateLeadPunk(keyRoot string, totalBars int, energy float64) []schema.NoteEvent {
	scale := getScaleDegrees(keyRoot, "minor")
	if len(scale) == 0 {
		scale = []int{0, 2, 3, 5, 7, 8, 10}
	}
	var events []schema.NoteEvent

	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		sec := songSection(bar, totalBars)
		root := scale[0] + 12*3

		// Punk lead is MINIMAL — only plays in intro and sometimes chorus.
		switch sec {
		case "intro":
			// Short 4-note lick, octave unison.
			lick := []int{root, root + 3, root + 5, root + 3}
			for i, p := range lick {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base + float64(i)*0.5, DurationBeat: 0.12,
					Velocity: 110,
				})
				// Octave unison.
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p + 12,
					StartBeat: base + float64(i)*0.5, DurationBeat: 0.12,
					Velocity: 105,
				})
			}

		case "chorus", "chorus2":
			// One short lick at the start of each chorus phrase.
			if bar%4 == 0 {
				lick := []int{root, root + 5, root + 7}
				for i, p := range lick {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p,
						StartBeat: base + float64(i)*0.4, DurationBeat: 0.1,
						Velocity: 110,
					})
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p + 12,
						StartBeat: base + float64(i)*0.4, DurationBeat: 0.1,
						Velocity: 105,
					})
				}
			}

		// Verse, bridge, outro: lead rests. Rhythm guitar carries everything.
		}
	}

	fmt.Printf("[Lead-Punk] %d events, %d bars (octave licks, mostly rests)\n", len(events), totalBars)
	return events
}

// getBluesScale returns MIDI pitches for blues scale (1-b3-4-b5-5-b7).
func getBluesScale(root string) []int {
	rootSemi := map[string]int{
		"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5,
		"F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11,
	}
	rs := rootSemi[root]
	intervals := []int{0, 3, 5, 6, 7, 10} // blues: 1-b3-4-b5-5-b7
	result := make([]int, 0, len(intervals)*4)
	for oct := 3; oct <= 5; oct++ {
		base := (oct + 1) * 12
		for _, iv := range intervals {
			p := base + rs + iv
			if p >= 36 && p <= 96 {
				result = append(result, p)
			}
		}
	}
	return result
}

func getDiminishedDegrees(root string) []int {
	rootSemi := map[string]int{
		"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5,
		"F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11,
	}
	rs := rootSemi[root]
	intervals := []int{0, 3, 6, 9} // diminished 7th: stacked minor 3rds
	result := make([]int, 0, len(intervals)*3)
	for oct := 3; oct <= 5; oct++ {
		base := (oct + 1) * 12
		for _, iv := range intervals {
			p := base + rs + iv
			if p >= 36 && p <= 96 {
				result = append(result, p)
			}
		}
	}
	return result
}

// getHarmonicMinorDegrees returns MIDI pitches for harmonic minor (1-2-b3-4-5-b6-7).
func getHarmonicMinorDegrees(root string) []int {
	rootSemi := map[string]int{
		"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5,
		"F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11,
	}
	rs := rootSemi[root]
	intervals := []int{0, 2, 3, 5, 7, 8, 11} // harmonic minor: raised 7th
	result := make([]int, 0, len(intervals)*4)
	for oct := 3; oct <= 6; oct++ {
		base := (oct + 1) * 12
		for _, iv := range intervals {
			p := base + rs + iv
			if p >= 36 && p <= 108 {
				result = append(result, p)
			}
		}
	}
	return result
}

// shapeMelodyArch reshapes a melody to have a clear arch contour:
// low start → climb → peak at 60-70% → descend → land on tonic.
func shapeMelodyArch(events []schema.NoteEvent, totalBars int, tonicPitch int) []schema.NoteEvent {
	if len(events) < 8 {
		return events
	}
	// Find the pitch range available.
	minP, maxP := 127, 0
	for _, ev := range events {
		if ev.Pitch < minP {
			minP = ev.Pitch
		}
		if ev.Pitch > maxP {
			maxP = ev.Pitch
		}
	}
	rangeSize := maxP - minP
	if rangeSize < 5 {
		rangeSize = 12 // force at least an octave
	}

	// Compute arch curve: 0→1→0 shaped parabola peaking at 65%.
	n := len(events)
	peakPos := float64(n) * 0.65
	for i := range events {
		// Parabolic arch: y = 1 - ((x - peak)/peak)^2
		x := float64(i)
		normDist := (x - peakPos) / peakPos // -1 to +0.5
		archFactor := 1.0 - normDist*normDist
		if archFactor < 0 {
			archFactor = 0
		}
		if archFactor > 1 {
			archFactor = 1
		}

		// Map arch factor to pitch range: 0 = bottom, 1 = top.
		targetPitch := minP + int(archFactor*float64(rangeSize))
		// Smooth: blend 40% toward target (avoids destroying motif character).
		events[i].Pitch = events[i].Pitch + int(float64(targetPitch-events[i].Pitch)*0.4)

		// Clamp.
		if events[i].Pitch < 21 {
			events[i].Pitch = 21
		}
		if events[i].Pitch > 108 {
			events[i].Pitch = 108
		}
	}

	// Last note: pull toward tonic.
	if n > 0 {
		last := &events[n-1]
		last.Pitch = last.Pitch + (tonicPitch-last.Pitch)/2
	}

	return events
}

// getScaleDegrees returns MIDI-compatible scale pitches for a given key.
// GetScaleDegrees returns MIDI pitches for a given root and mode (major/minor/dorian/etc).
func GetScaleDegrees(root, mode string) []int { return getScaleDegrees(root, mode) }

func getScaleDegrees(root, mode string) []int {
	rootSemi := map[string]int{
		"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5,
		"F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11,
	}
	rs, ok := rootSemi[root]
	if !ok {
		rs = 0
	}

	var intervals []int
	switch mode {
	case "minor", "natural_minor":
		intervals = []int{0, 2, 3, 5, 7, 8, 10}
	case "dorian":
		intervals = []int{0, 2, 3, 5, 7, 9, 10} // minor with raised 6th
	case "phrygian":
		intervals = []int{0, 1, 3, 5, 7, 8, 10} // minor with flat 2nd
	case "lydian":
		intervals = []int{0, 2, 4, 6, 7, 9, 11} // major with raised 4th
	case "mixolydian":
		intervals = []int{0, 2, 4, 5, 7, 9, 10} // major with flat 7th
	default:
		intervals = []int{0, 2, 4, 5, 7, 9, 11}
	}

	noteNames := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

	result := make([]int, 0, len(intervals)*4)
	for oct := 3; oct <= 6; oct++ {
		base := (oct + 1) * 12
		for _, iv := range intervals {
			p := base + rs + iv
			if p >= 21 && p <= 108 {
				result = append(result, p)
			}
		}
	}
	_ = noteNames
	return result
}

func copySlice(s []int) []int {
	out := make([]int, len(s))
	copy(out, s)
	return out
}

