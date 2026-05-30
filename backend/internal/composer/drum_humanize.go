package composer

import "github.com/ShowerBandV/text2midi/internal/schema"

// HumanizeHiHat adds articulation variety to hi-hat events.
// Alternates between closed(42), pedal(44), and open(46) with velocity variation.
func HumanizeHiHat(events []schema.NoteEvent, totalBars int) []schema.NoteEvent {
	for i := range events {
		if events[i].Type != "note" {
			continue
		}
		p := events[i].Pitch
		if p != 42 && p != 44 && p != 46 {
			continue // not a hi-hat
		}

		bar := int(events[i].StartBeat) / 4
		beatInBar := events[i].StartBeat - float64(bar)*4.0

		// Every 4th hit: use open hat (46) instead of closed (42).
		if int(beatInBar*4)%4 == 0 && p == 42 {
			events[i].Pitch = 46
			events[i].DurationBeat *= 1.5
		}

		// Every 8th bar: pedal chick (44) on beat 2 and 4.
		if bar%8 == 4 && (beatInBar == 1.0 || beatInBar == 3.0) && p == 42 {
			events[i].Pitch = 44
			events[i].DurationBeat = 0.06
			events[i].Velocity = 80
		}
	}
	return events
}
