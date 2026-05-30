package composer

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

var globalSeed = time.Now().UnixNano()

// SetGlobalSeed sets the seed used by all generators. Call before generation.
func SetGlobalSeed(s int64) {
	if s == 0 {
		s = time.Now().UnixNano()
	}
	globalSeed = s
	rand.Seed(s)
}


type SectionBlock struct {
	Name      string
	StartBar  int
	EndBar    int
	Bars      int
	Energy    float64
	MotifMode string
}

type Timeline struct {
	Sections  []SectionBlock
	TotalBars int
}

func BuildTimeline(sectionDefs map[string]int, totalBars int) *Timeline {
	tl := &Timeline{TotalBars: totalBars}
	sectionOrder := []string{"intro", "verse", "chorus", "verse", "chorus", "bridge", "chorus", "outro"}
	motifModes := map[string]string{
		"intro": "sparse", "verse": "partial", "chorus": "full", "bridge": "invert", "outro": "sparse",
	}
	energyMap := map[string]float64{
		"intro": 0.2, "verse": 0.4, "chorus": 0.85, "bridge": 0.5, "outro": 0.15,
	}
	currentBar := 0
	for _, name := range sectionOrder {
		bars := sectionDefs[name]
		if bars <= 0 {
			continue
		}
		if currentBar+bars > totalBars {
			bars = totalBars - currentBar
		}
		if bars <= 0 {
			break
		}
		tl.Sections = append(tl.Sections, SectionBlock{
			Name: name, StartBar: currentBar, EndBar: currentBar + bars,
			Bars: bars, Energy: energyMap[name], MotifMode: motifModes[name],
		})
		currentBar += bars
	}
	return tl
}

func ApplyMotif(motif []int, mode string) []int {
	if len(motif) == 0 {
		return motif
	}
	switch mode {
	case "full":
		return copySlice(motif)
	case "partial":
		return motif[:len(motif)/2+1]
	case "variant":
		return Invert(motif)
	case "sparse":
		return []int{motif[0], motif[1]}
	case "invert":
		return Invert(motif)
	default:
		return motif
	}
}

func chordPitchesForChord(chord string, octave int) []int {
	rootSemi := map[string]int{"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5, "F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11}
	root := chord
	isMinor := false
	if len(chord) > 1 && chord[len(chord)-1] == 'm' {
		root = chord[:len(chord)-1]
		isMinor = true
	}
	rs, ok := rootSemi[root]
	if !ok {
		rs = 0
	}
	base := (octave + 1) * 12
	r := base + rs
	third := r + 4
	if isMinor {
		third = r + 3
	}
	fifth := r + 7
	return []int{r, third, fifth, r + 12, third + 12, fifth + 12}
}

// ─── Style-aware section lengths ────────────────────────────────

func sectionDefsForStyle(darkness, energy, rhythmic, tension float64) map[string]int {
	// Metal / aggressive: shorter intro, longer chorus
	if darkness > 0.7 && energy > 0.7 {
		return map[string]int{"intro": 1, "verse": 2, "chorus": 4, "bridge": 1, "outro": 1}
	}
	// Pop / ballad: balanced
	if energy > 0.4 && rhythmic < 0.5 {
		return map[string]int{"intro": 2, "verse": 4, "chorus": 4, "bridge": 2, "outro": 2}
	}
	// Lo-fi / ambient: shorter overall
	if energy < 0.4 {
		return map[string]int{"intro": 2, "verse": 2, "chorus": 2, "bridge": 1, "outro": 1}
	}
	// Hip-hop: loop-oriented
	if rhythmic > 0.5 {
		return map[string]int{"intro": 1, "verse": 4, "chorus": 4, "bridge": 1, "outro": 1}
	}
	return map[string]int{"intro": 2, "verse": 4, "chorus": 4, "bridge": 2, "outro": 2}
}

// ─── Style-aware drums ─────────────────────────────────────────

func GenerateDrums(timeline *Timeline, darkness, energy, rhythmic, tension float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	pattern := drumPattern(darkness, energy, rhythmic, tension)

	for _, sec := range timeline.Sections {
		for bar := sec.StartBar; bar < sec.EndBar; bar++ {
			base := float64(bar) * 4.0
			for step := 0; step < 16; step++ {
				val := pattern[step]
				if val == 0 {
					continue
				}
				pitch := 36
				if val >= 2 {
					pitch = 38
				} // snare
				if step%2 == 1 && val == 1 {
					pitch = 42
				} // hi-hat
				vel := 60 + int(sec.Energy*50) + int(tension*20)
				// Metal: higher velocity on kick
				if darkness > 0.7 && energy > 0.7 && step%2 == 0 {
					vel += 10
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: pitch,
					StartBeat: base + float64(step)*0.25, DurationBeat: 0.1, Velocity: vel,
				})
			}
		}
	}
	return events
}

func drumPattern(darkness, energy, rhythmic, tension float64) [16]int {
	var p [16]int
	switch {
	case darkness > 0.7 && energy > 0.7 && tension > 0.5:
		for i := 0; i < 16; i += 2 {
			p[i] = 1
		}
		p[4] = 2
		p[12] = 2
		for i := 0; i < 16; i++ {
			if p[i] == 0 && i%3 == 1 {
				p[i] = 1
			}
		}
	case energy > 0.5 && rhythmic < 0.5:
		p[0] = 1
		p[4] = 2
		p[8] = 1
		p[12] = 2
		for i := 1; i < 16; i += 2 {
			p[i] = 1
		}
	case rhythmic > 0.5 && energy > 0.3:
		p[0] = 1
		p[4] = 1
		p[8] = 2
		p[12] = 1
		for i := 0; i < 16; i++ {
			if i%2 == 1 || i%4 == 3 {
				p[i] = 1
			}
		}
	case energy < 0.4:
		p[0] = 1
		p[8] = 2
		p[3] = 1
		p[7] = 1
		p[11] = 1
		p[15] = 1
	default:
		p[0] = 1
		p[4] = 2
		p[8] = 1
		p[12] = 2
		for i := 1; i < 16; i += 2 {
			p[i] = 1
		}
	}
	return p
}

// ─── Style-aware bass ──────────────────────────────────────────

// GenerateBassStyled generates bass with style-specific patterns.
func GenerateBassStyled(style string, chords []string, totalBars int) []schema.NoteEvent {
	switch style {
	case "metal":
		return bassMetal(chords, totalBars)
	case "punk":
		return bassPunk(chords, totalBars)
	case "emo":
		return bassEmo(chords, totalBars)
	case "pop", "victory":
		return bassPop(chords, totalBars)
	case "trap":
		return bassTrap(chords, totalBars)
	case "ambient", "casual", "healing", "tension":
		return bassCasual(chords, totalBars)
	case "jazz", "funk", "rpg":
		return GenerateBassMidra(chords, totalBars)
	default:
		return GenerateBassMidra(chords, totalBars)
	}
}

// bassMetal: tight 8th-note root chugs, synced with double-kick.
func bassMetal(chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		base := float64(bar) * 4.0
		// 8th note root chugs, palm-muted feel (short duration).
		for step := 0; step < 8; step++ {
			beat := float64(step) * 0.5
			vel := 95 + rng.Intn(15)
			// Accent downbeats.
			if step%2 == 0 {
				vel += 10
			}
			pitch := root
			// Octave jump every 4th bar.
			if bar%4 == 0 {
				pitch += 12
			}
			if pitch < 28 {
				pitch = 28
			}
			if pitch > 60 {
				pitch = 60
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat: base + beat, DurationBeat: 0.15,
				Velocity: vel,
			})
		}
		// Quick 16th-note fill before bar end every 2nd bar.
		if bar%2 == 1 {
			for _, b := range []float64{3.25, 3.5, 3.75} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: root,
					StartBeat: base + b, DurationBeat: 0.08,
					Velocity: 90 + rng.Intn(15),
				})
			}
		}
	}
	fmt.Printf("[Bass-Metal] %d events, %d bars\n", len(events), totalBars)
	return events
}

// bassPunk: straight 8th-note root notes, simple and relentless. No fills, no slides.
func bassPunk(chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		base := float64(bar) * 4.0
		// Straight 8th notes on root — simple, driving, exactly what punk bass does.
		for step := 0; step < 8; step++ {
			beat := float64(step) * 0.5
			vel := 100
			if step == 0 || step == 4 {
				vel = 108 // slight accent on downbeats
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: root,
				StartBeat: base + beat, DurationBeat: 0.18,
				Velocity: vel + rng.Intn(8),
			})
		}
	}
	fmt.Printf("[Bass-Punk] %d events, %d bars\n", len(events), totalBars)
	return events
}

// bassPop: melodic, syncopated, locked with kick. MJ pocket.
// Verse = sparse groove, Chorus = busier melodic fills.
func bassPop(chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		isChorus := bar >= 8 && bar%8 < 4
		base := float64(bar) * 4.0

		// Melodic pattern using chord tones: root, 5th, octave, 3rd.
		tones := []int{root, root + 7, root + 12, root + 4}
		if strings.Contains(chord, "m") {
			tones[3] = root + 3 // minor 3rd
		}

		// Verse: sparse, locked to kicks on 1 and 3.
		pattern := []struct {
			beat    float64
			toneIdx int
			dur     float64
		}{
			{0.0, 0, 0.8},  // root on 1
			{1.5, 1, 0.4},  // 5th on 2&
			{2.0, 0, 0.8},  // root on 3
		}
		if isChorus {
			pattern = []struct {
				beat    float64
				toneIdx int
				dur     float64
			}{
				{0.0, 0, 0.5},  // root
				{0.5, 2, 0.3},  // octave up (melodic jump)
				{1.0, 1, 0.4},  // 5th
				{1.5, 3, 0.3},  // 3rd (color)
				{2.0, 0, 0.5},  // root
				{2.5, 2, 0.3},  // octave
				{3.0, 1, 0.4},  // 5th
				{3.5, 0, 0.3},  // root leading to next bar
			}
		}

		for _, p := range pattern {
			pitch := tones[p.toneIdx%len(tones)]
			if pitch < 28 {
				pitch += 12
			}
			if pitch > 60 {
				pitch -= 12
			}
			vel := 90 + rng.Intn(12)
			if isChorus {
				vel += 10
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat: base + p.beat, DurationBeat: p.dur,
				Velocity: vel,
			})
		}
	}
	fmt.Printf("[Bass-Pop] %d events, %d bars\n", len(events), totalBars)
	return events
}

// bassCasual: Simple bass for casual games. Root on beat 1, gentle.
func bassCasual(chords []string, totalBars int) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		base := float64(bar) * 4.0
		// Single root note on beat 1, held for 3 beats.
		if bar%2 == 0 {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: root,
				StartBeat: base, DurationBeat: 3.0,
				Velocity: 100,
			})
		}
	}
	fmt.Printf("[Bass-Casual] %d events, %d bars\n", len(events), totalBars)
	return events
}

// bassTrap: 808 bass. Sparse, sustained, with pitch slides between notes. Dre low-end.
func bassTrap(chords []string, totalBars int) []schema.NoteEvent {
	var events []schema.NoteEvent
	prevPitch := 0
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 1) // C1 = 24, sub-bass range
		base := float64(bar) * 4.0

		// 808: only plays on beat 1 (and sometimes 3). Long sustained.
		if bar%2 == 0 {
			// Slide up from previous note if different.
			if prevPitch > 0 && root != prevPitch {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: root,
					StartBeat: base - 0.03, DurationBeat: 0.03,
					Velocity: 80, // slide grace note
				})
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: root,
				StartBeat: base, DurationBeat: 1.8, // sustained 808 boom
				Velocity: 115,
			})
			prevPitch = root
		}
		if bar%4 == 0 {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: root,
				StartBeat: base + 2.0, DurationBeat: 1.5,
				Velocity: 105,
			})
		}
	}
	fmt.Printf("[Bass-Trap] %d events, %d bars\n", len(events), totalBars)
	return events
}

// bassEmo: sparse, long sustained root notes, melancholic.
func bassEmo(chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		base := float64(bar) * 4.0
		// Only play on beats 1 and 3, long sustained.
		for _, beat := range []float64{0.0, 2.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: root,
				StartBeat: base + beat, DurationBeat: 1.5,
				Velocity: 60 + rng.Intn(20),
			})
		}
		// Occasional octave-up on bar 4 for subtle lift.
		if bar%4 == 3 {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: root + 12,
				StartBeat: base + 3.0, DurationBeat: 0.8,
				Velocity: 50 + rng.Intn(15),
			})
		}
	}
	fmt.Printf("[Bass-Emo] %d events, %d bars\n", len(events), totalBars)
	return events
}

// GenerateBassMidra is a Go port of Midra's generate_bass().
// Follows chord roots with octave shifts, random durations and velocities.
func GenerateBassMidra(chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	motifLen := 8
	motif := make([]int, motifLen)
	for i := range motif {
		motif[i] = rng.Intn(7)
	}
	motif[0] = 0
	motif[motifLen-1] = []int{0, 2, 4}[rng.Intn(3)]

	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2) // C2 = 36
		base := float64(bar) * 4.0
		beatPositions := []float64{0.0, 1.0, 2.0, 3.0}

		for idx, beat := range beatPositions {
			m := motif[(bar*2+idx)%motifLen]
			octave := 0
			if m >= 3 && m <= 4 {
				octave = 12
			} else if m >= 5 {
				octave = 24
			}
			pitch := root + octave
			if pitch < 28 {
				pitch = 28
			}
			if pitch > 60 {
				pitch = 60
			}
			events = append(events, schema.NoteEvent{
				Type:         "note",
				Pitch:        pitch,
				StartBeat:    base + beat,
				DurationBeat: []float64{0.5, 0.75, 1.0}[rng.Intn(3)],
				Velocity:     82 + rng.Intn(23), // 82-104
			})
		}
	}
	fmt.Printf("[MidraBass] %d events, %d bars\n", len(events), totalBars)
	return events
}

// bassTransitionFill adds a short bass fill before the chorus (bars 6-7).
func bassTransitionFill(totalBars int, chords []string) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent
	// Fill at bar 7 (last bar before chorus), or bar totalBars-2.
	fillBar := totalBars - 9
	if fillBar < 0 {
		fillBar = 0
	}
	if totalBars >= 8 {
		fillBar = totalBars - 1 // last bar fill
	}
	chord := chords[fillBar%len(chords)]
	root := chordRootMIDI(chord, 2)
	base := float64(fillBar)*4.0 + 3.0 // last beat
	// Ascending root→octave slide.
	steps := []int{root, root + 4, root + 7, root + 12}
	for i, p := range steps {
		events = append(events, schema.NoteEvent{
			Type: "note", Pitch: p,
			StartBeat: base + float64(i)*0.25, DurationBeat: 0.12,
			Velocity: 90 + rng.Intn(15),
		})
	}
	return events
}

func chordRootMIDI(chord string, octave int) int {
	// Strip chord quality suffixes to extract root note.
	root := chord
	// Remove trailing digits and qualifiers.
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
	return 36 // fallback C2
}

// GenerateDrumsStyled generates drums with style-specific patterns.
// Falls back to GenerateDrumsMidra for styles without specific handling.
func GenerateDrumsStyled(style string, totalBars int, energy float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	switch style {
	case "metal":
		events = drumsMetal(totalBars, energy)
	case "punk":
		events = drumsPunk(totalBars, energy)
	case "emo":
		events = drumsEmo(totalBars, energy)
	case "rock":
		events = drumsRock(totalBars, energy)
	case "pop", "victory":
		events = drumsPop(totalBars, energy)
	case "rpg":
		events = drumsCasual(totalBars, energy)
	case "trap":
		events = drumsTrap(totalBars, energy)
	case "ambient", "casual", "healing":
		events = drumsCasual(totalBars, energy)
	case "jazz":
		events = drumsJazz(totalBars, energy)
	case "funk":
		events = drumsFunk(totalBars, energy)
	default:
		density := energy * 0.5
		if density < 0.15 {
			density = 0.15
		}
		events = GenerateDrumsMidra(totalBars, density)
	}
	// Apply groove variant: half-time bridge, shuffle verse2.
	return applyGrooveVariant(events, totalBars)
}

// applyGrooveVariant modifies events per section: half-time thins out all instruments.
func applyGrooveVariant(events []schema.NoteEvent, totalBars int) []schema.NoteEvent {
	for i := range events {
		bar := int(events[i].StartBeat) / 4
		v := drumVariant(bar, totalBars)
		if v == "half" {
			// Half-time: mute every other kick/snare, halve hi-hat density.
			if events[i].Pitch == 36 || events[i].Pitch == 38 {
				// Only keep hits on beats 1 and 3 (0.0, 2.0).
				beatInBar := events[i].StartBeat - float64(bar)*4.0
				if beatInBar != 0.0 && beatInBar != 2.0 {
					events[i].Velocity = 1 // effectively mute
				}
			}
			if events[i].Pitch == 42 || events[i].Pitch == 46 {
				beatInBar := events[i].StartBeat - float64(bar)*4.0
				// Halve hi-hat: keep only on whole beats.
				if int(beatInBar*2)%2 != 0 {
					events[i].Velocity = int(float64(events[i].Velocity) * 0.1)
				}
			}
		}
	}
	return events
}

// drumsMetal: Metal drums. Verse = double-kick + ride. Chorus = blast beats + china + open hat.
// Off-beat kick accents on the "and" of beats for syncopated brutality.
func drumsMetal(totalBars int, energy float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		isChorus := bar >= 8 && bar%8 < 4
		isFillBar := bar%8 == 7 || (bar%8 == 3 && bar >= 8)

		kickVel := 110
		snareVel := 115
		if isChorus {
			kickVel = 125
			snareVel = 127
		}

		// ── Kick: double-kick 16ths + off-beat syncopation ─────
		for step := 0; step < 16; step++ {
			beat := float64(step) * 0.25
			// Downbeats: always kick.
			if step%2 == 0 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 36, DrumName: "kick",
					StartBeat: base + beat, DurationBeat: 0.06,
					Velocity: kickVel,
				})
			}
			// Off-beat kick accents: syncopated "push" on selected upbeats.
			if isChorus && (step == 3 || step == 7 || step == 11 || step == 15) {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 36, DrumName: "kick",
					StartBeat: base + beat, DurationBeat: 0.04,
					Velocity: 115,
				})
			}
			// Double-kick fill on last beat every 2nd bar.
			if bar%2 == 1 && step >= 12 && step%2 == 1 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 36, DrumName: "kick",
					StartBeat: base + beat, DurationBeat: 0.04,
					Velocity: 105,
				})
			}
		}

		// ── Snare: backbeat + ghost drag ────────────────────────
		for _, beat := range []float64{1.0, 3.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat - 0.08, DurationBeat: 0.03,
				Velocity: 40,
			})
			if isChorus {
				// Chorus flam.
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 38, DrumName: "snare",
					StartBeat: base + beat - 0.02, DurationBeat: 0.04,
					Velocity: 95,
				})
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat, DurationBeat: 0.06,
				Velocity: snareVel,
			})
		}

		// ── Cymbal ───────────────────────────────────────────────
		if isChorus {
			// Chorus: china (52) on downbeats + open hat wash.
			for _, down := range []float64{0.0, 1.0, 2.0, 3.0} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 52, DrumName: "china",
					StartBeat: base + down, DurationBeat: 0.15,
					Velocity: 110,
				})
			}
			for step := 0; step < 8; step++ {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 46, DrumName: "open_hat",
					StartBeat: base + float64(step)*0.5, DurationBeat: 0.4,
					Velocity: 95,
				})
			}
			if bar%8 == 0 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 49, DrumName: "crash",
					StartBeat: base, DurationBeat: 0.8,
					Velocity: 127,
				})
			}
		} else if bar >= 4 {
			// Verse: ride bell (53) with accent.
			for step := 0; step < 8; step++ {
				vel := 80
				if step%2 == 0 {
					vel = 105
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 53, DrumName: "ride_bell",
					StartBeat: base + float64(step)*0.5, DurationBeat: 0.1,
					Velocity: vel,
				})
			}
		} else {
			// Intro: closed hat, tight.
			for step := 0; step < 8; step++ {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 42, DrumName: "closed_hat",
					StartBeat: base + float64(step)*0.5, DurationBeat: 0.04,
					Velocity: 100,
				})
			}
		}

		// ── Fill ──────────────────────────────────────────────────
		if isFillBar {
			// Tom run: descending.
			toms := []struct {
				pitch int
				beat  float64
			}{{48, 0.0}, {47, 0.15}, {45, 0.28}, {43, 0.38}, {41, 0.48}}
			for _, t := range toms {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: t.pitch, DrumName: "tom",
					StartBeat: base + 3.0 + t.beat, DurationBeat: 0.07,
					Velocity: 115,
				})
			}
		}

		// ── Pre-chorus silence ───────────────────────────────────
		if bar%8 == 0 && bar > 0 {
			for i := len(events) - 1; i >= 0; i-- {
				e := &events[i]
				if e.StartBeat >= base-0.6 && e.StartBeat < base && (e.Pitch == 42 || e.Pitch == 46 || e.Pitch == 53) {
					e.Velocity = 1
					break
				}
			}
		}
	}
	fmt.Printf("[Drums-Metal] %d events, %d bars\n", len(events), totalBars)
	return events
}

// drumVariant returns the groove for a bar: "straight", "half", or "shuffle".
func drumVariant(bar, totalBars int) string {
	sec := songSection(bar, totalBars)
	switch sec {
	case "bridge":
		return "half"
	case "verse":
		if bar%16 >= 8 && bar%16 < 16 {
			return "shuffle"
		}
	}
	return "straight"
}

// drumsPunk: Verse = D-beat, Bridge = half-time, Verse2 = shuffle.
func drumsPunk(totalBars int, energy float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		isChorus := bar >= 8 && bar%8 < 4
		isFillBar := bar%8 == 7 || (bar%8 == 3 && bar >= 8)
		_ = drumVariant(bar, totalBars)

		kickVel := 110
		snareVel := 115
		if isChorus {
			kickVel = 125
			snareVel = 127
		}

		// ── Kick: D-beat ────────────────────────────────────────
		kickBeats := []float64{0.0, 0.5, 1.5, 2.0, 2.5, 3.5}
		for _, beat := range kickBeats {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 36, DrumName: "kick",
				StartBeat: base + beat, DurationBeat: 0.06,
				Velocity: kickVel,
			})
		}

		// ── Snare: backbeat + ghost drag + chorus flam ──────────
		for _, beat := range []float64{1.0, 3.0} {
			// Ghost drag: two soft snare taps just before the backbeat.
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat - 0.12, DurationBeat: 0.03,
				Velocity: 35,
			})
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat - 0.06, DurationBeat: 0.03,
				Velocity: 45,
			})
			if isChorus {
				// Flam: grace note 0.02s before main hit — explosive crack.
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 38, DrumName: "snare",
					StartBeat: base + beat - 0.02, DurationBeat: 0.04,
					Velocity: 90,
				})
			}
			// Main backbeat.
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat, DurationBeat: 0.06,
				Velocity: snareVel,
			})
		}

		// ── Cymbal ───────────────────────────────────────────────
		if isChorus {
			// Chorus: crash on downbeats, bark on upbeats — "CRASH-ts-TSH-ts".
			for _, down := range []float64{0.0, 1.0, 2.0, 3.0} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 49, DrumName: "crash",
					StartBeat: base + down, DurationBeat: 0.12,
					Velocity: 105,
				})
			}
			for _, up := range []float64{0.5, 1.5, 2.5, 3.5} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 46, DrumName: "open_hat",
					StartBeat: base + up, DurationBeat: 0.15,
					Velocity: 95,
				})
			}
			// Big crash on chorus entry.
			if bar%8 == 0 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 49, DrumName: "crash",
					StartBeat: base, DurationBeat: 0.8,
					Velocity: 127,
				})
			}
		} else {
			// Verse: closed hat 8th notes — all hard, all the time.
			for step := 0; step < 8; step++ {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 42, DrumName: "closed_hat",
					StartBeat: base + float64(step)*0.5, DurationBeat: 0.04,
					Velocity: 100,
				})
			}
		}

		// ── Fill (accelerating into the next section) ────────────
		if isFillBar {
			fillType := rng.Intn(3)
			lastBeat := base + 3.0
			switch fillType {
			case 0:
				// Tom fill with acceleration: spacing tightens.
				toms := []struct {
					pitch int
					beat  float64
				}{{48, 0.0}, {47, 0.22}, {43, 0.40}, {38, 0.55}}
				for _, t := range toms {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: t.pitch, DrumName: "tom",
						StartBeat: lastBeat + t.beat, DurationBeat: 0.08,
						Velocity: 110,
					})
				}
			case 1:
				// Kick-snare 16th — accelerating spacing.
				spacing := []float64{0.0, 0.25, 0.45, 0.62}
				for step := 0; step < 4; step++ {
					pitch := 36
					if step%2 == 1 {
						pitch = 38
					}
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: pitch,
						StartBeat: lastBeat + spacing[step], DurationBeat: 0.06,
						Velocity: 110 + step*5,
					})
				}
			case 2:
				// Flam + accelerating snare roll.
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 38, DrumName: "snare",
					StartBeat: lastBeat - 0.02, DurationBeat: 0.04,
					Velocity: 80,
				})
				accelSpacing := []float64{0.0, 0.25, 0.45, 0.62}
				for i, b := range accelSpacing {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: 38, DrumName: "snare",
						StartBeat: lastBeat + b, DurationBeat: 0.06,
						Velocity: 100 + i*8,
					})
				}
			}
		}

		// ── Pre-crash silence: lift hi-hat before chorus crash ───
		if bar%8 == 0 && bar > 0 {
			// Remove the last hi-hat hit before the crash for impact.
			// We do this by finding and silencing the last closed/open hat before bar start.
			for i := len(events) - 1; i >= 0; i-- {
				e := &events[i]
				if e.StartBeat >= base-0.6 && e.StartBeat < base && (e.Pitch == 42 || e.Pitch == 46) {
					e.Velocity = 1 // effectively silence
					break
				}
			}
		}
	}
	fmt.Printf("[Drums-Punk] %d events, %d bars\n", len(events), totalBars)
	return events
}

// humanizeDrums adds ghost notes, fills, and subtle variations to make drums feel human.
func humanizeDrums(events []schema.NoteEvent, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	result := make([]schema.NoteEvent, len(events))
	copy(result, events)

	// 1. Ghost notes: soft snare (38) hits between backbeats, ~30% chance per bar.
	for bar := 0; bar < totalBars; bar++ {
		if rng.Float64() < 0.3 {
			continue
		}
		base := float64(bar) * 4.0
		// Ghost note positions: between beats 2-3 or 3-4.
		ghostPos := []float64{2.25, 2.5, 2.75, 3.25, 3.5, 3.75}
		pos := ghostPos[rng.Intn(len(ghostPos))]
		result = append(result, schema.NoteEvent{
			Type: "note", Pitch: 38, DrumName: "snare_ghost",
			StartBeat: base + pos, DurationBeat: 0.04,
			Velocity: 25 + rng.Intn(20), // very soft
		})
	}

	// 2. Fill before section transitions (every 4 or 8 bars).
	for bar := 0; bar < totalBars; bar++ {
		if bar < 3 {
			continue // no fill in first 3 bars
		}
		if bar == totalBars-1 || bar%4 == 3 {
			base := float64(bar)*4.0 + 3.0 // last beat of the bar
			fillType := rng.Intn(3)
			switch fillType {
			case 0: // 16th note snare roll
				for step := 0; step < 4; step++ {
					result = append(result, schema.NoteEvent{
						Type: "note", Pitch: 38, DrumName: "snare",
						StartBeat: base + float64(step)*0.25, DurationBeat: 0.06,
						Velocity: 70 + rng.Intn(30),
					})
				}
			case 1: // Tom fill (high→mid→low→floor)
				toms := []int{48, 47, 45, 43} // hi-tom, mid-tom, low-tom, floor
				for step := 0; step < 4; step++ {
					result = append(result, schema.NoteEvent{
						Type: "note", Pitch: toms[step], DrumName: "tom",
						StartBeat: base + float64(step)*0.25, DurationBeat: 0.08,
						Velocity: 80 + rng.Intn(20),
					})
				}
			case 2: // Kick + snare flam fill
				for _, beat := range []float64{0.0, 0.25, 0.5, 0.75} {
					result = append(result, schema.NoteEvent{
						Type: "note", Pitch: 36, DrumName: "kick",
						StartBeat: base + beat, DurationBeat: 0.06,
						Velocity: 90 + rng.Intn(20),
					})
				}
				result = append(result, schema.NoteEvent{
					Type: "note", Pitch: 38, DrumName: "snare",
					StartBeat: base + 0.5, DurationBeat: 0.08,
					Velocity: 100,
				})
			}
		}
	}

	// 3. Transition effect: reverse cymbal swell before bar 8.
	for bar := 0; bar < totalBars; bar++ {
		if bar%8 == 0 && bar > 0 {
			base := float64(bar)*4.0 - 2.0 // 2 beats before the transition
			// Reverse cymbal: rising pitch illusion via velocity ramp.
			for step := 0; step < 8; step++ {
				result = append(result, schema.NoteEvent{
					Type: "note", Pitch: 50, DrumName: "reverse_cymbal",
					StartBeat: base + float64(step)*0.25, DurationBeat: 0.22,
					Velocity: 30 + step*12, // velocity ramp up
				})
			}
		}
	}

	// 5. Micro-timing: shift ~15% of hi-hat/ride events by ±3-5ms for human feel.
	for i := range result {
		if result[i].Pitch == 42 || result[i].Pitch == 46 || result[i].Pitch == 51 {
			if rng.Float64() < 0.15 {
				shift := (rng.Float64() - 0.5) * 0.01
				result[i].StartBeat += shift
				if result[i].StartBeat < 0 {
					result[i].StartBeat = 0
				}
			}
		}
	}

	// 6. Wider dynamics: stretch velocity range to use pp(20)-ff(125).
	for i := range result {
		if result[i].Velocity < 20 {
			continue // ghost notes stay ghost
		}
		// Map clamped range to wider spread.
		orig := float64(result[i].Velocity)
		// Stretch 40-110 → 20-125
		stretched := 20.0 + (orig-40.0)*(105.0/70.0)
		result[i].Velocity = int(stretched)
		if result[i].Velocity < 4 {
			result[i].Velocity = 4
		}
		if result[i].Velocity > 127 {
			result[i].Velocity = 127
		}
	}

	return result
}

// drumsEmo: sparse, rim-click, occasional floor tom, minimal.
func drumsEmo(totalBars int, energy float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		// Sparse kick: only on beat 1 (and sometimes 3).
		events = append(events, schema.NoteEvent{
			Type: "note", Pitch: 36, DrumName: "kick",
			StartBeat: base, DurationBeat: 0.15,
			Velocity: 70 + rng.Intn(20),
		})
		if bar%2 == 0 {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 36, DrumName: "kick",
				StartBeat: base + 2.0, DurationBeat: 0.15,
				Velocity: 65 + rng.Intn(15),
			})
		}
		// Rim-click (37) instead of snare — softer, more textured.
		for _, beat := range []float64{1.0, 3.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 37, DrumName: "rim_click",
				StartBeat: base + beat, DurationBeat: 0.12,
				Velocity: 55 + rng.Intn(20),
			})
		}
		// Floor tom (43) fill every 4 bars.
		if bar%4 == 3 {
			for _, beat := range []float64{3.0, 3.25, 3.5, 3.75} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 43, DrumName: "floor_tom",
					StartBeat: base + beat, DurationBeat: 0.12,
					Velocity: 70 + rng.Intn(20),
				})
			}
		}
		// Very sparse hi-hat.
		for step := 0; step < 8; step++ {
			if step%4 == 0 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 42, DrumName: "closed_hat",
					StartBeat: base + float64(step)*0.5, DurationBeat: 0.15,
					Velocity: 45 + rng.Intn(20),
				})
			}
		}
	}
	events = humanizeDrums(events, totalBars)
	fmt.Printf("[Drums-Emo] %d events, %d bars\n", len(events), totalBars)
	return events
}

// drumsJazz: Swing drums. Ride cymbal timekeeping, hi-hat on 2 and 4, soft kick.
func drumsJazz(totalBars int, energy float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		// Soft kick on 1 and 3 (feathering).
		for _, beat := range []float64{0.0, 2.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 36, DrumName: "kick",
				StartBeat: base + beat, DurationBeat: 0.1, Velocity: 100,
			})
		}
		// Hi-hat on 2 and 4 (chick).
		for _, beat := range []float64{1.0, 3.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 42, DrumName: "closed_hat",
				StartBeat: base + beat, DurationBeat: 0.06, Velocity: 100,
			})
		}
		// Ride cymbal swing pattern.
		swing := []float64{0.0, 0.55, 1.0, 1.55, 2.0, 2.55, 3.0, 3.55}
		for _, beat := range swing {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 51, DrumName: "ride",
				StartBeat: base + beat, DurationBeat: 0.1, Velocity: 100,
			})
		}
	}
	fmt.Printf("[Drums-Jazz] %d events, %d bars\n", len(events), totalBars)
	return events
}

// drumsFunk: Syncopated funk drums. Tight snare, hi-hat 16ths, ghost notes.
func drumsFunk(totalBars int, energy float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		// Kick on 1 (and sometimes 2&).
		events = append(events, schema.NoteEvent{
			Type: "note", Pitch: 36, DrumName: "kick",
			StartBeat: base, DurationBeat: 0.08, Velocity: 100,
		})
		if bar%2 == 0 {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 36, DrumName: "kick",
				StartBeat: base + 1.5, DurationBeat: 0.08, Velocity: 100,
			})
		}
		// Tight snare on 2 and 4.
		for _, beat := range []float64{1.0, 3.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat, DurationBeat: 0.06, Velocity: 100,
			})
		}
		// Hi-hat 16th notes.
		for step := 0; step < 16; step++ {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 42, DrumName: "closed_hat",
				StartBeat: base + float64(step)*0.25, DurationBeat: 0.03, Velocity: 100,
			})
		}
		// Ghost notes on snare between backbeats.
		for _, g := range []float64{1.25, 1.75, 3.25, 3.75} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + g, DurationBeat: 0.03, Velocity: 100,
			})
		}
	}
	fmt.Printf("[Drums-Funk] %d events, %d bars\n", len(events), totalBars)
	return events
}

// drumsCasual: Minimal drums for casual games. Soft kick on 1, rim-click on 3, nothing else.
func drumsCasual(totalBars int, energy float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		// Soft kick on beat 1.
		events = append(events, schema.NoteEvent{
			Type: "note", Pitch: 36, DrumName: "kick",
			StartBeat: base, DurationBeat: 0.2,
			Velocity: 100,
		})
		// Soft rim-click on beat 3.
		if bar%2 == 0 {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 37, DrumName: "rim_click",
				StartBeat: base + 2.0, DurationBeat: 0.1,
				Velocity: 100,
			})
		}
	}
	fmt.Printf("[Drums-Casual] %d events, %d bars\n", len(events), totalBars)
	return events
}

// drumsTrap: Trap drums. Sparse hard kick, 16th hi-hats with rolls, clap on 2/4. Dre bounce.
func drumsTrap(totalBars int, energy float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		isFill := bar%8 == 7

		// ── Kick: sparse, hard-hitting. Only on 1 (and sometimes 3). ──
		events = append(events, schema.NoteEvent{
			Type: "note", Pitch: 36, DrumName: "kick",
			StartBeat: base, DurationBeat: 0.3, // long 808 kick
			Velocity: 120,
		})
		if bar%4 == 0 {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 36, DrumName: "kick",
				StartBeat: base + 2.0, DurationBeat: 0.3,
				Velocity: 115,
			})
		}

		// ── Clap/Snare: clap (39) on 2 and 4. ──
		for _, beat := range []float64{1.0, 3.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 39, DrumName: "clap",
				StartBeat: base + beat, DurationBeat: 0.1,
				Velocity: 105,
			})
		}

		// ── Hi-hat: 16th notes with occasional triplet rolls. ──
		for step := 0; step < 16; step++ {
			beat := float64(step) * 0.25
			vel := 70
			// Accent every 3rd hit for triplet feel.
			if step%3 == 0 {
				vel = 90
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 42, DrumName: "closed_hat",
				StartBeat: base + beat, DurationBeat: 0.03,
				Velocity: vel,
			})
		}
		// Hi-hat roll: 32nd note burst on last beat.
		if isFill {
			for step := 0; step < 8; step++ {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 42, DrumName: "closed_hat",
					StartBeat: base + 3.0 + float64(step)*0.125, DurationBeat: 0.02,
					Velocity: 80,
				})
			}
		}

		// ── Open hat punctuation on beat 4 every 2 bars. ──
		if bar%2 == 1 {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 46, DrumName: "open_hat",
				StartBeat: base + 3.0, DurationBeat: 0.3,
				Velocity: 85,
			})
		}
	}
	fmt.Printf("[Drums-Trap] %d events, %d bars\n", len(events), totalBars)
	return events
}

// drumsPop: Pop/R&B drums. Swing hi-hat, verse rim-click, chorus full snare. MJ groove.
func drumsPop(totalBars int, energy float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		isIntro := bar < 4
		isPre := bar >= 4 && bar < 8
		isChorus := bar >= 8 && bar%8 < 4
		isFillBar := bar%8 == 7 || (bar%8 == 3 && bar >= 8)

		// ── Kick: 1 and 3 (classic pop) ────────────────────────
		kickVel := 100
		if isChorus {
			kickVel = 115
		}
		for _, beat := range []float64{0.0, 2.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 36, DrumName: "kick",
				StartBeat: base + beat, DurationBeat: 0.1,
				Velocity: kickVel,
			})
		}
		// Kick push on "3&" in pre-chorus and chorus.
		if isPre || isChorus {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 36, DrumName: "kick",
				StartBeat: base + 3.5, DurationBeat: 0.06,
				Velocity: kickVel - 10,
			})
		}

		// ── Snare: verse = rim-click, chorus = full snare ──────
		snarePitch := 37 // rim-click
		snareVel := 75
		if isChorus {
			snarePitch = 38 // full snare
			snareVel = 115
		} else if isPre {
			snarePitch = 38
			snareVel = 95
		}
		for _, beat := range []float64{1.0, 3.0} {
			// Ghost drag before backbeat.
			if !isIntro {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 38, DrumName: "snare",
					StartBeat: base + beat - 0.06, DurationBeat: 0.03,
					Velocity: 30,
				})
			}
			if isChorus {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 38, DrumName: "snare",
					StartBeat: base + beat - 0.02, DurationBeat: 0.04,
					Velocity: 85,
				})
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: snarePitch, DrumName: "snare",
				StartBeat: base + beat, DurationBeat: 0.08,
				Velocity: snareVel,
			})
		}

		// ── Hi-hat: swing feel + breathing ─────────────────────
		// Swing: triplet-based timing (not strict 8ths).
		swingOffset := []float64{0.0, 0.55, 1.0, 1.55, 2.0, 2.55, 3.0, 3.55}
		if isChorus {
			// Chorus: open hat wash.
			for _, beat := range swingOffset {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 46, DrumName: "open_hat",
					StartBeat: base + beat, DurationBeat: 0.4,
					Velocity: 90,
				})
			}
			if bar%8 == 0 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 49, DrumName: "crash",
					StartBeat: base, DurationBeat: 0.7, Velocity: 120,
				})
			}
		} else if isPre {
			// Pre-chorus: half-open hat (slightly open, more sizzle).
			for _, beat := range swingOffset {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 46, DrumName: "open_hat",
					StartBeat: base + beat, DurationBeat: 0.15,
					Velocity: 70,
				})
			}
		} else {
			// Verse/intro: closed hat, swing accents.
			for i, beat := range swingOffset {
				vel := 65
				if i%2 == 0 {
					vel = 90
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 42, DrumName: "closed_hat",
					StartBeat: base + beat, DurationBeat: 0.04,
					Velocity: vel,
				})
			}
		}

		// ── Fill ────────────────────────────────────────────────
		if isFillBar {
			// Snare roll with acceleration.
			for i, b := range []float64{3.0, 3.2, 3.35, 3.5} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 38, DrumName: "snare",
					StartBeat: base + b, DurationBeat: 0.05,
					Velocity: 90 + i*10,
				})
			}
		}
	}
	fmt.Printf("[Drums-Pop] %d events, %d bars\n", len(events), totalBars)
	return events
}

// drumsRock: Rock drums. Verse = backbeat + ride. Chorus = open hat + power fills.
// Off-beat kick pushes on the "and" for driving momentum.
func drumsRock(totalBars int, energy float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		isChorus := bar >= 8 && bar%8 < 4
		isFillBar := bar%8 == 7 || (bar%8 == 3 && bar >= 8)

		kickVel := 105
		snareVel := 110
		if isChorus {
			kickVel = 120
			snareVel = 125
		}

		// ── Kick: backbeat + off-beat pushes ────────────────────
		// Standard rock kick on 1 and 3.
		for _, beat := range []float64{0.0, 2.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 36, DrumName: "kick",
				StartBeat: base + beat, DurationBeat: 0.08,
				Velocity: kickVel,
			})
		}
		// Off-beat kick pushes: "and" of 2 and 4 for drive.
		if isChorus || bar%2 == 0 {
			for _, push := range []float64{1.5, 3.5} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 36, DrumName: "kick",
					StartBeat: base + push, DurationBeat: 0.06,
					Velocity: kickVel - 10,
				})
			}
		}

		// ── Snare: backbeat + ghost drag ────────────────────────
		for _, beat := range []float64{1.0, 3.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat - 0.08, DurationBeat: 0.03,
				Velocity: 35,
			})
			if bar%8 == 7 && beat == 3.0 {
				// Big accent on last snare before chorus.
				snareVel = 127
			}
			if isChorus {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 38, DrumName: "snare",
					StartBeat: base + beat - 0.02, DurationBeat: 0.04,
					Velocity: 90,
				})
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat, DurationBeat: 0.06,
				Velocity: snareVel,
			})
		}

		// ── Cymbal ───────────────────────────────────────────────
		if isChorus {
			// Chorus: crash on downbeats + open hat wash.
			for _, down := range []float64{0.0, 2.0} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 49, DrumName: "crash",
					StartBeat: base + down, DurationBeat: 0.15,
					Velocity: 108,
				})
			}
			for step := 0; step < 8; step++ {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 46, DrumName: "open_hat",
					StartBeat: base + float64(step)*0.5, DurationBeat: 0.4,
					Velocity: 90,
				})
			}
			if bar%8 == 0 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 49, DrumName: "crash",
					StartBeat: base, DurationBeat: 0.7,
					Velocity: 127,
				})
			}
		} else if bar >= 4 {
			// Verse: ride cymbal (51) with groove.
			for step := 0; step < 8; step++ {
				vel := 75
				if step%2 == 0 {
					vel = 100
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 51, DrumName: "ride",
					StartBeat: base + float64(step)*0.5, DurationBeat: 0.1,
					Velocity: vel,
				})
			}
		} else {
			// Intro: closed hat.
			for step := 0; step < 8; step++ {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 42, DrumName: "closed_hat",
					StartBeat: base + float64(step)*0.5, DurationBeat: 0.04,
					Velocity: 100,
				})
			}
		}

		// ── Fill ──────────────────────────────────────────────────
		if isFillBar {
			fillType := rng.Intn(2)
			lastBeat := base + 3.0
			if fillType == 0 {
				// Tom fill: hi → mid → floor.
				toms := []struct {
					pitch int
					beat  float64
				}{{48, 0.0}, {47, 0.25}, {43, 0.5}}
				for _, t := range toms {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: t.pitch, DrumName: "tom",
						StartBeat: lastBeat + t.beat, DurationBeat: 0.1,
						Velocity: 110,
					})
				}
			} else {
				// Kick-snare 16th flurry.
				for step := 0; step < 4; step++ {
					pitch := 36
					if step%2 == 1 {
						pitch = 38
					}
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: pitch,
						StartBeat: lastBeat + float64(step)*0.25, DurationBeat: 0.06,
						Velocity: 110 + step*5,
					})
				}
			}
		}

		// ── Pre-chorus silence ───────────────────────────────────
		if bar%8 == 0 && bar > 0 {
			for i := len(events) - 1; i >= 0; i-- {
				e := &events[i]
				if e.StartBeat >= base-0.6 && e.StartBeat < base && (e.Pitch == 42 || e.Pitch == 46 || e.Pitch == 51) {
					e.Velocity = 1
					break
				}
			}
		}
	}
	fmt.Printf("[Drums-Rock] %d events, %d bars\n", len(events), totalBars)
	return events
}

// GenerateDrumsMidra is a Go port of Midra's generate_drums().
func GenerateDrumsMidra(totalBars int, densityFactor float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	motifLen := 8
	hatMotif := make([]int, motifLen)
	// densityFactor controls hi-hat activity (0=sparse, 1=dense)
	for i := range hatMotif {
		if rng.Float64() < densityFactor {
			hatMotif[i] = 1
		} else {
			hatMotif[i] = 0
		}
	}
	hatMotif[0] = 1
	hatMotif[motifLen-1] = rng.Intn(2)

	kickCandidates := []float64{0.0, 0.75, 1.5, 2.0, 2.75, 3.5}
	kickMotif := make([]float64, 2)
	kickMotif[0] = kickCandidates[rng.Intn(len(kickCandidates))]
	kickMotif[1] = kickCandidates[rng.Intn(len(kickCandidates))]

	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0

		// Kick on strong beats + random positions
		for _, beat := range sortedUniqueFloats(append([]float64{0.0, 2.0}, kickMotif...)) {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 36, DrumName: "kick",
				StartBeat: base + beat, DurationBeat: 0.1, Velocity: 98 + rng.Intn(19),
			})
		}

		// Snare on 2 and 4
		for _, beat := range []float64{1.0, 3.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat, DurationBeat: 0.1, Velocity: 94 + rng.Intn(17),
			})
		}

		// Hi-hat 8th notes with random skip
		for i := 0; i < 8; i++ {
			if hatMotif[i%motifLen] == 0 {
				continue
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 42, DrumName: "closed_hat",
				StartBeat: base + float64(i)*0.5, DurationBeat: 0.1, Velocity: 70 + rng.Intn(27),
			})
		}
	}
	fmt.Printf("[MidraDrums] %d events, %d bars\n", len(events), totalBars)
	return events
}

func sortedUniqueFloats(s []float64) []float64 {
	m := make(map[float64]bool)
	var result []float64
	for _, v := range s {
		if !m[v] {
			m[v] = true
			result = append(result, v)
		}
	}
	// Simple bubble sort for small slices
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// GenerateChordsStyled generates chords/pad with style-specific voicings and rhythms.
func GenerateChordsStyled(style string, chords []string, totalBars int, blockRatio float64) []schema.NoteEvent {
	switch style {
	case "metal":
		return chordsPower(chords, totalBars, "metal")
	case "punk":
		return chordsPower(chords, totalBars, "punk")
	case "rock":
		return chordsPower(chords, totalBars, "rock")
	case "ambient", "casual", "healing", "tension":
		return chordsCasual(chords, totalBars)
	case "pop", "victory":
		return chordsPop(chords, totalBars)
	case "trap":
		return chordsTrap(chords, totalBars)
	default:
		return GenerateChordsMidra(chords, totalBars, blockRatio)
	}
}

// chordsPower: root + fifth power chords (no third), style-specific rhythm patterns.
func chordsPower(chords []string, totalBars int, style string) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		rootOct := 2
		if style == "metal" {
			rootOct = 1 // C1-C2 downtuned metal
		}
		root := chordRootMIDI(chord, rootOct)
		base := float64(bar) * 4.0
		pitches := []int{root, root + 7}
		if bar%4 >= 2 {
			pitches = []int{root + 7, root + 12} // 1st inversion
		}

		// Pick rhythm pattern per style.
		type hit struct{ beat, dur, vel float64 }
		var hits []hit
		isChorus := bar >= 8

		switch style {
		case "punk":
			// Punk: straight 8th notes, ALL downstrokes, relentless. Zero rhythmic tricks.
			// The punk sound comes from consistency, not complexity.
			// Occasional palm-muted bars for dynamics.
			palmMute := bar%8 >= 4 && bar%8 < 6 // bars 4-5 of each 8-bar cycle: palm mute
			for step := 0; step < 8; step++ {
				beat := float64(step) * 0.5
				vel := 95.0
				dur := 0.10
				if palmMute {
					vel = 70.0  // choked, percussive
					dur = 0.05  // very short, "chk-chk-chk"
				}
				if step == 0 || step == 4 {
					vel += 10 // slight accent on downbeats
				}
				hits = append(hits, hit{beat, dur, vel})
			}
		case "metal":
			// Metal: tight 8th-note chug, C2 register, locked with double-kick.
			if bar%2 == 0 {
				// Standard: every 8th note, palm-muted tight.
				for step := 0; step < 8; step++ {
					vel := 95.0
					if step%2 == 0 {
						vel = 110 // accent with kick
					}
					hits = append(hits, hit{float64(step) * 0.5, 0.06, vel})
				}
			} else {
				// Gallop fill: triplet feel, 16th note triplets.
				for step := 0; step < 16; step++ {
					if step%3 != 0 {
						continue
					}
					hits = append(hits, hit{float64(step) * 0.25, 0.04, 100})
				}
			}
		default: // rock
			// Rock: 8th notes with occasional rest for breathing.
			for step := 0; step < 8; step++ {
				if bar%4 == 0 && step == 4 {
					continue // rest every 4 bars
				}
				vel := 85.0
				if step == 0 || step == 4 {
					vel = 100
				}
				hits = append(hits, hit{float64(step) * 0.5, 0.15, vel})
			}
		}

		// Boost chorus dynamics.
		if isChorus {
			for i := range hits {
				hits[i].vel += 10
			}
		}

		for _, h := range hits {
			for _, p := range pitches {
				if p < 28 || p > 72 {
					continue
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base + h.beat, DurationBeat: h.dur,
					Velocity: int(h.vel),
				})
			}
		}
	}
	fmt.Printf("[Chords-Power] %d events, %d bars (style=%s)\n", len(events), totalBars, style)
	return events
}

// chordsCasual: Soft, simple chords for casual games. Gentle, not muddy.
func chordsCasual(chords []string, totalBars int) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 3) // C3 range — higher, cleaner
		base := float64(bar) * 4.0
		// Simple triad: root + 3rd + 5th, held for 2 bars, gentle.
		isMinor := strings.Contains(chord, "m")
		third := root + 4
		if isMinor {
			third = root + 3
		}
		if bar%2 == 0 {
			for _, p := range []int{root, third, root + 7} {
				if p < 40 || p > 72 {
					continue
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base, DurationBeat: 7.5,
					Velocity: 100,
				})
			}
		}
	}
	fmt.Printf("[Chords-Casual] %d events, %d bars\n", len(events), totalBars)
	return events
}

// chordsTrap: Dark, minimal pads. Sparse hits with long decay. Dre atmosphere.
func chordsTrap(chords []string, totalBars int) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		base := float64(bar) * 4.0
		fifth := root + 7

		// Only play on beat 1 — sparse and dark.
		// Minor voicing: root + 5th + octave (dark, no 3rd for ambiguity).
		pitches := []int{root, fifth, root + 12}
		for _, p := range pitches {
			if p < 28 || p > 72 {
				continue
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: p,
				StartBeat: base, DurationBeat: 3.5,
				Velocity: 45, // low, atmospheric
			})
		}
	}
	fmt.Printf("[Chords-Trap] %d events, %d bars\n", len(events), totalBars)
	return events
}

// chordsPop: Rich pop/R&B voicings — 7ths, 9ths, spread across keyboard. John Legend lush.
func chordsPop(chords []string, totalBars int) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		base := float64(bar) * 4.0
		isChorus := bar >= 8 && bar%8 < 4

		// Rich voicing: root + 5th (left hand) + 7th + 9th + 3rd (right hand).
		// Maj7 voicing: root, 5th, 7th, 9th, 3rd (octave up).
		isMinor := strings.Contains(chord, "m")

		third := root + 4
		if isMinor {
			third = root + 3
		}
		seventh := root + 10 // dominant 7th
		ninth := root + 14   // 9th

		// Verse: sparse block chords on 1 and 3.
		pattern := []struct {
			beat   float64
			dur    float64
			vel    int
			pitches []int
		}{
			{0.0, 1.5, 55, []int{root, root + 7, seventh, root + 12}},                     // 1: root position
			{2.0, 1.5, 55, []int{root + 7, root + 12, seventh + 12, ninth}},               // 3: 2nd inversion feel
		}

		if isChorus {
			pattern = []struct {
				beat   float64
				dur    float64
				vel    int
				pitches []int
			}{
				{0.0, 1.8, 70, []int{root, root + 7, seventh, third + 12}},               // rich Maj7
				{2.0, 0.8, 65, []int{root + 7, root + 12, seventh + 12}},                  // lighter
				{3.0, 0.8, 60, []int{root + 12, seventh + 12, ninth}},                     // upper extension
			}
		}

		for _, p := range pattern {
			for _, pitch := range p.pitches {
				if pitch < 28 || pitch > 84 {
					continue
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: pitch,
					StartBeat: base + p.beat, DurationBeat: p.dur,
					Velocity: p.vel,
				})
			}
		}
	}
	fmt.Printf("[Chords-Pop] %d events, %d bars\n", len(events), totalBars)
	return events
}

// chordsAmbient: open voicings, slow evolving pad with wide intervals.
func chordsAmbient(chords []string, totalBars int) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		base := float64(bar) * 4.0
		// Open voicing: root + fifth + tenth + octave up (spread across 2+ octaves).
		pitches := []int{root, root + 7, root + 16, root + 24}
		for _, p := range pitches {
			if p < 28 || p > 72 {
				continue
			}
			// Long sustained notes, slow attack feel.
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: p,
				StartBeat: base, DurationBeat: 3.8,
				Velocity: 40 + (bar%4)*5,
			})
		}
	}
	fmt.Printf("[Chords-Ambient] %d events, %d bars\n", len(events), totalBars)
	return events
}

// GenerateChordsMidra is a Go port of Midra's generate_chords().
// Alternates between block and arpeggiated patterns, random durations, random velocities.
func GenerateChordsMidra(chords []string, totalBars int, blockRatio float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	motifLen := 8
	motif := make([]int, motifLen)
	for i := range motif {
		motif[i] = rng.Intn(7)
	}
	motif[0] = 0
	motif[motifLen-1] = []int{0, 2, 4}[rng.Intn(3)]

	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		baseOctave := 2
		if m := motif[bar%motifLen]; m >= 3 {
			baseOctave = 3
		}
		notes := chordPitchesForChord(chord, baseOctave)
		base := float64(bar) * 4.0
		patternType := "arp"
		if rng.Float64() < blockRatio {
			patternType = "block"
		}
		// Fallback to motif-based if blockRatio is zero
		if blockRatio <= 0 {
			patternType = "block"
			if m := motif[bar%motifLen]; !(m == 0 || m == 3 || m == 6) {
				patternType = "arp"
			}
		}

		for _, p := range notes {
			if patternType == "block" {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat:    base,
					DurationBeat: []float64{2.0, 3.0, 4.0}[rng.Intn(3)],
					Velocity:     54 + rng.Intn(25), // 54-78
				})
			} else {
				step := 0.0
				for step < 4.0 {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: p,
						StartBeat:    base + step,
						DurationBeat: []float64{0.25, 0.5, 0.75}[rng.Intn(3)],
						Velocity:     52 + rng.Intn(23), // 52-74
					})
					step += 1.0
				}
			}
		}
	}
	fmt.Printf("[MidraChords] %d events, %d bars\n", len(events), totalBars)
	return events
}

func ExpandMelody(phrases []Phrase, basePitch, bpm int, darkness, energy, rhythmic, tension float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	bar := 0
	isMetal := darkness > 0.7 && energy > 0.7
	isHipHop := rhythmic > 0.5 && energy > 0.3
	isPop := energy > 0.4 && rhythmic < 0.5

	for _, phrase := range phrases {
		for bi, notes := range phrase.Bars {
			if len(notes) == 0 {
				bar++
				continue
			}
			beatStart := float64(bar) * 4.0
			notesPerBar := len(notes)

			for i, rel := range notes {
				pitch := basePitch + rel
				if pitch < 21 {
					pitch = 21
				}
				if pitch > 108 {
					pitch = 108
				}

				var startBeat, dur float64
				var vel int

				switch {
				case isMetal:
					// 16th note gallop, aggressive
					step := 4.0 / float64(notesPerBar)
					startBeat = beatStart + float64(i)*step
					dur = step * 0.6
					vel = 90 + 10*(bi%2)
					// Accent first note of each bar
					if i == 0 {
						vel = 110
					}

				case isHipHop:
					// Syncopated, off-beat emphasis
					syncPoints := []float64{0, 0.5, 1.5, 2.0, 3.0, 3.5}
					if i < len(syncPoints) {
						startBeat = beatStart + syncPoints[i]
					} else {
						startBeat = beatStart + float64(i)*0.75
					}
					dur = 0.4
					vel = 70 + 15*(i%3)

				case isPop:
					// On-beat, flowing
					step := 4.0 / float64(notesPerBar)
					startBeat = beatStart + float64(i)*step
					dur = step * 0.85
					vel = 60 + 15*(bi%3) + 5*i
					if i == 0 {
						vel = 85
					} // emphasize downbeat

				default:
					step := 4.0 / float64(notesPerBar)
					startBeat = beatStart + float64(i)*step
					dur = step * 0.8
					vel = 70 + 10*(bi%3)
				}

				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: pitch,
					StartBeat: startBeat, DurationBeat: dur, Velocity: vel,
				})
			}
			bar++
		}
	}
	return events
}

// ─── Song Composer (final, no hardcoded values) ─────────────────

// ComposeSong is the legacy wrapper.
func ComposeSong(motif []int, chords []string, totalBars, basePitch, bpm int,
	rng *rand.Rand, darkness, energy, rhythmic, tension float64) map[string][]schema.NoteEvent {
	ctx := &GenerationContext{
		Motif: motif, Chords: chords, TotalBars: totalBars,
		BasePitch: basePitch, BPM: bpm, RNG: rng,
		Darkness: darkness, Energy: energy, Rhythmic: rhythmic, Tension: tension,
	}
	return ComposeSongWithContext(ctx)
}

// ComposeSongWithContext is the DNA-aware composition entry point.
func ComposeSongWithContext(ctx *GenerationContext) map[string][]schema.NoteEvent {
	evMap := make(map[string][]schema.NoteEvent)
	if len(ctx.Motif) < 2 {
		ctx.Motif = []int{0, 2, 4, 3, 0}
	}
	if len(ctx.Chords) == 0 {
		ctx.Chords = []string{"C", "G", "Am", "F"}
	}
	if ctx.RNG == nil {
		ctx.RNG = rand.New(rand.NewSource(42))
	}

	// Determine style label from feature vector.
	label := ctx.StyleLabel()
	fmt.Printf("[ComposeSongWithContext] %s, %d bars\n", label, ctx.TotalBars)

	evMap["drums"] = GenerateDrumsStyled(label, ctx.TotalBars, ctx.Energy)
	fmt.Printf("  Drums: %d events\n", len(evMap["drums"]))

	evMap["bass"] = GenerateBassStyled(label, ctx.Chords, ctx.TotalBars)
	fmt.Printf("  Bass: %d events\n", len(evMap["bass"]))

	evMap["chords"] = GenerateChordsStyled(label, ctx.Chords, ctx.TotalBars, 0.8)
	fmt.Printf("  Chords: %d events\n", len(evMap["chords"]))

	fmt.Printf("[ComposeSongWithContext] done: %d tracks\n", len(evMap))
	return evMap
}
