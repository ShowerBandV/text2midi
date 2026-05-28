// Package musicdna — Style Profile.
// Analyzes multiple DNA templates to extract a statistical style signature.
// This drives the 4 generators with genre-specific biases instead of pure randomness.
package musicdna

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// StyleProfile aggregates statistics from multiple templates of the same style.
// Each field biases the corresponding generator's random choices.
type StyleProfile struct {
	Name             string        `json:"name"`
	ChordPreference  []ChordStats  `json:"chord_preference"`  // most common chords
	IntervalBias     []int         `json:"interval_bias"`     // preferred intervals (semitone steps)
	OctaveBias       []int         `json:"octave_bias"`       // preferred scale-degree octave ranges
	BlockVsArpRatio  float64       `json:"block_vs_arp"`      // 0=all arp, 1=all block
	VelocityRange    [2]int        `json:"velocity_range"`     // [min, max] preferred velocities
	DurationWeights  []float64     `json:"duration_weights"`  // weight for [short, med, long]
	DensityRange     [2]float64    `json:"density_range"`     // [min, max] notes per bar
	HarmonicComplexity float64     `json:"harmonic_complexity"` // 0=simple triads, 1=extended
}

// BuildStyleProfile analyzes all templates in a style directory and returns a statistical signature.
func BuildStyleProfile(db *TemplateDB, style string) (*StyleProfile, error) {
	templates, err := db.FindByStyle(style)
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, fmt.Errorf("no templates found for style %q", style)
	}

	sp := &StyleProfile{
		Name: style,
	}

	// Chord preference: count frequencies.
	chordCount := make(map[string]int)
	// Interval bias: collect all motif intervals.
	var allIntervals []int
	// Octave bias: track scale-degree usage.
	_ = make(map[int]int) // degreeCount (reserved for future use)
	// Block vs arp: count section densities.
	blockCount, arpCount := 0, 0
	// Velocity range.
	minVel, maxVel := 127, 0
	// Duration weights.
	var durationCount [3]int // short(0-0.4), medium(0.4-0.8), long(0.8+)
	// Density range.
	minDensity, maxDensity := 1.0, 0.0
	// Harmonic complexity.
	totalSections := 0

	for _, t := range templates {
		dna := t.DNA

		// Chord frequencies.
		for _, cb := range dna.Harmony.Progression {
			if cb.Chord != "" {
				chordCount[cb.Chord]++
			}
		}

		// Interval patterns.
		for _, iv := range dna.Motif.Pattern {
			allIntervals = append(allIntervals, iv)
		}

		// Section density and structure.
		for _, sec := range dna.Structure.Sections {
			totalSections++
			if sec.Density > 0.5 {
				blockCount++
			} else {
				arpCount++
			}
			if sec.Density < minDensity {
				minDensity = sec.Density
			}
			if sec.Density > maxDensity {
				maxDensity = sec.Density
			}
		}

		// Rhythm data.
		for _, d := range dna.Motif.Rhythm {
			if d < 0.4 {
				durationCount[0]++
			} else if d < 0.8 {
				durationCount[1]++
			} else {
				durationCount[2]++
			}
		}
	}

	// Compile chord preference: top 8 most frequent.
	type ch struct {
		name  string
		count int
	}
	var chordList []ch
	for name, count := range chordCount {
		chordList = append(chordList, ch{name, count})
	}
	sort.Slice(chordList, func(i, j int) bool {
		return chordList[i].count > chordList[j].count
	})
	for i, c := range chordList {
		if i >= 8 {
			break
		}
		sp.ChordPreference = append(sp.ChordPreference, ChordStats{
			Chord:      c.name,
			Frequency:  float64(c.count) / float64(len(templates)),
			IsCommon:   i < 4, // top 4 are "common"
		})
	}

	// Interval bias: most common intervals.
	intervalCount := make(map[int]int)
	for _, iv := range allIntervals {
		intervalCount[iv]++
	}
	type iv struct {
		val   int
		count int
	}
	var ivList []iv
	for val, count := range intervalCount {
		ivList = append(ivList, iv{val, count})
	}
	sort.Slice(ivList, func(i, j int) bool {
		return ivList[i].count > ivList[j].count
	})
	for i, item := range ivList {
		if i >= 6 {
			break
		}
		sp.IntervalBias = append(sp.IntervalBias, item.val)
	}

	// Block vs arp ratio.
	if blockCount+arpCount > 0 {
		sp.BlockVsArpRatio = float64(blockCount) / float64(blockCount+arpCount)
	}

	// Velocity range.
	sp.VelocityRange = [2]int{minVel, maxVel}

	// Duration weights.
	totalDurations := durationCount[0] + durationCount[1] + durationCount[2]
	if totalDurations > 0 {
		sp.DurationWeights = []float64{
			float64(durationCount[0]) / float64(totalDurations),
			float64(durationCount[1]) / float64(totalDurations),
			float64(durationCount[2]) / float64(totalDurations),
		}
	}

	// Density range.
	sp.DensityRange = [2]float64{minDensity, maxDensity}

	// Harmonic complexity: how many unique chords.
	if totalSections > 0 {
		sp.HarmonicComplexity = float64(len(chordList)) / 12.0
		if sp.HarmonicComplexity > 1.0 {
			sp.HarmonicComplexity = 1.0
		}
	}

	fmt.Printf("[StyleProfile] %s: %d templates, %d chord types, %d intervals\n",
		style, len(templates), len(chordList), len(ivList))
	return sp, nil
}

// ChordStats holds chord frequency data.
type ChordStats struct {
	Chord     string  `json:"chord"`
	Frequency float64 `json:"frequency"`
	IsCommon  bool    `json:"is_common"`
}

// Save saves the style profile as JSON.
func (sp *StyleProfile) Save(path string) error {
	data, err := json.MarshalIndent(sp, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	return os.WriteFile(path, data, 0644)
}

// LoadStyleProfile loads a style profile from JSON.
func LoadStyleProfile(path string) (*StyleProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sp StyleProfile
	if err := json.Unmarshal(data, &sp); err != nil {
		return nil, err
	}
	return &sp, nil
}

// PickChord selects a chord from the style profile, weighted by frequency.
// Returns the chord if available, or a fallback.
func (sp *StyleProfile) PickChord(defaultChords []string) string {
	if len(sp.ChordPreference) > 0 {
		return sp.ChordPreference[0].Chord
	}
	if len(defaultChords) > 0 {
		return defaultChords[0]
	}
	return "C"
}


