// Package musicdna — deterministic MIDI → DNA extractor.
// 3 modules: structure segmenter, chord detector, motif extractor.
package musicdna

import (
	"fmt"
	"math"
	"sort"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// Extractor converts raw MIDI events into structured MusicDNA.
// All analysis is rule-based, no LLM.
type Extractor struct{}

func NewExtractor() *Extractor { return &Extractor{} }

// Extract runs the full analysis pipeline.
func (e *Extractor) Extract(eventsByTrack map[string][]schema.NoteEvent, totalBars int, key string) *MusicDNA {
	dna := &MusicDNA{
		Structure: e.extractStructure(eventsByTrack, totalBars),
		Harmony:   e.extractHarmony(eventsByTrack, totalBars, key),
		Motif:     e.extractMotif(eventsByTrack, totalBars),
	}
	return dna
}

// ═══════════════════════════════════════════════════════════════════
// 1. Structure Segmenter
// ═══════════════════════════════════════════════════════════════════

// extractStructure divides the song into sections using 3 signals:
//   - note density change
//   - velocity shift
//   - chord change
func (e *Extractor) extractStructure(eventsByTrack map[string][]schema.NoteEvent, totalBars int) StructureDNA {
	if totalBars <= 0 {
		return StructureDNA{}
	}

	// Compute per-bar metrics.
	type barMetric struct {
		density  float64
		velocity float64
	}

	metrics := make([]barMetric, totalBars)

	for _, events := range eventsByTrack {
		for _, ev := range events {
			bar := int(ev.StartBeat) / 4
			if bar >= 0 && bar < totalBars {
				metrics[bar].density++
				metrics[bar].velocity += float64(ev.Velocity)
			}
		}
	}

	// Normalize.
	maxDensity := 1.0
	for _, m := range metrics {
		if m.density > maxDensity {
			maxDensity = m.density
		}
	}
	for i := range metrics {
		metrics[i].density /= maxDensity
		if metrics[i].density > 0 {
			metrics[i].velocity /= metrics[i].density * float64(maxDensity) * 127.0
		}
		if metrics[i].velocity > 1.0 {
			metrics[i].velocity = 1.0
		}
	}

	// Detect section boundaries: look for significant changes.
	type boundary struct{ bar int }
	var boundaries []boundary
	boundaries = append(boundaries, boundary{0}) // first bar is always a boundary

	for bar := 1; bar < totalBars; bar++ {
		densityJump := metrics[bar].density-metrics[bar-1].density > 0.3
		energyJump := metrics[bar].velocity-metrics[bar-1].velocity > 0.25
		if densityJump || energyJump {
			boundaries = append(boundaries, boundary{bar})
		}
	}

	// Name sections.
	sectionNames := []string{"intro", "verse", "chorus", "bridge", "outro"}
	sections := make([]Section, 0, len(boundaries))

	for i, b := range boundaries {
		endBar := totalBars
		if i+1 < len(boundaries) {
			endBar = boundaries[i+1].bar
		}

		// Calculate average energy and density for this section.
		var avgEnergy, avgDensity float64
		count := 0
		for bar := b.bar; bar < endBar && bar < totalBars; bar++ {
			avgDensity += metrics[bar].density
			avgEnergy += metrics[bar].velocity
			count++
		}
		if count > 0 {
			avgDensity /= float64(count)
			avgEnergy /= float64(count)
		}

		name := sectionNames[i]
		if i >= len(sectionNames) {
			name = fmt.Sprintf("section_%d", i)
		}

		sections = append(sections, Section{
			Name:     name,
			StartBar: b.bar,
			Bars:     endBar - b.bar,
			Energy:   avgEnergy,
			Density:  avgDensity,
		})
	}

	fmt.Printf("[Segmenter] %d sections detected\n", len(sections))
	return StructureDNA{Sections: sections}
}

// ═══════════════════════════════════════════════════════════════════
// 2. Chord Detector (rule-based, no LLM)
// ═══════════════════════════════════════════════════════════════════

func (e *Extractor) extractHarmony(eventsByTrack map[string][]schema.NoteEvent, totalBars int, key string) HarmonyDNA {
	hd := HarmonyDNA{Key: key}

	// Collect chord-capable events.
	var chordEvents []schema.NoteEvent
	for _, id := range []string{"chords", "piano", "pad", "strings", "rhythm_guitar"} {
		if evs, ok := eventsByTrack[id]; ok {
			chordEvents = append(chordEvents, evs...)
		}
	}
	if len(chordEvents) == 0 {
		return hd
	}

	// Group by bar.
	barPitches := make(map[int][]int)
	for _, ev := range chordEvents {
		bar := int(ev.StartBeat) / 4
		if bar >= 0 && bar < totalBars {
			barPitches[bar] = append(barPitches[bar], ev.Pitch%12)
		}
	}

	for bar := 0; bar < totalBars; bar++ {
		pc, ok := barPitches[bar]
		if !ok || len(pc) == 0 {
			continue
		}
		chord := detectChord(pc)
		hd.Progression = append(hd.Progression, ChordBar{Bar: bar, Chord: chord})
	}

	fmt.Printf("[ChordDetector] %d chords detected\n", len(hd.Progression))
	return hd
}

// detectChord uses pitch class histogram + root detection + template matching.
func detectChord(pcs []int) string {
	if len(pcs) == 0 {
		return "-"
	}

	// Histogram.
	hist := make(map[int]int)
	for _, p := range pcs {
		hist[p]++
	}

	// Find bass note (most frequent in lower range).
	// For simplicity, use the most frequent pitch class.
	root := 0
	maxCount := 0
	for p, c := range hist {
		if c > maxCount || (c == maxCount && (p == root || isConsonant(p, root))) {
			root = p
			maxCount = c
		}
	}

	// Check intervals from root.
	hasMinorThird := hist[(root+3)%12] > 0
	hasMajorThird := hist[(root+4)%12] > 0
	hasFifth := hist[(root+7)%12] > 0
	hasMinorSeventh := hist[(root+10)%12] > 0
	hasMajorSeventh := hist[(root+11)%12] > 0

	semiToNote := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	chord := semiToNote[root]

	switch {
	case hasMinorThird && hasMinorSeventh:
		chord += "m7"
	case hasMajorThird && hasMinorSeventh:
		chord += "7"
	case hasMajorThird && hasMajorSeventh:
		chord += "maj7"
	case hasMinorThird:
		chord += "m"
	case !hasMajorThird && !hasMinorThird && hasFifth:
		chord += "5" // power chord
	}

	return chord
}

func isConsonant(p, root int) bool {
	interval := (p - root + 12) % 12
	return interval == 0 || interval == 4 || interval == 5 || interval == 7
}

// ═══════════════════════════════════════════════════════════════════
// 3. Motif Extractor (find interval patterns via sliding window)
// ═══════════════════════════════════════════════════════════════════

func (e *Extractor) extractMotif(eventsByTrack map[string][]schema.NoteEvent, totalBars int) MotifDNA {
	md := MotifDNA{Confidence: 0}

	// Get lead melody.
	lead := eventsByTrack["lead"]
	if len(lead) == 0 {
		for _, id := range []string{"lead_guitar", "lead_vocal", "piano", "melody"} {
			if evs, ok := eventsByTrack[id]; ok && len(evs) > 0 {
				lead = evs
				break
			}
		}
	}
	if len(lead) < 4 {
		return md
	}

	// Sort by start beat.
	sort.Slice(lead, func(i, j int) bool {
		return lead[i].StartBeat < lead[j].StartBeat
	})

	// Convert to interval sequence: [0, +2, +4, +3, ...]
	// Base the intervals from the first note.
	base := lead[0].Pitch
	intervals := make([]int, 0, len(lead))
	for _, ev := range lead {
		intervals = append(intervals, ev.Pitch-base)
	}

	// Sliding window to find repeating patterns.
	windowSizes := []int{3, 4, 5, 6}
	type patternScore struct {
		pattern []int
		score   int
	}

	var scoredPatterns []patternScore

	for _, ws := range windowSizes {
		if ws > len(intervals)/2 {
			break
		}
		// Slide window across the interval sequence.
		patterns := make(map[string]int) // pattern string → count

		for start := 0; start <= len(intervals)-ws; start++ {
			pat := intervals[start : start+ws]
			key := fmt.Sprintf("%v", pat)
			patterns[key]++
		}

		// Find most frequent pattern for this window size.
		bestKey := ""
		bestCount := 0
		for key, count := range patterns {
			if count > bestCount {
				bestCount = count
				bestKey = key
			}
		}

		if bestCount > 1 {
			// Parse the pattern from the key.
			var pat []int
			fmt.Sscanf(bestKey, "%v", &pat)
			scoredPatterns = append(scoredPatterns, patternScore{
				pattern: pat,
				score:   bestCount * ws,
			})
		}
	}

	if len(scoredPatterns) == 0 && len(intervals) >= 4 {
		// Fallback: use first 4 notes as motif.
		md.Pattern = intervals[:4]
		md.Confidence = 0.3
	} else {
		// Pick the highest-scoring pattern.
		best := scoredPatterns[0]
		for _, sp := range scoredPatterns[1:] {
			if sp.score > best.score {
				best = sp
			}
		}
		md.Pattern = best.pattern
		md.Confidence = float64(best.score) / float64(len(intervals))
		if md.Confidence > 1.0 {
			md.Confidence = 1.0
		}
	}

	// Extract rhythm (durations of the motif notes).
	if len(md.Pattern) > 0 && len(lead) >= len(md.Pattern) {
		md.Rhythm = make([]float64, len(md.Pattern))
		for i := range md.Pattern {
			if i < len(lead) {
				md.Rhythm[i] = lead[i].DurationBeat
			}
		}
	}

	// Detect variants by comparing later phrases.
	md.Variants = detectVariants(intervals, md.Pattern)

	info := fmt.Sprintf("[MotifExtractor] pattern=%v, confidence=%.2f, variants=%d\n",
		md.Pattern, md.Confidence, len(md.Variants))
	_ = info // printed by caller
	return md
}

// detectVariants finds transpositions, inversions of the motif in the full sequence.
func detectVariants(fullIntervalSeq, motif []int) []MotifVariant {
	if len(motif) < 2 || len(fullIntervalSeq) < len(motif)*2 {
		return nil
	}

	var variants []MotifVariant

	// Scan the full sequence for pattern matches at different positions.
	for start := len(motif); start <= len(fullIntervalSeq)-len(motif); start++ {
		candidate := fullIntervalSeq[start : start+len(motif)]

		// Check transposition (all intervals shifted by same amount).
		isTranspose := true
		for i := 1; i < len(motif); i++ {
			if candidate[i]-candidate[i-1] != motif[i]-motif[i-1] {
				isTranspose = false
				break
			}
		}
		if isTranspose {
			variants = append(variants, MotifVariant{
				Type:    "transpose",
				Pattern: candidate,
			})
			continue
		}

		// Check inversion (intervals reversed).
		isInvert := true
		for i := 0; i < len(motif) && i < len(candidate); i++ {
			if math.Abs(float64(candidate[i]-motif[len(motif)-1-i])) > 2 {
				isInvert = false
				break
			}
		}
		if isInvert {
			variants = append(variants, MotifVariant{
				Type:    "invert",
				Pattern: candidate,
			})
			continue
		}

		// Check fragmentation (first 2-3 notes match).
		if len(motif) >= 3 && math.Abs(float64(candidate[0]-motif[0])) <= 2 &&
			math.Abs(float64(candidate[1]-motif[1])) <= 2 &&
			math.Abs(float64(candidate[2]-motif[2])) <= 2 {
			variants = append(variants, MotifVariant{
				Type:    "fragment",
				Pattern: candidate,
			})
		}
	}

	return variants
}
