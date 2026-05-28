// Package musicdna — MIDI → DNA extractor.
// Extracts structure, harmony, and motif from raw MIDI data.
// Designed for real-world polyphonic MIDI, not just LLM output.
package musicdna

import (
	"fmt"
	"math"
	"sort"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// Extractor converts raw MIDI events into structured MusicDNA.
type Extractor struct{}

func NewExtractor() *Extractor { return &Extractor{} }

// Extract runs the full pipeline on grouped track events.
func (e *Extractor) Extract(eventsByTrack map[string][]schema.NoteEvent, totalBars int, key string) *MusicDNA {
	dna := &MusicDNA{
		Motif:   e.extractMotif(eventsByTrack),
		Harmony: e.extractHarmony(eventsByTrack, totalBars, key),
	}
	if totalBars > 0 {
		dna.Structure = e.extractStructure(eventsByTrack, totalBars)
	}
	return dna
}

// ═══════════════════════════════════════════════════════════════════
// 1. Motif Extraction — find the most musically significant pattern
// ═══════════════════════════════════════════════════════════════════

func (e *Extractor) extractMotif(eventsByTrack map[string][]schema.NoteEvent) MotifDNA {
	md := MotifDNA{Confidence: 0}

	// Find the lead track: the most melodic track in the 60-84 range.
	leadTrack := findLeadTrack(eventsByTrack)
	if leadTrack == "" {
		return md
	}

	events := eventsByTrack[leadTrack]
	if len(events) < 8 {
		return md
	}

	// Sort by time.
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartBeat < events[j].StartBeat
	})

	// Convert to note sequence: just the pitches in order.
	// Filter out simultaneous notes (keep highest pitch).
	var pitches []int
	lastBeat := -1.0
	for _, ev := range events {
		if ev.StartBeat != lastBeat {
			pitches = append(pitches, ev.Pitch)
			lastBeat = ev.StartBeat
		}
	}

	if len(pitches) < 8 {
		return md
	}

	// Convert to intervals (relative semitone changes).
	intervals := make([]int, len(pitches)-1)
	for i := 0; i < len(pitches)-1; i++ {
		intervals[i] = pitches[i+1] - pitches[i]
	}

	// Sliding window: find the most repeated interval sequence.
	// Window sizes: 3, 4, 5 notes.
	type candidate struct {
		pattern []int
		score   int // frequency * length
	}

	var best candidate
	best.score = 0

	for ws := 3; ws <= 5 && ws <= len(intervals)/2; ws++ {
		freq := make(map[string]int)
		for i := 0; i <= len(intervals)-ws; i++ {
			key := fmt.Sprintf("%v", intervals[i:i+ws])
			freq[key]++
		}
		for key, count := range freq {
			if count > 1 { // at least 2 occurrences
				score := count * ws
				if score > best.score {
					best.score = score
					// Parse key back to pattern
					best.pattern = parsePattern(key)
				}
			}
		}
	}

	if best.score > 0 && len(best.pattern) > 0 {
		md.Pattern = best.pattern
		md.Confidence = float64(best.score) / float64(len(intervals))
		if md.Confidence > 1.0 {
			md.Confidence = 1.0
		}

		// Build rhythm from first occurrence of the motif.
		// Match the pattern start in intervals.
		motifLen := len(md.Pattern)
		for i := 0; i <= len(intervals)-motifLen; i++ {
			match := true
			for j := 0; j < motifLen; j++ {
				if intervals[i+j] != md.Pattern[j] {
					match = false
					break
				}
			}
			if match {
				for j := 0; j <= motifLen && i+j < len(pitches); j++ {
					md.Rhythm = append(md.Rhythm, 0.5)
				}
				break
			}
		}
	}

	fmt.Printf("[Motif] %s: interval=%v, score=%d, conf=%.2f\n",
		leadTrack, md.Pattern, best.score, md.Confidence)
	return md
}

// findLeadTrack picks the most melodic track from the event map.
func findLeadTrack(eventsByTrack map[string][]schema.NoteEvent) string {
	type trackScore struct {
		id    string
		score float64
	}

	var scores []trackScore
	for id, events := range eventsByTrack {
		if len(events) < 5 {
			continue
		}

		sumPitch := 0
		minPitch, maxPitch := 127, 0
		for _, ev := range events {
			sumPitch += ev.Pitch
			if ev.Pitch < minPitch {
				minPitch = ev.Pitch
			}
			if ev.Pitch > maxPitch {
				maxPitch = ev.Pitch
			}
		}
		avgPitch := float64(sumPitch) / float64(len(events))
		pitchRange := maxPitch - minPitch

		// Score: prefer 60-84 range, moderate range, not too dense.
		rangeScore := 1.0 - float64(pitchRange)/84.0
		if rangeScore < 0 {
			rangeScore = 0
		}

		pitchScore := 0.0
		if avgPitch >= 55 && avgPitch <= 85 {
			pitchScore = 1.0 - math.Abs(avgPitch-70)/30.0
		}

		// Penalize very dense tracks (chords, not melody).
		densityPenalty := 1.0
		if len(events) > 100 {
			densityPenalty = 0.5
		}
		if len(events) > 200 {
			densityPenalty = 0.2
		}

		score := (rangeScore*0.4 + pitchScore*0.6) * densityPenalty
		scores = append(scores, trackScore{id: id, score: score})
	}

	if len(scores) == 0 {
		return ""
	}

	best := scores[0]
	for _, s := range scores[1:] {
		if s.score > best.score {
			best = s
		}
	}

	return best.id
}

// parsePattern converts a fmt.Sprintf("%v") key back to []int.
func parsePattern(key string) []int {
	var result []int
	current := 0
	sign := 1
	inNumber := false

	for _, c := range key {
		if c >= '0' && c <= '9' {
			current = current*10 + int(c-'0')
			inNumber = true
		} else if c == '-' {
			sign = -1
		} else {
			if inNumber {
				result = append(result, current*sign)
				current = 0
				sign = 1
				inNumber = false
			}
		}
	}
	if inNumber {
		result = append(result, current*sign)
	}

	return result
}

// ═══════════════════════════════════════════════════════════════════
// 2. Harmony Extraction — chord detection from any track
// ═══════════════════════════════════════════════════════════════════

func (e *Extractor) extractHarmony(eventsByTrack map[string][]schema.NoteEvent, totalBars int, key string) HarmonyDNA {
	hd := HarmonyDNA{Key: key}

	// Collect ALL note events (any track) for chord detection.
	var allEvents []schema.NoteEvent
	for _, events := range eventsByTrack {
		allEvents = append(allEvents, events...)
	}

	if len(allEvents) < 10 {
		return hd
	}

	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].StartBeat < allEvents[j].StartBeat
	})

	// Group by bar, collect all pitch classes per bar.
	barPC := make(map[int]map[int]int) // bar → pitch class → count
	for _, ev := range allEvents {
		bar := int(ev.StartBeat) / 4
		if bar >= totalBars {
			continue
		}
		if barPC[bar] == nil {
			barPC[bar] = make(map[int]int)
		}
		barPC[bar][ev.Pitch%12]++
	}

	semiToNote := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

	for bar := 0; bar < totalBars; bar++ {
		pcs := barPC[bar]
		if len(pcs) < 2 {
			continue
		}

		// Find the most common pitch class (likely the root).
		root := 0
		maxCount := 0
		for pc, count := range pcs {
			if count > maxCount {
				maxCount = count
				root = pc
			}
		}

		// Determine chord quality.
		hasMinorThird := pcs[(root+3)%12] > 0
		hasMajorThird := pcs[(root+4)%12] > 0
			hasMinorSeventh := pcs[(root+10)%12] > 0
		hasMajorSeventh := pcs[(root+11)%12] > 0
		hasSus4 := pcs[(root+5)%12] > 0

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
		case hasSus4 && !hasMajorThird && !hasMinorThird:
			chord += "sus4"
		}

		hd.Progression = append(hd.Progression, ChordBar{Bar: bar, Chord: chord})
	}

	fmt.Printf("[Harmony] %d chords across %d bars (key=%s)\n", len(hd.Progression), totalBars, key)
	return hd
}

// ═══════════════════════════════════════════════════════════════════
// 3. Structure Extraction — section segmentation
// ═══════════════════════════════════════════════════════════════════

func (e *Extractor) extractStructure(eventsByTrack map[string][]schema.NoteEvent, totalBars int) StructureDNA {
	var sd StructureDNA
	if totalBars < 2 {
		return sd
	}

	// Compute per-bar energy and density.
	type barMetric struct {
		density  float64
		energy   float64
	}
	metrics := make([]barMetric, totalBars)

	for _, events := range eventsByTrack {
		for _, ev := range events {
			bar := int(ev.StartBeat) / 4
			if bar >= 0 && bar < totalBars {
				metrics[bar].density++
				metrics[bar].energy += float64(ev.Velocity) / 127.0
			}
		}
	}

	// Normalize density.
	maxDensity := 1.0
	for _, m := range metrics {
		if m.density > maxDensity {
			maxDensity = m.density
		}
	}
	for i := range metrics {
		metrics[i].density /= maxDensity
	}

	// Detect boundaries: look for significant energy/density shifts.
	type boundary struct{ bar int }
	var boundaries []boundary
	boundaries = append(boundaries, boundary{0})

	for bar := 1; bar < totalBars; bar++ {
		energyShift := math.Abs(metrics[bar].energy - metrics[bar-1].energy)
		densityShift := math.Abs(metrics[bar].density - metrics[bar-1].density)
		if energyShift > 0.2 || densityShift > 0.3 {
			boundaries = append(boundaries, boundary{bar})
		}
	}

	// Build sections.
	sectionNames := []string{"intro", "verse", "chorus", "bridge", "outro"}
	for i, b := range boundaries {
		endBar := totalBars
		if i+1 < len(boundaries) {
			endBar = boundaries[i+1].bar
		}

		avgEnergy, avgDensity := 0.0, 0.0
		count := 0
		for bar := b.bar; bar < endBar && bar < totalBars; bar++ {
			avgEnergy += metrics[bar].energy
			avgDensity += metrics[bar].density
			count++
		}
		if count > 0 {
			avgEnergy /= float64(count)
			avgDensity /= float64(count)
		}

		name := ""
		if i < len(sectionNames) {
			name = sectionNames[i]
		} else {
			name = fmt.Sprintf("section_%d", i)
		}

		if name == "" { name = "section" }
		sd.Sections = append(sd.Sections, Section{
			Name:     name,
			StartBar: b.bar,
			Bars:     endBar - b.bar,
			Energy:   avgEnergy,
			Density:  avgDensity,
		})
	}

	fmt.Printf("[Structure] %d sections\n", len(sd.Sections))
	return sd
}
