// Package musicdna — MusicDNA Extractor.
// Analyzes eventsByTrack and extracts structured DNA:
//   - Bar segmentation
//   - Chord inference
//   - Motif extraction (sliding window + interval clustering)
//   - Energy curve
//   - Instrument timeline
package musicdna

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// Extractor converts a raw MIDI event map into structured MusicDNA.
type Extractor struct{}

// NewExtractor creates a new extractor.
func NewExtractor() *Extractor {
	return &Extractor{}
}

// Extract analyzes events and returns a complete MusicDNA.
func (e *Extractor) Extract(eventsByTrack map[string][]schema.NoteEvent, totalBars int, key, bpm string) *MusicDNA {
	dna := &MusicDNA{}

	if totalBars <= 0 {
		totalBars = 1
	}

	dna.Structure = e.extractStructure(eventsByTrack, totalBars)
	dna.Harmony = e.extractHarmony(eventsByTrack, totalBars, key)
	dna.Motif = e.extractMotif(eventsByTrack, totalBars)
	dna.Rhythm = e.extractRhythm(eventsByTrack, totalBars)
	dna.Texture = e.extractTexture(eventsByTrack, totalBars)

	return dna
}

// ─── Structure Extraction ──────────────────────────────────────────

func (e *Extractor) extractStructure(eventsByTrack map[string][]schema.NoteEvent, totalBars int) StructureDNA {
	sd := StructureDNA{}

	// Divide into 4 sections (intro, verse, chorus, outro).
	sectionSize := totalBars / 4
	if sectionSize < 1 {
		sectionSize = 1
	}

	sectionNames := []string{"intro", "verse", "chorus", "outro"}
	for i, name := range sectionNames {
		startBar := i * sectionSize
		endBar := startBar + sectionSize
		if endBar > totalBars {
			endBar = totalBars
		}
		if i == len(sectionNames)-1 {
			endBar = totalBars
		}

		// Calculate energy and density for this section.
		totalVel := 0
		totalNotes := 0
		activeInsts := make(map[string]bool)
		velCount := 0

		for trackID, events := range eventsByTrack {
			for _, ev := range events {
				bar := int(ev.StartBeat) / 4
				if bar >= startBar && bar < endBar {
					totalVel += ev.Velocity
					velCount++
					totalNotes++
					activeInsts[trackID] = true
				}
			}
		}

		energy := 0.5
		if velCount > 0 {
			energy = float64(totalVel) / float64(velCount) / 127.0
		}
		density := float64(totalNotes) / float64(endBar-startBar) / 8.0
		if density > 1.0 {
			density = 1.0
		}

		insts := make([]string, 0, len(activeInsts))
		for id := range activeInsts {
			insts = append(insts, id)
		}
		sort.Strings(insts)

		sd.Sections = append(sd.Sections, Section{
			Name:        name,
			StartBar:    startBar,
			Bars:        endBar - startBar,
			Energy:      energy,
			Density:     density,
			Instruments: insts,
		})
	}

	return sd
}

// ─── Harmony Extraction ────────────────────────────────────────────

func (e *Extractor) extractHarmony(eventsByTrack map[string][]schema.NoteEvent, totalBars int, key string) HarmonyDNA {
	hd := HarmonyDNA{
		Key: key,
	}

	// Collect chord events grouped by bar.
	type barChord struct {
		bar      int
		pitches  []int
		chord    string
		function string
	}

	var barChords []barChord
	chordEvents := eventsByTrack["chords"]
	if len(chordEvents) == 0 {
		// Try "piano" or other chord-capable tracks.
		for _, try := range []string{"piano", "pad", "strings", "rhythm_guitar"} {
			if evs, ok := eventsByTrack[try]; ok && len(evs) > 0 {
				chordEvents = evs
				break
			}
		}
	}

	// Group chord notes by bar.
	barMap := make(map[int][]int)
	for _, ev := range chordEvents {
		bar := int(ev.StartBeat) / 4
		barMap[bar] = append(barMap[bar], ev.Pitch%12)
	}

	for bar := 0; bar < totalBars; bar++ {
		pc := barMap[bar]
		if len(pc) == 0 {
			continue
		}
		b := barChord{bar: bar, pitches: pc}
		b.chord = inferChord(pc)
		if key == "C major" || key == "C_Major" {
			b.function = chordFunction(b.chord)
		}
		barChords = append(barChords, b)
	}

	for _, bc := range barChords {
		hd.Progression = append(hd.Progression, ChordBar{
			Bar:      bc.bar,
			Chord:    bc.chord,
			Function: bc.function,
		})
	}

	return hd
}

// inferChord tries to determine the chord name from a set of pitch classes.
func inferChord(pcs []int) string {
	if len(pcs) == 0 {
		return "-"
	}
	// Count occurrences of each pitch class.
	counts := make(map[int]int)
	for _, p := range pcs {
		counts[p]++
	}

	// Find root (most common, or lowest note if tie).
	bestPC, bestCount := 0, 0
	for p, c := range counts {
		if c > bestCount || (c == bestCount && p < bestPC) {
			bestPC = p
			bestCount = c
		}
	}

	// Check for major/minor third.
	hasMajorThird := counts[(bestPC+4)%12] > 0
	hasMinorThird := counts[(bestPC+3)%12] > 0
	hasSeventh := counts[(bestPC+10)%12] > 0 || counts[(bestPC+11)%12] > 0

	semiToNote := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	root := semiToNote[bestPC]

	chord := root
	if hasMinorThird && !hasMajorThird {
		chord += "m"
	}
	if hasSeventh {
		chord += "7"
	}
	if !hasMinorThird && !hasMajorThird && !hasSeventh {
		chord += "5" // power chord
	}

	return chord
}

// chordFunction returns the harmonic function in C major.
func chordFunction(chord string) string {
	funcMap := map[string]string{
		"C": "T", "Dm": "S", "Em": "T_iii", "F": "S", "G": "D", "Am": "T_vi",
		"Bdim": "D_vii", "C7": "D", "Dm7": "S", "Em7": "T_iii", "Fmaj7": "S",
		"G7": "D", "Am7": "T_vi", "C5": "T", "G5": "D", "F5": "S",
	}
	if f, ok := funcMap[chord]; ok {
		return f
	}
	return "-"
}

// ─── Motif Extraction ──────────────────────────────────────────────

func (e *Extractor) extractMotif(eventsByTrack map[string][]schema.NoteEvent, totalBars int) MotifDNA {
	md := MotifDNA{}

	// Get lead melody events.
	leadEvents := eventsByTrack["lead"]
	if len(leadEvents) == 0 {
		for _, try := range []string{"lead_guitar", "lead_vocal", "piano"} {
			if evs, ok := eventsByTrack[try]; ok && len(evs) > 0 {
				leadEvents = evs
				break
			}
		}
	}
	if len(leadEvents) < 3 {
		return md
	}

	// Take first phrase as motif (first 4-6 notes).
	n := 5
	if len(leadEvents) < n {
		n = len(leadEvents)
	}

	// Calculate relative intervals from first note.
	base := leadEvents[0].Pitch
	md.Notes = make([]int, n)
	md.Rhythm = make([]float64, n)
	for i := 0; i < n; i++ {
		md.Notes[i] = leadEvents[i].Pitch - base
		md.Rhythm[i] = leadEvents[i].DurationBeat
	}

	// Detect variations in later phrases.
	phrases := splitPhrases(leadEvents, 4)
	for p := 1; p < len(phrases); p++ {
		if len(phrases[p]) < 3 {
			continue
		}
		// Compare with motif.
		v := detectVariant(phrases[0], phrases[p], base)
		if v != "" {
			md.Variants = append(md.Variants, MotifVariant{
				Type:  v,
				Notes: extractRelative(phrases[p], base),
			})
		}
	}

	fmt.Printf("[MusicDNA] motif: %v, variants: %d\n", md.Notes, len(md.Variants))
	return md
}

func splitPhrases(events []schema.NoteEvent, barsPerPhrase int) [][]schema.NoteEvent {
	if len(events) == 0 {
		return nil
	}
	phrases := make([][]schema.NoteEvent, 0)
	currentPhrase := -1

	for _, ev := range events {
		bar := int(ev.StartBeat) / 4
		if bar/barsPerPhrase != currentPhrase {
			currentPhrase = bar / barsPerPhrase
						phrases = append(phrases, []schema.NoteEvent{})
		}
		phrases[currentPhrase] = append(phrases[currentPhrase], ev)
	}
	return phrases
}

func detectVariant(motif, phrase []schema.NoteEvent, base int) string {
	if len(phrase) < len(motif)/2 {
		return ""
	}
	// Check transposition.
	motifInterval := motif[1].Pitch - motif[0].Pitch
	phraseInterval := phrase[1].Pitch - phrase[0].Pitch

	if phraseInterval == motifInterval {
		return "transpose"
	}
	if phraseInterval == -motifInterval {
		return "invert"
	}
	// Check retrograde.
	if len(phrase) >= len(motif) && phraseInterval == motif[len(motif)-1].Pitch-motif[len(motif)-2].Pitch {
		return "retrograde"
	}
	return "rhythm_shift"
}

func extractRelative(events []schema.NoteEvent, base int) []int {
	rel := make([]int, len(events))
	for i, ev := range events {
		rel[i] = ev.Pitch - base
	}
	return rel
}

// ─── Rhythm Extraction ─────────────────────────────────────────────

func (e *Extractor) extractRhythm(eventsByTrack map[string][]schema.NoteEvent, totalBars int) RhythmDNA {
	rd := RhythmDNA{
		DensityBySection: make(map[string]float64),
	}

	drumEvents := eventsByTrack["drums"]
	if len(drumEvents) == 0 {
		return rd
	}

	// Count drum hits per section.
	sectionSize := totalBars / 4
	if sectionSize < 1 {
		sectionSize = 1
	}
	sectionNames := []string{"intro", "verse", "chorus", "outro"}

	for i, name := range sectionNames {
		startBar := i * sectionSize
		endBar := startBar + sectionSize
		if endBar > totalBars {
			endBar = totalBars
		}

		count := 0
		for _, ev := range drumEvents {
			bar := int(ev.StartBeat) / 4
			if bar >= startBar && bar < endBar {
				count++
			}
		}
		perBar := float64(count) / float64(endBar-startBar)
		rd.DensityBySection[name] = perBar / 16.0 // normalize to 0-1
	}

	return rd
}

// ─── Texture Extraction ────────────────────────────────────────────

func (e *Extractor) extractTexture(eventsByTrack map[string][]schema.NoteEvent, totalBars int) TextureDNA {
	td := TextureDNA{
		InstrumentTimeline: make(map[string][]int),
	}

	for trackID, events := range eventsByTrack {
		if len(events) == 0 {
			continue
		}
		activeBars := make(map[int]bool)
		for _, ev := range events {
			bar := int(ev.StartBeat) / 4
			if bar >= 0 && bar < totalBars {
				activeBars[bar] = true
			}
		}
		bars := make([]int, 0, len(activeBars))
		for b := range activeBars {
			bars = append(bars, b)
		}
		sort.Ints(bars)
		td.InstrumentTimeline[trackID] = bars
	}

	// Detect layering events (instruments entering/exiting).
	for trackID, bars := range td.InstrumentTimeline {
		if len(bars) == 0 {
			continue
		}
		// Find gaps.
		for i := 0; i < len(bars)-1; i++ {
			if bars[i+1]-bars[i] > 2 {
				td.Layering = append(td.Layering, LayerEvent{
					Bar:        bars[i+1],
					Action:     "add",
					Instrument: trackID,
				})
			}
		}
	}

	return td
}

// Print returns a human-readable summary of the MusicDNA.
func (dna *MusicDNA) Print() string {
	var b strings.Builder
	b.WriteString("===== MusicDNA =====\n")

	b.WriteString("--- Structure ---\n")
	for _, s := range dna.Structure.Sections {
		b.WriteString(fmt.Sprintf("  %s: bars %d-%d, energy=%.2f, density=%.2f, insts=%v\n",
			s.Name, s.StartBar, s.StartBar+s.Bars-1, s.Energy, s.Density, s.Instruments))
	}

	b.WriteString("--- Harmony ---\n")
	b.WriteString(fmt.Sprintf("  Key: %s\n", dna.Harmony.Key))
	for _, c := range dna.Harmony.Progression {
		if len(c.Function) > 0 {
			b.WriteString(fmt.Sprintf("  bar %d: %s (%s)\n", c.Bar, c.Chord, c.Function))
		} else {
			b.WriteString(fmt.Sprintf("  bar %d: %s\n", c.Bar, c.Chord))
		}
	}

	b.WriteString("--- Motif ---\n")
	b.WriteString(fmt.Sprintf("  Notes (relative): %v\n", dna.Motif.Notes))
	b.WriteString(fmt.Sprintf("  Rhythm: %v\n", dna.Motif.Rhythm))
	for _, v := range dna.Motif.Variants {
		b.WriteString(fmt.Sprintf("  Variant: %s → %v\n", v.Type, v.Notes))
	}

	_ = dna.Rhythm
	_ = dna.Texture

	return b.String()
}
