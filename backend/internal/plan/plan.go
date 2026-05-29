// Package plan — Structured composition plan.
// A plan.json drives all 4 generators with per-section strategy, register, and density.
// This is the orchestrating layer between LLM intent and generator output.
package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Plan is the complete composition blueprint.
type Plan struct {
	FormatVersion string        `json:"format_version"`
	Tempo         float64       `json:"tempo"`
	Key           KeyInfo       `json:"key"`
	TotalBars     int           `json:"total_bars"`
	Sections      []SectionPlan `json:"sections"`
}

// KeyInfo defines the musical key.
type KeyInfo struct {
	Root  string `json:"root"`
	Mode  string `json:"mode"`
	Scale string `json:"scale"`
}

// SectionPlan defines one section in the composition.
type SectionPlan struct {
	Name      string  `json:"name"`
	Bars      int     `json:"bars"`
	Energy    float64 `json:"energy"`
	Density   float64 `json:"density"`
	Register  string  `json:"register"`   // "low", "mid", "high"
	LeadStrat string  `json:"lead_strategy"` // "new", "variation", "sequence", "recap", "climax"
}

// Build creates a composition plan from style + bar count.
// The plan is stored to disk and returned for use by all generators.
func Build(keyRoot, keyMode string, totalBars int, bpm int, profile *ProfileData) *Plan {
	p := &Plan{
		FormatVersion: "1.0",
		Tempo:         float64(bpm),
		Key:           KeyInfo{Root: keyRoot, Mode: keyMode, Scale: scaleName(keyMode)},
		TotalBars:     totalBars,
	}

	strategies := []string{"new", "variation", "development", "recap", "climax"}
	sectionNames := []string{"intro", "verse", "chorus", "bridge", "outro"}
	if totalBars <= 8 {
		sectionNames = []string{"intro", "verse", "chorus", "outro"}
	}

	barCursor := 0
	for i, name := range sectionNames {
		bars := totalBars / len(sectionNames)
		if i == len(sectionNames)-1 {
			bars = totalBars - barCursor
		}
		if bars <= 0 {
			break
		}

		energy := float64(i) / float64(len(sectionNames))
		density := 0.3 + energy*0.7
		register := "mid"
		if energy < 0.3 {
			register = "low"
		} else if energy > 0.7 {
			register = "high"
		}

		strategy := strategies[i]
		if i >= len(strategies) {
			strategy = strategies[len(strategies)-1]
		}

		p.Sections = append(p.Sections, SectionPlan{
			Name:      name,
			Bars:      bars,
			Energy:    energy,
			Density:   density,
			Register:  register,
			LeadStrat: strategy,
		})
		barCursor += bars
	}

	fmt.Printf("[Plan] %d sections, %d bars, key=%s %s, bpm=%d\n",
		len(p.Sections), totalBars, keyRoot, keyMode, bpm)
	return p
}

// Save writes the plan to disk as JSON.
func (p *Plan) Save(dir string) error {
	os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "plan.json")
	fmt.Printf("[Plan] saved to %s\n", path)
	return os.WriteFile(path, data, 0644)
}

// ProfileData packs style profile stats for plan building.
type ProfileData struct {
	ChordPref    []string
	IntervalBias []int
	BlockRatio   float64
	VelMin       int
	VelMax       int
	StepProb     float64
}

func scaleName(mode string) string {
	if mode == "minor" {
		return "natural_minor"
	}
	return "major"
}
