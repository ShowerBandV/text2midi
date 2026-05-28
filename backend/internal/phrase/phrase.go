// Package phrase — Phrase-level composition system.
// Music is structured in phrases (typically 4 bars), not individual bars.
// A phrase has a question (bars 1-2) and answer (bars 3-4) structure.
package phrase

import (
	"fmt"
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// Phrase is a 4-bar musical statement with tension and resolution.
type Phrase struct {
	Bars        [4][]int   // relative pitches per bar
	Rhythm      [4][]float64 // durations per bar
	Tension     float64    // 0-1: how unresolved this phrase feels
	Resolution  float64    // 0-1: how much it resolves
	HasCallback bool       // does this phrase reference a previous motif?
}

// Builder creates phrases from motifs.
type Builder struct {
	Motif []int
	Style string // "metal", "pop", "hiphop", "ambient"
	RNG   *rand.Rand
}

// NewBuilder creates a phrase builder.
func NewBuilder(motif []int, style string) *Builder {
	return &Builder{
		Motif: motif,
		Style: style,
		RNG:   rand.New(rand.NewSource(42)),
	}
}

// BuildPhrase creates one 4-bar phrase.
// tension controls how much variation to apply (0=repeat, 1=wild).
func (b *Builder) BuildPhrase(tension float64, sectionName string) Phrase {
	p := Phrase{
		Tension:    tension,
		Resolution: 1.0 - tension,
	}

	if b.Motif == nil || len(b.Motif) < 2 {
		b.Motif = []int{0, 2, 4, 3, 0}
	}

	switch b.Style {
	case "metal":
		b.buildMetal(&p)
	case "pop":
		b.buildPop(&p, tension)
	case "hiphop":
		b.buildHipHop(&p, sectionName)
	case "ambient":
		b.buildAmbient(&p, tension)
	default:
		b.buildPop(&p, tension)
	}

	return p
}

// metalPhrase: riff repetition with octave drops.
func (b *Builder) buildMetal(p *Phrase) {
	m := b.Motif
	// Bar 0: motif
	p.Bars[0] = copySlice(m)
	p.Rhythm[0] = repeatDuration(len(m), 0.48)
	// Bar 1: octave drop
	p.Bars[1] = transpose(m, -12)
	p.Rhythm[1] = repeatDuration(len(m), 0.48)
	// Bar 2: motif again
	p.Bars[2] = copySlice(m)
	p.Rhythm[2] = repeatDuration(len(m), 0.48)
	// Bar 3: fragment + slight variation
	p.Bars[3] = transpose(fragment(m, len(m)/2+1), 5)
	p.Rhythm[3] = repeatDuration(len(p.Bars[3]), 0.48)
}

// popPhrase: A-A'-B-A structure, singable.
func (b *Builder) buildPop(p *Phrase, tension float64) {
	m := b.Motif
	// Bar 0: A = motif
	p.Bars[0] = copySlice(m)
	p.Rhythm[0] = repeatDuration(len(m), 0.5+float64(b.RNG.Intn(3))*0.05)
	// Bar 1: A' = transpose up
	p.Bars[1] = transpose(m, 3)
	p.Rhythm[1] = p.Rhythm[0]
	// Bar 2: B = contrast
	if b.RNG.Float64() < 0.5 {
		p.Bars[2] = invert(m)
	} else {
		p.Bars[2] = fragment(m, len(m)-1)
	}
	p.Rhythm[2] = repeatDuration(len(p.Bars[2]), 0.4)
	// Bar 3: A = return
	p.Bars[3] = copySlice(m)
	p.Rhythm[3] = p.Rhythm[0]
}

// hiphopPhrase: loop-based, subtle changes.
func (b *Builder) buildHipHop(p *Phrase, sectionName string) {
	m := b.Motif
	if len(m) > 4 {
		m = m[:4]
	}
	// All bars use same loop
	for i := 0; i < 4; i++ {
		p.Bars[i] = copySlice(m)
		p.Rhythm[i] = repeatDuration(len(m), 0.5)
	}
	// Last bar slight variation
	if sectionName == "loop_b" {
		p.Bars[3] = transpose(m, 3)
	}
}

// ambientPhrase: sparse, evolving.
func (b *Builder) buildAmbient(p *Phrase, tension float64) {
	m := fragment(b.Motif, 2)
	shift := int(tension * 5)
	m = transpose(m, shift)
	for i := 0; i < 4; i++ {
		p.Bars[i] = transpose(m, i*2)
		p.Rhythm[i] = []float64{2.0, 2.0}
	}
}

// ExpandToNoteEvents converts phrases to schema.NoteEvent.
func (p Phrase) Expand(basePitch, bpm int, startBar int, energy float64) []schema.NoteEvent {
	var events []schema.NoteEvent
	for bi := 0; bi < 4; bi++ {
		notes := p.Bars[bi]
		durs := p.Rhythm[bi]
		if len(notes) == 0 {
			continue
		}
		barStart := float64(startBar+bi) * 4.0
		step := 4.0 / float64(len(notes))

		for i, rel := range notes {
			pitch := basePitch + rel
			if pitch < 21 {
				pitch = 21
			}
			if pitch > 108 {
				pitch = 108
			}
			dur := step * 0.85
			if i < len(durs) && durs[i] > 0 {
				dur = durs[i]
			}
			vel := 60 + int(energy*40)
			if i == 0 {
				vel += 15 // accent first note
			}

			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat:    barStart + float64(i)*step,
				DurationBeat: dur,
				Velocity:     vel,
			})
		}
	}
	return events
}

// helpers
func copySlice(s []int) []int {
	out := make([]int, len(s))
	copy(out, s)
	return out
}

func transpose(s []int, n int) []int {
	out := make([]int, len(s))
	for i, v := range s {
		out[i] = v + n
	}
	return out
}

func invert(s []int) []int {
	if len(s) == 0 {
		return s
	}
	out := make([]int, len(s))
	out[0] = s[0]
	for i := 1; i < len(s); i++ {
		out[i] = out[i-1] - (s[i] - s[i-1])
	}
	return out
}

func fragment(s []int, n int) []int {
	if n > len(s) {
		n = len(s)
	}
	out := make([]int, n)
	copy(out, s[:n])
	return out
}

func repeatDuration(n int, d float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = d
	}
	return out
}

// For logging
func init() {
	_ = fmt.Sprintf
}
