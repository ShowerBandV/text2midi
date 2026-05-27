// Package musicdna — structured music representation.
// A song is not a MIDI file. It's a 5-layer DNA structure:
// StructureDNA, HarmonyDNA, MotifDNA, RhythmDNA, TextureDNA.
package musicdna

// MusicDNA is the complete representation of a song.
type MusicDNA struct {
	Structure StructureDNA
	Harmony   HarmonyDNA
	Motif     MotifDNA
	Rhythm    RhythmDNA
	Texture   TextureDNA
}

// ─── 1. Structure Layer ────────────────────────────────────────────
type StructureDNA struct {
	Sections []Section
}

type Section struct {
	Name        string   // intro / verse / chorus / bridge / outro
	StartBar    int
	Bars        int
	Energy      float64  // 0-1 average velocity+density
	Density     float64  // 0-1 notes per bar
	Instruments []string // active instrument track IDs
}

// ─── 2. Harmony Layer ──────────────────────────────────────────────
type HarmonyDNA struct {
	Key         string   // "C major", "A minor"
	Progression []ChordBar
}

type ChordBar struct {
	Bar      int
	Chord    string // "C", "Am", "F", "G7", "Dm7"
	Function string // "T" (tonic), "S" (subdominant), "D" (dominant), "T_vi", etc.
}

// ─── 3. Motif Layer ────────────────────────────────────────────────
type MotifDNA struct {
	Notes    []int          // relative intervals from root: [0,2,4,3]
	Rhythm   []float64      // relative durations
	Variants []MotifVariant
}

type MotifVariant struct {
	Type  string // "invert", "transpose", "rhythm_shift", "retrograde"
	Notes []int
}

// ─── 4. Rhythm Layer ────────────────────────────────────────────────
type RhythmDNA struct {
	DrumPattern      string             // "trap", "boom_bap", "rock", "pop"
	Swing            float64            // 0-1
	DensityBySection map[string]float64 // section name → density
}

// ─── 5. Texture Layer ──────────────────────────────────────────────
type TextureDNA struct {
	InstrumentTimeline map[string][]int   // instrument track ID → active bars
	Layering           []LayerEvent
}

type LayerEvent struct {
	Bar        int
	Action     string // "add", "remove", "thin", "thicken"
	Instrument string
}

// ─── 6. MusicDNA Library ──────────────────────────────────────────
// Template is a named, reusable MusicDNA for a specific style/archetype.
type Template struct {
	Name        string
	Description string
	DNA         MusicDNA
}
