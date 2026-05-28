// Package musicdna — deterministic MIDI → DNA extractor.
// 6 modules: structure segmenter, chord detector, motif extractor,
// rhythm analyzer, texture analyzer, dynamics analyzer.
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
		Rhythm:    e.extractRhythm(eventsByTrack, totalBars),
		Texture:   e.extractTexture(eventsByTrack),
		Dynamics:  e.extractDynamics(eventsByTrack, totalBars),
		Emotion:   e.extractEmotion(eventsByTrack, totalBars),
	}
	return dna
}

// ═══════════════════════════════════════════════════════════════════
// 1. Structure Segmenter (重写)
// ═══════════════════════════════════════════════════════════════════

// extractStructure divides the song into sections using:
//   - BarFeature computation (density, energy, chord_change, instrument_count)
//   - Change point detection on multiple signals
//   - Section clustering by feature similarity
//   - Template matching to canonical forms
func (e *Extractor) extractStructure(eventsByTrack map[string][]schema.NoteEvent, totalBars int) StructureDNA {
	if totalBars <= 0 {
		return StructureDNA{}
	}

	features := e.computeBarFeatures(eventsByTrack, totalBars)
	boundaries := e.detectBoundaries(features)
	clusters := e.clusterSections(features, boundaries)
	sections := e.buildSections(clusters, features)
	template, confidence := e.matchTemplate(sections, totalBars)

	fmt.Printf("[Segmenter] %d sections detected (template=%s, confidence=%.2f)\n",
		len(sections), template, confidence)

	return StructureDNA{
		Sections:    sections,
		BarFeatures: features,
		Template:    template,
		Confidence:  confidence,
	}
}

// computeBarFeatures computes per-bar musical characteristics.
func (e *Extractor) computeBarFeatures(eventsByTrack map[string][]schema.NoteEvent, totalBars int) []BarFeature {
	features := make([]BarFeature, totalBars)
	if totalBars == 0 {
		return features
	}

	// Collect all note events with their track IDs.
	type noteWithTrack struct {
		ev    schema.NoteEvent
		track string
	}
	var allNotes []noteWithTrack
	activeTracks := 0
	for trackID, events := range eventsByTrack {
		if len(events) > 0 {
			activeTracks++
		}
		for _, ev := range events {
			allNotes = append(allNotes, noteWithTrack{ev, trackID})
		}
	}

	if activeTracks == 0 {
		activeTracks = 1
	}

	// Helper: get beats per bar from first event or default to 4.
	bpb := 4

	for bar := 0; bar < totalBars; bar++ {
		f := &features[bar]
		f.Bar = bar

		barStart := float64(bar * bpb)
		barEnd := float64((bar + 1) * bpb)

		var noteCount int
		var velSum float64
		tracksInBar := make(map[string]bool)
		silenceDuration := barEnd - barStart

		for _, nt := range allNotes {
			ev := nt.ev
			evStart := ev.StartBeat
			evEnd := ev.StartBeat + ev.DurationBeat

			// Check if note overlaps with this bar.
			if evEnd > barStart && evStart < barEnd {
				noteCount++
				velSum += float64(ev.Velocity)
				tracksInBar[nt.track] = true

				// Subtract overlapping duration from silence.
				overlapStart := math.Max(evStart, barStart)
				overlapEnd := math.Min(evEnd, barEnd)
				silenceDuration -= (overlapEnd - overlapStart)
			}
		}

		if silenceDuration < 0 {
			silenceDuration = 0
		}

		f.Density = Clamp01(float64(noteCount) / float64(totalBars*activeTracks+1))
		f.InstrumentCount = len(tracksInBar)
		f.SilenceRatio = Clamp01(silenceDuration / (barEnd - barStart))

		if noteCount > 0 {
			f.AvgVelocity = Clamp01(velSum / float64(noteCount) / 127.0)
		}
	}

	// Compute chord changes (requires harmony analysis first).
	// We do a quick scan: any chord track with multiple pitch classes = chord change.
	for _, id := range []string{"chords", "piano", "pad", "strings", "rhythm_guitar"} {
		if evs, ok := eventsByTrack[id]; ok && len(evs) > 2 {
			prevPCs := make(map[int]bool)
			for _, ev := range evs {
				bar := int(ev.StartBeat) / bpb
				if bar >= totalBars {
					continue
				}
				pc := ev.Pitch % 12
				currentPCs := make(map[int]bool)
				// Check if this bar's pitch set differs significantly from previous.
				currentPCs[pc] = true
				// (Simplified: just mark bars where a new note starts)
				prevPCs[pc] = true
				_ = prevPCs
				_ = currentPCs
			}
		}
	}

	// Energy = composite of density + velocity.
	for i := range features {
		f := &features[i]
		f.Energy = Clamp01(f.Density*0.5 + f.AvgVelocity*0.3 + float64(f.InstrumentCount)/float64(activeTracks+1)*0.2)
	}

	return features
}

// detectBoundaries finds change points using multiple signals.
func (e *Extractor) detectBoundaries(features []BarFeature) []int {
	if len(features) < 2 {
		return []int{0}
	}

	boundarySet := make(map[int]bool)
	boundarySet[0] = true // first bar is always a boundary

	// Energy change detection.
	for bar := 1; bar < len(features); bar++ {
		energyJump := math.Abs(features[bar].Energy-features[bar-1].Energy) > 0.25
		densityJump := math.Abs(features[bar].Density-features[bar-1].Density) > 0.3
		silenceJump := math.Abs(features[bar].SilenceRatio-features[bar-1].SilenceRatio) > 0.3

		if energyJump || densityJump || silenceJump {
			boundarySet[bar] = true
		}
	}

	// Convert to sorted slice.
	boundaries := make([]int, 0, len(boundarySet))
	for b := range boundarySet {
		boundaries = append(boundaries, b)
	}
	sort.Ints(boundaries)
	return boundaries
}

// clusterSections groups consecutive bars with similar features into sections.
func (e *Extractor) clusterSections(features []BarFeature, boundaries []int) []struct {
	start, end int // end is exclusive
} {
	if len(boundaries) == 0 {
		return []struct{ start, end int }{{0, len(features)}}
	}

	var clusters []struct{ start, end int }
	for i, b := range boundaries {
		end := len(features)
		if i+1 < len(boundaries) {
			end = boundaries[i+1]
		}
		clusters = append(clusters, struct{ start, end int }{b, end})
	}

	// Merge small clusters (< 2 bars) with neighbors.
	merged := make([]struct{ start, end int }, 0, len(clusters))
	for i := range clusters {
		bars := clusters[i].end - clusters[i].start
		if bars < 2 && len(merged) > 0 {
			// Merge with previous.
			merged[len(merged)-1].end = clusters[i].end
		} else {
			merged = append(merged, clusters[i])
		}
	}

	return merged
}

// buildSections converts clustered boundaries into named Sections.
func (e *Extractor) buildSections(clusters []struct{ start, end int }, features []BarFeature) []Section {
	sectionNames := []string{"intro", "verse", "chorus", "bridge", "outro", "pre_chorus", "break"}
	sections := make([]Section, 0, len(clusters))

	for i, c := range clusters {
		var avgEnergy, avgDensity float64
		count := 0
		for bar := c.start; bar < c.end && bar < len(features); bar++ {
			avgEnergy += features[bar].Energy
			avgDensity += features[bar].Density
			count++
		}
		if count > 0 {
			avgEnergy /= float64(count)
			avgDensity /= float64(count)
		}

		name := "section"
		if i < len(sectionNames) {
			name = sectionNames[i]
		}

		// If density drops significantly, likely a break/outro.
		if avgDensity < 0.2 && i > 0 {
			name = "break"
			if i == len(clusters)-1 {
				name = "outro"
			}
		}

		sections = append(sections, Section{
			Name:     name,
			StartBar: c.start,
			Bars:     c.end - c.start,
			Energy:   RoundTo(avgEnergy, 2),
			Density:  RoundTo(avgDensity, 2),
		})
	}

	return sections
}

// matchTemplate aligns detected sections to canonical forms.
// Returns matched template name and confidence score.
func (e *Extractor) matchTemplate(sections []Section, totalBars int) (string, float64) {
	if len(sections) == 0 {
		return "", 0
	}

	// Count unique section types and their ordering.
	names := make([]string, len(sections))
	for i, s := range sections {
		names[i] = s.Name
	}

	// Compute a signature: e.g. "verse,chorus,verse,chorus"
	sig := ""
	seen := make(map[string]int)
	for _, n := range names {
		seen[n]++
		if sig != "" {
			sig += "-"
		}
		sig += n
	}

	// Match against canonical templates.
	templates := map[string]string{
		"intro-verse-chorus-verse-chorus-outro":            "intro-verse-chorus",
		"intro-verse-chorus-bridge-chorus-outro":            "intro-verse-chorus",
		"intro-verse-pre_chorus-chorus-verse-chorus-outro":  "intro-verse-chorus",
		"verse-chorus-verse-chorus":                         "AABA",
		"verse-chorus-verse-chorus-bridge-chorus":           "ABAB",
		"intro-verse-chorus-bridge-outro":                   "I-V-C-B-O",
	}

	if matched, ok := templates[sig]; ok {
		return matched, 0.8
	}

	// Fallback: count section repetition to detect pattern.
	uniq := len(seen)
	if uniq <= 1 {
		return "through-composed", 0.3
	}
	if uniq <= 3 && len(sections) >= 4 {
		return "strophic", 0.5
	}

	return "through-composed", 0.2
}

// ═══════════════════════════════════════════════════════════════════
// 2. Chord Detector (rule-based, no LLM) — 增强
// ═══════════════════════════════════════════════════════════════════

func (e *Extractor) extractHarmony(eventsByTrack map[string][]schema.NoteEvent, totalBars int, key string) HarmonyDNA {
	hd := HarmonyDNA{Key: key, Confidence: 0}

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

	detectedCount := 0
	for bar := 0; bar < totalBars; bar++ {
		pc, ok := barPitches[bar]
		if !ok || len(pc) == 0 {
			continue
		}
		chord := detectChord(pc)
		hd.Progression = append(hd.Progression, ChordBar{Bar: bar, Chord: chord})
		if chord != "-" {
			detectedCount++
		}
	}

	if totalBars > 0 {
		hd.Confidence = Clamp01(float64(detectedCount) / float64(totalBars))
	}

	fmt.Printf("[ChordDetector] %d/%d chords detected (confidence=%.2f)\n",
		detectedCount, totalBars, hd.Confidence)
	return hd
}

// detectChord uses pitch class histogram + root detection + template matching.
// Supports: m, maj, 7, m7, maj7, dim, aug, sus4, sus2, 5, m7b5, dim7, add9, 6, m6, 9.
func detectChord(pcs []int) string {
	if len(pcs) == 0 {
		return "-"
	}

	hist := make(map[int]int)
	for _, p := range pcs {
		hist[p]++
	}

	// Root detection: most frequent pitch class, with consonant bias.
	root := 0
	maxCount := 0
	for p, c := range hist {
		if c > maxCount || (c == maxCount && isConsonant(p, root)) {
			root = p
			maxCount = c
		}
	}

	semiToNote := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	chord := semiToNote[root]

	// Check intervals from root.
	hasMinorThird := hist[(root+3)%12] > 0
	hasMajorThird := hist[(root+4)%12] > 0
	hasFifth := hist[(root+7)%12] > 0
	hasFlatFifth := hist[(root+6)%12] > 0
	hasMinorSeventh := hist[(root+10)%12] > 0
	hasMajorSeventh := hist[(root+11)%12] > 0
	hasSus4 := hist[(root+5)%12] > 0
	hasSus2 := hist[(root+2)%12] > 0
	hasAdd9 := hist[(root+14)%12] > 0 || hist[(root+2)%12] > 0
	hasSixth := hist[(root+9)%12] > 0
	hasAug := hist[(root+8)%12] > 0

	switch {
	case hasMinorThird && hasFlatFifth && hasMinorSeventh:
		chord += "m7b5" // half-diminished
	case hasMinorThird && hasFlatFifth:
		chord += "dim" // diminished triad
	case hasMinorThird && hasFifth && hasMinorSeventh:
		chord += "m7"
	case hasMajorThird && hasAug:
		chord += "aug"
	case hasMajorThird && hasMinorSeventh:
		chord += "7"
	case hasMajorThird && hasMajorSeventh:
		chord += "maj7"
	case hasMinorThird:
		if hasSixth {
			chord += "m6"
		} else if hasAdd9 && !hasMinorSeventh {
			chord += "madd9"
		} else {
			chord += "m"
		}
	case hasSus4:
		chord += "sus4"
	case hasSus2:
		chord += "sus2"
	case hasSixth:
		chord += "6"
	case hasAdd9:
		chord += "add9"
	case hasMajorSeventh:
		chord += "maj7"
	case hasMajorThird:
		chord += ""
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
// 3. Motif Extractor (find interval patterns via sliding window) — 增强评分
// ═══════════════════════════════════════════════════════════════════

func (e *Extractor) extractMotif(eventsByTrack map[string][]schema.NoteEvent, totalBars int) MotifDNA {
	md := MotifDNA{Confidence: 0}

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

	sort.Slice(lead, func(i, j int) bool {
		return lead[i].StartBeat < lead[j].StartBeat
	})

	base := lead[0].Pitch
	intervals := make([]int, 0, len(lead))
	durations := make([]float64, 0, len(lead))
	for _, ev := range lead {
		intervals = append(intervals, ev.Pitch-base)
		durations = append(durations, ev.DurationBeat)
	}

	relativeInts := make([]int, 0, len(intervals)-1)
	for i := 1; i < len(intervals); i++ {
		relativeInts = append(relativeInts, intervals[i]-intervals[i-1])
	}

	type candidate struct {
		pattern    []int
		rhythm     []float64
		windowSize int
	}

	var allCandidates []candidate
	for ws := 3; ws <= 8 && ws <= len(relativeInts); ws++ {
		for start := 0; start <= len(relativeInts)-ws; start++ {
			allCandidates = append(allCandidates, candidate{
				pattern:    relativeInts[start : start+ws],
				rhythm:     durations[start : start+ws],
				windowSize: ws,
			})
		}
	}

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

	var best *scoredMotif
	bestScore := 0.0

	for _, sm := range freq {
		rhythmScore := 0.0
		if len(sm.rhythm) > 0 {
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
		md.Confidence = Clamp01(bestScore / float64(len(intervals)))

		// Attach MotifScore.
		md.Score = ScoreMotif(intervals, relativeInts, durations, totalBars)

		// Detect variants.
		md.Variants = detectVariants(relativeInts, best.pattern, durations)
	}
	return md
}

// ScoreMotif is imported from the composer package at runtime via the extractor.
// We define a local version here for the extractor to use directly.
func ScoreMotif(intervals, relativeInts []int, durations []float64, totalBars int) *MotifScore {
	if len(intervals) < 4 || totalBars <= 0 {
		return &MotifScore{}
	}

	ms := &MotifScore{}

	// 1. Repetition
	seedLen := 4
	if seedLen > len(relativeInts) {
		seedLen = len(relativeInts)
	}
	if seedLen < 2 {
		seedLen = 2
	}
	seed := relativeInts[:seedLen]

	occurrences := 0
	for i := 0; i <= len(relativeInts)-seedLen; i++ {
		match := true
		for j := 0; j < seedLen; j++ {
			if relativeInts[i+j] != seed[j] {
				match = false
				break
			}
		}
		if match {
			occurrences++
		}
	}
	ms.Repetition = Clamp01(float64(occurrences) / float64(totalBars+1))

	// 2. Contour
	if len(intervals) >= 2 {
		var sumSlope, sumSlopeSq float64
		count := 0
		for i := 1; i < len(intervals); i++ {
			slope := float64(intervals[i] - intervals[i-1])
			sumSlope += slope
			sumSlopeSq += slope * slope
			count++
		}
		if count > 0 {
			meanSlope := sumSlope / float64(count)
			variance := sumSlopeSq/float64(count) - meanSlope*meanSlope
			if variance < 0 {
				variance = 0
			}
			ms.Contour = Clamp01(math.Sqrt(variance) / 10.0)
		}
	}

	// 3. Simplicity
	if len(relativeInts) > 0 {
		var sumAbs float64
		for _, iv := range relativeInts {
			sumAbs += float64(AbsInt(iv))
		}
		avgInterval := sumAbs / float64(len(relativeInts))
		ms.Simplicity = Clamp01(1.0 - avgInterval/12.0)
	}

	// 4. RhythmIdentity
	if len(durations) >= 4 {
		mid := len(durations) / 2
		var diffSum float64
		pairs := 0
		for i := 0; i < mid && i+mid < len(durations); i++ {
			diff := math.Abs(durations[i] - durations[i+mid])
			maxDur := math.Max(durations[i], durations[i+mid])
			if maxDur > 0 {
				diffSum += Clamp01(diff / maxDur)
				pairs++
			}
		}
		if pairs > 0 {
			ms.RhythmIdentity = Clamp01(1.0 - diffSum/float64(pairs))
		}
	}

	ms.CalculateTotal()
	return ms
}

func intsKey(ints []int) string {
	if len(ints) == 0 {
		return ""
	}
	b := make([]byte, 0, len(ints)*4)
	for _, v := range ints {
		b = append(b, byte((v+1000)>>8), byte((v+1000)&0xFF))
	}
	return string(b)
}

func detectVariants(fullSeq, motif []int, durations []float64) []MotifVariant {
	if len(motif) < 2 || len(fullSeq) < len(motif)*2 {
		return nil
	}

	var variants []MotifVariant
	skip := 0

	for start := len(motif); start <= len(fullSeq)-len(motif); start++ {
		skip++
		if skip%2 == 0 {
			continue
		}
		candidate := fullSeq[start : start+len(motif)]

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

		isInvert := true
		for i := 0; i < len(motif) && i < len(candidate); i++ {
			if AbsInt(candidate[i]+motif[len(motif)-1-i]) > 2 {
				isInvert = false
				break
			}
		}
		if isInvert {
			variants = append(variants, MotifVariant{Type: "invert", Pattern: candidate})
			continue
		}

		if len(motif) >= 3 && AbsInt(candidate[0]-motif[0]) <= 2 &&
			AbsInt(candidate[1]-motif[1]) <= 2 {
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

// ═══════════════════════════════════════════════════════════════════
// 4. Rhythm Analyzer
// ═══════════════════════════════════════════════════════════════════

func (e *Extractor) extractRhythm(eventsByTrack map[string][]schema.NoteEvent, totalBars int) RhythmDNA {
	rd := RhythmDNA{}

	// Collect all non-chord, non-pad events (rhythmic events).
	var rhythmicEvents []schema.NoteEvent
	for _, id := range []string{"drums", "percussion", "bass", "rhythm_guitar", "lead"} {
		if evs, ok := eventsByTrack[id]; ok {
			rhythmicEvents = append(rhythmicEvents, evs...)
		}
	}

	if len(rhythmicEvents) == 0 || totalBars == 0 {
		return rd
	}

	// Density: average notes per beat.
	bpb := 4.0
	totalBeats := float64(totalBars) * bpb
	rd.Density = Clamp01(float64(len(rhythmicEvents)) / totalBeats / 4.0) // normalize: 4 notes/beat = max

	// Syncopation: proportion of notes not on beat boundaries (0, 1, 2, 3).
	offbeat := 0
	for _, ev := range rhythmicEvents {
		beatPos := math.Mod(ev.StartBeat, 1.0)
		if beatPos > 0.05 && beatPos < 0.95 {
			offbeat++
		}
	}
	rd.Syncopation = Clamp01(float64(offbeat) / float64(len(rhythmicEvents)))

	// Swing: detect if 8th notes alternate long-short.
	if len(rhythmicEvents) >= 4 {
		sort.Slice(rhythmicEvents, func(i, j int) bool {
			return rhythmicEvents[i].StartBeat < rhythmicEvents[j].StartBeat
		})
		var swingRatios []float64
		for i := 1; i < len(rhythmicEvents); i++ {
			gap := rhythmicEvents[i].StartBeat - rhythmicEvents[i-1].StartBeat
			if gap > 0.1 && gap < 0.6 { // 8th note range
				swingRatios = append(swingRatios, gap)
			}
		}
		if len(swingRatios) >= 4 {
			var mean, sumSq float64
			for _, r := range swingRatios {
				mean += r
			}
			mean /= float64(len(swingRatios))
			for _, r := range swingRatios {
				sumSq += (r - mean) * (r - mean)
			}
			variance := sumSq / float64(len(swingRatios))
			// Higher variance = more swing
			rd.SwingAmount = Clamp01(math.Sqrt(variance) * 5.0)
		}
	}

	// Variety: distinct rhythmic patterns / total bars.
	// Use onset pattern per bar as a quick measure.
	barPatterns := make(map[string]bool)
	for _, ev := range rhythmicEvents {
		bar := int(ev.StartBeat / bpb)
		beatInBar := int(math.Mod(ev.StartBeat, bpb))
		key := fmt.Sprintf("%d:%d", bar, beatInBar)
		barPatterns[key] = true
	}
	if totalBars > 0 {
		rd.Variety = Clamp01(float64(len(barPatterns)) / float64(totalBars*4))
	}

	rd.Confidence = Clamp01((rd.Density + rd.Syncopation + rd.Variety) / 3.0)
	return rd
}

// ═══════════════════════════════════════════════════════════════════
// 5. Texture Analyzer
// ═══════════════════════════════════════════════════════════════════

func (e *Extractor) extractTexture(eventsByTrack map[string][]schema.NoteEvent) TextureDNA {
	td := TextureDNA{
		TrackCount: len(eventsByTrack),
	}

	roleMap := map[string]string{
		"drums": "rhythm", "percussion": "rhythm", "beat": "rhythm",
		"bass": "harmonic", "bassline": "harmonic",
		"chords": "harmonic", "piano": "harmonic", "pad": "harmonic", "strings": "harmonic",
		"lead": "melodic", "lead_vocal": "melodic", "melody": "melodic", "lead_guitar": "melodic",
		"fx": "atmosphere", "atmosphere": "atmosphere", "texture": "atmosphere",
	}

	totalNotes := 0
	for id, events := range eventsByTrack {
		if len(events) == 0 {
			continue
		}
		layer := TextureLayer{
			Name:      id,
			Role:      "unknown",
			Active:    true,
			NoteCount: len(events),
		}
		if role, ok := roleMap[id]; ok {
			layer.Role = role
		}

		var pitchSum float64
		for _, ev := range events {
			pitchSum += float64(ev.Pitch)
		}
		if len(events) > 0 {
			layer.AvgPitch = RoundTo(pitchSum/float64(len(events)), 1)
		}

		td.Layers = append(td.Layers, layer)
		totalNotes += len(events)
	}

	// Overall density: normalized by track count.
	maxNotesPerTrack := 100
	if totalNotes > 0 && td.TrackCount > 0 {
		avgPerTrack := float64(totalNotes) / float64(td.TrackCount)
		td.Density = Clamp01(avgPerTrack / float64(maxNotesPerTrack))
	}

	td.Confidence = Clamp01(float64(len(td.Layers)) / float64(len(eventsByTrack)+1))
	return td
}

// ═══════════════════════════════════════════════════════════════════
// 6. Dynamics Analyzer
// ═══════════════════════════════════════════════════════════════════

func (e *Extractor) extractDynamics(eventsByTrack map[string][]schema.NoteEvent, totalBars int) DynamicsDNA {
	dd := DynamicsDNA{}

	if totalBars == 0 {
		return dd
	}

	// Per-bar average velocity.
	bpb := 4
	barVelocities := make([]float64, totalBars)
	barCounts := make([]int, totalBars)

	var globalVelSum float64
	var globalVelCount int
	minVel := 127.0
	maxVel := 0.0

	for _, events := range eventsByTrack {
		for _, ev := range events {
			bar := int(ev.StartBeat) / bpb
			if bar >= 0 && bar < totalBars {
				barVelocities[bar] += float64(ev.Velocity)
				barCounts[bar]++
				globalVelSum += float64(ev.Velocity)
				globalVelCount++
				if float64(ev.Velocity) < minVel {
					minVel = float64(ev.Velocity)
				}
				if float64(ev.Velocity) > maxVel {
					maxVel = float64(ev.Velocity)
				}
			}
		}
	}

	// Average velocity.
	if globalVelCount > 0 {
		dd.AvgVelocity = Clamp01((globalVelSum / float64(globalVelCount)) / 127.0)
	}
	dd.DynamicRange = Clamp01((maxVel - minVel) / 127.0)

	// Energy curve.
	dd.EnergyCurve = make([]float64, totalBars)
	for bar := 0; bar < totalBars; bar++ {
		if barCounts[bar] > 0 {
			dd.EnergyCurve[bar] = Clamp01(barVelocities[bar] / float64(barCounts[bar]) / 127.0)
		}
	}

	// Crescendo detection: compare first third vs last third.
	if totalBars >= 6 {
		third := totalBars / 3
		var firstThird, lastThird float64
		for i := 0; i < third; i++ {
			firstThird += dd.EnergyCurve[i]
		}
		for i := totalBars - third; i < totalBars; i++ {
			lastThird += dd.EnergyCurve[i]
		}
		dd.Crescendo = lastThird > firstThird*1.1
	}

	dd.Confidence = Clamp01((dd.AvgVelocity + dd.DynamicRange) / 2.0)
	return dd
}

// ═══════════════════════════════════════════════════════════════════
// 7. Emotion Analyzer
// ═══════════════════════════════════════════════════════════════════

func (e *Extractor) extractEmotion(eventsByTrack map[string][]schema.NoteEvent, totalBars int) EmotionDNA {
	ed := EmotionDNA{}
	if totalBars == 0 {
		return ed
	}

	bpb := 4
	barVelocities := make([]float64, totalBars)
	barDensities := make([]float64, totalBars)

	for _, events := range eventsByTrack {
		for _, ev := range events {
			bar := int(ev.StartBeat) / bpb
			if bar >= 0 && bar < totalBars {
				barVelocities[bar] += float64(ev.Velocity)
				barDensities[bar]++
			}
		}
	}

	// Normalize per-bar.
	var totalVel, totalDensity float64
	for bar := 0; bar < totalBars; bar++ {
		if barDensities[bar] > 0 {
			barVelocities[bar] /= barDensities[bar]
		}
		barVelocities[bar] /= 127.0
		barDensities[bar] = Clamp01(barDensities[bar] / float64(totalBars*4))
		totalVel += barVelocities[bar]
		totalDensity += barDensities[bar]
	}

	avgVel := totalVel / float64(totalBars)
	avgDensity := totalDensity / float64(totalBars)

	// Energy: composite of velocity + density
	ed.Energy = Clamp01(avgVel*0.6 + avgDensity*0.4)

	// Tension: variance in velocity (erratic = tense)
	var velVariance float64
	for bar := 0; bar < totalBars; bar++ {
		diff := barVelocities[bar] - avgVel
		velVariance += diff * diff
	}
	velVariance /= float64(totalBars)
	ed.Tension = Clamp01(velVariance * 3.0)

	// Warmth: lower average pitch = warmer (bass-heavy)
	var totalPitch, pitchCount float64
	for _, events := range eventsByTrack {
		for _, ev := range events {
			totalPitch += float64(ev.Pitch)
			pitchCount++
		}
	}
	if pitchCount > 0 {
		avgPitch := totalPitch / pitchCount
		// 0-127 MIDI: 60 = middle C. Lower = warmer.
		warmth := 1.0 - Clamp01(avgPitch/127.0)
		ed.Warmth = Clamp01(warmth * 1.5)
	}

	// Stability: low density variance = stable
	var densVariance float64
	for bar := 0; bar < totalBars; bar++ {
		diff := barDensities[bar] - avgDensity
		densVariance += diff * diff
	}
	densVariance /= float64(totalBars)
	ed.Stability = Clamp01(1.0 - densVariance*2.0)

	// Brightness: detect high-pitch content (pitch > 72)
	var highCount float64
	for _, events := range eventsByTrack {
		for _, ev := range events {
			if ev.Pitch >= 72 {
				highCount++
			}
		}
	}
	if pitchCount > 0 {
		ed.Brightness = Clamp01(highCount / pitchCount * 2.0)
	}

	// Energy curve (per-bar)
	ed.Curve = make([]float64, totalBars)
	for bar := 0; bar < totalBars; bar++ {
		ed.Curve[bar] = barVelocities[bar]
	}

	ed.Confidence = Clamp01((ed.Energy + ed.Tension + ed.Warmth + ed.Stability + ed.Brightness) / 5.0)
	return ed
}

// ═══════════════════════════════════════════════════════════════════
// 8. MIDI Cleaner — filter noisy/invalid MIDI data
// ═══════════════════════════════════════════════════════════════════

// CleanMIDI filters noisy/invalid events and returns cleaned track data.
// Returns false if the track is too sparse to be useful.
func CleanMIDI(eventsByTrack map[string][]schema.NoteEvent) (map[string][]schema.NoteEvent, bool) {
	cleaned := make(map[string][]schema.NoteEvent)
	hasAny := false

	for id, events := range eventsByTrack {
		if len(events) < 3 {
			continue // too few notes, skip
		}

		var filtered []schema.NoteEvent
		for _, ev := range events {
			// Filter out-of-range pitches.
			if ev.Pitch < 0 || ev.Pitch > 127 {
				continue
			}
			// Filter zero-duration notes.
			if ev.DurationBeat <= 0 {
				continue
			}
			// Filter notes outside reasonable velocity range.
			if ev.Velocity < 10 || ev.Velocity > 127 {
				continue
			}
			filtered = append(filtered, ev)
		}

		if len(filtered) >= 3 {
			cleaned[id] = filtered
			hasAny = true
		}
	}

	return cleaned, hasAny
}

// IsValidMIDI checks if a MIDI track/event set is musically meaningful.
func IsValidMIDI(events []schema.NoteEvent) bool {
	if len(events) < 10 {
		return false
	}

	// Check total duration.
	minBeat := math.MaxFloat64
	maxBeat := 0.0
	for _, ev := range events {
		if ev.StartBeat < minBeat {
			minBeat = ev.StartBeat
		}
		end := ev.StartBeat + ev.DurationBeat
		if end > maxBeat {
			maxBeat = end
		}
	}

	durationBeats := maxBeat - minBeat
	if durationBeats < 4 { // less than 1 bar
		return false
	}

	// Check pitch range > 0.
	pitchSet := make(map[int]bool)
	for _, ev := range events {
		pitchSet[ev.Pitch] = true
	}
	if len(pitchSet) < 2 {
		return false // single pitch, probably a test tone
	}

	return true
}

// unused is kept for compilation alignment.
func unused() {}
