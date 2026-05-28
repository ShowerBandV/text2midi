// Package planner — Song Planning Engine.
// Takes LLM intent + feature vector → fully structured SongPlan.
// Each section has its own energy, density, instrumentation, and motif mode.
package planner

import (
	"fmt"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// SongPlan is the complete structural blueprint for a song.
type SongPlan struct {
	BPM      int
	Key      string
	Mood     string
	Sections []SectionPlan
}

// SectionPlan defines one section in the song.
type SectionPlan struct {
	Name      string   // "intro", "verse", "chorus", "bridge", "outro"
	Bars      int
	Energy    float64  // 0-1
	Density   float64  // 0-1
	Instruments []string // which tracks are active
	MotifMode string   // "full", "partial", "sparse", "invert"
}

// BuildSongPlan creates a structured song plan from an LLM intent.
func BuildSongPlan(fv schema.FeatureVector, totalBars int, key, mood string) *SongPlan {
	sp := &SongPlan{
		BPM:  120,
		Key:  key,
		Mood: mood,
	}

	// Determine section layout based on feature vector.
	sections := sectionLayout(fv.Darkness, fv.Energy, fv.RhythmicComplexity, fv.Tension, totalBars)
	sp.Sections = make([]SectionPlan, len(sections))

	for i, sec := range sections {
		sp.Sections[i] = SectionPlan{
			Name:      sec.name,
			Bars:      sec.bars,
			Energy:    sec.energy,
			Density:   sec.density,
			Instruments: sec.instruments,
			MotifMode: sec.motifMode,
		}
	}

	fmt.Printf("[Planner] %d sections, %d bars\n", len(sp.Sections), totalBars)
	return sp
}

type sectionDef struct {
	name        string
	bars        int
	energy      float64
	density     float64
	instruments []string
	motifMode   string
}

func sectionLayout(darkness, energy, rhythmic, tension float64, totalBars int) []sectionDef {
	var sections []sectionDef
	remaining := totalBars

	// Choose style template.
	switch {
	case darkness > 0.7 && energy > 0.7:
		// Metal: short intro, fast verses, explosive chorus
		if remaining >= 8 {
			sections = append(sections, sectionDef{"intro", 1, 0.3, 0.3, []string{"drums", "bass"}, "sparse"})
			sections = append(sections, sectionDef{"verse", 2, 0.5, 0.5, []string{"drums", "bass", "rhythm_guitar"}, "partial"})
			sections = append(sections, sectionDef{"chorus", 4, 0.9, 0.9, nil, "full"})
			sections = append(sections, sectionDef{"bridge", 1, 0.4, 0.3, []string{"bass", "lead_guitar"}, "invert"})
		}

	case energy > 0.4 && rhythmic < 0.5:
		// Pop: balanced intro-verse-pre-chorus-chorus
		if remaining >= 12 {
			sections = append(sections, sectionDef{"intro", 2, 0.2, 0.2, []string{"piano"}, "sparse"})
			sections = append(sections, sectionDef{"verse", 4, 0.4, 0.4, []string{"piano", "bass", "drums"}, "partial"})
			sections = append(sections, sectionDef{"pre", 2, 0.6, 0.5, []string{"piano", "bass", "drums", "strings"}, "variant"})
			sections = append(sections, sectionDef{"chorus", 4, 0.85, 0.8, nil, "full"})
			sections = append(sections, sectionDef{"bridge", 2, 0.5, 0.3, []string{"piano", "strings"}, "invert"})
			sections = append(sections, sectionDef{"outro", 2, 0.15, 0.15, []string{"piano"}, "sparse"})
		} else {
			sections = append(sections, sectionDef{"intro", 1, 0.2, 0.2, []string{"piano"}, "sparse"})
			sections = append(sections, sectionDef{"verse", 2, 0.4, 0.4, []string{"piano", "bass"}, "partial"})
			sections = append(sections, sectionDef{"chorus", 3, 0.85, 0.8, nil, "full"})
			sections = append(sections, sectionDef{"outro", 2, 0.15, 0.15, []string{"piano"}, "sparse"})
		}

	case rhythmic > 0.5 && energy > 0.3:
		// Hip-hop: loop-based, intro + loop + outro
		sections = append(sections, sectionDef{"intro", 1, 0.2, 0.2, []string{"pad", "drums"}, "sparse"})
		sections = append(sections, sectionDef{"loop_a", 3, 0.5, 0.5, []string{"drums", "bass", "lead"}, "partial"})
		sections = append(sections, sectionDef{"loop_b", 3, 0.7, 0.6, []string{"drums", "bass", "lead", "pad"}, "full"})
		sections = append(sections, sectionDef{"outro", 1, 0.15, 0.15, []string{"pad"}, "sparse"})

	default:
		// Ambient / default: minimal
		sections = append(sections, sectionDef{"intro", 2, 0.15, 0.15, []string{"pad"}, "sparse"})
		sections = append(sections, sectionDef{"verse", 4, 0.4, 0.3, []string{"pad", "bass", "drums_lite"}, "partial"})
		sections = append(sections, sectionDef{"chorus", 4, 0.7, 0.6, []string{"all"}, "full"})
		sections = append(sections, sectionDef{"outro", 2, 0.1, 0.1, []string{"pad"}, "sparse"})
	}

	// Trim to fit total bars.
	total := 0
	for _, s := range sections {
		total += s.bars
	}
	if total > totalBars {
		// Trim from last sections.
		for i := len(sections) - 1; i >= 0 && total > totalBars; i-- {
			excess := total - totalBars
			if sections[i].bars > excess {
				sections[i].bars -= excess
				total -= excess
			} else {
				total -= sections[i].bars
				sections = sections[:i]
			}
		}
	}

	return sections
}
