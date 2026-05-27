// Package composer — Song Composer.
// Integrates Motif + Harmony + Structure + Rhythm into a complete multi-track song.
// This is the final integration layer that turns "melody generator" into "music composer".
package composer

import (
	"fmt"
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ═══════════════════════════════════════════════════════════════════
// 1. Timeline Planner
// ═══════════════════════════════════════════════════════════════════

// SectionBlock defines one section in the song timeline.
type SectionBlock struct {
	Name      string
	StartBar  int
	EndBar    int
	Bars      int
	Energy    float64
	MotifMode string // "full", "partial", "variant", "sparse", "invert"
}

// Timeline is the complete song structure plan.
type Timeline struct {
	Sections []SectionBlock
	TotalBars int
}

// BuildTimeline creates a timeline from section definitions.
func BuildTimeline(sectionDefs map[string]int, totalBars int) *Timeline {
	tl := &Timeline{TotalBars: totalBars}

	sectionOrder := []string{"intro", "verse", "chorus", "verse", "chorus", "bridge", "chorus", "outro"}
	motifModes := map[string]string{
		"intro":  "sparse",
		"verse":  "partial",
		"chorus": "full",
		"bridge": "invert",
		"outro":  "sparse",
	}
	energyMap := map[string]float64{
		"intro":  0.2,
		"verse":  0.4,
		"chorus": 0.85,
		"bridge": 0.5,
		"outro":  0.15,
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
			Name:      name,
			StartBar:  currentBar,
			EndBar:    currentBar + bars,
			Bars:      bars,
			Energy:    energyMap[name],
			MotifMode: motifModes[name],
		})
		currentBar += bars
	}

	return tl
}

// ═══════════════════════════════════════════════════════════════════
// 2. Motif Allocation
// ═══════════════════════════════════════════════════════════════════

// ApplyMotif returns the motif variant for a section mode.
func ApplyMotif(motif []int, mode string) []int {
	if len(motif) == 0 {
		return motif
	}

	switch mode {
	case "full":
		return copySlice(motif)
	case "partial":
		if len(motif) <= 2 {
			return motif
		}
		return motif[:len(motif)/2+1]
	case "variant":
		return Invert(motif)
	case "sparse":
		if len(motif) <= 2 {
			return motif
		}
		// Take only first 2 notes, spaced out.
		return []int{motif[0], motif[1]}
	case "invert":
		return Invert(motif)
	default:
		return motif
	}
}

// ═══════════════════════════════════════════════════════════════════
// 3. Harmony Alignment
// ═══════════════════════════════════════════════════════════════════

// FitToChord snaps melody notes to the nearest chord tone.
func FitToChord(notes []int, chordPitches []int) []int {
	if len(chordPitches) == 0 {
		return notes
	}

	out := make([]int, len(notes))
	for i, n := range notes {
		nearest := chordPitches[0]
		bestDist := 999
		for _, cp := range chordPitches {
			d := n - cp
			if d < 0 {
				d = -d
			}
			if d < bestDist {
				bestDist = d
				nearest = cp
			}
		}
		out[i] = nearest
	}
	return out
}

// chordPitchesForChord returns MIDI pitches for a chord symbol at a given octave.
func chordPitchesForChord(chord string, octave int) []int {
	rootSemi := map[string]int{
		"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5,
		"F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11,
	}

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

// ═══════════════════════════════════════════════════════════════════
// 4. Track Generation
// ═══════════════════════════════════════════════════════════════════

// GenerateDrumsFromEnergy creates a style-aware kick-snare-hat pattern.
// Different styles get different rhythmic feels:
//   Metal: kick-heavy, double bass feel (steps 0,2,4,6,8,10,12,14)
//   Pop:   kick on 1&3 (0,8), snare on 2&4 (4,12)
//   HipHop: syncopated kick, snare on beat 3 (8)
//   LoFi:  simple, relaxed
func GenerateDrumsFromEnergy(timeline *Timeline, bpm int, rng *rand.Rand, darkness, energy, rhythmic, tension float64) []schema.NoteEvent {
	var events []schema.NoteEvent

	for _, sec := range timeline.Sections {
		pattern := drumPatternForStyle(darkness, energy, rhythmic, tension)
		for bar := sec.StartBar; bar < sec.EndBar; bar++ {
			base := float64(bar) * 4.0
			for step := 0; step < 16; step++ {
				if pattern[step] == 0 {
					continue
				}
				pitch := 36 // kick
				if step%4 == 2 || step%4 == 3 {
					pitch = 38 // snare on beats 2&4
				}
				if step%2 == 1 {
					pitch = 42 // hi-hat on offbeats
				}
				vel := 60 + int(sec.Energy*50)
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: pitch,
					StartBeat:    base + float64(step)*0.25,
					DurationBeat: 0.1,
					Velocity:     vel,
				})
			}
		}
	}
	return events
}

// drumPatternForStyle returns a 16-step drum pattern based on musical style.
// Style is determined by feature vector dimensions.
func drumPatternForStyle(darkness, energy, rhythmic, tension float64) [16]int {
	var p [16]int

	switch {
	case darkness > 0.7 && energy > 0.7 && tension > 0.5:
		// Metal / aggressive: double kick, snare on 2&4, heavy.
		// kick on all strong 8th notes (0,2,4,6,8,10,12,14)
		for i := 0; i < 16; i += 2 {
			p[i] = 1
		}
		// snare on 4, 12
		p[4] = 2
		p[12] = 2

	case energy > 0.5 && rhythmic < 0.5:
		// Pop / rock: kick on 1&3, snare on 2&4
		p[0] = 1  // kick beat 1
		p[4] = 2  // snare beat 2
		p[8] = 1  // kick beat 3
		p[12] = 2 // snare beat 4
		// hi-hat 8th notes (odd steps)
		for i := 1; i < 16; i += 2 {
			p[i] = 1
		}

	case energy > 0.3 && rhythmic > 0.5:
		// Hip-hop / trap: syncopated kick, snare on 3
		p[0] = 1  // kick
		p[4] = 1  // kick (syncopated)
		p[8] = 2  // snare on 3
		p[12] = 1 // kick
		// hi-hat rolls
		for i := 0; i < 16; i++ {
			if i%2 == 1 || i%4 == 3 {
				p[i] = 1
			}
		}

	case energy < 0.4:
		// Lo-fi / ambient: simple, sparse
		p[0] = 1  // kick downbeat
		p[8] = 2  // snare or clap on beat 3
		// gentle hi-hat on offbeats
		p[3] = 1
		p[7] = 1
		p[11] = 1
		p[15] = 1

	default:
		// Default: simple 4/4
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

// GenerateBassFromHarmony creates a bass line from the chord progression.
func GenerateBassFromHarmony(chordProg []string, timeline *Timeline) []schema.NoteEvent {
	var events []schema.NoteEvent

	for _, sec := range timeline.Sections {
		for bar := sec.StartBar; bar < sec.EndBar; bar++ {
			chord := chordProg[bar%len(chordProg)]
			base := float64(bar) * 4.0

			// Root on beat 1, fifth on beat 3.
			cp := chordPitchesForChord(chord, 2)
			if len(cp) >= 2 {
				// Root.
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: cp[0],
					StartBeat: base, DurationBeat: 2.0, Velocity: 85,
				})
				// Fifth or octave.
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: cp[0] + 7,
					StartBeat: base + 2.0, DurationBeat: 2.0, Velocity: 75,
				})
			}
		}
	}
	return events
}

// GeneratePadFromHarmony creates sustained pad chords.
func GeneratePadFromHarmony(chordProg []string, timeline *Timeline) []schema.NoteEvent {
	var events []schema.NoteEvent

	for _, sec := range timeline.Sections {
		energyFactor := 0.3 + sec.Energy*0.7
		for bar := sec.StartBar; bar < sec.EndBar; bar++ {
			chord := chordProg[bar%len(chordProg)]
			base := float64(bar) * 4.0

			cp := chordPitchesForChord(chord, 3)
			for _, p := range cp {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat:    base,
					DurationBeat: 4.0,
					Velocity:     30 + int(energyFactor*30),
				})
			}
		}
	}
	return events
}

// ═══════════════════════════════════════════════════════════════════
// 5. Song Composer — Full Pipeline
// ═══════════════════════════════════════════════════════════════════

// ComposeSong runs the full composition pipeline with style-aware drums.
// Input: motif, chords, feature vector dimensions for style-aware generation.
func ComposeSong(motif []int, chords []string, totalBars, basePitch, bpm int, rng *rand.Rand, darkness, energy, rhythmic, tension float64) map[string][]schema.NoteEvent {
	evMap := make(map[string][]schema.NoteEvent)

	if len(motif) < 2 {
		// Fallback motif.
		motif = []int{0, 2, 4, 3, 0}
	}

	// Step 1: Build timeline.
	sectionDefs := map[string]int{
		"intro":  2,
		"verse":  4,
		"chorus": 4,
		"bridge": 2,
		"outro":  2,
	}
	timeline := BuildTimeline(sectionDefs, totalBars)
	fmt.Printf("[SongComposer] timeline: %d sections, %d bars\n", len(timeline.Sections), timeline.TotalBars)

	// Step 2: Generate drums (style-aware).
	evMap["drums"] = GenerateDrumsFromEnergy(timeline, bpm, rng, darkness, energy, rhythmic, tension)
	fmt.Printf("[SongComposer] drums: %d events\n", len(evMap["drums"]))

	// Step 3: Generate bass.
	if len(chords) == 0 {
		chords = []string{"C", "G", "Am", "F"}
	}
	evMap["bass"] = GenerateBassFromHarmony(chords, timeline)
	fmt.Printf("[SongComposer] bass: %d events\n", len(evMap["bass"]))

	// Step 4: Generate lead melody from motif.
	if rng == nil {
		rng = rand.New(rand.NewSource(42))
	}
	
	plan := MotifPlan{
		UseRate:        0.7,
		VariationLevel: 0.4,
		CallResponse:   true,
		OctaveStrategy: "chorus_up",
		BarsPerPhrase:  4,
		TotalBars:      totalBars,
	}

	var allPhrases []Phrase
	for _, sec := range timeline.Sections {
		motifVar := ApplyMotif(motif, sec.MotifMode)
		phrases := BuildSection(motifVar, sec.Name, sec.Bars, plan, rng)
		allPhrases = append(allPhrases, phrases...)
	}

	leadEvents := ExpandMelody(allPhrases, basePitch, bpm)
	evMap["lead"] = leadEvents
	fmt.Printf("[SongComposer] lead: %d notes from %d-note motif\n", len(leadEvents), len(motif))

	// Step 5: Generate pad.
	evMap["pad"] = GeneratePadFromHarmony(chords, timeline)
	fmt.Printf("[SongComposer] pad: %d events\n", len(evMap["pad"]))

	fmt.Printf("[SongComposer] done: %d tracks, %d total events\n", len(evMap),
		len(evMap["drums"])+len(evMap["bass"])+len(evMap["lead"])+len(evMap["pad"]))

	return evMap
}
