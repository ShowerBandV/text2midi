package composer

import (
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// HookMemory stores a hook phrase and replays it in intro and outro for memorability.
// The hook appears identically at bars 0-3 (intro) and bars N-4 to N-1 (outro),
// creating a "remember this" feeling.
type HookMemory struct {
	pitches []int
	rhythm  []float64
}

// NewHookMemory creates a hook from a scale and stores it for later recall.
func NewHookMemory(scale []int, rng *rand.Rand) *HookMemory {
	root := scale[0]
	third := scale[2]
	fifth := scale[4]
	// Memorable shapes: rising then falling.
	shapes := [][]int{
		{root, third, fifth, third},           // 1-3-5-3
		{root, scale[1], third, root + 12},    // 1-2-3-8
		{root, fifth, third, root},            // 1-5-3-1
		{fifth, third, root, root - 7},        // 5-3-1-5(low)
	}
	shape := shapes[rng.Intn(len(shapes))]
	return &HookMemory{
		pitches: shape,
		rhythm:  []float64{0.5, 0.5, 0.5, 0.5},
	}
}

// Render plays the hook at the given bar with optional octave shift.
func (h *HookMemory) Render(bar int, octave int) []schema.NoteEvent {
	if h == nil {
		return nil
	}
	base := float64(bar) * 4.0
	var events []schema.NoteEvent
	var t float64
	for i, p := range h.pitches {
		pitch := p + 12*octave
		if pitch < 48 { pitch += 12 }
		if pitch > 96 { pitch -= 12 }
		dur := h.rhythm[i%len(h.rhythm)] * 0.8
		if dur < 0.06 { dur = 0.06 }
		events = append(events, schema.NoteEvent{
			Type: "note", Pitch: pitch,
			StartBeat: base + t, DurationBeat: dur,
			Velocity: 100,
		})
		t += h.rhythm[i%len(h.rhythm)]
	}
	return events
}

// GenerateLeadWithHook creates a melody with an exact hook that repeats in intro and outro.
func GenerateLeadWithHook(scale []int, chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent
	hook := NewHookMemory(scale, rng)

	for bar := 0; bar < totalBars; bar++ {
		sec := songSection(bar, totalBars)

		switch sec {
		case "intro":
			// Play hook in low octave.
			if bar%2 == 0 {
				events = append(events, hook.Render(bar, 4)...)
			}
		case "verse":
			// Sparse: just the first note of the hook, every 4 bars.
			if bar%4 == 0 && hook != nil {
				p := hook.pitches[0] + 12*4
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat: float64(bar)*4 + 1.0, DurationBeat: 1.5,
					Velocity: 100,
				})
			}
		case "chorus", "chorus2":
			// Full hook, high octave, fast rhythm.
			if bar%2 == 0 {
				events = append(events, hook.Render(bar, 5)...)
			}
		case "bridge":
			// No hook — rest for other instruments.
		case "outro":
			// Hook again, low octave, fading — identical to intro.
			if bar%2 == 0 {
				events = append(events, hook.Render(bar, 4)...)
			}
		}
	}

	return events
}
