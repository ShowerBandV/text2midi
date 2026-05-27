// Package composer — Motif memory and thematic development.
// Extracts the initial melodic motif and applies variations to later phrases,
// creating thematic coherence throughout the piece.
package composer

import (
	"fmt"
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// MotifExtractor analyzes a lead melody to find and develop thematic material.
type MotifExtractor struct {
	BarsPerPhrase int
	MinMotifNotes int
}

// NewMotifExtractor creates a motif extractor for 4-bar phrases.
func NewMotifExtractor() *MotifExtractor {
	return &MotifExtractor{
		BarsPerPhrase: 4,
		MinMotifNotes: 4,
	}
}

// PhraseInfo holds the notes belonging to one phrase.
type PhraseInfo struct {
	StartBar int
	EndBar   int
	Notes    []schema.NoteEvent
}

// splitPhrases divides the melody into equal-length phrases.
func (m *MotifExtractor) splitPhrases(events []schema.NoteEvent, totalBars int) []PhraseInfo {
	if len(events) == 0 {
		return nil
	}

	numPhrases := (totalBars + m.BarsPerPhrase - 1) / m.BarsPerPhrase
	phrases := make([]PhraseInfo, numPhrases)

	for i := range phrases {
		phrases[i] = PhraseInfo{
			StartBar: i * m.BarsPerPhrase,
			EndBar:   (i + 1) * m.BarsPerPhrase,
		}
	}

	// Assign notes to phrases.
	for _, ev := range events {
		bar := int(ev.StartBeat) / 4
		idx := bar / m.BarsPerPhrase
		if idx >= numPhrases {
			idx = numPhrases - 1
		}
		phrases[idx].Notes = append(phrases[idx].Notes, ev)
	}

	return phrases
}

// extractMotif extracts the core motif from the first phrase.
// The motif is the first MinMotifNotes notes that define the melodic shape.
func (m *MotifExtractor) extractMotif(phrase PhraseInfo) []schema.NoteEvent {
	if len(phrase.Notes) < m.MinMotifNotes {
		return phrase.Notes
	}
	return phrase.Notes[:m.MinMotifNotes]
}

// VariationType describes how to transform a motif.
type VariationType int

const (
	VarOriginal     VariationType = iota // 0: no change
	VarTransposeUp                       // 1: transpose up a fifth
	VarTransposeDown                     // 2: transpose down a fourth
	VarInvert                            // 3: invert intervals
	VarRhythmicDbl                       // 4: double durations (slower)
	VarFragment                          // 5: use first half of motif
	VarSequence                          // 6: sequence (repeat at different pitch)
)

// ApplyMotifDevelopment interleaves motif variations with original phrases.
// First phrase stays as LLM composed (the "theme").
// EVERY OTHER phrase gets a motif variation; the rest keep the LLM's original.
// This preserves the composer's intent while adding thematic coherence.
func (m *MotifExtractor) ApplyMotifDevelopment(events []schema.NoteEvent, totalBars int) []schema.NoteEvent {
	if len(events) < m.MinMotifNotes*2 {
		return events // Not enough notes for development
	}

	phrases := m.splitPhrases(events, totalBars)
	if len(phrases) < 2 {
		return events // Only one phrase, nothing to develop
	}

	// Extract motif from first phrase.
	motif := m.extractMotif(phrases[0])
	if len(motif) < m.MinMotifNotes {
		return events
	}

	fmt.Printf("[Motif] extracted %d-note motif from phrase 0\n", len(motif))

	// Interleave: every OTHER phrase gets a motif variation.
	// Phrases 1, 3, 5, ... stay as LLM composed (the "original").
	// Phrases 2, 4, 6, ... get a motif variation (the "development").
	variations := []VariationType{
		VarTransposeUp,
		VarTransposeDown,
		VarInvert,
		VarRhythmicDbl,
		VarFragment,
		VarSequence,
	}
	varIdx := 0

	for i := 1; i < len(phrases); i++ {
		if len(phrases[i].Notes) == 0 {
			continue
		}

		if i%2 == 1 {
			// Odd phrase (1, 3, 5...) — keep LLM original.
			fmt.Printf("[Motif] phrase %d: keep original (%d notes)\n",
				i, len(phrases[i].Notes))
			continue
		}

		// Even phrase (2, 4, 6...) — replace with motif variation.
		varied := m.applyVariation(motif, variations[varIdx%len(variations)], phrases[i].StartBar)
		varIdx++

		// Scale to match the target phrase's approximate energy.
		targetCount := len(phrases[i].Notes)
		if targetCount > 0 && len(varied) > targetCount*2 {
			varied = varied[:targetCount]
		}

		phrases[i].Notes = varied

		fmt.Printf("[Motif] phrase %d: %s variation (%d notes)\n",
			i, variationName(variations[(varIdx-1)%len(variations)]), len(varied))
	}

	// Reassemble.
	var result []schema.NoteEvent
	for _, p := range phrases {
		result = append(result, p.Notes...)
	}

	return result
}

// applyVariation creates a variation of the motif.
func (m *MotifExtractor) applyVariation(motif []schema.NoteEvent, varType VariationType, startBar int) []schema.NoteEvent {
	barStartBeat := float64(startBar) * 4.0

	switch varType {
	case VarOriginal:
		// Copy as-is.
		notes := make([]schema.NoteEvent, len(motif))
		copy(notes, motif)
		for i := range notes {
			notes[i].StartBeat += barStartBeat
		}
		return notes

	case VarTransposeUp:
		// Transpose up a perfect fifth (7 semitones).
		notes := make([]schema.NoteEvent, len(motif))
		for i, n := range motif {
			notes[i] = n
			notes[i].Pitch += 7
			if notes[i].Pitch > 108 {
				notes[i].Pitch -= 12
			}
			notes[i].StartBeat = n.StartBeat - float64(int(n.StartBeat/4)*4) + barStartBeat
		}
		return notes

	case VarTransposeDown:
		// Transpose down a perfect fourth (5 semitones).
		notes := make([]schema.NoteEvent, len(motif))
		for i, n := range motif {
			notes[i] = n
			notes[i].Pitch -= 5
			if notes[i].Pitch < 21 {
				notes[i].Pitch += 12
			}
			notes[i].StartBeat = n.StartBeat - float64(int(n.StartBeat/4)*4) + barStartBeat
		}
		return notes

	case VarInvert:
		// Invert intervals around the first note.
		if len(motif) == 0 {
			return nil
		}
		basePitch := motif[0].Pitch
		notes := make([]schema.NoteEvent, len(motif))
		for i, n := range motif {
			notes[i] = n
			if i > 0 {
				interval := n.Pitch - motif[i-1].Pitch
				notes[i].Pitch = notes[i-1].Pitch - interval
				if notes[i].Pitch < 21 {
					notes[i].Pitch += 12
				}
				if notes[i].Pitch > 108 {
					notes[i].Pitch -= 12
				}
			} else {
				notes[i].Pitch = basePitch
			}
			notes[i].StartBeat = n.StartBeat - float64(int(n.StartBeat/4)*4) + barStartBeat
		}
		return notes

	case VarRhythmicDbl:
		// Double all durations (stretch by 2x).
		notes := make([]schema.NoteEvent, 0, len(motif)*2)
		for _, n := range motif {
			beatInBar := n.StartBeat - float64(int(n.StartBeat/4)*4)
			notes = append(notes, schema.NoteEvent{
				Type: n.Type, Pitch: n.Pitch,
				StartBeat:    barStartBeat + beatInBar*2,
				DurationBeat: n.DurationBeat * 2,
				Velocity:     n.Velocity,
			})
			// Add a rest by skipping the next slot.
		}
		return notes

	case VarFragment:
		// Use only the first half of the motif, repeat it.
		half := len(motif) / 2
		if half < 2 {
			half = 2
		}
		frag := motif[:half]
		notes := make([]schema.NoteEvent, 0, half*2)
		fragDur := frag[len(frag)-1].StartBeat + frag[len(frag)-1].DurationBeat - frag[0].StartBeat
		for rep := 0; rep < 2; rep++ {
			for _, n := range frag {
				beatInBar := n.StartBeat - float64(int(n.StartBeat/4)*4)
				notes = append(notes, schema.NoteEvent{
					Type: n.Type, Pitch: n.Pitch,
					StartBeat:    barStartBeat + beatInBar + float64(rep)*fragDur,
					DurationBeat: n.DurationBeat,
					Velocity:     n.Velocity - 5 + rand.Intn(10),
				})
			}
		}
		return notes

	case VarSequence:
		// Repeat motif at ascending pitch levels (sequence).
		notes := make([]schema.NoteEvent, 0, len(motif)*3)
		seqLen := len(motif)
		for rep := 0; rep < 3; rep++ {
			transpose := rep * 3 // up 3 semitones each repetition
			for _, n := range motif {
				beatInBar := n.StartBeat - float64(int(n.StartBeat/4)*4)
				p := n.Pitch + transpose
				if p > 108 {
					p -= 12
				}
				notes = append(notes, schema.NoteEvent{
					Type: n.Type, Pitch: p,
					StartBeat:    barStartBeat + beatInBar + float64(rep)*float64(seqLen)*0.5,
					DurationBeat: n.DurationBeat * 0.75,
					Velocity:     n.Velocity,
				})
			}
		}
		return notes
	}

	return motif
}

// GenerateCounterMelody creates a secondary melody that harmonizes with the lead.
// Uses mostly 3rds and 6ths below the lead melody, with occasional unison/octave.
func GenerateCounterMelody(leadNotes []schema.NoteEvent, totalBars int) []schema.NoteEvent {
	if len(leadNotes) < 4 {
		return nil
	}

	// Create a sparser, lower counter-melody.
	// Pick every 2nd or 3rd note from lead and harmonize at a 3rd or 6th below.
	var counter []schema.NoteEvent
	noteCount := 0

	for i, n := range leadNotes {
		// Only harmonize ~40% of lead notes (sparser than lead).
		if i%3 != 0 {
			continue
		}

		// Pick interval: prefer 3rd or 6th below.
		interval := -3 // minor third below
		if (i/3)%2 == 0 {
			interval = -4 // major third below
		}
		if (i/3)%3 == 0 {
			interval = -8 // minor sixth below
		}

		counterPitch := n.Pitch + interval
		if counterPitch < 21 {
			counterPitch += 12
		}
		if counterPitch > 84 {
			counterPitch -= 12
		}

		// Slightly behind the lead for "echo" effect.
		delay := 0.0
		if noteCount > 0 && noteCount%2 == 0 {
			delay = 0.08 // slight delay for call-response feel
		}

		counter = append(counter, schema.NoteEvent{
			Type: n.Type, Pitch: counterPitch,
			StartBeat:    n.StartBeat + delay,
			DurationBeat: n.DurationBeat * 1.2, // slightly longer
			Velocity:     n.Velocity - 15,       // quieter than lead
		})
		noteCount++
	}

	fmt.Printf("[CounterMelody] generated %d notes (harmonizing lead %d)\n", len(counter), len(leadNotes))
	return counter
}

// ApplyCallResponse splits the lead melody into call and response phrases.
// Even bars = call (lead), odd bars = response (counter-melody or bass fill).
// This creates the classic "question and answer" musical对话.
func ApplyCallResponse(leadEvents []schema.NoteEvent) []schema.NoteEvent {
	if len(leadEvents) < 8 {
		return leadEvents
	}

	// Group notes by bar.
	barNotes := make(map[int][]schema.NoteEvent)
	for _, ev := range leadEvents {
		bar := int(ev.StartBeat) / 4
		barNotes[bar] = append(barNotes[bar], ev)
	}

	var result []schema.NoteEvent
	for bar := 0; ; bar++ {
		notes, ok := barNotes[bar]
		if !ok {
			break
		}

		if bar%2 == 0 {
			// Even bar = CALL (keep original energy).
			for _, n := range notes {
				n.Velocity = int(float64(n.Velocity) * 1.1)
				if n.Velocity > 127 {
					n.Velocity = 127
				}
				result = append(result, n)
			}
		} else {
			// Odd bar = RESPONSE (quieter, shorter, answering feel).
			for _, n := range notes {
				n.Velocity = int(float64(n.Velocity) * 0.75)
				if n.Velocity < 20 {
					n.Velocity = 20
				}
				n.DurationBeat *= 0.8
				result = append(result, n)
			}
		}
	}

	fmt.Printf("[CallResponse] applied to %d bars\n", len(barNotes))
	return result
}

// ApplyRegisterExpansion gradually increases the melody's pitch range across sections.
// Verse = narrow range (5th), Pre-chorus = medium (octave), Chorus = full range (10th+).
func ApplyRegisterExpansion(events []schema.NoteEvent, totalBars int) []schema.NoteEvent {
	if len(events) < 4 {
		return events
	}

	barsPerPhrase := 4
	numPhrases := (totalBars + barsPerPhrase - 1) / barsPerPhrase

	for i := range events {
		bar := int(events[i].StartBeat) / 4
		phraseIdx := bar / barsPerPhrase
		if phraseIdx >= numPhrases {
			phraseIdx = numPhrases - 1
		}

		// Expand register: later phrases play higher.
		// Phrase 0: keep original. Phrase 1: +2 semi. Phrase 2: +5 semi. Phrase 3: +7 semi.
		registerShift := 0
		switch phraseIdx {
		case 0:
			registerShift = 0
		case 1:
			registerShift = 2 // up a major 2nd
		case 2:
			registerShift = 5 // up a 4th
		default:
			registerShift = 7 + phraseIdx - 3 // up a 5th or more
		}

		if registerShift > 12 {
			registerShift = 12 // cap at octave
		}

		newPitch := events[i].Pitch + registerShift
		if newPitch > 100 {
			newPitch = events[i].Pitch // don't go too high
		}
		events[i].Pitch = newPitch
	}

	fmt.Printf("[RegisterExpansion] applied across %d phrases\n", numPhrases)
	return events
}

// ApplySyncopation randomly shifts ~20% of melody notes by an 8th note offset.
// This creates rhythmic interest without changing the melodic contour.
func ApplySyncopation(events []schema.NoteEvent) []schema.NoteEvent {
	if len(events) < 4 {
		return events
	}

	shifted := 0
	for i := range events {
		// Skip the first note (it sets the groove).
		if i == 0 {
			continue
		}
		// ~20% chance of syncopation.
		beatInBar := events[i].StartBeat - float64(int(events[i].StartBeat/4))*4
		// Only shift if the note isn't already on an offbeat.
		onBeat := beatInBar - float64(int(beatInBar))
		if onBeat < 0.05 || onBeat > 0.45 && onBeat < 0.55 {
			if rand.Float64() < 0.2 {
				// Shift forward by half a beat (eighth note).
				events[i].StartBeat += 0.5
				shifted++
			}
		}
	}

	if shifted > 0 {
		fmt.Printf("[Syncopation] shifted %d/%d notes\n", shifted, len(events))
	}
	return events
}

// ApplyAnacrusis shifts the start of the melody by a half beat with probability p.
// This creates pickup notes (anacrusis) that sound more natural than always starting on beat 1.
func ApplyAnacrusis(events []schema.NoteEvent, totalBars int) []schema.NoteEvent {
	if len(events) < 4 {
		return events
	}

	// Shift the entire melody forward by 0.5 beats (eighth note pickup).
	shift := 0.0
	// Use position of first note to decide.
	firstBarStart := float64(int(events[0].StartBeat/4)) * 4
	if events[0].StartBeat-firstBarStart < 0.1 {
		// Starts on beat 1 — shift to create pickup.
		shift = 0.5
	}

	if shift > 0 {
		for i := range events {
			events[i].StartBeat += shift
		}
		fmt.Printf("[Anacrusis] shifted melody +%.1f beat (pickup feel)\n", shift)
	}
	return events
}

func variationName(v VariationType) string {
	switch v {
	case VarOriginal:
		return "original"
	case VarTransposeUp:
		return "transpose+5th"
	case VarTransposeDown:
		return "transpose-4th"
	case VarInvert:
		return "invert"
	case VarRhythmicDbl:
		return "rhythm_double"
	case VarFragment:
		return "fragment"
	case VarSequence:
		return "sequence"
	default:
		return "unknown"
	}
}
