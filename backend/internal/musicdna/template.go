// Package musicdna — Template Library.
// Saves/loads MusicDNA as JSON files, indexed by style.
// Templates serve as "RAG for Music" — pre-composed structural patterns
// that guide generation instead of generating from scratch.
package musicdna

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// Template wraps a MusicDNA with metadata for storage and retrieval.
type Template struct {
	Name        string    `json:"name"`
	Style       string    `json:"style"`
	Description string    `json:"description"`
	DNA         MusicDNA  `json:"dna"`
	Source      string    `json:"source,omitempty"` // MIDI file path
}

// TemplateDB manages a collection of MusicDNA templates on disk.
type TemplateDB struct {
	BaseDir string
}

// NewTemplateDB creates a template database in the given directory.
// Templates are stored as JSON files under {BaseDir}/{style}/*.json
func NewTemplateDB(baseDir string) *TemplateDB {
	return &TemplateDB{BaseDir: baseDir}
}

// Save writes a template to disk, indexed by style.
// Creates directory structure: {BaseDir}/{style}/{name}.json
func (lib *TemplateDB) Save(t *Template) error {
	if t.Style == "" {
		t.Style = "general"
	}
	dir := filepath.Join(lib.BaseDir, sanitize(t.Style))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("template mkdir: %w", err)
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("template marshal: %w", err)
	}

	filename := filepath.Join(dir, sanitize(t.Name)+".json")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("template write: %w", err)
	}

	fmt.Printf("[TemplateLib] saved: %s/%s\n", t.Style, t.Name)
	return nil
}

// Load retrieves a template by exact style and name.
func (lib *TemplateDB) Load(style, name string) (*Template, error) {
	path := filepath.Join(lib.BaseDir, sanitize(style), sanitize(name)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("template read %s/%s: %w", style, name, err)
	}
	var t Template
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("template parse %s/%s: %w", style, name, err)
	}
	return &t, nil
}

// FindByStyle returns all templates matching a style keyword.
// Does substring matching so "pop" matches "jpop", "pop_ballad", etc.
func (lib *TemplateDB) FindByStyle(keyword string) ([]*Template, error) {
	styleDir := filepath.Join(lib.BaseDir, sanitize(keyword))
	var results []*Template

	// Try exact style dir first.
	if entries, err := os.ReadDir(styleDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				t, err := lib.Load(keyword, strings.TrimSuffix(e.Name(), ".json"))
				if err == nil {
					results = append(results, t)
				}
			}
		}
	}

	// Also scan all style dirs for keyword match.
	parentDir := lib.BaseDir
	if entries, err := os.ReadDir(parentDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && e.Name() != sanitize(keyword) &&
				(strings.Contains(e.Name(), strings.ToLower(keyword)) ||
					strings.Contains(strings.ToLower(keyword), e.Name())) {
				if subEntries, err := os.ReadDir(filepath.Join(parentDir, e.Name())); err == nil {
					for _, se := range subEntries {
						if !se.IsDir() && strings.HasSuffix(se.Name(), ".json") {
							t, err := lib.Load(e.Name(), strings.TrimSuffix(se.Name(), ".json"))
							if err == nil {
								results = append(results, t)
							}
						}
					}
				}
			}
		}
	}

	// Sort by closest match.
	sort.Slice(results, func(i, j int) bool {
		return len(results[i].Name) < len(results[j].Name)
	})

	return results, nil
}

// ListStyles returns all style directories in the library.
func (lib *TemplateDB) ListStyles() ([]string, error) {
	entries, err := os.ReadDir(lib.BaseDir)
	if err != nil {
		return nil, err
	}
	var styles []string
	for _, e := range entries {
		if e.IsDir() {
			styles = append(styles, e.Name())
		}
	}
	sort.Strings(styles)
	return styles, nil
}

// ListTemplates returns all templates for a given style.
func (lib *TemplateDB) ListTemplates(style string) ([]string, error) {
	dir := filepath.Join(lib.BaseDir, sanitize(style))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return names, nil
}

// FromMIDI extracts MusicDNA from events and saves it as a template.
func (lib *TemplateDB) FromMIDI(eventsByTrack map[string][]schema.NoteEvent, totalBars int, key, style, name, source string) (*Template, error) {
	ext := NewExtractor()
	dna := ext.Extract(eventsByTrack, totalBars, key)

	t := &Template{
		Name:        name,
		Style:       style,
		Description: fmt.Sprintf("Auto-extracted from %s (%d bars, %s)", source, totalBars, key),
		DNA:         *dna,
		Source:      source,
	}

	if err := lib.Save(t); err != nil {
		return nil, err
	}
	return t, nil
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
	return strings.Trim(s, "_")
}
