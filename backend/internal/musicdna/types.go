// Package musicdna — core music representation.
// 6 layers: Structure, Harmony, Motif, Rhythm, Texture, Dynamics.
// Purpose: record "music generation rules", not raw MIDI data.
package musicdna

import (
	"fmt"
	"math"
)

// MusicDNA is the complete generative representation of a song.
type MusicDNA struct {
	Structure StructureDNA `json:"structure"`
	Harmony   HarmonyDNA   `json:"harmony"`
	Motif     MotifDNA     `json:"motif"`
	Rhythm    RhythmDNA    `json:"rhythm"`
	Texture   TextureDNA   `json:"texture"`
	Dynamics  DynamicsDNA  `json:"dynamics"`
	Emotion   EmotionDNA   `json:"emotion"`
}

// Print returns a human-readable summary.
func (d *MusicDNA) Print() string {
	s := "===== MusicDNA =====\n"
	s += d.Structure.Print()
	s += d.Harmony.Print()
	s += d.Motif.Print()
	s += d.Rhythm.Print()
	s += d.Texture.Print()
	s += d.Dynamics.Print()
	s += d.Emotion.Print()
	return s
}

// ─── Structure ─────────────────────────────────────────────────────

type StructureDNA struct {
	Sections   []Section    `json:"sections"`
	BarFeatures []BarFeature `json:"bar_features,omitempty"`
	Template   string       `json:"template,omitempty"` // matched template: "AABA", "ABAB", "intro-verse-chorus", etc.
	Confidence float64      `json:"confidence"`         // 0-1 how well the structure fits
}

func (s *StructureDNA) Print() string {
	out := "--- Structure ---\n"
	if s.Template != "" {
		out += fmt.Sprintf("  Template: %s (confidence=%.2f)\n", s.Template, s.Confidence)
	}
	for _, sec := range s.Sections {
		out += fmt.Sprintf("  %s: bars %d-%d (energy=%.2f, density=%.2f)\n",
			sec.Name, sec.StartBar, sec.StartBar+sec.Bars-1, sec.Energy, sec.Density)
	}
	return out
}

type Section struct {
	Name      string  `json:"name"`
	StartBar  int     `json:"start_bar"`
	Bars      int     `json:"bars"`
	Energy    float64 `json:"energy"`
	Density   float64 `json:"density"`
}

// BarFeature captures per-bar musical characteristics.
type BarFeature struct {
	Bar             int     `json:"bar"`
	Density         float64 `json:"density"`          // note count normalized
	AvgVelocity     float64 `json:"avg_velocity"`     // average velocity (0-1)
	ChordChange     bool    `json:"chord_change"`      // chord changed this bar
	InstrumentCount int     `json:"instrument_count"`  // active tracks this bar
	SilenceRatio    float64 `json:"silence_ratio"`     // proportion of silence (0=none, 1=all silent)
	Energy          float64 `json:"energy"`            // composite energy
}

// ─── Harmony ───────────────────────────────────────────────────────

type HarmonyDNA struct {
	Key          string     `json:"key"`
	Progression  []ChordBar `json:"progression"`
	Confidence   float64    `json:"confidence"`
}

func (h *HarmonyDNA) Print() string {
	out := "--- Harmony ---\n"
	out += fmt.Sprintf("  Key: %s\n", h.Key)
	for _, c := range h.Progression {
		out += fmt.Sprintf("  bar %d: %s\n", c.Bar, c.Chord)
	}
	return out
}

type ChordBar struct {
	Bar   int    `json:"bar"`
	Chord string `json:"chord"`
}

// ─── Motif ─────────────────────────────────────────────────────────

type MotifDNA struct {
	Pattern    []int          `json:"pattern"`     // interval sequence from root, e.g. [0,2,4,3]
	Rhythm     []float64      `json:"rhythm"`      // relative durations
	Score      *MotifScore    `json:"score,omitempty"`
	Confidence float64        `json:"confidence"`  // 0-1 how well this represents the song
	Variants   []MotifVariant `json:"variants,omitempty"`
}

type MotifVariant struct {
	Type    string `json:"type"`    // "transpose", "invert", "fragment", "rhythm_shift"
	Pattern []int  `json:"pattern"`
}

func (m *MotifDNA) Print() string {
	out := "--- Motif ---\n"
	out += fmt.Sprintf("  Pattern (intervals): %v\n", m.Pattern)
	out += fmt.Sprintf("  Rhythm: %v\n", m.Rhythm)
	out += fmt.Sprintf("  Confidence: %.2f\n", m.Confidence)
	if m.Score != nil {
		out += m.Score.Print()
	}
	for _, v := range m.Variants {
		out += fmt.Sprintf("  Variant %s: %v\n", v.Type, v.Pattern)
	}
	return out
}

// MotifScore evaluates melody quality across 4 dimensions.
type MotifScore struct {
	Repetition     float64 `json:"repetition"`      // occurrence / total bars (0-1)
	Contour        float64 `json:"contour"`         // slope variance (0-1, higher = more angular)
	Simplicity     float64 `json:"simplicity"`      // 1 - (avg interval / 12) (0-1, higher = simpler)
	RhythmIdentity float64 `json:"rhythm_identity"` // duration pattern self-similarity (0-1)
	Total          float64 `json:"total"`           // weighted sum
}

func (s *MotifScore) Print() string {
	return fmt.Sprintf("  Score: repetition=%.2f contour=%.2f simplicity=%.2f rhythm=%.2f total=%.2f\n",
		s.Repetition, s.Contour, s.Simplicity, s.RhythmIdentity, s.Total)
}

// CalculateTotal computes the weighted total from the 4 sub-scores.
func (s *MotifScore) CalculateTotal() {
	s.Total = s.Repetition*0.4 + s.Contour*0.2 + s.Simplicity*0.2 + s.RhythmIdentity*0.2
}

// ─── Rhythm ───────────────────────────────────────────────────────

type RhythmDNA struct {
	Density      float64 `json:"density"`       // average notes per beat (0-1 normalized)
	SwingAmount  float64 `json:"swing_amount"`  // 0=straight, 0.5=heavy swing
	Syncopation  float64 `json:"syncopation"`   // proportion of offbeat notes (0-1)
	Variety      float64 `json:"variety"`       // rhythmic pattern diversity (0-1)
	Confidence   float64 `json:"confidence"`
}

func (r *RhythmDNA) Print() string {
	return fmt.Sprintf("--- Rhythm ---\n  density=%.2f swing=%.2f syncopation=%.2f variety=%.2f\n",
		r.Density, r.SwingAmount, r.Syncopation, r.Variety)
}

// ─── Texture ───────────────────────────────────────────────────────

type TextureDNA struct {
	TrackCount  int                `json:"track_count"`
	Layers      []TextureLayer     `json:"layers"`
	Density     float64            `json:"density"`     // overall arrangement density (0-1)
	Confidence  float64            `json:"confidence"`
}

type TextureLayer struct {
	Name     string  `json:"name"`     // "drums", "bass", "chords", "lead", "pad", "fx"
	Role     string  `json:"role"`     // "rhythm", "harmonic", "melodic", "atmosphere"
	Active   bool    `json:"active"`
	NoteCount int   `json:"note_count"`
	AvgPitch  float64 `json:"avg_pitch"` // average MIDI pitch
}

func (t *TextureDNA) Print() string {
	out := "--- Texture ---\n"
	out += fmt.Sprintf("  Tracks: %d, density=%.2f\n", t.TrackCount, t.Density)
	for _, l := range t.Layers {
		out += fmt.Sprintf("  %s (%s): %d notes, avg pitch=%.0f\n", l.Name, l.Role, l.NoteCount, l.AvgPitch)
	}
	return out
}

// ─── Dynamics ──────────────────────────────────────────────────────

type DynamicsDNA struct {
	EnergyCurve  []float64 `json:"energy_curve"`   // per-bar energy values
	DynamicRange float64   `json:"dynamic_range"`  // max - min velocity (0-1)
	AvgVelocity  float64   `json:"avg_velocity"`   // overall average velocity (0-1)
	Crescendo    bool      `json:"crescendo"`      // energy increases toward end
	Confidence   float64   `json:"confidence"`
}

func (d *DynamicsDNA) Print() string {
	return fmt.Sprintf("--- Dynamics ---\n  range=%.2f avg_vel=%.2f crescendo=%v\n",
		d.DynamicRange, d.AvgVelocity, d.Crescendo)
}

// ─── Emotion ──────────────────────────────────────────────────────

type EmotionDNA struct {
	Tension    float64   `json:"tension"`    // 0-1 tension/anxiety
	Energy     float64   `json:"energy"`     // 0-1 overall energy
	Warmth     float64   `json:"warmth"`     // 0-1 warm vs cold
	Stability  float64   `json:"stability"`  // 0-1 stable vs chaotic
	Brightness float64   `json:"brightness"` // 0-1 bright vs dark
	Curve      []float64 `json:"curve,omitempty"`  // per-bar energy values
	Confidence float64   `json:"confidence"`
}

func (e *EmotionDNA) Print() string {
	return fmt.Sprintf("--- Emotion ---\n  tension=%.2f energy=%.2f warmth=%.2f stability=%.2f brightness=%.2f\n",
		e.Tension, e.Energy, e.Warmth, e.Stability, e.Brightness)
}

// ─── Helpers ───────────────────────────────────────────────────────

// Clamp01 clamps a float64 to [0, 1].
func Clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// AbsInt returns the absolute value of an int.
func AbsInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// RoundTo rounds a float64 to a given number of decimal places.
func RoundTo(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}
