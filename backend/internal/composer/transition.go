// Package composer — Transition Engine for smooth section transitions.
// Automatically inserts fills, risers, cymbal reverses, bass slides,
// and silence bars between sections with different energy levels.
package composer

import (
	"fmt"
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// TransitionType defines the kind of transition.
type TransitionType int

const (
	TNone           TransitionType = iota // no transition
	TDrumFill                             // drum fill (snare + toms)
	TRiser                                // ascending noise/sweep
	TCymbalReverse                        // reversed cymbal swell
	TBassSlide                            // bass slide up/down
	TSilence                              // sudden stop
	TBreath                               // half-beat pause before downbeat
)

// Transition describes an automatic section transition.
type Transition struct {
	Type      TransitionType
	Duration  float64 // in beats
	Intensity float64 // 0.0-1.0
	TargetBar int     // which bar this transition leads into
}

// DetectTransition automatically selects a transition based on energy change.
// fromEnergy: energy of the outgoing section (0-1)
// toEnergy: energy of the incoming section (0-1)
// targetBar: the bar number where the new section starts
func DetectTransition(fromEnergy, toEnergy float64, targetBar int) *Transition {
	diff := toEnergy - fromEnergy
	absDiff := diff
	if absDiff < 0 {
		absDiff = -absDiff
	}

	t := &Transition{
		TargetBar: targetBar,
		Duration:  1.0, // default: 1 beat
	}

	switch {
	case toEnergy > 0.8 && diff > 0.3:
		// Big energy jump UP → riser + crash
		t.Type = TRiser
		t.Duration = 2.0
		t.Intensity = 0.9
	case toEnergy > 0.6 && diff > 0.2:
		// Medium energy jump UP → drum fill
		t.Type = TDrumFill
		t.Duration = 1.0
		t.Intensity = 0.7
	case toEnergy < 0.3 && diff < -0.3:
		// Big energy drop → silence or breath
		if rand.Float64() < 0.5 {
			t.Type = TSilence
			t.Duration = 0.5
		} else {
			t.Type = TBreath
			t.Duration = 0.25
		}
		t.Intensity = 0.3
	case diff < -0.1:
		// Small energy drop → cymbal or bass slide
		if rand.Float64() < 0.5 {
			t.Type = TCymbalReverse
			t.Duration = 1.0
		} else {
			t.Type = TBassSlide
			t.Duration = 0.5
		}
		t.Intensity = 0.5
	default:
		// Same energy → subtle drum fill
		t.Type = TDrumFill
		t.Duration = 0.5
		t.Intensity = 0.4
	}

	return t
}

// ApplyTransition modifies eventsByTrack to include the transition.
func ApplyTransition(eventsByTrack map[string][]schema.NoteEvent, t *Transition, bpm int) {
	if t.Type == TNone {
		return
	}

	barStart := float64(t.TargetBar) * 4.0
	transitionStart := barStart - t.Duration

	fmt.Printf("[Transition] %s → bar %d (intensity=%.1f)\n",
		tname(t.Type), t.TargetBar, t.Intensity)

	switch t.Type {
	case TDrumFill:
		// Snare + tom fill: rapid hits accelerating toward downbeat.
		if drums, ok := eventsByTrack["drums"]; ok {
			hits := 4 + int(t.Intensity*4)
			for i := 0; i < hits; i++ {
				beat := transitionStart + float64(i)*(t.Duration/float64(hits))
				pitch := 38 + rand.Intn(12) // snare + tom range
				vel := 70 + int(t.Intensity*40)
				drums = append(drums, schema.NoteEvent{
					Type: "note", Pitch: pitch, DrumName: "fill",
					StartBeat: beat, DurationBeat: 0.08, Velocity: vel,
				})
			}
			// Crash on downbeat.
			drums = append(drums, schema.NoteEvent{
				Type: "note", Pitch: 49, DrumName: "crash",
				StartBeat: barStart, DurationBeat: 0.5, Velocity: 110,
			})
			eventsByTrack["drums"] = drums
		}

	case TRiser:
		// Ascending pitch sweep using noise/sweep effect.
		riserTopPitch := 84
		if lead, ok := eventsByTrack["lead"]; ok {
			steps := 8
			for i := 0; i < steps; i++ {
				progress := float64(i) / float64(steps)
				beat := transitionStart + progress*t.Duration
				// Rapid ascending notes in higher register.
				basePitch := 72 + int(progress*24)
				lead = append(lead, schema.NoteEvent{
					Type: "note", Pitch: basePitch,
					StartBeat:    beat,
					DurationBeat: t.Duration / float64(steps) * 0.8,
					Velocity:     60 + int(progress*50),
				})
			}
			// Crash cymbal on downbeat.
			lead = append(lead, schema.NoteEvent{
				Type: "note", Pitch: riserTopPitch + 12,
				StartBeat:    barStart,
				DurationBeat: 1.0,
				Velocity:     110,
			})
			eventsByTrack["lead"] = lead
		}
		// Also add crash to drums.
		if drums, ok := eventsByTrack["drums"]; ok {
			drums = append(drums, schema.NoteEvent{
				Type: "note", Pitch: 49, DrumName: "crash",
				StartBeat: barStart, DurationBeat: 0.5, Velocity: 120,
			})
			eventsByTrack["drums"] = drums
		}

	case TCymbalReverse:
		// Simulate reversed cymbal by gradually increasing velocity.
		if drums, ok := eventsByTrack["drums"]; ok {
			steps := 4
			for i := 0; i < steps; i++ {
				progress := float64(i) / float64(steps)
				beat := transitionStart + progress*t.Duration
				vel := int(progress * 100)
				if vel < 10 {
					vel = 10
				}
				drums = append(drums, schema.NoteEvent{
					Type: "note", Pitch: 49, DrumName: "crash",
					StartBeat: beat, DurationBeat: 0.1, Velocity: vel,
				})
			}
			eventsByTrack["drums"] = drums
		}

	case TBassSlide:
		// Bass slides from current root to next root.
		if bass, ok := eventsByTrack["bass"]; ok {
			startPitch := 36
			endPitch := 40
			if len(bass) > 0 {
				startPitch = bass[len(bass)-1].Pitch
				endPitch = startPitch + 5
				if endPitch > 60 {
					endPitch = startPitch - 5
				}
			}
			steps := 4
			for i := 0; i < steps; i++ {
				progress := float64(i) / float64(steps)
				beat := transitionStart + progress*t.Duration
				pitch := startPitch + int(float64(endPitch-startPitch)*progress)
				bass = append(bass, schema.NoteEvent{
					Type: "note", Pitch: pitch,
					StartBeat:    beat,
					DurationBeat: t.Duration / float64(steps),
					Velocity:     80 + int(progress*30),
				})
			}
			eventsByTrack["bass"] = bass
		}

	case TSilence:
		// Remove all notes in the transition zone (create silence).
		for trackID, events := range eventsByTrack {
			if trackID == "drums" {
				continue // drums still need the crash
			}
			filtered := make([]schema.NoteEvent, 0, len(events))
			for _, ev := range events {
				if ev.StartBeat < transitionStart || ev.StartBeat >= barStart {
					filtered = append(filtered, ev)
				}
			}
			eventsByTrack[trackID] = filtered
		}

	case TBreath:
		// Half-beat pause: shift all notes in the first beat of the new section.
		for trackID, events := range eventsByTrack {
			if trackID == "drums" {
				continue
			}
			for i := range events {
				if events[i].StartBeat >= barStart && events[i].StartBeat < barStart+0.5 {
					events[i].StartBeat += 0.5 // push forward
				}
			}
		}
	}
}

func tname(t TransitionType) string {
	switch t {
	case TDrumFill:
		return "drum_fill"
	case TRiser:
		return "riser"
	case TCymbalReverse:
		return "cymbal_rev"
	case TBassSlide:
		return "bass_slide"
	case TSilence:
		return "silence"
	case TBreath:
		return "breath_pause"
	default:
		return "none"
	}
}

// BuildSectionEnergyProfile creates an energy curve from section names.
// Sections not in the map get a default energy of 0.5.
func BuildSectionProfile(sections []string) []float64 {
	profile := map[string]float64{
		"intro":   0.2,
		"verse":   0.4,
		"pre":     0.6,
		"chorus":  0.85,
		"bridge":  0.5,
		"solo":    0.75,
		"outro":   0.2,
		"build":   0.5,
		"climax":  0.95,
		"resolve": 0.3,
	}
	energies := make([]float64, len(sections))
	for i, s := range sections {
		if e, ok := profile[s]; ok {
			energies[i] = e
		} else {
			energies[i] = 0.5
		}
	}
	return energies
}

// ApplyAllTransitions scans the arrangement timeline and inserts transitions.
func ApplyAllTransitions(eventsByTrack map[string][]schema.NoteEvent, sectionEnergies []float64, sectionBarStarts []int, bpm int) {
	if len(sectionEnergies) < 2 {
		return
	}
	for i := 1; i < len(sectionEnergies); i++ {
		from := sectionEnergies[i-1]
		to := sectionEnergies[i]
		bar := sectionBarStarts[i]
		t := DetectTransition(from, to, bar)
		if t.Type != TNone {
			ApplyTransition(eventsByTrack, t, bpm)
		}
	}
}
