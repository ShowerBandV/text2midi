// Package composer — Chord Substitution Engine.
// Replaces generic chord progressions with sophisticated alternatives:
// - Secondary dominants (V7/vi, V7/ii, etc.)
// - Modal mixture (borrowed chords from parallel key)
// - Tritone substitutions
// - Deceptive cadences
package composer

import (
	"math/rand"
	"strings"

	"github.com/yourname/text2midi/internal/schema"
)

// ChordSubstituter applies harmonic substitutions to chord progressions.
type ChordSubstituter struct {
	rng *rand.Rand
}

// NewChordSubstituter creates a substituter with a given seed.
func NewChordSubstituter(seed int64) *ChordSubstituter {
	return &ChordSubstituter{rng: rand.New(rand.NewSource(seed))}
}

// ApplySubstitutions processes a chord progression and substitutes generic chords.
// tension controls how adventurous the substitutions are (0.0-1.0).
func (cs *ChordSubstituter) ApplySubstitutions(prog []schema.ChordChange, key string, tension float64) []schema.ChordChange {
	if len(prog) < 2 {
		return prog
	}

	// Determine key root and mode.
	keyRoot := key
	keyIsMinor := false
	if fields := strings.Fields(key); len(fields) >= 2 {
		keyRoot = fields[0]
		keyIsMinor = fields[1] == "minor" || fields[1] == "natural_minor"
	}

	result := make([]schema.ChordChange, len(prog))
	copy(result, prog)

	substitutionRate := tension * 0.4 // up to 40% of chords get substituted

	for i := 0; i < len(result); i++ {
		if cs.rng.Float64() >= substitutionRate {
			continue
		}

		chord := result[i].Chord
		sub := cs.findSubstitution(chord, keyRoot, keyIsMinor, i, result)
		if sub != "" {
			result[i].Chord = sub
		}
	}

	return result
}

// findSubstitution returns a substitute chord for a given chord symbol.
func (cs *ChordSubstituter) findSubstitution(chord, keyRoot string, keyIsMinor bool, pos int, prog []schema.ChordChange) string {
	// Extract root and quality.
	root := chord
	isMinor := strings.HasSuffix(chord, "m")
	if isMinor {
		root = chord[:len(chord)-1]
	}

	// Remove any extensions for matching.
	cleanRoot := root
	for _, suf := range []string{"maj7", "m7", "7", "maj", "dim", "aug", "sus4", "sus2", "9", "11", "13"} {
		if strings.HasSuffix(cleanRoot, suf) {
			cleanRoot = strings.TrimSuffix(cleanRoot, suf)
		}
	}

	// Substitution rules based on harmonic function.
	// I → iii or vi (tonic substitution)
	// ii → IV (subdominant substitution)
	// IV → ii (subdominant substitution)
	// V → vii° (dominant substitution) or add secondary dominant
	// vi → I (tonic substitution) or add V7/vi

	// Pre-chorus / build section: add secondary dominants.
	if pos > 0 && pos < len(prog)-1 && cs.rng.Float64() < 0.5 {
		// Before a minor chord, insert its secondary dominant (V7/vi, V7/ii).
		nextRaw := ""
		if pos+1 < len(prog) {
			nextRaw = strings.TrimSuffix(strings.TrimSuffix(prog[pos+1].Chord, "m"), "7")
		}
		if nextRaw != "" {
			// Build secondary dominant: V7 of the next chord.
			semi := map[string]int{"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3, "E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8, "Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11}
			nextSemi, ok := semi[nextRaw]
			if ok {
				// Dominant of target = target + 7 semitones.
				domSemi := (nextSemi + 7) % 12
				semiToNote := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
				domRoot := semiToNote[domSemi]
				// Check if it's the same as current root.
				curSemi, _ := semi[strings.TrimSuffix(strings.TrimSuffix(chord, "m"), "7")]
				if domSemi != curSemi {
					return domRoot + "7" // Secondary dominant
				}
			}
		}
	}

	// Substitute based on chord quality.
	switch {
	case cleanRoot == keyRoot && !isMinor && pos > 0:
		// Tonic chord in major: substitute with vi or iii.
		choices := []string{"Am7", "Em7"}
		return choices[cs.rng.Intn(len(choices))]

	case isMinor && pos < len(prog)-1:
		// Minor chord (not last): substitute with IV of parallel major.
		// e.g., Am → F (subdominant of C major)
		return "F"

	case !isMinor && pos == len(prog)-1:
		// Last chord: deceptive cadence.
		// Instead of resolving to I, go to vi.
		if keyIsMinor {
			return "VI" // bVI in minor
		}
		return "Am" // vi in major

	default:
		return "" // No substitution
	}
}
