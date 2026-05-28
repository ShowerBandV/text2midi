// Package motif — global motif registry and manipulation.
// Tracks all motifs across sections and enables theme regression.
package motif

import "fmt"

// Motif is a short musical idea (3-8 notes, relative intervals).
type Motif struct {
	ID             string
	Notes          []int     // relative intervals from root
	Rhythm         []float64 // relative durations
	EmotionalLabel string    // "happy", "sad", "aggressive"
	ReuseCount     int
}

// Registry holds all motifs used in a composition.
type Registry struct {
	Motifs    map[string]*Motif
	UsageLog  []string // ordered list of motif IDs used
	CurrentID int
}

// NewRegistry creates an empty motif registry.
func NewRegistry() *Registry {
	return &Registry{
		Motifs:   make(map[string]*Motif),
		UsageLog: make([]string, 0),
	}
}

// Register adds a motif and returns its ID.
func (r *Registry) Register(notes []int, rhythm []float64, label string) string {
	if len(notes) < 2 {
		return ""
	}
	id := fmt.Sprintf("motif_%d", r.CurrentID)
	r.CurrentID++
	r.Motifs[id] = &Motif{
		ID:             id,
		Notes:          copyInts(notes),
		Rhythm:         copyFloats(rhythm),
		EmotionalLabel: label,
		ReuseCount:     0,
	}
	r.UsageLog = append(r.UsageLog, id)
	fmt.Printf("[MotifRegistry] %s: %v (label=%s)\n", id, notes, label)
	return id
}

// Get returns a motif by ID.
func (r *Registry) Get(id string) *Motif {
	return r.Motifs[id]
}

// Latest returns the most recently registered motif.
func (r *Registry) Latest() *Motif {
	if len(r.UsageLog) == 0 {
		return nil
	}
	return r.Motifs[r.UsageLog[len(r.UsageLog)-1]]
}

// GetVariant returns a developed version of a motif.
// variant: 0=original, 1=transpose+3, 2=transpose+5, 3=invert, 4=retrograde
func (r *Registry) GetVariant(id string, variant int) *Motif {
	base := r.Get(id)
	if base == nil {
		return nil
	}
	if variant == 0 {
		return base
	}

	m := &Motif{
		ID:             fmt.Sprintf("%s_v%d", id, variant),
		Notes:          copyInts(base.Notes),
		Rhythm:         copyFloats(base.Rhythm),
		EmotionalLabel: base.EmotionalLabel,
		ReuseCount:     0,
	}

	switch variant {
	case 1:
		m.Notes = transpose(m.Notes, 3)
	case 2:
		m.Notes = transpose(m.Notes, 5)
	case 3:
		m.Notes = invert(m.Notes)
	case 4:
		m.Notes = retrograde(m.Notes)
	}

	r.Motifs[m.ID] = m
	r.UsageLog = append(r.UsageLog, m.ID)
	return m
}

// ReuseCount returns how many times the given motif has been referenced.
func (r *Registry) ReuseCount(id string) int {
	if m := r.Get(id); m != nil {
		return m.ReuseCount
	}
	return 0
}

// TransformMotif applies a random transformation for live variation.
func TransformMotif(m *Motif) *Motif {
	return &Motif{
		ID:             m.ID + "_x",
		Notes:          transpose(m.Notes, 2),
		Rhythm:         m.Rhythm,
		EmotionalLabel: m.EmotionalLabel,
		ReuseCount:     0,
	}
}

// Helpers
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

func retrograde(s []int) []int {
	out := make([]int, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}

func copyInts(s []int) []int {
	out := make([]int, len(s))
	copy(out, s)
	return out
}

func copyFloats(s []float64) []float64 {
	out := make([]float64, len(s))
	copy(out, s)
	return out
}
