// Package composer — Dynamic Control Engine.
// Controls which instruments are active based on section energy.
// True dynamic range comes from removing/re-adding instruments, not just velocity.
package composer

import (
	"fmt"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

var rng struct{ n int }

// DynamicLayer defines which instruments play at a given energy threshold.
type DynamicLayer struct {
	EnergyThreshold float64
	Instruments     []string
	Description     string
}

// DynamicLayers is the master layer table.
// Lower energy = fewer instruments = more contrast when energy rises.
var DynamicLayers = []DynamicLayer{
	{
		EnergyThreshold: 0.0,
		Instruments:     []string{"pad", "drums"},
		Description:     "Ambient + minimal drums",
	},
	{
		EnergyThreshold: 0.3,
		Instruments:     []string{"pad", "bass", "drums", "piano"},
		Description:     "Foundation: bass + drums + piano",
	},
	{
		EnergyThreshold: 0.5,
		Instruments:     []string{"pad", "bass", "drums", "chords", "piano", "lead"},
		Description:     "Full rhythm section",
	},
	{
		EnergyThreshold: 0.7,
		Instruments:     []string{"all", "counter_melody", "texture"},
		Description:     "All instruments + counter melody",
	},
}

// AdjustDrumDensity removes hi-hat and snare hits based on energy.
// Low energy = simpler drums (kick only). High energy = full pattern.
func AdjustDrumDensity(events []schema.NoteEvent, energy float64, totalBars int, styleName string) []schema.NoteEvent {
	if len(events) == 0 {
		return events
	}

	// Metal always keeps full drums.
	if containsIC(styleName, "metal") || containsIC(styleName, "heavy") {
		return events
	}

	keepRatio := 0.3 + energy*0.7 // 0.3 at energy=0, 1.0 at energy=1.0

	var result []schema.NoteEvent
	for _, ev := range events {
		// Always keep kick (pitch 36).
		if ev.Pitch == 36 {
			result = append(result, ev)
			continue
		}
		// Snare (38), hihat (42,46): keep probabilistically.
		if rng.n%100 < int(keepRatio*100) {
			result = append(result, ev)
		}
		rng.n++
	}
	return result
}

func containsIC(s, substr string) bool {
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

// GetActiveInstruments returns the instrument set for a given energy level.
// Resolves "all" to mean all tracks currently in the map.
func GetActiveInstruments(energy float64, allTracks []string) []string {
	// Find the highest layer that this energy exceeds.
	layer := DynamicLayers[0]
	for _, l := range DynamicLayers {
		if energy >= l.EnergyThreshold {
			layer = l
		}
	}

	if len(layer.Instruments) == 1 && layer.Instruments[0] == "all" {
		return allTracks
	}

	// Expand "all" placeholder.
	result := make([]string, 0, len(layer.Instruments))
	for _, inst := range layer.Instruments {
		if inst == "all" {
			result = append(result, allTracks...)
		} else {
			result = append(result, inst)
		}
	}
	return result
}

// MuteInstruments removes note events for tracks not in the active set.
// Instruments below the energy threshold get their events cleared.
func MuteInstruments(evMap map[string][]schema.NoteEvent, activeInstruments []string, sectionIndex int) {
	// Build a set for O(1) lookup.
	active := make(map[string]bool)
	for _, inst := range activeInstruments {
		active[inst] = true
	}

	muted := 0
	for trackID := range evMap {
		if !active[trackID] {
			// Clear events for muted tracks in this section.
			// But keep the track entry (it may become active later).
			evMap[trackID] = []schema.NoteEvent{}
			muted++
		}
	}
	if muted > 0 {
		fmt.Printf("[Dynamics] section %d: %d tracks muted (energy=%.1f)\n",
			sectionIndex, muted, 0.0)
	}
}

// ApplyLayeredDynamics applies energy-based instrument muting per section.
// sections: list of section names (e.g. ["intro","verse","chorus","outro"])
// sectionBarStarts: the starting bar for each section
// totalBars: total number of bars
func ApplyLayeredDynamics(evMap map[string][]schema.NoteEvent, sectionEnergies []float64, sectionBarStarts []int) {
	if len(sectionEnergies) < 1 {
		return
	}

	// Get all track IDs.
	allTracks := make([]string, 0, len(evMap))
	for id := range evMap {
		allTracks = append(allTracks, id)
	}

	for i := 0; i < len(sectionEnergies); i++ {
		energy := sectionEnergies[i]
		startBar := 0
		if i < len(sectionBarStarts) {
			startBar = sectionBarStarts[i]
		}
		endBar := startBar + 2 // default section length

		active := GetActiveInstruments(energy, allTracks)

		// Mute tracks not in the active set.
		activeSet := make(map[string]bool)
		for _, a := range active {
			activeSet[a] = true
		}

		for trackID, events := range evMap {
			if !activeSet[trackID] {
				// Remove events that fall within this section's bar range.
				filtered := make([]schema.NoteEvent, 0, len(events))
				for _, ev := range events {
					bar := int(ev.StartBeat) / 4
					if bar >= startBar && bar < endBar {
						continue // remove
					}
					filtered = append(filtered, ev)
				}
				evMap[trackID] = filtered
			}
		}

		fmt.Printf("[Dynamics] section %d (energy=%.1f): %d/%d instruments active\n",
			i, energy, len(active), len(allTracks))
	}
}
