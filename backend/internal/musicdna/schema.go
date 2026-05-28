// Package musicdna — JSON serialization and DNA Library persistence.
package musicdna

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ─── MusicDNA Serialization ───────────────────────────────────────

// ToJSON serializes MusicDNA to indented JSON bytes.
func (d *MusicDNA) ToJSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

// FromJSON deserializes MusicDNA from JSON bytes.
func FromJSON(data []byte) (*MusicDNA, error) {
	var dna MusicDNA
	if err := json.Unmarshal(data, &dna); err != nil {
		return nil, fmt.Errorf("musicdna from JSON: %w", err)
	}
	return &dna, nil
}

// ─── DNA Library (Phase 2) ────────────────────────────────────────

// DNATemplate is a stored DNA template with metadata.
type DNATemplate struct {
	Name     string    `json:"name"`
	Style    string    `json:"style"`
	DNA      MusicDNA  `json:"dna"`
	Quality  float64   `json:"quality"`  // 0-1 quality score
	Source   string    `json:"source"`   // e.g., "midi_output/song.mid"
}

// Library manages a directory of .dna template files.
type Library struct {
	Dir string
}

// NewLibrary creates a library rooted at dir.
func NewLibrary(dir string) *Library {
	return &Library{Dir: dir}
}

// Save writes a DNATemplate to {dir}/{name}.dna.
func (lib *Library) Save(tmpl *DNATemplate) error {
	if err := os.MkdirAll(lib.Dir, 0755); err != nil {
		return fmt.Errorf("create library dir: %w", err)
	}

	data, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal template: %w", err)
	}

	path := filepath.Join(lib.Dir, tmpl.Name+".dna")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write template: %w", err)
	}
	return nil
}

// Load reads a DNATemplate from {dir}/{name}.dna.
func (lib *Library) Load(name string) (*DNATemplate, error) {
	path := filepath.Join(lib.Dir, name+".dna")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template %q: %w", name, err)
	}

	var tmpl DNATemplate
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parse template %q: %w", name, err)
	}
	return &tmpl, nil
}

// List returns all .dna files in the library, filtered by optional style.
func (lib *Library) List(style string) ([]DNATemplate, error) {
	entries, err := os.ReadDir(lib.Dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list library: %w", err)
	}

	var templates []DNATemplate
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".dna" {
			continue
		}
		name := entry.Name()[:len(entry.Name())-4] // strip .dna
		tmpl, err := lib.Load(name)
		if err != nil {
			continue // skip invalid files
		}
		if style != "" && tmpl.Style != style {
			continue
		}
		templates = append(templates, *tmpl)
	}
	return templates, nil
}

// Delete removes a .dna file from the library.
func (lib *Library) Delete(name string) error {
	path := filepath.Join(lib.Dir, name+".dna")
	if err := os.Remove(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("delete template %q: %w", name, err)
	}
	return nil
}

// ─── Quality Scoring for DNA Library ──────────────────────────────

// ScoreTemplate computes a quality score (0-1) for a MusicDNA.
// Higher scores indicate more usable/musical templates.
func ScoreTemplate(dna *MusicDNA) float64 {
	if dna == nil {
		return 0
	}

	score := 0.0
	weights := 0.0

	// Structure quality: more sections = more interesting
	if len(dna.Structure.Sections) > 0 {
		structScore := Clamp01(float64(len(dna.Structure.Sections)) / 6.0)
		score += structScore * 0.25
		weights += 0.25
	}

	// Harmony quality: chord progression length / total bars
	if len(dna.Harmony.Progression) > 0 {
		harmonyScore := dna.Harmony.Confidence
		score += harmonyScore * 0.25
		weights += 0.25
	}

	// Motif quality: confidence + score
	if dna.Motif.Confidence > 0 {
		motifScore := dna.Motif.Confidence
		if dna.Motif.Score != nil {
			motifScore = (motifScore + dna.Motif.Score.Total) / 2.0
		}
		score += motifScore * 0.2
		weights += 0.2
	}

	// Rhythm quality
	if dna.Rhythm.Confidence > 0 {
		score += dna.Rhythm.Confidence * 0.1
		weights += 0.1
	}

	// Texture quality: more tracks = richer arrangement
	if dna.Texture.TrackCount > 0 {
		texScore := Clamp01(float64(dna.Texture.TrackCount) / 8.0)
		score += texScore * 0.1
		weights += 0.1
	}

	// Dynamics quality
	if dna.Dynamics.Confidence > 0 {
		score += dna.Dynamics.Confidence * 0.1
		weights += 0.1
	}

	if weights == 0 {
		return 0
	}
	return Clamp01(score / weights)
}
