// Package harmony provides harmonic constraint checking for melody generation.

// Ensures all notes stay in-key, prioritize chord tones on strong beats,

// and avoid harsh intervals (tritone, minor 2nd, major 7th in exposed positions).

package harmony



import (

	"fmt"

	"sort"



	"github.com/ShowerBandV/text2midi/internal/schema"

)



// noteToSemi maps note names to semitone offsets.

var noteToSemi = map[string]int{

	"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3,

	"E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8,

	"Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11,

}



// semiToNote maps semitone offsets to note names.

var semiToNote = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}



// blueNoteOffsets defines the semitone intervals (from key root) that are

// tolerated as "blue notes" in blues/hard rock contexts instead of being

// snapped to the nearest in-key pitch. b3=3, b5=6, b7=10 in major.

var blueNoteOffsets = map[int]bool{3: true, 6: true, 10: true}



// Constraint holds precomputed harmonic data for a song.

type Constraint struct {

	KeyRoot      string

	KeyMode      string

	ScaleSemitones []int // semitone offsets of the scale (e.g. C minor: [0,2,3,5,7,8,10])

	KeyPitches   []int  // all MIDI pitches in key within 21-108

	PentPitches  []int  // pentatonic subset (root, m3, P4, P5, m7)

	Bars         int

	BluesTolerant bool  // when true, blue notes (b3, b5, b7) are NOT snapped to scale

	ChordAtBar   func(bar int) (root string, isMinor bool) // returns chord root + quality at bar

}



// BuildConstraint precomputes scale data from a song plan.

// Detects blues/hard rock from the feature vector's Tension dimension.

func BuildConstraint(plan *schema.SongPlan) *Constraint {

	c := &Constraint{

		KeyRoot:       plan.Key.Root,

		KeyMode:       plan.Key.Mode,

		Bars:          plan.TotalBars,

		BluesTolerant: plan.FeatureVector.Tension > 0.3,

	}



	// Build scale semitones.

	switch plan.Key.Mode {

	case "minor", "natural_minor":

		c.ScaleSemitones = []int{0, 2, 3, 5, 7, 8, 10}

	case "major":

		c.ScaleSemitones = []int{0, 2, 4, 5, 7, 9, 11}

	default:

		c.ScaleSemitones = []int{0, 2, 3, 5, 7, 8, 10}

	}



	// Build key pitches across MIDI range 21-108.

	rootSemi, ok := noteToSemi[plan.Key.Root]

	if !ok {

		rootSemi = 0

	}

	for midi := 21; midi <= 108; midi++ {

		noteInOctave := (midi - rootSemi) % 12

		if noteInOctave < 0 {

			noteInOctave += 12

		}

		for _, s := range c.ScaleSemitones {

			if noteInOctave == s {

				c.KeyPitches = append(c.KeyPitches, midi)

				break

			}

		}

	}



	// Build pentatonic subset: root, minor third, fourth, fifth, minor seventh.

	pentOffsets := []int{0, 3, 5, 7, 10}

	for _, midi := range c.KeyPitches {

		noteInOctave := (midi - rootSemi) % 12

		if noteInOctave < 0 {

			noteInOctave += 12

		}

		for _, po := range pentOffsets {

			if noteInOctave == po {

				c.PentPitches = append(c.PentPitches, midi)

				break

			}

		}

	}



	// Build chord-at-bar lookup.

	c.ChordAtBar = func(bar int) (string, bool) {

		if plan == nil || len(plan.ChordProgression) == 0 {

			return plan.Key.Root, plan.Key.Mode == "minor"

		}

		idx := bar % len(plan.ChordProgression)

		if idx >= len(plan.ChordProgression) {

			idx = len(plan.ChordProgression) - 1

		}

		chord := plan.ChordProgression[idx].Chord

		if len(chord) == 0 {

			return plan.Key.Root, plan.Key.Mode == "minor"

		}

		isMinor := chord[len(chord)-1] == 'm'

		if isMinor {

			return chord[:len(chord)-1], true

		}

		return chord, false

	}



	return c

}



// ChordTones returns the triad for a chord root at a given octave.

func (c *Constraint) ChordTones(root string, isMinor bool, octave int) []int {

	rootSemi, ok := noteToSemi[root]

	if !ok {

		rootSemi = 0

	}

	base := (octave + 1) * 12

	third := 4

	if isMinor {

		third = 3

	}

	return []int{

		base + rootSemi,           // root

		base + rootSemi + third,   // third

		base + rootSemi + 7,       // fifth

		base + rootSemi + 12,      // octave

	}

}



// InKey returns true if a MIDI pitch is in the key scale.

func (c *Constraint) InKey(pitch int) bool {

	for _, p := range c.KeyPitches {

		if p == pitch {

			return true

		}

	}

	return false

}



// NearestInKey returns the closest in-key pitch to the given pitch.

func (c *Constraint) NearestInKey(pitch int) int {

	if len(c.KeyPitches) == 0 {

		return pitch

	}

	// Binary search for closest.

	idx := sort.SearchInts(c.KeyPitches, pitch)

	if idx >= len(c.KeyPitches) {

		return c.KeyPitches[len(c.KeyPitches)-1]

	}

	if idx == 0 {

		return c.KeyPitches[0]

	}

	// Compare pitch with both neighbors.

	low := c.KeyPitches[idx-1]

	high := c.KeyPitches[idx]

	if pitch-low < high-pitch {

		return low

	}

	return high

}



// IsHarshInterval returns true if the melodic interval is harsh for melody.

func IsHarshInterval(prev, curr int) bool {

	interval := prev - curr

	if interval < 0 {

		interval = -interval

	}

	interval %= 12

	// Tritone (6 semitones), minor 2nd (1), major 7th (11).

	return interval == 6 || interval == 1 || interval == 11

}



// SnapToScale post-processes all events to snap out-of-scale notes to the nearest

// in-key pitch. For melody tracks, it also applies chord-tone priority on strong beats.

func SnapToScale(eventsByTrack map[string][]schema.NoteEvent, c *Constraint) {

	for trackID := range eventsByTrack {

		if trackID == "drums" {

			continue // drums don't need harmonic correction

		}

		track := eventsByTrack[trackID]

		for i := range track {

			e := &track[i]

			bar := int(e.StartBeat) / 4

			if bar >= c.Bars {

				bar = c.Bars - 1

			}

			if bar < 0 {

				bar = 0

			}



			// Step 1: If blues-tolerant, check if this is a blue note before snapping.

			if !c.InKey(e.Pitch) && c.BluesTolerant {

				// Compute semitone offset from key root.

				rootSemi, _ := noteToSemi[c.KeyRoot]

				offset := (e.Pitch - rootSemi) % 12

				if offset < 0 {

					offset += 12

				}

				if blueNoteOffsets[offset] {

					// Keep the blue note --it adds character.

					continue

				}

			}



			// Step 2: Snap to in-key if outside scale.

			if !c.InKey(e.Pitch) {

				e.Pitch = c.NearestInKey(e.Pitch)

			}



			// Step 3: On strong beats (beat 1, beat 3), prefer chord tones.

			beatPos := int(e.StartBeat*4) % 4

			isStrongBeat := beatPos == 0 || beatPos == 2

			if isStrongBeat {

				root, minor := c.ChordAtBar(bar)

				chordTones := c.ChordTones(root, minor, e.Pitch/12-1)

				// If pitch is not a chord tone, snap to nearest chord tone.

				isChordTone := false

				for _, ct := range chordTones {

					if ct == e.Pitch {

						isChordTone = true

						break

					}

				}

				if !isChordTone {

					// Find nearest chord tone.

					nearest := chordTones[0]

					bestDist := 999

					for _, ct := range chordTones {

						d := e.Pitch - ct

						if d < 0 {

							d = -d

						}

						if d < bestDist {

							bestDist = d

							nearest = ct

						}

					}

					if nearest >= 21 && nearest <= 108 {

						e.Pitch = nearest

					}

				}

			}

		}



		// Step 3: Fix harsh melodic intervals.

		for i := 1; i < len(track); i++ {

			prev := track[i-1]

			curr := &track[i]

			gap := curr.StartBeat - (prev.StartBeat + prev.DurationBeat)



			// Only fix if notes are close in time (within 1 beat).

			if gap < 1.0 && IsHarshInterval(prev.Pitch, curr.Pitch) {

				// Move current pitch up/down 1-2 semitones in scale.

				candidates := []int{

					c.NearestInKey(curr.Pitch + 1),

					c.NearestInKey(curr.Pitch - 1),

					c.NearestInKey(curr.Pitch + 2),

					c.NearestInKey(curr.Pitch - 2),

					prev.Pitch,

				}

				best := curr.Pitch

				bestDist := 999

				for _, cand := range candidates {

					if cand < 21 || cand > 108 {

						continue

					}

					if IsHarshInterval(prev.Pitch, cand) {

						continue

					}

					d := cand - curr.Pitch

					if d < 0 {

						d = -d

					}

					if d < bestDist {

						bestDist = d

						best = cand

					}

				}

				curr.Pitch = best

			}

		}

	}

}



// Describe returns a human-readable summary of the harmonic setup.

func (c *Constraint) Describe() string {

	scaleNames := make([]string, len(c.ScaleSemitones))

	rootSemi, _ := noteToSemi[c.KeyRoot]

	for i, s := range c.ScaleSemitones {

		noteName := semiToNote[(s+rootSemi)%12]

		scaleNames[i] = noteName

	}

	return fmt.Sprintf("%s %s scale: [%s]  pentatonic subset: [root, m3, P4, P5, m7]  %d in-key pitches",

		c.KeyRoot, c.KeyMode, joinStrings(scaleNames, " "), len(c.KeyPitches))

}



func joinStrings(a []string, sep string) string {

	if len(a) == 0 {

		return ""

	}

	s := a[0]

	for _, v := range a[1:] {

		s += sep + v

	}

	return s

}

