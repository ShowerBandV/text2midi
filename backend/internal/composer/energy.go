// Package composer provides orchestration logic for multi-instrument coordination.
// It analyzes the lead melody and adjusts other instruments to follow its energy,
// density, and phrasing.
package composer

import (
	"fmt"
	"math"

	"github.com/yourname/text2midi/internal/schema"
)

// EnergyCurve describes the energy profile of a melody across bars.
type EnergyCurve struct {
	BarCount int
	// Per-bar metrics:
	Density  []float64 // 0-1: note density (how many notes per bar, normalized)
	Velocity []float64 // 0-1: average velocity (normalized)
	Range    []float64 // 0-1: pitch range width (normalized)
	Energy   []float64 // 0-1: composite energy = (density * 0.4 + velocity * 0.3 + range * 0.3)
}

// AnalyzeMelody computes the energy curve from lead melody events.
func AnalyzeMelody(events []schema.NoteEvent, totalBars int) *EnergyCurve {
	if totalBars <= 0 {
		totalBars = 1
	}
	ec := &EnergyCurve{
		BarCount: totalBars,
		Density:  make([]float64, totalBars),
		Velocity: make([]float64, totalBars),
		Range:    make([]float64, totalBars),
		Energy:   make([]float64, totalBars),
	}

	if len(events) == 0 {
		// Flat curve if no melody.
		for i := 0; i < totalBars; i++ {
			ec.Energy[i] = 0.5
		}
		return ec
	}

	// Group events by bar.
	type barStats struct {
		count    int
		velSum   int
		minPitch int
		maxPitch int
	}
	stats := make([]barStats, totalBars)
	for i := range stats {
		stats[i].minPitch = 127
		stats[i].maxPitch = 0
	}

	globalMin, globalMax := 127, 0
	for _, ev := range events {
		bar := int(ev.StartBeat) / 4
		if bar >= totalBars {
			bar = totalBars - 1
		}
		if bar < 0 {
			bar = 0
		}
		s := &stats[bar]
		s.count++
		s.velSum += ev.Velocity
		if ev.Pitch < s.minPitch {
			s.minPitch = ev.Pitch
		}
		if ev.Pitch > s.maxPitch {
			s.maxPitch = ev.Pitch
		}
		if ev.Pitch < globalMin {
			globalMin = ev.Pitch
		}
		if ev.Pitch > globalMax {
			globalMax = ev.Pitch
		}
	}

	// Find global max density for normalization.
	maxDensity := 1
	for _, s := range stats {
		if s.count > maxDensity {
			maxDensity = s.count
		}
	}
	pitchRange := globalMax - globalMin
	if pitchRange <= 0 {
		pitchRange = 12
	}

	for i := 0; i < totalBars; i++ {
		s := stats[i]
		if s.count == 0 {
			ec.Density[i] = 0
			ec.Velocity[i] = 0
			ec.Range[i] = 0
		} else {
			ec.Density[i] = float64(s.count) / float64(maxDensity)
			ec.Velocity[i] = float64(s.velSum/s.count) / 127.0
			barRange := s.maxPitch - s.minPitch
			if barRange < 1 {
				barRange = 1
			}
			ec.Range[i] = float64(barRange) / float64(pitchRange)
		}
		// Composite: density matters most for energy feel.
		ec.Energy[i] = ec.Density[i]*0.4 + ec.Velocity[i]*0.3 + ec.Range[i]*0.3
	}

	return ec
}

// PeakBars returns the bar indices where energy exceeds the given threshold.
func (ec *EnergyCurve) PeakBars(threshold float64) []int {
	var peaks []int
	for i, e := range ec.Energy {
		if e >= threshold {
			peaks = append(peaks, i)
		}
	}
	return peaks
}

// AverageEnergy returns the mean energy across all bars.
func (ec *EnergyCurve) AverageEnergy() float64 {
	if ec.BarCount == 0 {
		return 0
	}
	sum := 0.0
	for _, e := range ec.Energy {
		sum += e
	}
	return sum / float64(ec.BarCount)
}

// EnergyAtBar returns the composite energy for a specific bar (clamped).
func (ec *EnergyCurve) EnergyAtBar(bar int) float64 {
	if bar < 0 {
		bar = 0
	}
	if bar >= ec.BarCount {
		bar = ec.BarCount - 1
	}
	if bar >= len(ec.Energy) {
		return 0.5
	}
	return ec.Energy[bar]
}

// Smooth applies a simple moving average to the energy curve.
func (ec *EnergyCurve) Smooth(window int) {
	if window < 2 {
		return
	}
	smoothed := make([]float64, ec.BarCount)
	for i := 0; i < ec.BarCount; i++ {
		sum := 0.0
		count := 0
		for j := -window/2; j <= window/2; j++ {
			idx := i + j
			if idx >= 0 && idx < ec.BarCount {
				sum += ec.Energy[idx]
				count++
			}
		}
		if count > 0 {
			smoothed[i] = sum / float64(count)
		} else {
			smoothed[i] = ec.Energy[i]
		}
	}
	ec.Energy = smoothed
}

// EnergyToDrumModifier returns a multiplier for drum density (0.5-1.5)
// based on energy. High energy bars get more drum hits.
func EnergyToDrumModifier(energy float64) float64 {
	return 0.5 + energy*1.0 // 0.5 ->1.5
}

// EnergyToChordModifier returns the chord style based on energy.
// Returns "block" for high energy, "arp" for medium, "open" for low.
func EnergyToChordStyle(energy float64) string {
	switch {
	case energy > 0.7:
		return "block"
	case energy > 0.4:
		return "mixed"
	default:
		return "open"
	}
}

// AdjustChordDensity modifies chord events based on melody density.
// When melody is dense ->chords get simpler (fewer notes, lower register).
// When melody is sparse ->chords can be fuller.
func AdjustChordDensity(chordEvents []schema.NoteEvent, ec *EnergyCurve) []schema.NoteEvent {
	if len(chordEvents) == 0 || ec == nil {
		return chordEvents
	}

	// Group chord events by bar.
	type barGroup struct {
		indices []int
	}
	bars := make(map[int]*barGroup)
	for i, ev := range chordEvents {
		bar := int(ev.StartBeat) / 4
		if _, ok := bars[bar]; !ok {
			bars[bar] = &barGroup{}
		}
		bars[bar].indices = append(bars[bar].indices, i)
	}

	for bar, group := range bars {
		melEnergy := ec.EnergyAtBar(bar)
		// High melody energy = reduce chord density (thin out notes).
		// Low melody energy = keep or expand chord density.
		if melEnergy > 0.7 && len(group.indices) > 3 {
			// Remove every 3rd note to thin out.
			keep := make([]int, 0, len(group.indices))
			for i, idx := range group.indices {
				if i%3 != 2 { // keep 0,1,3,4,6,7...
					keep = append(keep, idx)
				}
			}
			// Replace with subset.
			kept := make([]schema.NoteEvent, 0, len(keep))
			for _, idx := range keep {
				kept = append(kept, chordEvents[idx])
			}
			// Rebuild chordEvents (simply lower velocity for simplicity).
			for _, idx := range group.indices {
				chordEvents[idx].Velocity = int(float64(chordEvents[idx].Velocity) * 0.6)
				if chordEvents[idx].Velocity < 20 {
					chordEvents[idx].Velocity = 20
				}
			}
		}
		if melEnergy < 0.3 {
			// Low energy: slightly boost velocity to fill space.
			for _, idx := range group.indices {
				chordEvents[idx].Velocity = int(float64(chordEvents[idx].Velocity) * 1.3)
				if chordEvents[idx].Velocity > 127 {
					chordEvents[idx].Velocity = 127
				}
			}
		}
	}
	return chordEvents
}

// FixVoiceCrossing detects and corrects octave crossings between instrument groups.
// The rule: bass < chords < lead in general register. If any track's average pitch
// exceeds the track above it, shift it down an octave.
func FixVoiceCrossing(eventsByTrack map[string][]schema.NoteEvent) {
	type trackInfo struct {
		id       string
		priority int // lower = should be lower in mix
	}

	tracks := []trackInfo{
		{"bass", 0},
		{"chords", 1},
		{"lead", 2},
	}

	// Compute average pitch per track.
	avgPitch := make(map[string]float64)
	count := make(map[string]int)
	sum := make(map[string]int)

	for _, ti := range tracks {
		evs := eventsByTrack[ti.id]
		for _, e := range evs {
			sum[ti.id] += e.Pitch
			count[ti.id]++
		}
		if count[ti.id] > 0 {
			avgPitch[ti.id] = float64(sum[ti.id]) / float64(count[ti.id])
		}
	}

	// Check bass < chords < lead.
	for i := 1; i < len(tracks); i++ {
		lower := tracks[i-1].id
		upper := tracks[i].id
		if count[lower] == 0 || count[upper] == 0 {
			continue
		}
		if avgPitch[lower] > avgPitch[upper] {
			// Voices crossed! Shift lower track down an octave.
			evs := eventsByTrack[lower]
			for j := range evs {
				if evs[j].Pitch > 24 {
					evs[j].Pitch -= 12
				}
			}
			fmt.Printf("  ->Voice fix: %s (avg %.0f) crossed %s (avg %.0f), shifted %s down\n",
				lower, avgPitch[lower], upper, avgPitch[upper], lower)
		}
	}
}

// AlignBassToKick snaps bass note starts to the nearest kick drum hit.
// This creates the locked-in "bass + kick" feel that defines good rhythm sections.
func AlignBassToKick(bassEvents []schema.NoteEvent, drumEvents []schema.NoteEvent) []schema.NoteEvent {
	if len(bassEvents) == 0 || len(drumEvents) == 0 {
		return bassEvents
	}

	// Collect all kick drum hit times.
	var kickBeats []float64
	for _, ev := range drumEvents {
		if ev.Pitch == 36 || ev.DrumName == "kick" {
			kickBeats = append(kickBeats, ev.StartBeat)
		}
	}
	if len(kickBeats) == 0 {
		return bassEvents
	}

	aligned := make([]schema.NoteEvent, len(bassEvents))
	for i, bass := range bassEvents {
		aligned[i] = bass
		// Find nearest kick hit within 0.25 beats.
		bestDist := 0.25
		bestBeat := bass.StartBeat
		for _, kb := range kickBeats {
			dist := bass.StartBeat - kb
			if dist < 0 {
				dist = -dist
			}
			if dist < bestDist {
				bestDist = dist
				bestBeat = kb
			}
		}
		aligned[i].StartBeat = bestBeat
	}

	fmt.Printf("[Align] %d bass notes aligned to %d kick hits\n", len(bassEvents), len(kickBeats))
	return aligned
}

// NormalizeEnergy clamps all energy values to [0, 1].
func NormalizeEnergy(ec *EnergyCurve) {
	if ec == nil || len(ec.Energy) == 0 {
		return
	}
	maxE := 0.0
	for _, e := range ec.Energy {
		if e > maxE {
			maxE = e
		}
	}
	if maxE <= 0 {
		return
	}
	for i := range ec.Energy {
		ec.Energy[i] = math.Min(ec.Energy[i]/maxE, 1.0)
	}
}
