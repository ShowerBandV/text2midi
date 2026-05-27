package composer

import "math/rand"

// BuildSection generates phrases for a given section type, style-aware.
func BuildSection(motif []int, name string, bars int, plan MotifPlan, rng *rand.Rand, style string) []Phrase {
	numPhrases := bars / plan.BarsPerPhrase
	if numPhrases < 1 { numPhrases = 1 }
	phrases := make([]Phrase, numPhrases)

	for i := 0; i < numPhrases; i++ {
		var phrase Phrase
		switch style {
		case "metal":
			// Metal: REPETITIVE, minimal variation. The same riff over and over.
			phrase = metalPhrase(motif, rng)
		case "hiphop":
			// Hip-hop: LOOP-BASED, slight variations each loop.
			phrase = hiphopPhrase(motif, i, rng)
		case "pop":
			// Pop: clear A-A'-B-A structure, singable melody.
			phrase = popPhrase(motif, rng)
		default:
			// Ambient / default: sparse, slow evolution.
			phrase = ambientPhrase(motif, i, rng)
		}

		// Section-specific transformations
		switch name {
		case "intro":
			for b := range phrase.Bars {
				for j := range phrase.Bars[b] {
					phrase.Bars[b][j] = phrase.Bars[b][j]/2 - 12
				}
				if len(phrase.Bars[b]) > 3 { phrase.Bars[b] = phrase.Bars[b][:3] }
			}
		case "chorus":
			for b := range phrase.Bars {
				for j := range phrase.Bars[b] {
					phrase.Bars[b][j] += 12
				}
			}
		case "bridge":
			for b := range phrase.Bars {
				phrase.Bars[b] = Invert(phrase.Bars[b])
			}
		}
		phrases[i] = phrase
	}
	return phrases
}

// metalPhrase: same riff, repeated with slight accent variation.
func metalPhrase(motif []int, rng *rand.Rand) Phrase {
	p := Phrase{}
	// Bar 0: riff
	p.Bars[0] = copySlice(motif)
	// Bar 1: same riff, octave drop
	p.Bars[1] = Transpose(motif, -12)
	// Bar 2: same riff
	p.Bars[2] = copySlice(motif)
	// Bar 3: riff with slight variation (fragment + extend)
	p.Bars[3] = Extend(Fragment(motif, len(motif)/2+1), 1)
	return p
}

// hiphopPhrase: loop-based, slight changes each repetition.
func hiphopPhrase(motif []int, loopIdx int, rng *rand.Rand) Phrase {
	p := Phrase{}
	base := motif
	if len(motif) > 4 { base = motif[:4] } // shorter loops

	// All 4 bars use same loop
	p.Bars[0] = copySlice(base)
	p.Bars[1] = copySlice(base)
	p.Bars[2] = copySlice(base)
	p.Bars[3] = copySlice(base)

	// Each loop iteration adds slight variation on the last bar
	if loopIdx > 0 {
		p.Bars[3] = Transpose(base, 3)
	}
	return p
}

// popPhrase: clear A-A'-B-A structure, singable.
func popPhrase(motif []int, rng *rand.Rand) Phrase {
	p := Phrase{}
	// Bar 0: A = motif
	p.Bars[0] = copySlice(motif)
	// Bar 1: A' = slight variation (transpose up 2-3 semitones)
	p.Bars[1] = Transpose(motif, 3)
	// Bar 2: B = contrast (invert or fragment)
	p.Bars[2] = Fragment(Invert(motif), len(motif)-1)
	// Bar 3: A = return to motif
	p.Bars[3] = copySlice(motif)
	return p
}

// ambientPhrase: sparse, evolving slowly.
func ambientPhrase(motif []int, phraseIdx int, rng *rand.Rand) Phrase {
	p := Phrase{}
	notes := Fragment(motif, 2) // just 2 notes per bar
	// Each phrase evolves
	shift := phraseIdx * 2
	notes = Transpose(notes, shift)
	p.Bars[0] = notes
	p.Bars[1] = Invert(notes)
	p.Bars[2] = append(notes, notes[0]+2)
	p.Bars[3] = notes
	return p
}
