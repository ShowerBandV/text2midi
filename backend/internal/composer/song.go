package composer

import (
	"fmt"
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/generator"
	"github.com/ShowerBandV/text2midi/internal/schema"
)

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

// GenerateBassMidra is a Go port of Midra's generate_bass().
// Follows chord roots with octave shifts, random durations and velocities.
func GenerateBassMidra(chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(42))
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

func chordRootMIDI(chord string, octave int) int {
	root := chord
	if len(chord) > 1 && chord[len(chord)-1] == 'm' {
		root = chord[:len(chord)-1]
	}
	if len(chord) > 1 && chord[len(chord)-1] == '7' {
		root = chord[:len(chord)-1]
	}
	if root == root {
		if len(root) > 1 && root[len(root)-1] == 'm' {
			root = root[:len(root)-1]
		}
		if len(root) > 1 && (root[len(root)-1] == '7' || root[len(root)-1] == '5') {
			root = root[:len(root)-1]
		}
		rootSemi := map[string]int{
			"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5,
			"F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11,
		}
		if rs := rootSemi[root]; rs >= 0 || root == "" {
			return (octave+1)*12 + rootSemi[root]
		}
	}
	return 36 // fallback C2
}

// GenerateDrumsMidra is a Go port of Midra's generate_drums().
func GenerateDrumsMidra(totalBars int, densityFactor float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(42))
	motifLen := 8
	hatMotif := make([]int, motifLen)
	for i := range hatMotif {
		hatMotif[i] = rng.Intn(2)
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

// GenerateChordsMidra is a Go port of Midra's generate_chords().
// Alternates between block and arpeggiated patterns, random durations, random velocities.
func GenerateChordsMidra(chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(42))
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
		patternType := "block"
		if m := motif[bar%motifLen]; !(m == 0 || m == 3 || m == 6) {
			patternType = "arp"
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

// Original GenerateBass kept for compatibility.
func GenerateBass(chords []string, timeline *Timeline, darkness, energy, rhythmic, tension float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	isMetal := darkness > 0.7 && energy > 0.7
	isHipHop := rhythmic > 0.5 && energy > 0.3
	isPop := energy > 0.4 && rhythmic < 0.5

	for _, sec := range timeline.Sections {
		octave := 2
		if isHipHop {
			octave = 1
		} // 808 sub-bass
		if isMetal {
			octave = 2
		} // low but punchy

		for bar := sec.StartBar; bar < sec.EndBar; bar++ {
			chord := chords[bar%len(chords)]
			base := float64(bar) * 4.0
			cp := chordPitchesForChord(chord, octave)
			if len(cp) < 1 {
				continue
			}
			root := cp[0]

			switch {
			case isMetal:
				// Octave gallop: root - octave - root - fifth
				events = append(events, schema.NoteEvent{Type: "note", Pitch: root, StartBeat: base, DurationBeat: 0.25, Velocity: 100})
				events = append(events, schema.NoteEvent{Type: "note", Pitch: root + 12, StartBeat: base + 0.25, DurationBeat: 0.25, Velocity: 90})
				events = append(events, schema.NoteEvent{Type: "note", Pitch: root, StartBeat: base + 0.75, DurationBeat: 0.25, Velocity: 95})
				events = append(events, schema.NoteEvent{Type: "note", Pitch: root + 7, StartBeat: base + 2.0, DurationBeat: 0.5, Velocity: 85})

			case isHipHop:
				// 808 sliding bass: long root + slide to fifth
				events = append(events, schema.NoteEvent{Type: "note", Pitch: root, StartBeat: base, DurationBeat: 1.5, Velocity: 95})
				events = append(events, schema.NoteEvent{Type: "note", Pitch: root + 7, StartBeat: base + 1.5, DurationBeat: 0.5, Velocity: 85})
				events = append(events, schema.NoteEvent{Type: "note", Pitch: root - 12, StartBeat: base + 2.5, DurationBeat: 1.5, Velocity: 80})

			case isPop:
				// Walking bass: root - third - fifth - root
				third := root + 4
				fifth := root + 7
				events = append(events, schema.NoteEvent{Type: "note", Pitch: root, StartBeat: base, DurationBeat: 1.0, Velocity: 80})
				events = append(events, schema.NoteEvent{Type: "note", Pitch: third, StartBeat: base + 1.0, DurationBeat: 1.0, Velocity: 70})
				events = append(events, schema.NoteEvent{Type: "note", Pitch: fifth, StartBeat: base + 2.0, DurationBeat: 1.0, Velocity: 75})
				events = append(events, schema.NoteEvent{Type: "note", Pitch: root + 12, StartBeat: base + 3.0, DurationBeat: 1.0, Velocity: 70})

			default:
				// Root on 1, fifth on 3
				events = append(events, schema.NoteEvent{Type: "note", Pitch: root, StartBeat: base, DurationBeat: 2.0, Velocity: 85})
				events = append(events, schema.NoteEvent{Type: "note", Pitch: root + 7, StartBeat: base + 2.0, DurationBeat: 2.0, Velocity: 75})
			}
		}
	}
	return events
}

// ─── Style-aware pad ───────────────────────────────────────────

func GeneratePad(chords []string, timeline *Timeline, darkness, energy, rhythmic, tension float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	isMetal := darkness > 0.7 && energy > 0.7
	isSlow := energy < 0.4 && rhythmic < 0.4
	isPop := energy > 0.4 && rhythmic < 0.5

	if isMetal {
		return events
	}

	for _, sec := range timeline.Sections {
		if sec.Energy < 0.2 && !isPop {
			continue
		}
		energyFactor := 0.3 + sec.Energy*0.7

		for bar := sec.StartBar; bar < sec.EndBar; bar++ {
			chord := chords[bar%len(chords)]
			base := float64(bar) * 4.0
			cp := chordPitchesForChord(chord, 3)

			if isSlow && len(cp) >= 3 {
				// Ballad: arpeggiated broken chords (彩虹-like)
				pat := []int{cp[0], cp[1], cp[2], cp[1], cp[0] + 12, cp[2], cp[1], cp[0]}
				for i, p := range pat {
					events = append(events, schema.NoteEvent{Type: "note", Pitch: p,
						StartBeat: base + float64(i)*0.5, DurationBeat: 0.4, Velocity: 35 + int(energyFactor*25)})
				}
			} else if isPop && len(cp) >= 3 {
				events = append(events, schema.NoteEvent{Type: "note", Pitch: cp[0], StartBeat: base, DurationBeat: 4.0, Velocity: 25 + int(energyFactor*25)})
				events = append(events, schema.NoteEvent{Type: "note", Pitch: cp[2], StartBeat: base, DurationBeat: 4.0, Velocity: 25 + int(energyFactor*25)})
				events = append(events, schema.NoteEvent{Type: "note", Pitch: cp[0] + 12, StartBeat: base, DurationBeat: 4.0, Velocity: 20 + int(energyFactor*20)})
			} else {
				for _, p := range cp {
					events = append(events, schema.NoteEvent{Type: "note", Pitch: p,
						StartBeat: base, DurationBeat: 4.0, Velocity: 30 + int(energyFactor*30)})
				}
			}
		}
	}
	return events
}

// ─── Style-aware ExpandMelody ───────────────────────────────────

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

// ComposeSong is the legacy wrapper. Use ComposeSongWithContext for new code.
func ComposeSong(motif []int, chords []string, totalBars, basePitch, bpm int,
	rng *rand.Rand, darkness, energy, rhythmic, tension float64) map[string][]schema.NoteEvent {
	ctx := &GenerationContext{
		Motif: motif, Chords: chords, TotalBars: totalBars,
		BasePitch: basePitch, BPM: bpm, RNG: rng,
		Darkness: darkness, Energy: energy, Rhythmic: rhythmic, Tension: tension,
		DNA: DefaultDNA(),
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
	if ctx.DNA == nil {
		ctx.DNA = DefaultDNA()
	}

	// Step 1: Style-aware section lengths
	sd := sectionDefsForStyle(ctx.Darkness, ctx.Energy, ctx.Rhythmic, ctx.Tension)
	timeline := BuildTimeline(sd, ctx.TotalBars)
	label := ctx.StyleLabel()
	fmt.Printf("[SongComposer] %s, %d sections, %d bars (DNA=%s)\n",
		label, len(timeline.Sections), ctx.TotalBars, ctx.DNA.Name)

	// Step 2: Bass (DNA-aware)
	evMap["bass"] = GenerateBass(ctx.Chords, timeline, ctx.Darkness, ctx.Energy, ctx.Rhythmic, ctx.Tension)
	fmt.Printf("[Bass] %d events\n", len(evMap["bass"]))

	// Step 3: Drums (DNA-aware)
	evMap["drums"] = GenerateDrums(timeline, ctx.Darkness, ctx.Energy, ctx.Rhythmic, ctx.Tension)
	fmt.Printf("[Drums] %d events\n", len(evMap["drums"]))

	// Step 4: Lead melody is generated by LLM in main.go
	evMap["lead"] = nil
	// Step 5: Pad (DNA-aware)
	evMap["pad"] = GeneratePad(ctx.Chords, timeline, ctx.Darkness, ctx.Energy, ctx.Rhythmic, ctx.Tension)
	fmt.Printf("[Pad] %d events\n", len(evMap["pad"]))

	// Step 6: FX / special effects (energy-driven)
	evMap["fx"] = generator.GenerateFX(ctx.TotalBars, ctx.BPM, ctx.Energy, ctx.Tension)
	fmt.Printf("[FX] %d events\n", len(evMap["fx"]))

	fmt.Printf("[SongComposer] done: %d tracks\n", len(evMap))
	return evMap
}
