package composer

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// bassJazzWalking creates a walking bass line: quarter notes, chord tones on 1&3, passing tones on 2&4.
func bassJazzWalking(chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent

	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		base := float64(bar) * 4.0
		isMinor := strings.Contains(chord, "m")

		// Build chord tones and approach notes.
		third := root + 4
		fifth := root + 7
		if isMinor {
			third = root + 3
		}

		// Walking pattern: quarter notes.
		// Beat 1 = root. Beat 2 = approach (chromatic or scale). Beat 3 = fifth. Beat 4 = approach.
		approaches := []int{root - 1, root + 1, third - 1, third + 1, fifth - 1, fifth + 1}
		pattern := []int{
			root,                            // beat 1: root (strong)
			approaches[rng.Intn(6)],         // beat 2: chromatic approach
			fifth,                           // beat 3: fifth
			approaches[rng.Intn(6)],         // beat 4: leading to next bar
		}

		// Every other bar: walk up instead.
		if bar%2 == 1 {
			pattern = []int{root, third, fifth, root + 12}
		}

		for i, p := range pattern {
			pitch := p
			if pitch < 28 { pitch += 12 }
			if pitch > 60 { pitch -= 12 }
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat: base + float64(i)*1.0, DurationBeat: 0.85,
				Velocity: 100,
			})
		}
	}

	fmt.Printf("[Bass-Jazz] %d events, %d bars (walking)\n", len(events), totalBars)
	return events
}

// bassFunkSlap creates a slap bass line: syncopated, ghost notes, octave pops.
func bassFunkSlap(chords []string, totalBars int) []schema.NoteEvent {
	var events []schema.NoteEvent

	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		base := float64(bar) * 4.0

		// Funk pattern: root on 1 (slap), pop octave on 1&, ghost on 2&, root on 3, pop on 3&.
		type funkNote struct {
			beat float64
			pitch int
			dur   float64
			vel   int
		}

		pattern := []funkNote{
			{0.0, root, 0.15, 110},        // slap root
			{0.5, root + 12, 0.1, 105},    // pop octave
			{1.25, root, 0.05, 60},        // ghost
			{1.5, root - 5, 0.1, 100},     // slide
			{2.0, root, 0.15, 110},        // slap root
			{2.5, root + 12, 0.1, 105},    // pop octave
			{3.0, root + 7, 0.12, 100},    // fifth
			{3.5, root + 12, 0.1, 105},    // pop octave
		}

		// Variation every 4 bars.
		if bar%4 == 2 {
			pattern[4].beat = 2.25
			pattern[4].pitch = root
			pattern[4].vel = 65 // ghost
		}

		for _, n := range pattern {
			pitch := n.pitch
			if pitch < 28 { pitch += 12 }
			if pitch > 72 { pitch -= 12 }
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat: base + n.beat, DurationBeat: n.dur,
				Velocity: n.vel,
			})
		}
	}

	fmt.Printf("[Bass-Funk] %d events, %d bars (slap)\n", len(events), totalBars)
	return events
}

// bassRockBlues creates a rock bass line with blues inflections.
func bassRockBlues(chords []string, totalBars int) []schema.NoteEvent {
	var events []schema.NoteEvent

	for bar := 0; bar < totalBars; bar++ {
		chord := chords[bar%len(chords)]
		root := chordRootMIDI(chord, 2)
		base := float64(bar) * 4.0

		// Rock pattern: root on 1, fifth on 2&, root octave on 3, b7 walk on 4.
		pattern := []struct {
			beat float64
			pitch int
			dur   float64
		}{
			{0.0, root, 0.8},         // beat 1: root
			{1.5, root + 7, 0.4},     // 2&: fifth
			{2.0, root + 12, 0.8},    // beat 3: octave
			{3.5, root + 10, 0.3},    // 4&: b7 (blues flavor)
		}

		if bar%2 == 0 {
			// Even bars: walking down.
			pattern[3].pitch = root - 2
		}

		for _, n := range pattern {
			pitch := n.pitch
			if pitch < 28 { pitch += 12 }
			if pitch > 60 { pitch -= 12 }
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat: base + n.beat, DurationBeat: n.dur,
				Velocity: 100,
			})
		}
	}

	fmt.Printf("[Bass-Rock] %d events, %d bars (blues)\n", len(events), totalBars)
	return events
}
