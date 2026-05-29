package validator

import (
	"fmt"
	"sort"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// FixMeasureDurations adjusts note durations to match the time signature.
// If a bar has too few beats, extends the last note. If too many, trims the last note.
// Drums (ch9) are skipped.
// Returns a list of what was fixed.
func FixMeasureDurations(tracks []schema.TrackIR, totalBars int, ts schema.TimeSignature) []string {
	beatsPerBar := ts.Numerator
	if beatsPerBar <= 0 {
		beatsPerBar = 4
	}
	var fixes []string

	for ti := range tracks {
		t := &tracks[ti]
		if t.Channel == 9 || !t.Enabled || len(t.Events) == 0 {
			continue
		}

		// Group events by bar.
		barEvents := make([][]*schema.NoteEvent, totalBars)
		for i := range barEvents {
			barEvents[i] = nil
		}
		for i := range t.Events {
			ev := &t.Events[i]
			if ev.Type != "note" {
				continue
			}
			bar := int(ev.StartBeat) / beatsPerBar
			if bar >= totalBars {
				bar = totalBars - 1
			}
			if bar < 0 {
				bar = 0
			}
			barEvents[bar] = append(barEvents[bar], ev)
		}

		for bar, evs := range barEvents {
			if len(evs) == 0 {
				continue
			}

			// Sort by start beat.
			sort.Slice(evs, func(i, j int) bool {
				return evs[i].StartBeat < evs[j].StartBeat
			})

			// Compute total duration in this bar.
			barStart := float64(bar * beatsPerBar)
			barEnd := barStart + float64(beatsPerBar)
			var totalDur float64
			for _, ev := range evs {
				// Clamp note to bar boundaries for duration calculation.
				noteStart := ev.StartBeat
				noteEnd := ev.StartBeat + ev.DurationBeat
				if noteStart < barStart {
					noteStart = barStart
				}
				if noteEnd > barEnd {
					noteEnd = barEnd
				}
				if noteEnd > noteStart {
					totalDur += noteEnd - noteStart
				}
			}

			deviation := barEnd - barStart - totalDur
			if deviation <= 0.05 && deviation >= -0.05 {
				continue // close enough
			}

			// Adjust the last note's duration.
			last := evs[len(evs)-1]
			oldDur := last.DurationBeat
			last.DurationBeat += deviation
			if last.DurationBeat < 0.05 {
				last.DurationBeat = 0.05
			}
			// Don't let duration push past bar end.
			if last.StartBeat+last.DurationBeat > barEnd {
				last.DurationBeat = barEnd - last.StartBeat
				if last.DurationBeat < 0.05 {
					last.DurationBeat = 0.05
				}
			}

			fixes = append(fixes, fmt.Sprintf(
				"track %q bar %d: adjusted last note duration %.2f → %.2f (off by %.2f)",
				t.ID, bar, oldDur, last.DurationBeat, deviation))
		}
	}
	return fixes
}

// Lint runs formatting checks on NoteEvents and reports issues.
// Port of Clef's abc_lint.py: checks for invalid values, missing fields, etc.
func Lint(tracks []schema.TrackIR) []string {
	var issues []string

	for _, t := range tracks {
		if !t.Enabled {
			continue
		}
		for i, ev := range t.Events {
			prefix := fmt.Sprintf("track %q event %d", t.ID, i)

			// Type check.
			if ev.Type == "" {
				issues = append(issues, prefix+": missing type field")
				ev.Type = "note"
			}

			// Pitch for non-drum tracks.
			if t.Channel != 9 {
				if ev.Pitch < 21 {
					issues = append(issues, fmt.Sprintf("%s: pitch %d below audible range", prefix, ev.Pitch))
				}
				if ev.Pitch > 108 {
					issues = append(issues, fmt.Sprintf("%s: pitch %d above typical range", prefix, ev.Pitch))
				}
			}

			// Velocity.
			if ev.Velocity < 1 {
				issues = append(issues, fmt.Sprintf("%s: velocity %d is silent", prefix, ev.Velocity))
				ev.Velocity = 64
			}
			if ev.Velocity > 127 {
				issues = append(issues, fmt.Sprintf("%s: velocity %d > 127, clamped", prefix, ev.Velocity))
				ev.Velocity = 127
			}

			// Duration.
			if ev.DurationBeat <= 0 {
				issues = append(issues, fmt.Sprintf("%s: duration %.3f ≤ 0, set to 0.25", prefix, ev.DurationBeat))
				ev.DurationBeat = 0.25
			}
			if ev.DurationBeat > 16.0 {
				issues = append(issues, fmt.Sprintf("%s: duration %.1f unusually long (>4 bars)", prefix, ev.DurationBeat))
			}

			// Start beat.
			if ev.StartBeat < 0 {
				issues = append(issues, fmt.Sprintf("%s: negative start beat %.2f, set to 0", prefix, ev.StartBeat))
				ev.StartBeat = 0
			}
		}
	}
	return issues
}
