// Package musicdna — core music representation.
// 3 layers only: Structure, Harmony, Motif.
// Purpose: record "music generation rules", not MIDI data.
package musicdna

import "fmt"

// MusicDNA is the complete generative representation of a song.
type MusicDNA struct {
	Structure StructureDNA
	Harmony   HarmonyDNA
	Motif     MotifDNA
}

// Print returns a human-readable summary.
func (d *MusicDNA) Print() string {
	s := "===== MusicDNA =====\n"
	s += d.Structure.Print()
	s += d.Harmony.Print()
	s += d.Motif.Print()
	return s
}

// ─── Structure ─────────────────────────────────────────────────────

type StructureDNA struct {
	Sections []Section
}

func (s *StructureDNA) Print() string {
	out := "--- Structure ---\n"
	for _, sec := range s.Sections {
		out += fmt.Sprintf("  %s: bars %d-%d (energy=%.2f, density=%.2f)\n",
			sec.Name, sec.StartBar, sec.StartBar+sec.Bars-1, sec.Energy, sec.Density)
	}
	return out
}

type Section struct {
	Name      string
	StartBar  int
	Bars      int
	Energy    float64
	Density   float64
}

// ─── Harmony ───────────────────────────────────────────────────────

type HarmonyDNA struct {
	Key         string
	Progression []ChordBar
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
	Bar   int
	Chord string
}

// ─── Motif ─────────────────────────────────────────────────────────

type MotifDNA struct {
	Pattern     []int     // interval sequence from root, e.g. [0,2,4,3]
	Rhythm      []float64 // relative durations
	Confidence  float64   // 0-1 how well this represents the song
	Variants    []MotifVariant
}

type MotifVariant struct {
	Type    string // "transpose", "invert", "fragment", "rhythm_shift"
	Pattern []int
}

func (m *MotifDNA) Print() string {
	out := "--- Motif ---\n"
	out += fmt.Sprintf("  Pattern (intervals): %v\n", m.Pattern)
	out += fmt.Sprintf("  Rhythm: %v\n", m.Rhythm)
	out += fmt.Sprintf("  Confidence: %.2f\n", m.Confidence)
	for _, v := range m.Variants {
		out += fmt.Sprintf("  Variant %s: %v\n", v.Type, v.Pattern)
	}
	return out
}
