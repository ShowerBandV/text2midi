// Package composer — generation context for DNA-driven composition.
package composer

import (
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/musicdna"
)

// GenerationContext bundles all parameters for ComposeSong.
// Consolidates the growing list of float64 params into one struct,
// enabling DNA-driven and emotion-driven generation.
type GenerationContext struct {
	// Core musical parameters.
	Motif      []int    // pitch-interval seed, e.g. [0,2,4,3,0]
	Chords     []string // chord progression, e.g. ["C","G","Am","F"]
	TotalBars  int
	BasePitch  int // MIDI reference note (60 = C4)
	BPM        int
	RNG        *rand.Rand

	// Feature vector (4 dimensions — passed to sub-generators).
	Darkness  float64 // 0-1
	Energy    float64 // 0-1
	Rhythmic  float64 // 0-1 rhythmic complexity
	Tension   float64 // 0-1 harmonic tension

	// ComposerDNA personality (from archetype or DNA library).
	DNA *ComposerDNA

	// MotifSource controls motif selection: "dna", "default", "random"
	MotifSource string

	// Emotion curve for bar-granular dynamics.
	EmotionCurve *EmotionCurve
}

// NewDefaultContext creates a GenerationContext with sensible defaults.
func NewDefaultContext(totalBars, bpm int) *GenerationContext {
	return &GenerationContext{
		Motif:     []int{0, 2, 4, 3, 0},
		Chords:    []string{"C", "G", "Am", "F"},
		TotalBars: totalBars,
		BasePitch: 60,
		BPM:       bpm,
		RNG:       NewRNG(),
		Darkness:  0.3,
		Energy:    0.6,
		Rhythmic:  0.4,
		Tension:   0.3,
		DNA:       DefaultDNA(),
	}
}

// WithStyle sets feature vector from a style's default values.
func (ctx *GenerationContext) WithStyle(darkness, energy, rhythmic, tension float64) *GenerationContext {
	ctx.Darkness = darkness
	ctx.Energy = energy
	ctx.Rhythmic = rhythmic
	ctx.Tension = tension
	return ctx
}

// WithDNA sets ComposerDNA from a DNA template or archetype.
func (ctx *GenerationContext) WithDNA(dna *ComposerDNA) *GenerationContext {
	if dna != nil {
		ctx.DNA = dna
	}
	return ctx
}

// StyleLabel returns a label based on feature vector thresholds.
// Used by sub-generators to select style-specific behavior.
func (ctx *GenerationContext) StyleLabel() string {
	d, e, r, t := ctx.Darkness, ctx.Energy, ctx.Rhythmic, ctx.Tension
	if d > 0.7 && e > 0.7 && t > 0.5 {
		return "metal"
	}
	if e > 0.5 && d < 0.4 && t < 0.4 {
		return "pop"
	}
	if d < 0.3 && e < 0.4 && r < 0.4 {
		return "ambient"
	}
	if r > 0.6 && e > 0.5 {
		return "complex"
	}
	if d > 0.5 && e > 0.5 {
		return "rock"
	}
	return "default"
}

// BeatEmotion returns the EmotionState for a specific bar.
// Falls back to a mid-range state if no curve is set.
func (ctx *GenerationContext) BeatEmotion(bar int) EmotionState {
	if ctx.EmotionCurve != nil && bar >= 0 && bar < len(ctx.EmotionCurve.Bars) {
		return ctx.EmotionCurve.Bars[bar]
	}
	return EmotionState{
		Tension:    ctx.Tension,
		Energy:     ctx.Energy,
		Warmth:     0.5,
		Stability:  0.6,
		Brightness: 0.5,
	}
}

// MotifUseRate returns how often to repeat the main motif (0-1).
// Driven by ComposerDNA.MotifObsession, with feature vector modifier.
func (ctx *GenerationContext) MotifUseRate() float64 {
	rate := 0.5
	if ctx.DNA != nil {
		rate = ctx.DNA.MotifObsession
	}
	// Dark/high-tension pieces repeat more.
	if ctx.Darkness > 0.6 {
		rate += 0.1
	}
	if ctx.Tension > 0.6 {
		rate += 0.1
	}
	if rate > 1.0 {
		rate = 1.0
	}
	return rate
}

// NewRNG creates a new seeded random number generator.
func NewRNG() *rand.Rand {
	return rand.New(rand.NewSource(globalSeed))
}

// DefaultDNA returns a neutral ComposerDNA archetype.
func DefaultDNA() *ComposerDNA {
	archetype := ComposerArchetypes["Classical Purist"]
	return &archetype
}

// GenerateContextFromDNA converts extracted MusicDNA into a GenerationContext
// for MIDI generation. This closes the DNA ↔ MIDI loop.
func GenerateContextFromDNA(dna *musicdna.MusicDNA, bpm, bars int) *GenerationContext {
	ctx := NewDefaultContext(8, 120)
	if dna == nil {
		return ctx
	}

	// Structure → TotalBars
	totalBars := 0
	for _, s := range dna.Structure.Sections {
		totalBars += s.Bars
	}
	if totalBars > 0 {
		ctx.TotalBars = totalBars
	}
	if bars > 0 {
		ctx.TotalBars = bars
	}
	if bpm > 0 {
		ctx.BPM = bpm
	}

	// Harmony → Chords
	if len(dna.Harmony.Progression) > 0 {
		chords := make([]string, 0, len(dna.Harmony.Progression))
		for _, c := range dna.Harmony.Progression {
			if c.Chord != "" && c.Chord != "-" {
				chords = append(chords, c.Chord)
			}
		}
		if len(chords) > 0 {
			ctx.Chords = chords
		}
	}

	// Motif → seed
	if len(dna.Motif.Pattern) >= 2 {
		ctx.Motif = dna.Motif.Pattern
	}

	// Feature vector from EmotionDNA
	ctx.Darkness = 1.0 - dna.Emotion.Brightness
	ctx.Energy = dna.Emotion.Energy
	ctx.Tension = dna.Emotion.Tension
	ctx.Rhythmic = dna.Rhythm.Syncopation

	// ComposerDNA from MusicDNA
	cd := DNAFromMusicDNA(dna)
	ctx.DNA = &cd

	// Emotion curve
	if len(dna.Emotion.Curve) > 0 {
		bCount := len(dna.Emotion.Curve)
		curve := &EmotionCurve{Bars: make([]EmotionState, bCount)}
		for i, e := range dna.Emotion.Curve {
			curve.Bars[i] = EmotionState{
				Tension:    dna.Emotion.Tension * e,
				Energy:     e,
				Warmth:     dna.Emotion.Warmth,
				Stability:  dna.Emotion.Stability,
				Brightness: dna.Emotion.Brightness,
			}
		}
		ctx.EmotionCurve = curve
	}

	return ctx
}

// MotifFromLibrary loads the highest-quality motif from the DNA library.
func MotifFromLibrary(libDir, style string) []int {
	lib := musicdna.NewLibrary(libDir)
	templates, err := lib.List(style)
	if err != nil || len(templates) == 0 {
		return nil
	}
	best := templates[0]
	for _, t := range templates[1:] {
		if t.Quality > best.Quality {
			best = t
		}
	}
	if len(best.DNA.Motif.Pattern) >= 2 {
		return best.DNA.Motif.Pattern
	}
	return nil
}
