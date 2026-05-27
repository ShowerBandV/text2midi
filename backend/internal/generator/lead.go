package generator

import (
	"fmt"
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/music"
	"github.com/ShowerBandV/text2midi/internal/schema"
)

// GenerateLead generates a lead melody with phrase-aware contour constraints.
// Phrase rules: start on root/fifth, rise then fall, resolve at phrase end.
// Feature vector: Darkness ->interval, Energy ->density, Tension ->chromaticism.
func GenerateLead(plan schema.SongPlan, track schema.ArrangementTrack) []schema.NoteEvent {
	root := plan.Key.Root
	scaleName := plan.Key.Scale
	if scaleName == "" {
		scaleName = "natural_minor"
	}
	fv := plan.FeatureVector

	scale, err := music.GetScale(root, scaleName)
	if err != nil {
		scale, _ = music.GetScale("C", "natural_minor")
	}

	// Phrase: 4 bars = one phrase. Total bars = N phrases.
	barsPerPhrase := 4
	numPhrases := (plan.TotalBars + barsPerPhrase - 1) / barsPerPhrase

	// Notes per bar based on energy.
	notesPerBar := 3 + int(fv.Energy*8)
	if notesPerBar < 3 {
		notesPerBar = 3
	}
	if notesPerBar > 12 {
		notesPerBar = 12
	}
	notesPerPhrase := notesPerBar * barsPerPhrase

	// Build phrase contour: arc shape (low ->high ->low).
	// Motif values are scale degree indices (0-6).
	phraseMotif := make([]int, notesPerPhrase)
	peakPos := rand.Intn(notesPerPhrase-2) + 1 // not first or last

	// Fill with arc: rise to peak, fall to end.
	for i := 0; i < notesPerPhrase; i++ {
		progress := float64(i) / float64(notesPerPhrase-1)
		peakRatio := float64(peakPos) / float64(notesPerPhrase-1)
		// Arc formula: rise to peakPos, fall to end.
		var height float64
		if i <= peakPos {
			height = progress / peakRatio // 0->
		} else {
			height = (1.0 - progress) / (1.0 - peakRatio) // 1->
		}

		// Scale to motif range.
		motifRange := 6 - int(fv.Darkness*3) // dark=3-6, bright=3-6
		if motifRange < 3 {
			motifRange = 3
		}
		val := int(height * float64(motifRange))
		if val < 0 {
			val = 0
		}
		if val > 6 {
			val = 6
		}
		phraseMotif[i] = val
	}

	// First note should be root or fifth (0 or 4 in scale).
	phraseMotif[0] = []int{0, 4}[rand.Intn(2)]
	// Last note resolves to root (0) for closure.
	phraseMotif[notesPerPhrase-1] = 0

	// Smooth transitions: limit interval jumps.
	for i := 1; i < notesPerPhrase-1; i++ {
		prev := phraseMotif[i-1]
		if phraseMotif[i]-prev > 3 {
			phraseMotif[i] = prev + 2
		}
		if prev-phraseMotif[i] > 3 {
			phraseMotif[i] = prev - 2
		}
	}

	// Octave based on darkness.
	octave := 5 - int(fv.Darkness*1.5)
	if octave < 3 {
		octave = 3
	}
	if octave > 5 {
		octave = 5
	}

	// Duration pattern: long+short variety (not all same).
	durationPattern := []float64{0.5, 0.25, 0.75, 0.25, 0.5, 0.125, 0.5, 0.375}

	tensionProb := fv.Tension * 0.25
	lofiJitter := fv.LoFi * 0.04
	velBase := 70 + int(fv.Energy*35)
	stepDuration := 4.0 / float64(notesPerBar)

	var events []schema.NoteEvent
	for phrase := 0; phrase < numPhrases; phrase++ {
		phraseStartBar := phrase * barsPerPhrase
		for i := 0; i < notesPerPhrase; i++ {
			bar := phraseStartBar + i/notesPerBar
			if bar >= plan.TotalBars {
				break
			}
			beatInBar := float64(i%notesPerBar) * stepDuration
			base := float64(bar) * 4.0

			step := phraseMotif[i]
			scaleIdx := step % len(scale)

			pitch, err := music.NoteNameToMIDI(fmt.Sprintf("%s%d", scale[scaleIdx], octave))
			if err != nil {
				pitch = 60 + octave*12
			}

			// Chromatic tension.
			if rand.Float64() < tensionProb {
				pitch += []int{-1, 1}[rand.Intn(2)]
				if pitch < 24 {
					pitch = 24
				}
				if pitch > 108 {
					pitch = 108
				}
			}

			jitter := 0.0
			if lofiJitter > 0 {
				jitter = (rand.Float64() - 0.5) * 2 * lofiJitter
			}

			vel := velBase + rand.Intn(20)
			if vel > 127 {
				vel = 127
			}

			dur := durationPattern[i%len(durationPattern)]
			if fv.Energy < 0.3 {
				dur = []float64{1.0, 1.5, 2.0}[i%3] // longer, sustained
			}

			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat:    base + beatInBar + jitter,
				DurationBeat: dur,
				Velocity:     vel,
			})
		}
	}
	return events
}
