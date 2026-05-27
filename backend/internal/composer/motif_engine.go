// Package composer — Motif Engine.
// Takes a short motif (3-8 notes) and generates a full song structure through
// repetition, variation, contrast, and return. This replaces LLM melody generation.
package composer

import (
	"fmt"
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ─── Motif Plan ────────────────────────────────────────────────────

type MotifPlan struct {
	UseRate         float64 // 0-1: how often motif appears
	VariationLevel  float64 // 0-1: how aggressive variations are
	CallResponse    bool
	OctaveStrategy  string // "flat", "chorus_up", "gradual"
	BarsPerPhrase   int
	TotalBars       int
}

// ─── Motif Variator ────────────────────────────────────────────────

// Transpose shifts all notes by an interval.
func Transpose(motif []int, interval int) []int {
	out := make([]int, len(motif))
	for i, n := range motif {
		out[i] = n + interval
	}
	return out
}

// Invert mirrors intervals: +2 +3 → -2 -3
func Invert(motif []int) []int {
	if len(motif) == 0 {
		return motif
	}
	out := make([]int, len(motif))
	out[0] = motif[0] // keep first note
	for i := 1; i < len(motif); i++ {
		interval := motif[i] - motif[i-1]
		out[i] = out[i-1] - interval
	}
	return out
}

// Retrograde reverses the note order.
func Retrograde(motif []int) []int {
	out := make([]int, len(motif))
	for i, n := range motif {
		out[len(motif)-1-i] = n
	}
	return out
}

// Fragment takes the first n notes.
func Fragment(motif []int, n int) []int {
	if n > len(motif) {
		n = len(motif)
	}
	out := make([]int, n)
	copy(out, motif[:n])
	return out
}

// Extend appends passing notes to lengthen the motif.
func Extend(motif []int, extra int) []int {
	if len(motif) < 2 {
		return motif
	}
	out := make([]int, len(motif)+extra)
	copy(out, motif)
	last := motif[len(motif)-1]
	avgStep := 0
	for i := 1; i < len(motif); i++ {
		avgStep += motif[i] - motif[i-1]
	}
	avgStep /= len(motif) - 1
	for i := 0; i < extra; i++ {
		out[len(motif)+i] = last + avgStep*(i+1)
	}
	return out
}

// ─── Phrase Builder ────────────────────────────────────────────────

// Phrase = 4 bars.
type Phrase struct {
	Bars [4][]int // each bar = []int of relative pitches
}

// BuildPhrase creates a 4-bar phrase from a motif.
// Structure: A (motif), A' (variation), B (contrast), A (return).
func BuildPhrase(motif []int, plan MotifPlan, rng *rand.Rand) Phrase {
	p := Phrase{}

	// Bar 0: Motif A (original or slightly varied).
	if rng.Float64() < 0.3 && plan.VariationLevel > 0.3 {
		p.Bars[0] = Transpose(motif, []int{2, 3, 5}[rng.Intn(3)])
	} else {
		p.Bars[0] = copySlice(motif)
	}

	// Bar 1: Motif A' (variation).
	switch rng.Intn(3) {
	case 0:
		p.Bars[1] = Invert(motif)
	case 1:
		p.Bars[1] = Transpose(motif, []int{-5, -3, 3, 5}[rng.Intn(4)])
	case 2:
		p.Bars[1] = Fragment(Retrograde(motif), len(motif)-1)
	}

	// Bar 2: Motif B (contrast — different rhythm/interval direction).
	if rng.Float64() < 0.5 {
		p.Bars[2] = Extend(motif, 2)
	} else {
		p.Bars[2] = Transpose(Invert(motif), 7)
	}

	// Bar 3: Motif A (return).
	if rng.Float64() < 0.4 {
		p.Bars[3] = copySlice(motif) // exact return
	} else {
		p.Bars[3] = Transpose(motif, -2) // return with slight shift
	}

	return p
}

// ─── Section Composer ──────────────────────────────────────────────

// BuildSection generates phrases for a given section type.
func BuildSection(motif []int, name string, bars int, plan MotifPlan, rng *rand.Rand) []Phrase {
	numPhrases := bars / plan.BarsPerPhrase
	if numPhrases < 1 {
		numPhrases = 1
	}

	phrases := make([]Phrase, numPhrases)
	for i := 0; i < numPhrases; i++ {

		phrase := BuildPhrase(motif, plan, rng)

		// Apply section-specific transformations.
		switch name {
		case "intro":
			// Sparser, lower register.
			for b := range phrase.Bars {
				for j := range phrase.Bars[b] {
					phrase.Bars[b][j] = phrase.Bars[b][j]/2 - 12 // lower + quieter
				}
				if len(phrase.Bars[b]) > 3 {
					phrase.Bars[b] = phrase.Bars[b][:3] // fewer notes
				}
			}
		case "verse":
			// Keep mostly original.
		case "chorus":
			// Higher register, fuller.
			for b := range phrase.Bars {
				for j := range phrase.Bars[b] {
					phrase.Bars[b][j] += 12 // octave up
				}
				// Double the motif (octave doubling).
				if len(phrase.Bars[b]) > 0 {
					doubled := make([]int, len(phrase.Bars[b])*2)
					for j, n := range phrase.Bars[b] {
						doubled[j] = n
						doubled[len(phrase.Bars[b])+j] = n + 12
					}
					phrase.Bars[b] = doubled
				}
			}
		case "bridge":
			// Invert + rhythm shift.
			for b := range phrase.Bars {
				phrase.Bars[b] = Invert(phrase.Bars[b])
			}
		case "outro":
			// Return to original, slow down.
			for b := range phrase.Bars {
				phrase.Bars[b] = copySlice(motif)
			}
		}

		phrases[i] = phrase
	}
	return phrases
}

// ─── Expand to NoteEvents ──────────────────────────────────────────

// ExpandMelody converts motif-based phrases into MIDI NoteEvents.
// basePitch: the MIDI root pitch (e.g. 60 for C4).
// bpm: used for timing calculations.
func ExpandMelody(phrases []Phrase, basePitch, bpm int) []schema.NoteEvent {
	var events []schema.NoteEvent
	bar := 0

	for _, phrase := range phrases {
		for bi, notes := range phrase.Bars {
			if len(notes) == 0 {
				bar++
				continue
			}

			beatStart := float64(bar) * 4.0
			// Distribute notes across the bar.
			notesPerBar := len(notes)
			step := 4.0 / float64(notesPerBar)

			for i, rel := range notes {
				pitch := basePitch + rel
				if pitch < 21 {
					pitch = 21
				}
				if pitch > 108 {
					pitch = 108
				}

				events = append(events, schema.NoteEvent{
					Type:         "note",
					Pitch:        pitch,
					StartBeat:    beatStart + float64(i)*step,
					DurationBeat: step * 0.8,
					Velocity:     70 + 10*(bi%3), // dynamic: increase through phrase
				})
			}
			bar++
		}
	}

	return events
}

// ─── Full Pipeline ─────────────────────────────────────────────────

// GenerateMelodyFromMotif runs the full Motif Engine pipeline.
// Takes a motif and a plan, returns a full melody as NoteEvents.
func GenerateMelodyFromMotif(motif []int, totalBars int, basePitch, bpm int) []schema.NoteEvent {
	if len(motif) < 2 {
		return nil
	}

	rng := rand.New(rand.NewSource(42))

	plan := MotifPlan{
		UseRate:         0.7,
		VariationLevel:  0.4,
		CallResponse:    true,
		OctaveStrategy:  "chorus_up",
		BarsPerPhrase:   4,
		TotalBars:       totalBars,
	}

	// Build sections.
	sections := map[string]int{
		"intro":  2,
		"verse":  4,
		"chorus": 4,
		"bridge": 2,
		"outro":  2,
	}

	var allPhrases []Phrase
	for name, bars := range sections {
		if bars <= 0 {
			continue
		}
		phrases := BuildSection(motif, name, bars, plan, rng)
		allPhrases = append(allPhrases, phrases...)
		fmt.Printf("[MotifEngine] %s: %d phrases\n", name, len(phrases))
	}

	events := ExpandMelody(allPhrases, basePitch, bpm)
	fmt.Printf("[MotifEngine] total: %d notes from %d-note motif\n", len(events), len(motif))
	return events
}

func copySlice(s []int) []int {
	out := make([]int, len(s))
	copy(out, s)
	return out
}
