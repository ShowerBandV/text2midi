// Package composer — Motif Engine (Midra-style: simple, musical).
// Generates lead melody from scale degrees with stepwise bias and random velocities.
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
// BuildSection moved to phrase.go (style-aware)

// GenerateLeadMidra is a Go port of Midra's generate_lead().
// Generates a random scale-degree motif with stepwise bias, anchors, and random velocities.
// secDensity: per-bar density [0-1]. secRegister: per-bar octave shift (nil = auto from density).
func GenerateLeadMidra(keyRoot, keyMode string, totalBars int, stepProb float64, velMin, velMax int, secDensity []float64, secRegister []int) []schema.NoteEvent {
	scale := getScaleDegrees(keyRoot, keyMode)
	if len(scale) == 0 {
		scale = []int{0, 2, 3, 5, 7, 8, 10} // fallback: C minor
	}

	rng := rand.New(rand.NewSource(42))
	motifLen := 8
	motif := make([]int, motifLen)

	// Generate random motif from scale degrees (0-6).
	for i := range motif {
		motif[i] = rng.Intn(7)
	}
	// Anchor: first note = root (0), last = root/third (0, 2, 4).
	motif[0] = 0
	motif[motifLen-1] = rng.Intn(3) * 2 // 0, 2, or 4

	// Stepwise bias: 65% chance to move by ±1 or ±2 instead of random jump.
	for i := 1; i < motifLen-1; i++ {
		if rng.Float64() < stepProb {
			step := []int{-2, -1, 1, 2}[rng.Intn(4)]
			motif[i] = motif[i-1] + step
			if motif[i] < 0 {
				motif[i] = 0
			}
			if motif[i] > 6 {
				motif[i] = 6
			}
		}
	}

	// Generate events with per-section density and register.
	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		density := 0.5 // default
		if bar < len(secDensity) && secDensity[bar] > 0 {
			density = secDensity[bar]
		}
		// Low density: only play first few notes of motif.
		// High density: full motif.
		noteCount := int(float64(motifLen) * density)
		if noteCount < 2 { noteCount = 2 }
		if noteCount > motifLen { noteCount = motifLen }
		// Register: use explicit section register if provided, else auto from density.
		octave := 5
		if bar < len(secRegister) && secRegister[bar] != 0 {
			octave = secRegister[bar]
		} else {
			if density > 0.7 {
				octave = 6
			}
			if density < 0.3 {
				octave = 4
			}
		}

		for i := 0; i < noteCount; i++ {
			step := motif[i]
			scaleIdx := step % len(scale)
			if scaleIdx < 0 { scaleIdx += len(scale) }
			pitch := scale[scaleIdx] + 12*(octave-1)
			velocity := velMin + rng.Intn(velMax-velMin)
			duration := []float64{0.25, 0.4, 0.5, 0.75}[rng.Intn(4)]
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat:    base + float64(i)*0.5,
				DurationBeat: duration,
				Velocity:     velocity,
			})
		}
	}

	fmt.Printf("[MidraLead] %d-note motif from %d-note scale, %d events, %d bars\n",
		motifLen, len(scale), len(events), totalBars)
	return events
}

// getScaleDegrees returns MIDI-compatible scale pitches for a given key.
func getScaleDegrees(root, mode string) []int {
	rootSemi := map[string]int{
		"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5,
		"F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11,
	}
	rs, ok := rootSemi[root]
	if !ok {
		rs = 0
	}

	var intervals []int
	switch mode {
	case "minor", "natural_minor":
		intervals = []int{0, 2, 3, 5, 7, 8, 10}
	default:
		intervals = []int{0, 2, 4, 5, 7, 9, 11}
	}

	noteNames := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

	result := make([]int, 0, len(intervals)*4)
	for oct := 3; oct <= 6; oct++ {
		base := (oct + 1) * 12
		for _, iv := range intervals {
			p := base + rs + iv
			if p >= 21 && p <= 108 {
				result = append(result, p)
			}
		}
	}
	_ = noteNames
	return result
}

func copySlice(s []int) []int {
	out := make([]int, len(s))
	copy(out, s)
	return out
}

