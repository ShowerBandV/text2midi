// Package composer — Texture Layer Engine.
// Adds atmospheric layers (drone, pad, noise, arp, reverse) that fill
// the sonic background. Essential for game music immersion.
package composer

import (
	"math/rand"

	"github.com/yourname/text2midi/internal/schema"
)

// TextureType defines the kind of atmospheric layer.
type TextureType int

const (
	TXNone    TextureType = iota
	TXPad                 // long chord pad
	TXDrone               // sustained single note
	TXNoise               // noise sweep
	TXArp                 // rapid arpeggio
	TXReverse             // reversed piano swell
)

// GenerateTexture creates atmospheric texture events for a given section.
// The texture is added as a new track "atmosphere" in the events map.
func GenerateTexture(evMap map[string][]schema.NoteEvent, ttype TextureType, keyRoot string, barCount, bpm int, energy float64) {
	if ttype == TXNone {
		return
	}

	events := make([]schema.NoteEvent, 0)
	root := KeyToMIDI(keyRoot, 3) // C3-ish

	switch ttype {
	case TXPad:
		// Long sustained chords, very quiet.
		third := root + 4
		fifth := root + 7
		vel := 20 + int(energy*20)
		for bar := 0; bar < barCount; bar++ {
			base := float64(bar) * 4.0
			for _, p := range []int{root, third, fifth, root + 12} {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat:    base,
					DurationBeat: 4.0,
					Velocity:     vel + rand.Intn(5),
				})
			}
		}

	case TXDrone:
		// Single sustained note, very low.
		vel := 15 + int(energy*15)
		for bar := 0; bar < barCount; bar++ {
			base := float64(bar) * 4.0
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: root - 12,
				StartBeat:    base,
				DurationBeat: 4.0,
				Velocity:     vel,
			})
		}

	case TXNoise:
		// Rapid random high-pitched noise sparkles.
		vel := 30 + int(energy*30)
		for i := 0; i < barCount*4; i++ {
			beat := float64(rand.Intn(barCount*4)) * 1.0
			pitch := 72 + rand.Intn(24)
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat:    beat,
				DurationBeat: 0.05 + float64(rand.Intn(5))*0.01,
				Velocity:     vel,
			})
		}

	case TXArp:
		// Rapid ascending arpeggio, repeating.
		vel := 25 + int(energy*25)
		notes := []int{root, root + 7, root + 12, root + 7} // root-fifth-octave-fifth
		for bar := 0; bar < barCount; bar++ {
			base := float64(bar) * 4.0
			for step := 0; step < 8; step++ {
				p := notes[step%len(notes)]
				beat := float64(step) * 0.5
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: p,
					StartBeat:    base + beat,
					DurationBeat: 0.4,
					Velocity:     vel + rand.Intn(5),
				})
			}
		}

	case TXReverse:
		// Reverse swell: crescendo over the last beat of each phrase.
		vel := 40 + int(energy*30)
		for bar := 3; bar < barCount; bar += 4 {
			base := float64(bar) * 4.0
			for i := 0; i < 8; i++ {
				progress := float64(i) / 8.0
				beat := 3.0 + progress*1.0
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: root + 12 + int(progress*12),
					StartBeat:    base + beat,
					DurationBeat: 0.05,
					Velocity:     int(float64(vel) * progress),
				})
			}
		}
	}

	evMap["atmosphere"] = events
}

// SelectTexture chooses a texture type based on section energy and style.
func SelectTexture(energy float64) TextureType {
	switch {
	case energy < 0.2:
		return TXPad
	case energy < 0.5:
		return TXDrone
	case energy < 0.7:
		return TXArp
	default:
		return TXNoise
	}
}

// KeyToMIDI converts a key name to a MIDI pitch at a given octave.
func KeyToMIDI(key string, octave int) int {
	semi := map[string]int{
		"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3,
		"E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8,
		"Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11,
	}
	if s, ok := semi[key]; ok {
		return (octave+1)*12 + s
	}
	return 60
}
