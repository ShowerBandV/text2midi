// Package generator implements rule-based MIDI note generators for each instrument role.
// Ported from music_agent/generators/.
package generator

import (
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// GenerateNotes dispatches to the correct generator based on track ID.
// It mirrors the dispatch logic in music_agent/agents/note_generator.py.
func GenerateNotes(plan schema.SongPlan, track schema.ArrangementTrack) []schema.NoteEvent {
	switch track.ID {
	case "drums":
		return GenerateDrums(plan, track)
	case "bass":
		return GenerateBass(plan, track)
	case "chords":
		return GenerateChords(plan, track)
	case "lead":
		return GenerateLead(plan, track)
	case "rhythm_guitar":
		return GenerateRhythmGuitar(plan, track)
	case "lead_guitar":
		return GenerateLead(plan, track)
	default:
		return GenerateGeneric(plan, track)
	}
}

// GenerateGeneric creates a simple placeholder pattern for unknown tracks.
func GenerateGeneric(plan schema.SongPlan, track schema.ArrangementTrack) []schema.NoteEvent {
	totalBars := plan.TotalBars
	basePitch := 60 + min(track.Channel, 12)
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		start := float64(bar) * 4.0
		events = append(events, schema.NoteEvent{
			Type: "note", Pitch: basePitch,
			StartBeat: start, DurationBeat: 1.0, Velocity: 72,
		})
		events = append(events, schema.NoteEvent{
			Type: "note", Pitch: basePitch + 7,
			StartBeat: start + 2.0, DurationBeat: 1.0, Velocity: 68,
		})
	}
	return events
}

// makeMotif creates a motif slice of length n with random values 0-6,
// then fixes first=0 and last to a random 0,2,4.
func makeMotif(n int) []int {
	m := make([]int, n)
	for i := range m {
		m[i] = rand.Intn(7) // 0..6
	}
	m[0] = 0
	m[n-1] = []int{0, 2, 4}[rand.Intn(3)]
	return m
}
