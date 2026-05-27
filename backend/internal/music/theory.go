// Package music provides music theory utilities: note↔MIDI conversion,
// scale/chord generation, and drum mapping.
// Ported from music_agent/core/music_theory.py and core/drum_map.py.
package music

import (
	"fmt"
	"math"
	"strings"
)

// Note-to-semitone mapping (same as NOTE_TO_SEMITONE in Python).
var noteToSemitone = map[string]int{
	"C": 0, "C#": 1, "Db": 1,
	"D": 2, "D#": 3, "Eb": 3,
	"E": 4,
	"F": 5, "F#": 6, "Gb": 6,
	"G": 7, "G#": 8, "Ab": 8,
	"A": 9, "A#": 10, "Bb": 10,
	"B": 11,
}

// Semitone index to canonical note name.
var semitoneToNote = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

// NoteNameToMIDI converts a note name like "C4" to a MIDI pitch.
// Supports negative octaves: "C-1" = 0, "C4" = 60, "A4" = 69.
func NoteNameToMIDI(noteName string) (int, error) {
	if len(noteName) < 2 {
		return 0, fmt.Errorf("invalid note name: %q", noteName)
	}
	// Split note name at the first digit to get note part and octave part.
	// Handles "C4", "C-1", "Bb3", "F#5", etc.
	i := 0
	for i < len(noteName) && (noteName[i] < '0' || noteName[i] > '9') {
		if i > 0 && noteName[i] == '-' {
			break // octave sign starts at the digit after '-'
		}
		i++
	}
	if i == 0 || i > len(noteName)-1 {
		return 0, fmt.Errorf("invalid note name: %q", noteName)
	}
	name := noteName[:i]
	octaveStr := noteName[i:]
	octave := 0
	for _, c := range octaveStr {
		if c == '-' {
			continue
		}
		octave = octave*10 + int(c-'0')
	}
	if octaveStr[0] == '-' {
		octave = -octave
	}
	semi, ok := noteToSemitone[name]
	if !ok {
		return 0, fmt.Errorf("unknown note: %q", name)
	}
	return (octave+1)*12 + semi, nil
}

// MIDIToNoteName converts a MIDI pitch to a note name like "C4".
func MIDIToNoteName(midiNote int) string {
	octave := midiNote/12 - 1
	note := semitoneToNote[midiNote%12]
	return fmt.Sprintf("%s%d", note, octave)
}

// GetScale returns the note names in a given scale.
// Supports "major" and "natural_minor" (or "minor").
func GetScale(root, scale string) ([]string, error) {
	rootSemi, ok := noteToSemitone[root]
	if !ok {
		return nil, fmt.Errorf("unknown root note: %q", root)
	}
	var intervals []int
	switch strings.ToLower(scale) {
	case "major":
		intervals = []int{0, 2, 4, 5, 7, 9, 11}
	case "natural_minor", "minor":
		intervals = []int{0, 2, 3, 5, 7, 8, 10}
	default:
		return nil, fmt.Errorf("unsupported scale: %q", scale)
	}
	result := make([]string, len(intervals))
	for i, interval := range intervals {
		result[i] = semitoneToNote[(rootSemi+interval)%12]
	}
	return result, nil
}

// ParseChord parses extended chord symbols: C, Dm, G7, Cmaj7, Dm7, Am9, Fsus4,
// Cdim, Caug, Cm7b5, Cdim7, Cmaj9, C9, Cm9, and slash chords like C/G, Dm/F, Am/C, G/B.
// Returns note names for the full chord voicing.
func ParseChord(chord string) ([]string, error) {
	// Handle slash chords: "C/G" → root C, bass G
	bassNote := ""
	if idx := strings.Index(chord, "/"); idx >= 0 {
		bassNote = chord[idx+1:]
		chord = chord[:idx]
	}

	// Determine suffix type
	suffix := ""
	root := chord
	for _, suf := range []string{
		"maj7b5", "maj7#5", "m7b5", "m7#5", "dim7", "aug7",
		"maj7", "maj9", "maj13",
		"m7", "m9", "m11", "m13",
		"7sus4", "7#9", "7b9", "7#11", "7b13",
		"dim", "aug", "sus4", "sus2",
		"7", "9", "11", "13",
		"m", "",
	} {
		if strings.HasSuffix(root, suf) {
			suffix = suf
			root = root[:len(root)-len(suf)]
			if root == "" {
				root = suf
				suffix = ""
			}
			break
		}
	}
	if root == "" {
		return nil, fmt.Errorf("invalid chord: %q", chord)
	}

	rootSemi, ok := noteToSemitone[root]
	if !ok {
		return nil, fmt.Errorf("unknown chord root: %q", root)
	}

	// Build intervals relative to root (in semitones).
	type interval struct {
		name     string
		semitones int
	}
	var intervals []interval

	// Start with root.
	intervals = append(intervals, interval{root, 0})

	// Third / sus
	isMinor := false
	switch {
	case suffix == "sus4":
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+5)%12], 5})
	case suffix == "sus2":
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+2)%12], 2})
	case strings.HasPrefix(suffix, "m"):
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+3)%12], 3})
		isMinor = true
	default:
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+4)%12], 4})
	}

	// Fifth
	isDim := strings.HasPrefix(suffix, "dim") || suffix == "m7b5"
	isAug := strings.HasPrefix(suffix, "aug")
	switch {
	case isDim && suffix == "dim7":
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+6)%12], 6})
	case isDim:
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+6)%12], 6})
	case isAug:
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+8)%12], 8})
	default:
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+7)%12], 7})
	}

	// Seventh - logic:
	// "maj7" = major 7th (11), "maj9" also uses major 7th
	// "7" = dominant 7th (10), "9" also uses dominant 7th
	// "m7" = minor 7th (10), "m9" also uses minor 7th
	// "dim7" = diminished 7th (9)
	hasSeventh := false
	hasNinth := false
	switch {
	case suffix == "maj7" || suffix == "maj9" || suffix == "maj13":
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+11)%12], 11})
		hasSeventh = true
	case suffix == "7" || suffix == "9" || suffix == "11" || suffix == "13" || suffix == "7sus4":
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+10)%12], 10})
		hasSeventh = true
	case suffix == "m7" || suffix == "m9" || suffix == "m11":
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+10)%12], 10})
		hasSeventh = true
	case suffix == "dim7":
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+9)%12], 9})
		hasSeventh = true
	case suffix == "m7b5":
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+10)%12], 10})
		hasSeventh = true
	}

	// Ninth
	if suffix == "maj9" || suffix == "m9" || suffix == "9" {
		intervals = append(intervals, interval{semitoneToNote[(rootSemi+2)%12], 2})
		hasNinth = true
	}

	// Handle altered chords (7#9, 7b9, etc.)
	if strings.Contains(suffix, "#9") || strings.Contains(suffix, "b9") {
		// Already have seventh from "7". Add altered ninth.
		if strings.Contains(suffix, "#9") {
			intervals = append(intervals, interval{semitoneToNote[(rootSemi+3)%12], 3})
		} else {
			intervals = append(intervals, interval{semitoneToNote[(rootSemi+1)%12], 1})
		}
		hasNinth = true
	}

	result := make([]string, 0, len(intervals))
	for _, iv := range intervals {
		result = append(result, iv.name)
	}

	// Add bass note from slash chord if present.
	if bassNote != "" {
		if bassSemi, ok := noteToSemitone[bassNote]; ok {
			result = append(result, bassNote)
			_ = bassSemi // bass note name stored
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("empty chord: %q", chord)
	}

	_ = hasSeventh
	_ = hasNinth
	_ = isMinor
	return result, nil
}

// ChordToMIDINotes converts a chord symbol to MIDI pitches at a given octave.
// E.g., ChordToMIDINotes("Cm", 3) ->[48, 51, 55] (C3, Eb3, G3).
func ChordToMIDINotes(chord string, baseOctave int) ([]int, error) {
	notes, err := ParseChord(chord)
	if err != nil {
		return nil, err
	}
	result := make([]int, len(notes))
	for i, n := range notes {
		pitch, err := NoteNameToMIDI(fmt.Sprintf("%s%d", n, baseOctave))
		if err != nil {
			return nil, err
		}
		result[i] = pitch
	}
	return result, nil
}

// RootPitch returns the MIDI pitch of a chord's root note at C1-based octave.
// Used by bass generator. C1 = 24, so root C ->24, D ->26, etc.
func RootPitch(chord string) (int, error) {
	root := chord
	if strings.HasSuffix(chord, "m") {
		root = chord[:len(chord)-1]
	}
	semi, ok := noteToSemitone[root]
	if !ok {
		return 0, fmt.Errorf("unknown root note: %q", root)
	}
	return 24 + semi, nil // C1 = 24 base
}

// ClampInt clamps an integer to a range [min, max].
func ClampInt(val, minVal, maxVal int) int {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

// RoundToInt rounds a float64 to the nearest int.
func RoundToInt(x float64) int {
	return int(math.Round(x))
}
