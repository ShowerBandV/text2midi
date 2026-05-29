package composer

import (
	"fmt"
	"math/rand"

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

// GenerateBassStyled generates bass with style-specific patterns.
func GenerateBassStyled(style string, chords []string, totalBars int) []schema.NoteEvent {
	switch style {
	case "metal":
		return bassMetal(chords, totalBars)
	case "punk":
		return bassPunk(chords, totalBars)
	case "emo":
		return bassEmo(chords, totalBars)
	default:
		return GenerateBassMidra(chords, totalBars)
	}
}

// bassMetal: tight 8th-note root chugs, synced with double-kick.
func bassMetal(chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(42))
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

// bassPunk: driving 8th-note root notes, simple and relentless.
func bassPunk(chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(42))
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		base := float64(bar) * 4.0
		// 8th notes on root, punk style (fast, driving, simple).
		for step := 0; step < 8; step++ {
			beat := float64(step) * 0.5
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: root,
				StartBeat: base + beat, DurationBeat: 0.2,
				Velocity: 100 + rng.Intn(15),
			})
		}
	}
	fmt.Printf("[Bass-Punk] %d events, %d bars\n", len(events), totalBars)
	return events
}

// bassEmo: sparse, long sustained root notes, melancholic.
func bassEmo(chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(42))
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

// GenerateDrumsStyled generates drums with style-specific patterns.
// Falls back to GenerateDrumsMidra for styles without specific handling.
func GenerateDrumsStyled(style string, totalBars int, energy float64) []schema.NoteEvent {
	switch style {
	case "metal":
		return drumsMetal(totalBars, energy)
	case "punk":
		return drumsPunk(totalBars, energy)
	case "emo":
		return drumsEmo(totalBars, energy)
	case "rock":
		return drumsRock(totalBars, energy)
	default:
		density := energy * 0.5
		if density < 0.15 {
			density = 0.15
		}
		return GenerateDrumsMidra(totalBars, density)
	}
}

// drumsMetal: double-kick 16th notes, china cymbal accents, aggressive.
func drumsMetal(totalBars int, energy float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(42))
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		// Double-kick: 16th note pattern (eighth notes on beats, 16th fills).
		for step := 0; step < 16; step++ {
			beat := float64(step) * 0.25
			// Kick on every 8th note (steps 0,2,4,6,8,10,12,14).
			if step%2 == 0 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 36, DrumName: "kick",
					StartBeat: base + beat, DurationBeat: 0.08,
					Velocity: 100 + int(energy*25),
				})
			}
			// Extra double-kick fills on last beat of every 2nd bar.
			if bar%2 == 1 && step >= 12 && step%2 == 1 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 36, DrumName: "kick",
					StartBeat: base + beat, DurationBeat: 0.06,
					Velocity: 95 + int(energy*20),
				})
			}
		}
		// Snare on 2 and 4 (heavy).
		for _, beat := range []float64{1.0, 3.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat, DurationBeat: 0.1,
				Velocity: 105 + rng.Intn(15),
			})
		}
		// Ride cymbal (51) or china (52) instead of hi-hat.
		for step := 0; step < 8; step++ {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 51, DrumName: "ride",
				StartBeat: base + float64(step)*0.5, DurationBeat: 0.1,
				Velocity: 75 + rng.Intn(20),
			})
		}
		// Crash on bar start.
		if bar%4 == 0 {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 49, DrumName: "crash",
				StartBeat: base, DurationBeat: 0.3,
				Velocity: 115,
			})
		}
	}
	fmt.Printf("[Drums-Metal] %d events, %d bars\n", len(events), totalBars)
	return events
}

// drumsPunk: fast d-beat / skate punk pattern.
func drumsPunk(totalBars int, energy float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(42))
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		// D-beat: kick on 1 + 3&, snare on 2 + 4.
		kickBeats := []float64{0.0, 0.5, 1.5, 2.0, 2.5, 3.5}
		for _, beat := range kickBeats {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 36, DrumName: "kick",
				StartBeat: base + beat, DurationBeat: 0.08,
				Velocity: 100 + rng.Intn(15),
			})
		}
		for _, beat := range []float64{1.0, 3.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat, DurationBeat: 0.08,
				Velocity: 105 + rng.Intn(15),
			})
		}
		// Fast 8th note hi-hat.
		for step := 0; step < 8; step++ {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 42, DrumName: "closed_hat",
				StartBeat: base + float64(step)*0.5, DurationBeat: 0.06,
				Velocity: 80 + rng.Intn(20),
			})
		}
	}
	fmt.Printf("[Drums-Punk] %d events, %d bars\n", len(events), totalBars)
	return events
}

// drumsEmo: sparse, rim-click, occasional floor tom, minimal.
func drumsEmo(totalBars int, energy float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(42))
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
	fmt.Printf("[Drums-Emo] %d events, %d bars\n", len(events), totalBars)
	return events
}

// drumsRock: solid backbeat with fills.
func drumsRock(totalBars int, energy float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(42))
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		// Kick on 1 and 3 (and syncopation).
		for _, beat := range []float64{0.0, 2.0} {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 36, DrumName: "kick",
				StartBeat: base + beat, DurationBeat: 0.1,
				Velocity: 95 + rng.Intn(20),
			})
		}
		// Snare on 2 and 4.
		for _, beat := range []float64{1.0, 3.0} {
			vel := 100 + rng.Intn(15)
			// Accent on beat 4 every 4th bar.
			if bar%4 == 3 && beat == 3.0 {
				vel = 120
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 38, DrumName: "snare",
				StartBeat: base + beat, DurationBeat: 0.1,
				Velocity: vel,
			})
		}
		// 8th note hi-hat with accents.
		for step := 0; step < 8; step++ {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 42, DrumName: "closed_hat",
				StartBeat: base + float64(step)*0.5, DurationBeat: 0.1,
				Velocity: 75 + rng.Intn(25),
			})
		}
		// Crash + floor tom fill every 8 bars.
		if bar%8 == 7 {
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: 49, DrumName: "crash",
				StartBeat: base, DurationBeat: 0.4, Velocity: 110,
			})
			for _, beat := range []float64{3.0, 3.25, 3.5, 3.75} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 43 + rng.Intn(3), DrumName: "tom",
					StartBeat: base + beat, DurationBeat: 0.1,
					Velocity: 90 + rng.Intn(20),
				})
			}
		}
	}
	fmt.Printf("[Drums-Rock] %d events, %d bars\n", len(events), totalBars)
	return events
}

// GenerateDrumsMidra is a Go port of Midra's generate_drums().
func GenerateDrumsMidra(totalBars int, densityFactor float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(42))
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

// GenerateChordsStyled generates chords/pad with style-specific voicings.
func GenerateChordsStyled(style string, chords []string, totalBars int, blockRatio float64) []schema.NoteEvent {
	switch style {
	case "metal", "punk", "rock":
		return chordsPower(chords, totalBars) // power chords (root + fifth, no third)
	case "ambient":
		return chordsAmbient(chords, totalBars) // open voicings, slow attack
	default:
		return GenerateChordsMidra(chords, totalBars, blockRatio)
	}
}

// chordsPower: root + fifth power chords (no third), perfect for distorted guitar styles.
func chordsPower(chords []string, totalBars int) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 3) // C3 range for guitar
		base := float64(bar) * 4.0
		// Power chord: root + fifth at two octaves.
		pitches := []int{root, root + 7, root + 12, root + 19}
		for _, p := range pitches {
			if p < 36 || p > 84 {
				continue
			}
			// Strummed power chord: short attack on downbeats.
			for _, beat := range []float64{0.0, 2.0} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: base + beat, DurationBeat: 1.5,
					Velocity: 80 + (bar%3)*5,
				})
			}
		}
	}
	fmt.Printf("[Chords-Power] %d events, %d bars\n", len(events), totalBars)
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

