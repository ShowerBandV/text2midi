// Package composer — Emotion Engine.
// Maps emotional states to musical parameters: harmony, rhythm, motif, arrangement.
// The control bus that makes composition expressive rather than mechanical.
package composer

import (
	"fmt"
	"math/rand"
)

// ─── Emotion State ─────────────────────────────────────────────────

type EmotionState struct {
	Tension    float64 // 0-1: harmonic tension (consonant → dissonant)
	Energy     float64 // 0-1: rhythmic density (calm → intense)
	Warmth     float64 // 0-1: timbral warmth (cold synth → warm acoustic)
	Stability  float64 // 0-1: structural stability (chaotic → stable)
	Brightness float64 // 0-1: register brightness (dark → bright)
}

// DefaultEmotions returns preset emotions for common sections.
func DefaultEmotions() map[string]EmotionState {
	return map[string]EmotionState{
		"intro":  {Tension: 0.2, Energy: 0.2, Warmth: 0.6, Stability: 0.8, Brightness: 0.4},
		"verse":  {Tension: 0.4, Energy: 0.4, Warmth: 0.7, Stability: 0.7, Brightness: 0.5},
		"chorus": {Tension: 0.6, Energy: 0.9, Warmth: 0.5, Stability: 0.6, Brightness: 0.8},
		"bridge": {Tension: 0.8, Energy: 0.5, Warmth: 0.3, Stability: 0.3, Brightness: 0.3},
		"outro":  {Tension: 0.1, Energy: 0.1, Warmth: 0.8, Stability: 0.9, Brightness: 0.3},
	}
}

// ─── Emotion Curve ─────────────────────────────────────────────────

// EmotionCurve maps a timeline of emotional states across the song.
type EmotionCurve struct {
	Bars []EmotionState // one emotion per bar
}

// BuildEmotionCurve creates a full-song emotion timeline from section emotions.
func BuildEmotionCurve(sectionEmotions map[string]EmotionState, sectionBars map[string]int, sectionOrder []string, totalBars int) *EmotionCurve {
	curve := &EmotionCurve{Bars: make([]EmotionState, totalBars)}
	barCursor := 0

	for _, name := range sectionOrder {
		bars := sectionBars[name]
		if bars <= 0 {
			continue
		}

		emotion := sectionEmotions[name]

		for i := 0; i < bars && barCursor+i < totalBars; i++ {
			progress := float64(i) / float64(bars)
			// Smooth transitions within sections.
			curve.Bars[barCursor+i] = EmotionState{
				Tension:    clamp(emotion.Tension + progress*0.1 - 0.05),
				Energy:     clamp(emotion.Energy + progress*0.15 - 0.075),
				Warmth:     clamp(emotion.Warmth - progress*0.1 + 0.05),
				Stability:  clamp(emotion.Stability - progress*0.05 + 0.025),
				Brightness: clamp(emotion.Brightness + progress*0.1 - 0.05),
			}
		}
		barCursor += bars
	}

	return curve
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ─── Emotion → Music Mappings ─────────────────────────────────────

// SelectChordEmotion picks a chord type based on emotional state.
func SelectChordEmotion(emotion EmotionState, keyRoot int, rng *rand.Rand) string {
	semiToNote := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	root := semiToNote[keyRoot%12]

	switch {
	case emotion.Tension > 0.7:
		// High tension: dominant, altered, diminished.
		choices := []string{root + "7", root + "dim", root + "aug", nextDominant(root, keyRoot)}
		return choices[rng.Intn(len(choices))]
	case emotion.Stability > 0.7 && emotion.Energy > 0.6:
		// Stable + energetic: major, power chord.
		return root
	case emotion.Warmth > 0.6:
		// Warm: minor, add9, maj7.
		choices := []string{root + "maj7", root + "m", root + "m9"}
		return choices[rng.Intn(len(choices))]
	case emotion.Brightness > 0.7:
		// Bright: major, sus.
		choices := []string{root, root + "sus4", root + "maj7"}
		return choices[rng.Intn(len(choices))]
	default:
		return root
	}
}

func nextDominant(current string, keyRoot int) string {
	semi := map[string]int{"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11}
	names := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	root := current
	if len(current) > 1 && current[1] == '#' || current[1] == 'b' {
		root = current[:2]
	} else {
		root = current[:1]
	}
	rs, ok := semi[root]
	if !ok {
		rs = 0
	}
	// Dominant of the dominant: V7/V → e.g. D7 → G7
	domRoot := (rs + 7) % 12
	return names[domRoot] + "7"
}

// RhythmDensity returns note density multiplier based on emotion.
func RhythmDensity(emotion EmotionState) float64 {
	return 0.3 + emotion.Energy*0.7
}

// MotifMutationRate returns how aggressively to mutate the motif.
func MotifMutationRate(emotion EmotionState) float64 {
	// High tension + low stability → mutate more.
	rate := emotion.Tension*0.6 + (1-emotion.Stability)*0.4
	return clamp(rate)
}

// RegisterShift returns octave offset based on brightness.
func RegisterShift(emotion EmotionState) int {
	if emotion.Brightness > 0.7 {
		return 12 // octave up
	}
	if emotion.Brightness < 0.3 {
		return -12 // octave down
	}
	return 0
}

// ─── Emotion-Driven Chord Progression ──────────────────────────────

// GenerateChordsFromEmotion creates a chord progression driven by the emotion curve.
func GenerateChordsFromEmotion(curve *EmotionCurve, keyRoot int, bars int, rng *rand.Rand) []string {
	chords := make([]string, bars)
	for bar := 0; bar < bars && bar < len(curve.Bars); bar++ {
		emotion := curve.Bars[bar]
		chords[bar] = SelectChordEmotion(emotion, keyRoot, rng)
	}
	return chords
}

// ─── Emotion-Driven Motif Mutation ─────────────────────────────────

// MutateMotifByEmotion applies emotion-aware mutations to a motif.
func MutateMotifByEmotion(motif []int, emotion EmotionState, rng *rand.Rand) []int {
	if len(motif) == 0 {
		return motif
	}

	rate := MotifMutationRate(emotion)
	if rng.Float64() > rate {
		// No mutation: repeat motif as-is.
		return copySlice(motif)
	}

	// Apply mutation based on emotional profile.
	switch {
	case emotion.Tension > 0.6 && emotion.Stability < 0.4:
		// High tension + unstable → invert + add leading tones.
		result := Invert(motif)
		result = append(result, motif[len(motif)-1]+1) // leading tone
		return result
	case emotion.Energy > 0.7:
		// High energy → transpose up + fragment.
		result := Transpose(motif, 5)
		return Fragment(result, len(result)/2+1)
	case emotion.Warmth > 0.6 && emotion.Brightness < 0.4:
		// Warm + dark → lower register, slower rhythm.
		return Transpose(motif, -3)
	default:
		// Moderate → slight variation.
		if rng.Float64() < 0.5 {
			return Retrograde(motif)
		}
		return Transpose(motif, []int{-2, 2, 3}[rng.Intn(3)])
	}
}

// ─── Emotion-Driven Section Detection ──────────────────────────────

// DetectEmotionFromLLM parses LLM mood descriptions into EmotionState.
// This bridges the gap between "nostalgic" and the numeric emotion space.
func DetectEmotionFromLLM(mood string) EmotionState {
	// Default: neutral.
	e := EmotionState{Tension: 0.4, Energy: 0.5, Warmth: 0.5, Stability: 0.6, Brightness: 0.5}

	// Simple keyword matching.
	keywords := map[string]struct {
		tension, energy, warmth, stability, brightness float64
	}{
		"calm":     {0.1, 0.2, 0.7, 0.9, 0.3},
		"sad":      {0.5, 0.2, 0.4, 0.5, 0.2},
		"happy":    {0.2, 0.7, 0.7, 0.8, 0.8},
		"angry":    {0.8, 0.9, 0.2, 0.3, 0.6},
		"peaceful": {0.1, 0.2, 0.8, 0.9, 0.4},
		"dark":     {0.7, 0.4, 0.2, 0.4, 0.1},
		"bright":   {0.2, 0.6, 0.6, 0.7, 0.9},
		"nostalgic": {0.3, 0.3, 0.7, 0.6, 0.4},
		"epic":     {0.6, 0.9, 0.4, 0.5, 0.8},
		"gentle":   {0.2, 0.2, 0.8, 0.8, 0.4},
		"intense":  {0.8, 0.9, 0.3, 0.3, 0.7},
	}
	for kw, vals := range keywords {
		if containsIgnoreCase(mood, kw) {
			e.Tension = vals.tension
			e.Energy = vals.energy
			e.Warmth = vals.warmth
			e.Stability = vals.stability
			e.Brightness = vals.brightness
			break
		}
	}

	fmt.Printf("[EmotionEngine] mood=%q → tension=%.1f energy=%.1f warmth=%.1f stability=%.1f brightness=%.1f\n",
		mood, e.Tension, e.Energy, e.Warmth, e.Stability, e.Brightness)
	return e
}
