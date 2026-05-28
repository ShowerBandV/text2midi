// Package critic — Post-generation quality evaluation.
// Scores generated music on 4 dimensions and triggers regeneration when quality is low.
package critic

import (
	"fmt"
	"math"
	"sort"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// MusicScore is the quality evaluation of generated music.
type MusicScore struct {
	Repetition     float64 // 0-1: enough repetition for memorability
	Tension        float64 // 0-1: harmonic tension curve has shape
	Groove         float64 // 0-1: rhythmic feel is engaging
	Climax         float64 // 0-1: there is a clear climax section
	Density        float64 // 0-1: density varies across sections
	Total          float64 // weighted total
}

// Evaluate runs all quality checks on generated music.
func Evaluate(eventsByTrack map[string][]schema.NoteEvent, totalBars int) *MusicScore {
	s := &MusicScore{}

	if totalBars <= 0 {
		return s
	}

	s.Repetition = evaluateRepetition(eventsByTrack, totalBars)
	s.Tension = evaluateTension(eventsByTrack, totalBars)
	s.Groove = evaluateGroove(eventsByTrack, totalBars)
	s.Climax = evaluateClimax(eventsByTrack, totalBars)
	s.Density = evaluateDensityVariety(eventsByTrack, totalBars)

	s.Total = s.Repetition*0.25 + s.Tension*0.2 + s.Groove*0.2 + s.Climax*0.2 + s.Density*0.15

	fmt.Printf("[Critic] rep=%.2f ten=%.2f gro=%.2f cli=%.2f den=%.2f total=%.2f\n",
		s.Repetition, s.Tension, s.Groove, s.Climax, s.Density, s.Total)

	return s
}

// evaluateRepetition: checks if the lead melody has enough repeated patterns.
func evaluateRepetition(eventsByTrack map[string][]schema.NoteEvent, totalBars int) float64 {
	lead := getTrack(eventsByTrack, "lead", "lead_guitar", "lead_vocal")
	if len(lead) < 8 {
		return 0.3
	}

	// Calculate interval sequence.
	intervals := make([]int, 0)
	sort.Slice(lead, func(i, j int) bool {
		return lead[i].StartBeat < lead[j].StartBeat
	})
	for i := 1; i < len(lead); i++ {
		intervals = append(intervals, lead[i].Pitch-lead[i-1].Pitch)
	}

	if len(intervals) < 4 {
		return 0.3
	}

	// Count repeated 3-note patterns.
	patternCount := 0
	for i := 0; i <= len(intervals)-3; i++ {
		for j := i + 3; j <= len(intervals)-3; j++ {
			if intervals[i] == intervals[j] &&
				intervals[i+1] == intervals[j+1] &&
				intervals[i+2] == intervals[j+2] {
				patternCount++
			}
		}
	}

	maxPossible := (len(intervals) - 2) * (len(intervals) - 5) / 2
	if maxPossible <= 0 {
		return 0.3
	}
	score := float64(patternCount) / float64(maxPossible) * 10
	if score > 1.0 {
		score = 1.0
	}
	// A score of 0 means no repetition at all — bad.
	// A score of 0.5-0.8 means healthy repetition.
	if score < 0.3 {
		score = score * 0.5 // penalize insufficient repetition
	}
	return score
}

// evaluateTension: checks if energy varies across sections.
func evaluateTension(eventsByTrack map[string][]schema.NoteEvent, totalBars int) float64 {
	if totalBars < 4 {
		return 0.5
	}

	// Compute energy per bar.
	barEnergy := make([]float64, totalBars)
	for _, events := range eventsByTrack {
		for _, ev := range events {
			bar := int(ev.StartBeat) / 4
			if bar >= 0 && bar < totalBars {
				barEnergy[bar] += float64(ev.Velocity) * 0.01
			}
		}
	}

	// Check that energy varies (isn't flat).
	if len(barEnergy) < 2 {
		return 0.5
	}
	maxE, minE := barEnergy[0], barEnergy[0]
	for _, e := range barEnergy {
		if e > maxE {
			maxE = e
		}
		if e < minE {
			minE = e
		}
	}
	range_ := maxE - minE
	if range_ < 0.5 {
		return range_ * 2 // penalize flat energy
	}
	return 0.7 + range_*0.3
}

// evaluateGroove: checks drum variety.
func evaluateGroove(eventsByTrack map[string][]schema.NoteEvent, totalBars int) float64 {
	drums := eventsByTrack["drums"]
	if len(drums) == 0 {
		return 0.5
	}

	// Check velocity variety.
	velSum := 0
	velCount := 0
	velSeen := make(map[int]bool)
	for _, ev := range drums {
		velSum += ev.Velocity
		velCount++
		velSeen[ev.Velocity] = true
	}

	if velCount == 0 {
		return 0.5
	}

	avgVel := float64(velSum) / float64(velCount)
	stdDev := 0.0
	for _, ev := range drums {
		d := float64(ev.Velocity) - avgVel
		stdDev += d * d
	}
	stdDev = math.Sqrt(stdDev / float64(velCount))

	// More velocity variety = more groove.
	score := stdDev / 30.0
	if score > 1.0 {
		score = 1.0
	}
	return 0.4 + score*0.6
}

// evaluateClimax: checks if the last third has higher energy.
func evaluateClimax(eventsByTrack map[string][]schema.NoteEvent, totalBars int) float64 {
	if totalBars < 4 {
		return 0.5
	}

	oneThird := totalBars / 3
	if oneThird < 1 {
		oneThird = 1
	}

	// Energy in first third vs last third.
	firstEnergy := 0.0
	lastEnergy := 0.0
	firstCount := 0
	lastCount := 0

	for _, events := range eventsByTrack {
		for _, ev := range events {
			bar := int(ev.StartBeat) / 4
			if bar < oneThird {
				firstEnergy += float64(ev.Velocity)
				firstCount++
			}
			if bar >= totalBars-oneThird {
				lastEnergy += float64(ev.Velocity)
				lastCount++
			}
		}
	}

	if firstCount == 0 || lastCount == 0 {
		return 0.5
	}

	avgFirst := firstEnergy / float64(firstCount)
	avgLast := lastEnergy / float64(lastCount)
	ratio := avgLast / avgFirst

	if ratio > 1.5 {
		return 1.0
	}
	if ratio > 1.0 {
		return 0.7
	}
	return ratio / 1.5
}

// evaluateDensityVariety: checks if notes-per-bar varies.
func evaluateDensityVariety(eventsByTrack map[string][]schema.NoteEvent, totalBars int) float64 {
	if totalBars < 4 {
		return 0.5
	}

	notesPerBar := make([]int, totalBars)
	for _, events := range eventsByTrack {
		for _, ev := range events {
			bar := int(ev.StartBeat) / 4
			if bar >= 0 && bar < totalBars {
				notesPerBar[bar]++
			}
		}
	}

	maxN, minN := notesPerBar[0], notesPerBar[0]
	for _, n := range notesPerBar {
		if n > maxN {
			maxN = n
		}
		if n < minN {
			minN = n
		}
	}

	diff := maxN - minN
	if diff < 3 {
		return 0.3 // too uniform
	}
	if diff > 20 {
		return 1.0
	}
	return 0.5 + float64(diff)/40.0
}

// NeedsRegeneration returns true if quality is too low.
func (s *MusicScore) NeedsRegeneration() bool {
	return s.Total < 0.4
}

func getTrack(m map[string][]schema.NoteEvent, ids ...string) []schema.NoteEvent {
	for _, id := range ids {
		if evs, ok := m[id]; ok && len(evs) > 0 {
			return evs
		}
	}
	return nil
}
