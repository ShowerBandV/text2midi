// Package style — Instrumentation templates (style → instrument mapping).
// Each template defines the exact MIDI program, channel, pitch range, and role
// for every instrument in a given style. This prevents AI from choosing
// inappropriate instruments or playing in conflicting registers.
package style

import "sort"

// InstrumentSpec defines one instrument in an arrangement.
type InstrumentSpec struct {
	Name        string `json:"name"`
	Program     int    `json:"midi_program"` // GM program number
	Channel     int    `json:"midi_channel"`
	MinPitch    int    `json:"min_pitch"`    // lowest allowed pitch
	MaxPitch    int    `json:"max_pitch"`    // highest allowed pitch
	Role        string `json:"role"`
	EntryBar    int    `json:"entry_bar"`    // which bar this instrument enters (0 = intro)
	EntryEnergy float64 `json:"entry_energy"` // min energy before this instrument plays
}

// Template is a complete instrumentation template for a musical style.
type Template struct {
	Name        string
	DisplayName string
	Instruments []InstrumentSpec
}

// GetInstrumentTemplate returns the instrumentation template for a given style.
// Falls back to a generic template if the style isn't explicitly mapped.
func GetInstrumentTemplate(styleName string) *Template {
	if t, ok := instrumentTemplates[styleName]; ok {
		copy := t
		return &copy
	}
	// Fallback: try matching by prefix or return generic.
	for key, t := range instrumentTemplates {
		if len(key) <= len(styleName) && styleName[:len(key)] == key {
			copy := t
			return &copy
		}
	}
	t := instrumentTemplates["generic"]
	return &t
}

// FindBestTemplate tries to match by keywords in the style description.
func FindBestTemplate(styleName, styleDesc string) *Template {
	// Exact match first.
	if t := GetInstrumentTemplate(styleName); t.Name != "generic" {
		return t
	}
	// Keyword match.
	keywords := map[string]string{
		"rock": "rock", "metal": "rock", "grunge": "rock",
		"pop": "pop", "ballad": "pop", "rnb": "pop", "soul": "pop",
		"trap": "hiphop", "hip": "hiphop", "rap": "hiphop", "drill": "hiphop",
		"lofi": "hiphop", "boom": "hiphop", "west": "hiphop",
		"orchestra": "cinematic", "epic": "cinematic", "cinematic": "cinematic",
		"film": "cinematic", "fantasy": "cinematic", "battle": "cinematic",
		"jazz": "jazz", "blues": "jazz",
		"chinese": "chinese", "古风": "chinese", "国风": "chinese",
		"electronic": "electronic", "edm": "electronic", "synth": "electronic",
	}
	for kw, template := range keywords {
		if contains(styleName, kw) || contains(styleDesc, kw) {
			return GetInstrumentTemplate(template)
		}
	}
	return GetInstrumentTemplate("generic")
}

func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			ca := s[i+j]
			cb := substr[j]
			if ca >= 'A' && ca <= 'Z' {
				ca += 32
			}
			if cb >= 'A' && cb <= 'Z' {
				cb += 32
			}
			if ca != cb {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// instrumentTemplates is the master template database.
var instrumentTemplates = map[string]Template{
	// ─── Rock / Hard Rock ───────────────────────────────────────
	"rock": {
		Name: "rock", DisplayName: "Classic Rock",
		Instruments: []InstrumentSpec{
			{Name: "Power Chord Guitar", Program: 30, Channel: 0, MinPitch: 52, MaxPitch: 72, Role: "rhythm_guitar", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "Lead Guitar", Program: 29, Channel: 1, MinPitch: 64, MaxPitch: 88, Role: "lead_guitar", EntryBar: 4, EntryEnergy: 0.5},
			{Name: "Rock Bass", Program: 34, Channel: 2, MinPitch: 28, MaxPitch: 48, Role: "bass", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "Rock Drums", Program: 0, Channel: 9, MinPitch: 35, MaxPitch: 81, Role: "drums", EntryBar: 2, EntryEnergy: 0.3},
		},
	},
	// ─── Pop / Ballad ───────────────────────────────────────────
	"pop": {
		Name: "pop", DisplayName: "Pop / Ballad",
		Instruments: []InstrumentSpec{
			{Name: "Acoustic Piano", Program: 0, Channel: 0, MinPitch: 40, MaxPitch: 76, Role: "piano", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "Synth Pad", Program: 91, Channel: 1, MinPitch: 48, MaxPitch: 72, Role: "pad", EntryBar: 2, EntryEnergy: 0.2},
			{Name: "Finger Bass", Program: 33, Channel: 2, MinPitch: 28, MaxPitch: 48, Role: "bass", EntryBar: 4, EntryEnergy: 0.4},
			{Name: "Pop Drums", Program: 0, Channel: 9, MinPitch: 35, MaxPitch: 81, Role: "drums", EntryBar: 4, EntryEnergy: 0.4},
			{Name: "String Ensemble", Program: 48, Channel: 3, MinPitch: 48, MaxPitch: 76, Role: "strings", EntryBar: 6, EntryEnergy: 0.6},
		},
	},
	// ─── Hip-Hop / Trap ─────────────────────────────────────────
	"hiphop": {
		Name: "hiphop", DisplayName: "Hip-Hop / Trap",
		Instruments: []InstrumentSpec{
			{Name: "Synth Lead", Program: 80, Channel: 0, MinPitch: 60, MaxPitch: 84, Role: "lead", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "808 Bass", Program: 38, Channel: 1, MinPitch: 24, MaxPitch: 48, Role: "bass", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "Synth Pad", Program: 90, Channel: 2, MinPitch: 48, MaxPitch: 72, Role: "pad", EntryBar: 2, EntryEnergy: 0.2},
			{Name: "Trap Drums", Program: 0, Channel: 9, MinPitch: 35, MaxPitch: 81, Role: "drums", EntryBar: 1, EntryEnergy: 0.1},
		},
	},
	// ─── Cinematic / Orchestral ─────────────────────────────────
	"cinematic": {
		Name: "cinematic", DisplayName: "Cinematic / Orchestral",
		Instruments: []InstrumentSpec{
			{Name: "String Ensemble", Program: 48, Channel: 0, MinPitch: 48, MaxPitch: 84, Role: "strings", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "French Horn", Program: 60, Channel: 1, MinPitch: 48, MaxPitch: 72, Role: "brass", EntryBar: 2, EntryEnergy: 0.3},
			{Name: "Timpani", Program: 47, Channel: 2, MinPitch: 36, MaxPitch: 55, Role: "timpani", EntryBar: 2, EntryEnergy: 0.3},
			{Name: "Choir", Program: 52, Channel: 3, MinPitch: 48, MaxPitch: 76, Role: "choir", EntryBar: 6, EntryEnergy: 0.6},
			{Name: "Orchestral Percussion", Program: 0, Channel: 9, MinPitch: 35, MaxPitch: 81, Role: "drums", EntryBar: 4, EntryEnergy: 0.5},
		},
	},
	// ─── Chinese Style (国风) ─────────────────────────────────────
	"chinese": {
		Name: "chinese", DisplayName: "Chinese Style",
		Instruments: []InstrumentSpec{
			{Name: "Guzheng", Program: 107, Channel: 0, MinPitch: 48, MaxPitch: 84, Role: "guzheng", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "Pipa", Program: 106, Channel: 1, MinPitch: 48, MaxPitch: 84, Role: "pipa", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "Dizi", Program: 73, Channel: 2, MinPitch: 60, MaxPitch: 88, Role: "dizi", EntryBar: 2, EntryEnergy: 0.2},
			{Name: "String Ensemble", Program: 48, Channel: 3, MinPitch: 40, MaxPitch: 72, Role: "strings", EntryBar: 2, EntryEnergy: 0.2},
			{Name: "Chinese Percussion", Program: 0, Channel: 9, MinPitch: 35, MaxPitch: 81, Role: "drums", EntryBar: 4, EntryEnergy: 0.4},
		},
	},
	// ─── Electronic / EDM ────────────────────────────────────────
	"electronic": {
		Name: "electronic", DisplayName: "Electronic / EDM",
		Instruments: []InstrumentSpec{
			{Name: "Synth Lead", Program: 80, Channel: 0, MinPitch: 60, MaxPitch: 96, Role: "lead", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "Synth Bass", Program: 38, Channel: 1, MinPitch: 24, MaxPitch: 48, Role: "bass", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "Warm Pad", Program: 89, Channel: 2, MinPitch: 48, MaxPitch: 72, Role: "pad", EntryBar: 1, EntryEnergy: 0.1},
			{Name: "EDM Drums", Program: 0, Channel: 9, MinPitch: 35, MaxPitch: 81, Role: "drums", EntryBar: 2, EntryEnergy: 0.3},
		},
	},
	// ─── Jazz ────────────────────────────────────────────────────
	"jazz": {
		Name: "jazz", DisplayName: "Jazz",
		Instruments: []InstrumentSpec{
			{Name: "Jazz Piano", Program: 1, Channel: 0, MinPitch: 40, MaxPitch: 76, Role: "piano", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "Acoustic Bass", Program: 32, Channel: 1, MinPitch: 28, MaxPitch: 48, Role: "bass", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "Jazz Drums", Program: 0, Channel: 9, MinPitch: 35, MaxPitch: 81, Role: "drums", EntryBar: 1, EntryEnergy: 0.1},
		},
	},
	// ─── Generic fallback ────────────────────────────────────────
	"generic": {
		Name: "generic", DisplayName: "Generic",
		Instruments: []InstrumentSpec{
			{Name: "Warm Piano", Program: 0, Channel: 0, MinPitch: 40, MaxPitch: 76, Role: "piano", EntryBar: 0, EntryEnergy: 0.0},
			{Name: "Synth Bass", Program: 33, Channel: 1, MinPitch: 28, MaxPitch: 48, Role: "bass", EntryBar: 2, EntryEnergy: 0.2},
			{Name: "String Pad", Program: 48, Channel: 2, MinPitch: 48, MaxPitch: 72, Role: "pad", EntryBar: 3, EntryEnergy: 0.3},
			{Name: "Acoustic Drums", Program: 0, Channel: 9, MinPitch: 35, MaxPitch: 81, Role: "drums", EntryBar: 4, EntryEnergy: 0.4},
		},
	},
}

// GetTemplateNames returns all available template names for UI selection.
func GetTemplateNames() []string {
	var names []string
	for _, t := range instrumentTemplates {
		names = append(names, t.Name)
	}
	sort.Strings(names)
	return names
}
