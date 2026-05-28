// Package arranger — Arrangement coordination engine.
// Manages register conflicts, rhythm density balancing, and auto silence.
// The arranger ensures all tracks work together rather than competing.
package arranger

import (
	"fmt"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ArrangementState tracks the current arrangement state per bar.
type ArrangementState struct {
	Density         float64             // 0-1: how busy the arrangement is
	RegisterUsage   map[string]string   // track → "low"/"mid"/"high"
	RhythmComplexity float64            // 0-1: how rhythmically dense
	ActiveTracks    int                 // how many tracks are playing
}

// ConflictType describes a conflict detected in the arrangement.
type ConflictType int

const (
	NoConflict     ConflictType = iota
	RegisterClash              // two tracks in same register
	RhythmClash                // too many busy tracks simultaneously
	EmptyBar                   // nothing playing
	DensitySpike               // sudden density jump
)

// Conflict describes an arrangement issue that needs fixing.
type Conflict struct {
	Type     ConflictType
	Bar      int
	Tracks   []string
	Severity float64 // 0-1
	Message  string
}

// CheckArrangement scans eventsByTrack for arrangement conflicts.
func CheckArrangement(eventsByTrack map[string][]schema.NoteEvent, totalBars int) []Conflict {
	var conflicts []Conflict

	for bar := 0; bar < totalBars; bar++ {
		barStart := float64(bar) * 4.0
		barEnd := barStart + 4.0

		// Collect track activity and register usage.
		type trackActivity struct {
			id       string
			avgPitch float64
			noteCount int
		}
		var active []trackActivity

		for trackID, events := range eventsByTrack {
			count := 0
			sumPitch := 0
			for _, ev := range events {
				if ev.StartBeat >= barStart && ev.StartBeat < barEnd {
					count++
					sumPitch += ev.Pitch
				}
			}
			if count > 0 {
				active = append(active, trackActivity{
					id:        trackID,
					avgPitch:  float64(sumPitch) / float64(count),
					noteCount: count,
				})
			}
		}

		// Check register clash: bass and lead in same register.
		var bassPitch, leadPitch float64
		var hasBass, hasLead bool
		for _, a := range active {
			if a.id == "bass" {
				bassPitch = a.avgPitch
				hasBass = true
			}
			if a.id == "lead" || a.id == "lead_guitar" {
				leadPitch = a.avgPitch
				hasLead = true
			}
		}
		if hasBass && hasLead && bassPitch > leadPitch-12 {
			conflicts = append(conflicts, Conflict{
				Type:     RegisterClash,
				Bar:      bar,
				Tracks:   []string{"bass", "lead"},
				Severity: 0.6,
				Message:  fmt.Sprintf("bar %d: bass (%.0f) and lead (%.0f) in same register", bar, bassPitch, leadPitch),
			})
		}

		// Check density: too many notes from too many tracks.
		if len(active) >= 4 {
			totalNotes := 0
			for _, a := range active {
				totalNotes += a.noteCount
			}
			if totalNotes > 32 {
				conflicts = append(conflicts, Conflict{
					Type:     DensitySpike,
					Bar:      bar,
					Severity: 0.4,
					Message:  fmt.Sprintf("bar %d: density spike (%d notes, %d tracks)", bar, totalNotes, len(active)),
				})
			}
		}

		// Check empty bar.
		if len(active) == 0 {
			conflicts = append(conflicts, Conflict{
				Type:     EmptyBar,
				Bar:      bar,
				Severity: 0.8,
				Message:  fmt.Sprintf("bar %d: completely empty", bar),
			})
		}
	}

	if len(conflicts) > 0 {
		fmt.Printf("[Arranger] %d conflicts detected\n", len(conflicts))
	}
	return conflicts
}

// ResolveConflicts applies fixes to detected conflicts.
// Modifies eventsByTrack in place.
func ResolveConflicts(conflicts []Conflict, eventsByTrack map[string][]schema.NoteEvent) int {
	fixed := 0
	for _, c := range conflicts {
		switch c.Type {
		case RegisterClash:
			// Lower bass by an octave.
			if events, ok := eventsByTrack["bass"]; ok {
				for i := range events {
					bar := int(events[i].StartBeat) / 4
					if bar == c.Bar && events[i].Pitch > 48 {
						events[i].Pitch -= 12
						fixed++
					}
				}
			}
		case DensitySpike:
			// Slightly reduce velocities.
			for _, trackID := range c.Tracks {
				if events, ok := eventsByTrack[trackID]; ok {
					for i := range events {
						bar := int(events[i].StartBeat) / 4
						if bar == c.Bar {
							events[i].Velocity = int(float64(events[i].Velocity) * 0.85)
							fixed++
						}
					}
				}
			}
		}
	}
	if fixed > 0 {
		fmt.Printf("[Arranger] resolved %d conflicts\n", fixed)
	}
	return fixed
}
