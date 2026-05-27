// Package composer — SongMemory: full-song composition logic.
// Records motif, rhythm, harmony across the entire piece.
// Enables cross-section references (theme regression, callback, variation).
package composer

import (
	"fmt"
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ComposerDNA defines a composer's musical personality.
type ComposerDNA struct {
	Name                string
	MotifObsession      float64 // 0-1: how much to repeat the main motif
	RepetitionTolerance float64 // 0-1: how much repetition is acceptable
	HarmonicAggression  float64 // 0-1: chord substitution aggressiveness
	SilenceTolerance    float64 // 0-1: how much silence/space to leave
	Chaos               float64 // 0-1: intentional "mistakes" probability
	SyncopationBias     float64 // 0-1: offbeat accent preference
	RegisterJumpBias    float64 // 0-1: how often to leap between registers
	AllowVoiceCrossing  bool    // allow bass > chords > lead violations
	AllowParallelFifth  bool    // allow parallel fifths/octaves
}

var ComposerArchetypes = map[string]ComposerDNA{
	"Toby Fox (Undertale)": {
		Name: "Toby Fox", MotifObsession: 0.90, RepetitionTolerance: 0.90,
		SilenceTolerance: 0.70, Chaos: 0.30, SyncopationBias: 0.40,
		RegisterJumpBias: 0.30, AllowVoiceCrossing: true, AllowParallelFifth: true,
	},
	"Nobuo Uematsu (FF)": {
		Name: "Nobuo Uematsu", MotifObsession: 0.70, RepetitionTolerance: 0.60,
		HarmonicAggression: 0.40, SyncopationBias: 0.30, RegisterJumpBias: 0.40,
		AllowVoiceCrossing: true,
	},
	"Hans Zimmer": {
		Name: "Hans Zimmer", MotifObsession: 0.85, RepetitionTolerance: 0.80,
		HarmonicAggression: 0.30, RegisterJumpBias: 0.60, Chaos: 0.20,
	},
	"JRPG Default": {
		Name: "JRPG Default", MotifObsession: 0.75, RepetitionTolerance: 0.70,
		HarmonicAggression: 0.30, SilenceTolerance: 0.40, SyncopationBias: 0.30,
		Chaos: 0.15,
	},
	"Retro Game": {
		Name: "Retro Game", MotifObsession: 0.85, RepetitionTolerance: 0.90,
		SilenceTolerance: 0.60, RegisterJumpBias: 0.50, SyncopationBias: 0.20,
	},
	"Hyperpop Maniac": {
		Name: "Hyperpop Maniac", Chaos: 0.90, SyncopationBias: 0.90,
		HarmonicAggression: 0.80, RegisterJumpBias: 0.80,
		AllowVoiceCrossing: true, AllowParallelFifth: true,
	},
	"Classical Purist": {
		Name: "Classical Purist", MotifObsession: 0.60, RepetitionTolerance: 0.50,
		HarmonicAggression: 0.20, SilenceTolerance: 0.30, Chaos: 0.05,
		AllowVoiceCrossing: false, AllowParallelFifth: false,
	},
	"Default": {
		Name: "Default", MotifObsession: 0.50, RepetitionTolerance: 0.50,
		HarmonicAggression: 0.20, SilenceTolerance: 0.30, Chaos: 0.10,
		SyncopationBias: 0.30, RegisterJumpBias: 0.30,
		AllowVoiceCrossing: false, AllowParallelFifth: false,
	},
}

// SongMemory is the full-song compositional memory.
// Every section can reference and develop material from previous sections.
type SongMemory struct {
	Sections         []SectionMemory
	MainMotif        Motif
	RhythmSignature  RhythmCell
	HarmonicProfile  []int   // pitch classes used most
	EnergyCurve      []float64
	MelodicInterval  []int   // preferred intervals
	CurrentSection   int
}

// SectionMemory stores what happened in one section.
type SectionMemory struct {
	Name        string
	StartBar    int
	MotifVariant int    // which variant of the main motif was used
	Energy      float64
	ActiveInsts []string
	Density     float64
}

// Motif is a short musical idea (3-5 notes).
type Motif struct {
	Pitches   []int    // absolute pitches
	PCs       []int    // pitch classes (0-11)
	Intervals []int    // interval sequence
	Rhythm    []float64 // relative durations
}

// RhythmCell is a rhythmic identity pattern.
type RhythmCell struct {
	Durations []float64
	Accents   []bool
}

// NewSongMemory creates empty song memory.
func NewSongMemory() *SongMemory {
	return &SongMemory{
		Sections:        make([]SectionMemory, 0),
		MelodicInterval: []int{1, 2, 3, 4, 5, 7}, // default permitted intervals
	}
}

// RecordSection stores what happened in a section.
func (sm *SongMemory) RecordSection(name string, startBar int, energy float64, activeInsts []string) {
	sm.Sections = append(sm.Sections, SectionMemory{
		Name:        name,
		StartBar:    startBar,
		Energy:      energy,
		ActiveInsts: activeInsts,
		MotifVariant: len(sm.Sections) % 4, // cycle through 4 variants
	})
}

// LearnMotif extracts the motif from the first phrase of lead melody.
func (sm *SongMemory) LearnMotif(events []schema.NoteEvent) {
	if len(events) < 3 {
		return
	}
	n := 5
	if len(events) < n {
		n = len(events)
	}
	m := Motif{}
	for i := 0; i < n; i++ {
		m.Pitches = append(m.Pitches, events[i].Pitch)
		m.PCs = append(m.PCs, events[i].Pitch%12)
		if i > 0 {
			m.Intervals = append(m.Intervals, events[i].Pitch-events[i-1].Pitch)
		}
		// Store beat-aligned positions as rhythm (normalized).
		m.Rhythm = append(m.Rhythm, events[i].DurationBeat)
	}
	sm.MainMotif = m
	fmt.Printf("[SongMemory] learned %d-note motif: %v\n", n, m.PCs)
}

// GetThemeVariant returns a development of the main motif based on section type.
// Variants: 0=original, 1=transpose+5, 2=invert, 3=retrograde
func (sm *SongMemory) GetThemeVariant(sectionName string, sectionBar int) []int {
	if len(sm.MainMotif.PCs) < 3 {
		return nil
	}

	// Map section to variant.
	var variant int
	switch sectionName {
	case "intro":
		variant = 0 // original
	case "verse":
		variant = 0 // original
	case "pre", "build":
		variant = 1 // transpose up
	case "chorus", "climax":
		variant = 2 // invert + transpose up
	case "bridge":
		variant = 3 // retrofit or fragment
	case "solo":
		variant = 1 // transpose up
	case "outro":
		variant = 0 // original, slower
	default:
		variant = sectionBar % 4
	}

	// Apply variant.
	motif := sm.MainMotif
	result := make([]int, len(motif.PCs))

	switch variant {
	case 0:
		copy(result, motif.Pitches)
	case 1:
		// Transpose up a fifth (7 semitones) or fourth (5 semitones).
		trans := 7
		if sectionBar%2 == 0 {
			trans = 5
		}
		for i, p := range motif.Pitches {
			result[i] = p + trans
			if result[i] > 96 {
				result[i] -= 12
			}
		}
	case 2:
		// Invert around the first note's octave.
		base := motif.Pitches[0]
		for i, p := range motif.Pitches {
			interval := p - base
			result[i] = base - interval
			if result[i] < 36 {
				result[i] += 12
			}
			if result[i] > 84 {
				result[i] -= 12
			}
		}
	case 3:
		// Retrograde: reverse order, keep pitches.
		for i := range motif.Pitches {
			result[i] = motif.Pitches[len(motif.Pitches)-1-i]
		}
	}

	fmt.Printf("[Theme] section %q: variant %d (%d notes)\n", sectionName, variant, len(result))
	return result
}

// RecordEnergy saves the energy curve for later reference.
func (sm *SongMemory) RecordEnergy(energies []float64) {
	sm.EnergyCurve = energies
}

// PickComposer selects a composer archetype based on keyword matching.
func PickComposer(styleName, styleDesc, mood string) ComposerDNA {
	keywords := map[string]string{
		"epic": "Hans Zimmer", "cinematic": "Hans Zimmer", "metal": "Hans Zimmer",
		"heavy": "Hans Zimmer", "fantasy": "Nobuo Uematsu (FF)", "rpg": "JRPG Default",
		"rock": "Toby Fox (Undertale)", "punk": "Retro Game",
		"retro": "Retro Game", "chiptune": "Retro Game", "8bit": "Retro Game",
		"hyper": "Hyperpop Maniac", "weird": "Hyperpop Maniac",
		"classical": "Classical Purist", "orchestral": "Classical Purist",
	}
	for kw, name := range keywords {
		if containsIgnoreCase(mood, kw) || containsIgnoreCase(styleDesc, kw) {
			if arch, ok := ComposerArchetypes[name]; ok {
				return arch
			}
		}
	}
	return ComposerArchetypes["Default"]
}

func containsIgnoreCase(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			ca := s[i+j]
			cb := substr[j]
			if ca >= 'A' && ca <= 'Z' {
				ca += 32
			}
			if cb >= 'A' && cb <= 'Z' {
				cb += 32
			}
			if ca != cb {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Roll returns true with probability p.
func Roll(rng *rand.Rand, p float64) bool {
	return rng.Float64() < p
}
