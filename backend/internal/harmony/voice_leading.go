// Package harmony --Voice Leading: smooth chord connection algorithm.
// Connects consecutive chords by keeping common tones and moving other voices
// by the smallest possible interval, avoiding parallel fifths and octaves.
package harmony

import (
	"math"
	"strings"
)

// noteSemiVL is a local semitone map for voice leading calculations.
var noteSemiVL = map[string]int{
	"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3,
	"E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8,
	"Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11,
}

// ChordPitches returns the MIDI pitches for a chord symbol (e.g. "Cm7", "Dm", "F")
// at a given base octave. Supports triads + 7th extensions.
func ChordPitches(chord string, baseOct int) []int {
	root := chord
	isMinor := false
	hasSeventh := false
	hasNinth := false

	if strings.HasSuffix(chord, "maj7") {
		root = chord[:len(chord)-4]
		hasSeventh = true
	} else if strings.HasSuffix(chord, "m7") {
		root = chord[:len(chord)-2]
		isMinor = true
		hasSeventh = true
	} else if strings.HasSuffix(chord, "7") {
		root = chord[:len(chord)-1]
		hasSeventh = true
	} else if strings.HasSuffix(chord, "m9") {
		root = chord[:len(chord)-3]
		isMinor = true
		hasSeventh = true
		hasNinth = true
	} else if strings.HasSuffix(chord, "9") {
		root = chord[:len(chord)-1]
		hasSeventh = true
		hasNinth = true
	} else if strings.HasSuffix(chord, "m") {
		root = chord[:len(chord)-1]
		isMinor = true
	}

	rootSemi, ok := noteSemiVL[root]
	if !ok {
		rootSemi = 0
	}

	baseMIDI := (baseOct + 1) * 12
	r := baseMIDI + rootSemi

	tones := []int{r} // root

	// Third.
	third := r + 4
	if isMinor {
		third = r + 3
	}
	tones = append(tones, third)

	// Fifth.
	tones = append(tones, r+7)

	// Seventh.
	if hasSeventh {
		if isMinor {
			tones = append(tones, r+10) // m7
		} else {
			tones = append(tones, r+11) // maj7, or r+10 for dom7
			if strings.HasSuffix(chord, "7") && !strings.HasSuffix(chord, "maj7") {
				tones[len(tones)-1] = r + 10 // dom7
			}
		}
	}

	// Ninth.
	if hasNinth {
		tones = append(tones, r+2)
	}

	// Clamp to MIDI range.
	var out []int
	for _, p := range tones {
		if p >= 21 && p <= 108 {
			out = append(out, p)
		}
	}
	return out
}

// voice represents a single voice in the voice leading algorithm.
type voice struct {
	currentPitch int // current MIDI pitch
	targetPitch  int // assigned target pitch in next chord
}

// ConnectChords applies voice leading between two chord pitch sets.
// prevPitches: current chord's MIDI pitches (from actual voicing).
// nextPitches: next chord's available MIDI pitches (raw chord tones).
// Returns the best matching next voicing that minimizes voice movement.
//
// Algorithm:
//  1. Find common tones ->keep them in the same voice.
//  2. For remaining voices, greedily assign closest target pitch.
//  3. Check for parallel fifths/octaves and correct if found.
//  4. Return connected voicing.
func ConnectChords(prevPitches, nextPitches []int) []int {
	if len(prevPitches) == 0 || len(nextPitches) == 0 {
		return nextPitches
	}

	// Step 1: Build pitch class lookup for next chord.
	nextPC := make(map[int]bool) // pitch class ->exists in next chord
	for _, p := range nextPitches {
		nextPC[p%12] = true
	}

	// Step 2: Greedy assignment.
	used := make(map[int]bool) // indices in nextPitches that are taken
	result := make([]int, len(prevPitches))

	// First pass: keep common tones.
	assigned := make([]bool, len(prevPitches))
	for i, prev := range prevPitches {
		pc := prev % 12
		if nextPC[pc] {
			// Find the closest candidate in the same pitch class.
			bestIdx := -1
			bestDist := math.MaxInt32
			for j, cand := range nextPitches {
				if used[j] {
					continue
				}
				if cand%12 != pc {
					continue
				}
				dist := absInt(cand - prev)
				if dist < bestDist {
					bestDist = dist
					bestIdx = j
				}
			}
			if bestIdx >= 0 {
				result[i] = nextPitches[bestIdx]
				used[bestIdx] = true
				assigned[i] = true
			}
		}
	}

	// Second pass: assign remaining voices to closest unused target pitch.
	unassignedTargets := []int{}
	for j := range nextPitches {
		if !used[j] {
			unassignedTargets = append(unassignedTargets, nextPitches[j])
		}
	}

	for i := range prevPitches {
		if assigned[i] {
			continue
		}
		if len(unassignedTargets) == 0 {
			// Fallback: use any close pitch from next chord.
			bestIdx := -1
			bestDist := math.MaxInt32
			for j, cand := range nextPitches {
				if used[j] {
					continue
				}
				dist := absInt(cand - prevPitches[i])
				if dist < bestDist {
					bestDist = dist
					bestIdx = j
				}
			}
			if bestIdx >= 0 {
				result[i] = nextPitches[bestIdx]
				used[bestIdx] = true
				assigned[i] = true
			}
			continue
		}

		// Find closest unassigned target.
		bestIdx := 0
		bestDist := math.MaxInt32
		for j, cand := range unassignedTargets {
			dist := absInt(cand - prevPitches[i])
			if dist < bestDist {
				bestDist = dist
				bestIdx = j
			}
		}
		result[i] = unassignedTargets[bestIdx]
		unassignedTargets = append(unassignedTargets[:bestIdx], unassignedTargets[bestIdx+1:]...)
		assigned[i] = true
	}

	// Step 3: Check for parallel fifths/octaves and correct.
	// Parallel fifth: two voices both move by the same interval and end up a fifth apart.
	// Parallel octave: two voices both end up an octave apart.

	// Simple correction: if two adjacent voices form a perfect fifth or octave,
	// and they also did in the previous chord, adjust one voice.
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			prevInterval := absInt(prevPitches[i] - prevPitches[j])
			nextInterval := absInt(result[i] - result[j])

			// Parallel fifth: both were P5 apart and both are P5 apart.
			if prevInterval%12 == 7 && nextInterval%12 == 7 {
				// Move one voice up/down by a semitone to break the parallel.
				if result[j] < 108 {
					result[j]++
				} else if result[i] > 21 {
					result[i]--
				}
			}

			// Parallel octave: both were P8 apart and both are P8 apart.
			if prevInterval%12 == 0 && nextInterval%12 == 0 && prevInterval != 0 {
				if result[j] < 108 {
					result[j]++
				} else if result[i] > 21 {
					result[i]--
				}
			}
		}
	}

	return result
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
