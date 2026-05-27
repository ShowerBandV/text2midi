// Package composer — Section structure templates.
// Defines common song arrangements (intro → verse → chorus → bridge → outro)
// with per-section energy, instrument density, and drum pattern.
package composer

import "github.com/yourname/text2midi/internal/schema"

// Section describes one part of a song structure.
type Section struct {
	Name     string  // "intro", "verse", "chorus", "bridge", "outro"
	Bars     int     // length in bars
	Energy   float64 // 0-1 target energy level
	Density  float64 // 0-1 instrument density
	DrumPattern string // "full", "simple", "kick_snare", "off"
	Instruments []string // which instruments are active
}

// Structure is a complete song arrangement template.
type Structure struct {
	Name     string
	BPM      int
	Sections []Section
}

// PopBallad:  intro → verse → pre-chorus → chorus → verse → chorus → bridge → chorus → outro
var PopBallad = Structure{
	Name: "pop_ballad",
	Sections: []Section{
		{Name: "intro",   Bars: 2, Energy: 0.3, Density: 0.2, DrumPattern: "off", Instruments: []string{"piano", "strings"}},
		{Name: "verse",   Bars: 4, Energy: 0.4, Density: 0.3, DrumPattern: "simple", Instruments: []string{"piano", "bass", "drums"}},
		{Name: "pre",     Bars: 2, Energy: 0.6, Density: 0.5, DrumPattern: "full", Instruments: []string{"piano", "bass", "drums", "strings"}},
		{Name: "chorus",  Bars: 4, Energy: 0.8, Density: 0.7, DrumPattern: "full", Instruments: []string{"all"}},
		{Name: "verse",   Bars: 4, Energy: 0.4, Density: 0.3, DrumPattern: "simple", Instruments: []string{"piano", "bass", "drums"}},
		{Name: "chorus",  Bars: 4, Energy: 0.8, Density: 0.7, DrumPattern: "full", Instruments: []string{"all"}},
		{Name: "bridge",  Bars: 4, Energy: 0.5, Density: 0.4, DrumPattern: "simple", Instruments: []string{"piano", "strings", "pad"}},
		{Name: "chorus",  Bars: 6, Energy: 0.9, Density: 0.8, DrumPattern: "full", Instruments: []string{"all"}},
		{Name: "outro",   Bars: 2, Energy: 0.2, Density: 0.2, DrumPattern: "off", Instruments: []string{"piano"}},
	},
}

// RockAnthem: intro → verse → chorus → verse → chorus → solo → chorus → outro
var RockAnthem = Structure{
	Name: "rock_anthem",
	Sections: []Section{
		{Name: "intro",   Bars: 2, Energy: 0.5, Density: 0.4, DrumPattern: "kick_snare", Instruments: []string{"drums", "bass"}},
		{Name: "verse",   Bars: 4, Energy: 0.5, Density: 0.4, DrumPattern: "simple", Instruments: []string{"drums", "bass", "rhythm_guitar"}},
		{Name: "chorus",  Bars: 4, Energy: 0.8, Density: 0.7, DrumPattern: "full", Instruments: []string{"all"}},
		{Name: "verse",   Bars: 4, Energy: 0.5, Density: 0.4, DrumPattern: "simple", Instruments: []string{"drums", "bass", "rhythm_guitar"}},
		{Name: "chorus",  Bars: 4, Energy: 0.9, Density: 0.8, DrumPattern: "full", Instruments: []string{"all"}},
		{Name: "solo",    Bars: 4, Energy: 0.7, Density: 0.6, DrumPattern: "full", Instruments: []string{"drums", "bass", "lead_guitar"}},
		{Name: "chorus",  Bars: 4, Energy: 0.9, Density: 0.9, DrumPattern: "full", Instruments: []string{"all"}},
		{Name: "outro",   Bars: 2, Energy: 0.3, Density: 0.3, DrumPattern: "simple", Instruments: []string{"drums", "bass"}},
	},
}

// EpicCinematic: introduces → builds → climax → resolves
var EpicCinematic = Structure{
	Name: "epic_cinematic",
	Sections: []Section{
		{Name: "intro",     Bars: 2, Energy: 0.3, Density: 0.2, DrumPattern: "off", Instruments: []string{"strings", "choir"}},
		{Name: "build",     Bars: 4, Energy: 0.5, Density: 0.4, DrumPattern: "simple", Instruments: []string{"strings", "brass", "timpani"}},
		{Name: "climax",    Bars: 4, Energy: 0.9, Density: 0.9, DrumPattern: "full", Instruments: []string{"all"}},
		{Name: "sustain",   Bars: 2, Energy: 0.7, Density: 0.6, DrumPattern: "full", Instruments: []string{"strings", "brass", "drums"}},
		{Name: "climax",    Bars: 4, Energy: 0.9, Density: 0.9, DrumPattern: "full", Instruments: []string{"all"}},
		{Name: "resolve",   Bars: 2, Energy: 0.3, Density: 0.3, DrumPattern: "off", Instruments: []string{"strings", "choir"}},
	},
}

// SelectStructure returns the best structure template based on style and energy.
func SelectStructure(energy float64, primaryStyle string) *Structure {
	// Simple heuristic based on energy and style keywords.
	switch {
	case energy > 0.7 && (primaryStyle == "epic" || primaryStyle == "orchestral" || primaryStyle == "cinematic"):
		return &EpicCinematic
	case energy > 0.6 && (primaryStyle == "rock" || primaryStyle == "metal" || primaryStyle == "hard"):
		return &RockAnthem
	default:
		return &PopBallad
	}
}

// TotalBars returns the total bar count for a structure.
func (s *Structure) TotalBars() int {
	total := 0
	for _, sec := range s.Sections {
		total += sec.Bars
	}
	return total
}

// SectionAtBar returns which section is active at a given bar.
func (s *Structure) SectionAtBar(bar int) *Section {
	cursor := 0
	for i := range s.Sections {
		sec := &s.Sections[i]
		if bar >= cursor && bar < cursor+sec.Bars {
			return sec
		}
		cursor += sec.Bars
	}
	// Default to last section.
	return &s.Sections[len(s.Sections)-1]
}

// ApplyStructure modifies eventsByTrack per-section energy/density.
func ApplyStructure(eventsByTrack map[string][]schema.NoteEvent, structure *Structure, totalBars int) {
	if structure == nil {
		return
	}

	for _, events := range eventsByTrack {
		for i := range events {
			bar := int(events[i].StartBeat) / 4
			if bar >= totalBars {
				bar = totalBars - 1
			}
			sec := structure.SectionAtBar(bar)

			// Adjust velocity by section energy with wider dynamic range.
			// Real music: intro=pp(0.3x), verse=mp(0.6x), chorus=ff(1.5x).
			dynamicMap := map[string]float64{
				"intro":  0.3,
				"verse":  0.7,
				"pre":    0.9,
				"chorus": 1.4,
				"bridge": 0.6,
				"solo":   1.2,
				"build":  0.5,
				"climax": 1.5,
				"sustain": 1.1,
				"resolve": 0.4,
				"outro":  0.3,
			}
			factor := 0.5 + sec.Energy*0.8
			if df, ok := dynamicMap[sec.Name]; ok {
				factor = df
			}
			events[i].Velocity = int(float64(events[i].Velocity) * factor)
			if events[i].Velocity > 127 {
				events[i].Velocity = 127
			}
			if events[i].Velocity < 8 {
				events[i].Velocity = 8
			}
		}
	}
}
