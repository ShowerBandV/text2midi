package composer

import (
	"fmt"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ApplyEnding adds a proper song ending: last 4 bars get ritardando
// (notes stretch to fill space) + final chord hold on the last bar.
func ApplyEnding(evMap map[string][]schema.NoteEvent, totalBars int) {
	if totalBars < 8 {
		return
	}
	endingStart := float64(totalBars-4) * 4.0 // last 4 bars

	// Find the root of the last chord for final hold.
	var finalRoot int
	for _, key := range []string{"chords", "rhythm", "pad"} {
		if evs, ok := evMap[key]; ok && len(evs) > 0 {
			finalRoot = evs[len(evs)-1].Pitch
			break
		}
	}

	for key, evs := range evMap {
		if key == "drums" {
			continue // drums don't get ritardando
		}
		for i := range evs {
			e := &evs[i]
			if e.StartBeat < endingStart {
				continue
			}
			// Ritardando: stretch note duration by factor of 1.5-2x.
			progress := (e.StartBeat - endingStart) / 16.0 // 0 to 1 over 4 bars
			if progress > 1.0 {
				progress = 1.0
			}
			stretch := 1.0 + progress*1.0 // 1x → 2x
			e.DurationBeat *= stretch
		}
	}

	// Final chord hold: add a long sustained note on the root.
	lastBar := float64(totalBars-1) * 4.0
	if finalRoot > 0 {
		for _, key := range []string{"lead", "chords", "rhythm"} {
			if evs, ok := evMap[key]; ok {
				evs[len(evs)-1] = schema.NoteEvent{
					Type: "note", Pitch: finalRoot - 12,
					StartBeat: lastBar, DurationBeat: 6.0,
					Velocity: 80,
				}
				break
			}
		}
	}

	fmt.Println("  [Ending] applied ritardando + final chord hold")
}
