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

	// Step 1: Normalize — remove absolute pitch, keep intervals from first note.
	base := lead[0].Pitch
	intervals := make([]int, 0, len(lead))
	durations := make([]float64, 0, len(lead))
	for _, ev := range lead {
		intervals = append(intervals, ev.Pitch-base)
		durations = append(durations, ev.DurationBeat)
	}

	// Step 2: Convert to relative interval changes (the core recognition space).
	relativeInts := make([]int, 0, len(intervals)-1)
	for i := 1; i < len(intervals); i++ {
		relativeInts = append(relativeInts, intervals[i]-intervals[i-1])
	}

	// Step 3: Sliding window at multiple sizes.
	type candidate struct {
		pattern    []int
		rhythm     []float64
		startIdx   int
		windowSize int
	}

	var allCandidates []candidate
	for ws := 3; ws <= 8 && ws <= len(relativeInts); ws++ {
		for start := 0; start <= len(relativeInts)-ws; start++ {
			c := candidate{
				pattern:    relativeInts[start : start+ws],
				rhythm:     durations[start : start+ws],
				startIdx:   start,
				windowSize: ws,
			}
			allCandidates = append(allCandidates, c)
		}
	}

	// Step 4: Hash-based frequency counting.
	type scoredMotif struct {
		pattern []int
		rhythm  []float64
		freq    int
		length  int
	}

	freq := make(map[string]*scoredMotif)
	for _, c := range allCandidates {
		key := intsKey(c.pattern)
		if _, ok := freq[key]; !ok {
			freq[key] = &scoredMotif{
				pattern: c.pattern,
				rhythm:  c.rhythm,
				length:  c.windowSize,
			}
		}
		freq[key].freq++
	}

	// Step 5: Score and select dominant motif.
	// score = freq*0.6 + length*0.2 + (1 if rhythm varies)*0.2
	var best *scoredMotif
	bestScore := 0.0

	for _, sm := range freq {
		rhythmScore := 0.0
		if len(sm.rhythm) > 0 {
			// Check if rhythm has variation (not all same duration).
			allSame := true
			for i := 1; i < len(sm.rhythm); i++ {
				if math.Abs(sm.rhythm[i]-sm.rhythm[0]) > 0.05 {
					allSame = false
					break
				}
			}
			if !allSame {
				rhythmScore = 0.2
			}
		}
		score := float64(sm.freq)*0.6 + float64(sm.length)*0.2 + rhythmScore
		if score > bestScore {
			bestScore = score
			best = sm
		}
	}

	if best != nil && len(best.pattern) > 0 {
		md.Pattern = best.pattern
		md.Rhythm = best.rhythm
		md.Confidence = bestScore / float64(len(intervals))
		if md.Confidence > 1.0 {
			md.Confidence = 1.0
		}

		// Detect variants.
		md.Variants = detectVariants(relativeInts, best.pattern, durations)
	}
	return md
}

func intsKey(ints []int) string {
	if len(ints) == 0 {
		return ""
	}
	b := make([]byte, 0, len(ints)*4)
	for _, v := range ints {
		// Compact encoding: each int as 2 bytes (+-999 range)
		b = append(b, byte((v+1000)>>8), byte((v+1000)&0xFF))
	}
	return string(b)
}

// detectVariants finds transpositions, inversions of the motif in the full interval sequence.
func detectVariants(fullSeq, motif []int, durations []float64) []MotifVariant {
	if len(motif) < 2 || len(fullSeq) < len(motif)*2 {
		return nil
	}

	var variants []MotifVariant
	skip := 0

	for start := len(motif); start <= len(fullSeq)-len(motif); start++ {
		skip++
		if skip%2 == 0 {
			continue // check every other position for efficiency
		}
		candidate := fullSeq[start : start+len(motif)]

		// Check transposition: same interval pattern, shifted by constant.
		isSameShape := true
		for i := 1; i < len(motif); i++ {
			if candidate[i]-candidate[i-1] != motif[i]-motif[i-1] {
				isSameShape = false
				break
			}
		}
		if isSameShape {
			variants = append(variants, MotifVariant{Type: "transpose", Pattern: candidate})
			continue
		}

		// Check inversion: intervals reversed.
		isInvert := true
		for i := 0; i < len(motif) && i < len(candidate); i++ {
			if abs(candidate[i]+motif[len(motif)-1-i]) > 2 {
				isInvert = false
				break
			}
		}
		if isInvert {
			variants = append(variants, MotifVariant{Type: "invert", Pattern: candidate})
			continue
		}

		// Check fragment: first 2-3 notes match.
		if len(motif) >= 3 && abs(candidate[0]-motif[0]) <= 2 &&
			abs(candidate[1]-motif[1]) <= 2 {
			// Check rhythm similarity.
			if len(durations) > start+1 {
				rhythmRatio := durations[start] / durations[0]
				if rhythmRatio > 0.5 && rhythmRatio < 2.0 {
					variants = append(variants, MotifVariant{Type: "fragment", Pattern: candidate})
				}
			}
		}
	}

	return variants
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func unused() {}
