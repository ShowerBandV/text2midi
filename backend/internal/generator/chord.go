package generator

import (
	"math/rand"
	"strings"

	"github.com/ShowerBandV/text2midi/internal/harmony"
	"github.com/ShowerBandV/text2midi/internal/schema"
)

// noteToSemi and semiToNote for extension calculations.
var chordSemi = map[string]int{
	"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3,
	"E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8,
	"Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11,
}

// buildChordVoicing returns MIDI pitches for a chord with extensions and voicing.
//   tension: 0-1 ->higher adds 7th (0.3+), 9th (0.6+)
//   darkness: 0-1 ->higher uses lower inversions
//   density: 0-1 ->higher adds doubled octaves
func buildChordVoicing(chord string, baseOct int, tension, darkness, density float64) []int {
	root := chord
	isMinor := strings.HasSuffix(chord, "m")
	if isMinor {
		root = chord[:len(chord)-1]
	}
	rootSemi, ok := chordSemi[root]
	if !ok {
		return []int{60, 64, 67}
	}

	baseMIDI := (baseOct + 1) * 12
	r := baseMIDI + rootSemi // root pitch
	t := r + 3               // minor third
	if !isMinor {
		t = r + 4 // major third
	}
	f := r + 7 // fifth

	// Build chord tone set starting with triad.
	tones := []int{r, t, f}

	// Add 7th based on tension.
	if tension > 0.3 {
		var seventh int
		if isMinor {
			seventh = r + 10 // minor 7th (e.g. Dm ->C)
		} else if tension < 0.6 {
			seventh = r + 11 // major 7th (bright)
		} else {
			seventh = r + 10 // dominant 7th (bluesy)
		}
		if seventh > 21 && seventh < 108 {
			tones = append(tones, seventh)
		}
	}

	// Add 9th at high tension.
	if tension > 0.6 {
		ninth := r + 2 // major 9th
		// Adjust to same octave region as 7th.
		if ninth < 21 {
			ninth += 12
		}
		if ninth > 108 {
			ninth -= 12
		}
		if ninth > 21 && ninth < 108 {
			tones = append(tones, ninth)
		}
	}

	// Inversion based on darkness.
	// Darkness 0-0.3: root position (keep as-is)
	// Darkness 0.3-0.7: 1st inversion (move root up octave)
	// Darkness 0.7-1.0: 2nd inversion (move root+third up octave)
	var voicing []int
	switch {
	case darkness > 0.7:
		// 2nd inversion: fifth in bass, then root+third above.
		for _, p := range tones {
			if p == r || p == t {
				voicing = append(voicing, p+12)
			} else {
				voicing = append(voicing, p)
			}
		}
	case darkness > 0.3:
		// 1st inversion: third in bass, then rest above.
		for _, p := range tones {
			if p == r {
				voicing = append(voicing, p+12)
			} else {
				voicing = append(voicing, p)
			}
		}
	default:
		voicing = tones
	}

	// Add doubled octave for high density.
	if density > 0.7 {
		voicing = append(voicing, r+12) // root + octave
	}

	// Add high extension for very bright (low darkness + high tension).
	if darkness < 0.3 && tension > 0.4 {
		voicing = append(voicing, r+12+7) // octave + fifth
	}

	// Clamp and deduplicate.
	seen := map[int]bool{}
	var out []int
	for _, p := range voicing {
		if p < 21 || p > 108 {
			continue
		}
		// Remove duplicates within same octave.
		normalized := p % 12
		if seen[normalized] && (p-r)%12 == 0 {
			continue
		}
		seen[normalized] = true
		out = append(out, p)
	}
	if len(out) == 0 {
		out = []int{r, t, f}
	}
	return out
}

// GenerateChords generates chord pad/arpeggio events following the progression.
// Tension ->7th/9th extensions, Darkness ->inversion, Density ->voicing width.
func GenerateChords(plan schema.SongPlan, track schema.ArrangementTrack) []schema.NoteEvent {
	totalBars := plan.TotalBars
	prog := plan.ChordProgression
	fv := plan.FeatureVector

	motif := makeMotif(8)

	// Arpeggiation probability.
	arpProb := fv.Density * 0.7
	// Base octave (darker = lower).
	baseOct := 3 - int(fv.Darkness*1.5)
	if baseOct < 1 {
		baseOct = 1
	}
	if baseOct > 4 {
		baseOct = 4
	}

	velBase := 45 + int(fv.Energy*35)

	var events []schema.NoteEvent
	var prevPitches []int // for voice leading between bars

	for bar := 0; bar < totalBars; bar++ {
		chord := prog[bar%len(prog)].Chord

		m := motif[bar%len(motif)]
		octOffset := 0
		if m <= 2 && fv.Darkness < 0.5 {
			octOffset = -12
		}

		// Build chord voicing with extensions + inversion.
		rawPitches := buildChordVoicing(chord, baseOct+octOffset/12, fv.Tension, fv.Darkness, fv.Density)

		// Apply voice leading from previous bar if available.
		var allNotes []int
		if len(prevPitches) > 0 && len(rawPitches) > 0 {
			allNotes = harmony.ConnectChords(prevPitches, rawPitches)
		} else {
			allNotes = rawPitches
		}
		prevPitches = make([]int, len(allNotes))
		copy(prevPitches, allNotes)

		base := float64(bar) * 4.0
		isArp := m != 0 && m != 3 && m != 6 || rand.Float64() < arpProb

		for _, pitch := range allNotes {
			if !isArp {
				dur := []float64{2.0, 3.0, 4.0}
				durIdx := bar % len(dur)
				if fv.Energy > 0.7 {
					durIdx = 0
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: pitch,
					StartBeat:    base,
					DurationBeat: dur[durIdx],
					Velocity:     velBase + rand.Intn(20),
				})
			} else {
				stepSize := 1.0
				if fv.Energy > 0.6 {
					stepSize = 0.5
				}
				step := 0.0
				for step < 4.0 {
					dur := []float64{0.25, 0.5, 0.75}
					durIdx := int(step*2) % len(dur)
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: pitch,
						StartBeat:    base + step,
						DurationBeat: dur[durIdx],
						Velocity:     velBase - 5 + rand.Intn(18),
					})
					step += stepSize
				}
			}
		}
	}
	return events
}
