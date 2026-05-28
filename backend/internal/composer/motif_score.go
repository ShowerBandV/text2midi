// Package composer — music composition and arrangement engine.
package composer

import (
	"math"

	"github.com/ShowerBandV/text2midi/internal/musicdna"
)

// ScoreMotif evaluates a motif across 4 dimensions and returns a MotifScore.
// Implements the ROADMAP.md Motif Scoring System:
//
//	Total = repetition*0.4 + contour*0.2 + simplicity*0.2 + rhythm_identity*0.2
//
// Scoring drives generation: high score → more repetition, low score → strong variation.
func ScoreMotif(intervals, relativeInts []int, durations []float64, totalBars int) *musicdna.MotifScore {
	if len(intervals) < 4 || totalBars <= 0 {
		return &musicdna.MotifScore{}
	}

	ms := &musicdna.MotifScore{}

	// 1. Repetition: how many times the core pattern appears / total bars.
	// Count occurrences of the first 3-5 intervals as the "seed pattern".
	seedLen := 4
	if seedLen > len(relativeInts) {
		seedLen = len(relativeInts)
	}
	if seedLen < 2 {
		seedLen = 2
	}
	seed := relativeInts[:seedLen]

	occurrences := 0
	for i := 0; i <= len(relativeInts)-seedLen; i++ {
		match := true
		for j := 0; j < seedLen; j++ {
			if relativeInts[i+j] != seed[j] {
				match = false
				break
			}
		}
		if match {
			occurrences++
		}
	}
	// Normalize: max expected occurrences ≈ totalBars
	ms.Repetition = musicdna.Clamp01(float64(occurrences) / float64(totalBars+1))

	// 2. Contour: slope variance of the pitch sequence.
	// Higher variance = more angular/interesting contour.
	if len(intervals) >= 2 {
		var sumSlope, sumSlopeSq float64
		count := 0
		for i := 1; i < len(intervals); i++ {
			slope := float64(intervals[i] - intervals[i-1])
			sumSlope += slope
			sumSlopeSq += slope * slope
			count++
		}
		if count > 0 {
			meanSlope := sumSlope / float64(count)
			variance := sumSlopeSq/float64(count) - meanSlope*meanSlope
			if variance < 0 {
				variance = 0
			}
			// Normalize: variance of ~10 is quite angular; cap at 1
			ms.Contour = musicdna.Clamp01(math.Sqrt(variance) / 10.0)
		}
	}

	// 3. Simplicity: 1 - (avg interval size / 12 semitones).
	if len(relativeInts) > 0 {
		var sumAbs float64
		for _, iv := range relativeInts {
			sumAbs += float64(musicdna.AbsInt(iv))
		}
		avgInterval := sumAbs / float64(len(relativeInts))
		ms.Simplicity = musicdna.Clamp01(1.0 - avgInterval/12.0)
	}

	// 4. RhythmIdentity: self-similarity of duration patterns.
	if len(durations) >= 4 {
		// Divide into first half and second half, compare pattern similarity.
		mid := len(durations) / 2
		var diffSum float64
		pairs := 0
		for i := 0; i < mid && i+mid < len(durations); i++ {
			diff := math.Abs(durations[i] - durations[i+mid])
			// Normalize duration diff by max duration
			maxDur := math.Max(durations[i], durations[i+mid])
			if maxDur > 0 {
				diffSum += musicdna.Clamp01(diff / maxDur)
				pairs++
			}
		}
		if pairs > 0 {
			ms.RhythmIdentity = musicdna.Clamp01(1.0 - diffSum/float64(pairs))
		}
	}

	ms.CalculateTotal()
	return ms
}

// ScoreDrivesMutation returns a mutation factor based on motif score.
// High score (>0.7): conservative mutation (keep motif recognizable)
// Low score (<0.4): aggressive mutation (try harder to find a good motif)
func ScoreDrivesMutation(score *musicdna.MotifScore) float64 {
	if score == nil {
		return 0.5 // default
	}
	// Low total score → high mutation; high score → low mutation.
	return musicdna.Clamp01(1.0 - score.Total)
}
