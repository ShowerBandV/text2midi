// Package composer — Personality engine for musical identity.
// Instead of every song sounding like "the engine", each ComposerDNA produces
// music with a distinct character. Combined with SongDNA for cross-section memory.
package composer

import "math/rand"

// ComposerDNA defines a composer's musical personality.
// These traits control how post-processing is applied, what "mistakes" to make,
// and how much repetition/obsession to allow.
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

// ComposerArchetypes is a library of named composer personalities.
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

// SongDNA stores cross-section musical memory.
// This allows later sections to reference earlier material.
type SongDNA struct {
	MainMotif     []int     // pitch classes of the main motif
	RhythmCell    []float64 // durations of the rhythm pattern
	IntervalBias  []int     // preferred intervals
	FirstPhrase   []int     // first phrase pitches (for callback)
	MotifStartBar int       // which bar the motif starts in
}

// RecordMotif captures the first notable phrase as the song's identity.
func RecordMotif(events []NoteEventLike, dna *SongDNA) {
	if len(events) < 3 {
		return
	}
	// Take first 3-5 notes as the core motif.
	n := 4
	if len(events) < n {
		n = len(events)
	}
	for i := 0; i < n; i++ {
		dna.MainMotif = append(dna.MainMotif, events[i].GetPitch()%12)
		dna.RhythmCell = append(dna.RhythmCell, events[i].GetDuration())
	}
	dna.MotifStartBar = int(events[0].GetStartBeat()) / 4
}

// ReferenceMotif checks if a later phrase references the main motif.
// Returns a similarity score (0-1).
func (dna *SongDNA) ReferenceMotif(events []NoteEventLike) float64 {
	if len(dna.MainMotif) < 3 || len(events) < 3 {
		return 0
	}
	pitches := make([]int, len(events))
	for i, e := range events {
		pitches[i] = e.GetPitch() % 12
	}
	// Simple: count how many pitches match the motif.
	matches := 0
	for _, p := range pitches {
		for _, m := range dna.MainMotif {
			if p == m {
				matches++
				break
			}
		}
	}
	// Also check interval shape match.
	intervalMatch := 0
	end := len(dna.MainMotif)
	if end > len(pitches) {
		end = len(pitches)
	}
	for i := 1; i < end; i++ {
		mi := dna.MainMotif[i] - dna.MainMotif[i-1]
		pi := pitches[i] - pitches[i-1]
		if (mi < 0 && pi < 0) || (mi > 0 && pi > 0) || (mi == 0 && pi == 0) {
			intervalMatch++
		}
	}
	return (float64(matches)/float64(len(pitches))*0.5 +
		float64(intervalMatch)/float64(end-1)*0.5)
}

// NoteEventLike abstracts the minimal interface for note events.
type NoteEventLike interface {
	GetPitch() int
	GetStartBeat() float64
	GetDuration() float64
}

// AdaptNoteEvent adapts a NoteEvent to NoteEventLike.
// We need this since schema.NoteEvent is a concrete type.

// PickComposer selects a composer archetype based on the requested style.
func PickComposer(styleName, styleDesc, mood string) ComposerDNA {
	// Keyword-based selection.
	moodKeywords := map[string]string{
		"epic": "Hans Zimmer", "cinematic": "Hans Zimmer",
		"fantasy": "Nobuo Uematsu (FF)", "rpg": "JRPG Default",
		"metal": "Hans Zimmer", "heavy": "Hans Zimmer", "rock": "Toby Fox (Undertale)",
		"punk": "Retro Game",
		"retro": "Retro Game", "chiptune": "Retro Game", "8bit": "Retro Game",
		"hyper": "Hyperpop Maniac", "weird": "Hyperpop Maniac",
		"classical": "Classical Purist", "orchestral": "Classical Purist",
	}
	for kw, name := range moodKeywords {
		if containsIgnoreCase(styleName, kw) || containsIgnoreCase(styleDesc, kw) || containsIgnoreCase(mood, kw) {
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
