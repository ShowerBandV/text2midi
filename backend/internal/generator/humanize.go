// Package generator --Instrument-specific humanization.
// Applies different timing/velocity offsets per instrument family
// to make MIDI sound more like real musicians playing.
package generator

import (
	"math/rand"

	"github.com/yourname/text2midi/internal/schema"
)

// InstrumentFamily categorizes a track for humanization.
type InstrumentFamily int

const (
	FamilyPiano   InstrumentFamily = iota // 0: piano, keys
	FamilyGuitar                          // 1: guitar (acoustic/electric)
	FamilyBrass                           // 2: brass, winds
	FamilyStrings                         // 3: strings, pads
	FamilyDrums                           // 4: drums, percussion
	FamilyBass                            // 5: bass
	FamilyLead                            // 6: lead synth / melody
	FamilyDefault                         // 7: other
)

// DetectFamily returns the instrument family for a track based on its ID and role.
func DetectFamily(trackID, role string) InstrumentFamily {
	switch trackID {
	case "drums", "percussion", "taiko", "taiko_drums", "driving_percussion":
		return FamilyDrums
	case "bass", "bassline", "synth_bass":
		return FamilyBass
	case "piano", "keys", "keyboard", "epiano":
		return FamilyPiano
	case "rhythm_guitar", "lead_guitar", "guitar", "acoustic_guitar":
		return FamilyGuitar
	case "strings", "string_ensemble", "celli", "violin", "viola", "orchestral_strings":
		return FamilyStrings
	case "brass", "heroic_brass", "horn", "trumpet", "trombone", "orchestral_hits":
		return FamilyBrass
	case "lead", "synth_lead", "vocal", "choir":
		return FamilyLead
	case "chords", "pad", "synth_pad", "warm_pad":
		return FamilyStrings // strings family for pads
	}
	// Fallback: detect by role keywords.
	switch role {
	case "piano", "keys":
		return FamilyPiano
	case "guitar":
		return FamilyGuitar
	case "brass":
		return FamilyBrass
	case "strings", "pad":
		return FamilyStrings
	case "bass":
		return FamilyBass
	case "melody", "lead":
		return FamilyLead
	}
	return FamilyDefault
}

// HumanizeTiming applies instrument-specific timing offsets to events.
// LoFi level controls overall looseness (0.0 = tight, 1.0 = very loose).
func HumanizeTiming(events []schema.NoteEvent, family InstrumentFamily, lofi float64) []schema.NoteEvent {
	if len(events) == 0 {
		return events
	}

	// Base jitter range in beats (at 120 BPM, 1 beat = 500ms).
	// We use beats so it scales with tempo automatically.
	var (
		noteOnJitter  float64 // timing offset for note start
		noteOffJitter float64 // timing offset for note duration
	)

	switch family {
	case FamilyPiano:
		// Piano: subtle rubato, slight delay for thick chords.
		noteOnJitter = 0.005 + lofi*0.015  // 2.5-10ms at 120BPM
		noteOffJitter = 0.01 + lofi*0.02

	case FamilyGuitar:
		// Guitar: slightly rushed upstrokes, loose timing.
		noteOnJitter = 0.01 + lofi*0.02   // 5-15ms
		noteOffJitter = 0.005 + lofi*0.015

	case FamilyBrass:
		// Brass: slight attack delay, breath feel.
		noteOnJitter = 0.02 + lofi*0.025  // 10-22.5ms delay
		noteOffJitter = 0.01 + lofi*0.015

	case FamilyStrings:
		// Strings: smooth, connected --more duration jitter.
		noteOnJitter = 0.008 + lofi*0.018
		noteOffJitter = 0.025 + lofi*0.035

	case FamilyDrums:
		// Drums: snare slightly behind the beat for backbeat feel.
		// This is handled differently --see below.
		noteOnJitter = 0.003 + lofi*0.01
		noteOffJitter = 0.003 + lofi*0.008

	case FamilyBass:
		// Bass: locked to kick drum --tight timing.
		noteOnJitter = 0.003 + lofi*0.01
		noteOffJitter = 0.005 + lofi*0.012

	case FamilyLead:
		// Lead: expressive, slight push/pull.
		noteOnJitter = 0.008 + lofi*0.02
		noteOffJitter = 0.008 + lofi*0.015

	default:
		noteOnJitter = 0.008 + lofi*0.015
		noteOffJitter = 0.008 + lofi*0.015
	}

	for i := range events {
		e := &events[i]

		// Apply timing jitter to start.
		jitter := (rand.Float64() - 0.5) * 2 * noteOnJitter
		e.StartBeat += jitter
		if e.StartBeat < 0 {
			e.StartBeat = 0
		}

		// Apply duration jitter.
		durJitter := (rand.Float64() - 0.5) * 2 * noteOffJitter
		e.DurationBeat += durJitter
		if e.DurationBeat < 0.05 {
			e.DurationBeat = 0.05
		}
	}

	// Special: snare backbeat delay for drums.
	if family == FamilyDrums {
		for i := range events {
			e := &events[i]
			if e.DrumName == "snare" {
				// Snare slightly behind the beat (backbeat feel).
				if e.StartBeat-float64(int(e.StartBeat)) > 0.45 &&
					e.StartBeat-float64(int(e.StartBeat)) < 0.55 {
					e.StartBeat += 0.01 + lofi*0.015
				}
			}
		}
	}

	return events
}

// HumanizeVelocity applies instrument-specific velocity shaping.
func HumanizeVelocity(events []schema.NoteEvent, family InstrumentFamily, energy float64) []schema.NoteEvent {
	if len(events) == 0 {
		return events
	}

	var (
		variation int    // max velocity variation
		accentLvl int    // extra velocity for accents
	)

	switch family {
	case FamilyPiano:
		variation = 12
		accentLvl = 8
	case FamilyGuitar:
		variation = 15
		accentLvl = 12
	case FamilyBrass:
		variation = 10
		accentLvl = 15
	case FamilyStrings:
		variation = 8
		accentLvl = 6
	case FamilyDrums:
		variation = 10
		accentLvl = 10
	case FamilyBass:
		variation = 8
		accentLvl = 5
	case FamilyLead:
		variation = 14
		accentLvl = 10
	default:
		variation = 10
		accentLvl = 8
	}

	for i := range events {
		e := &events[i]
		// Add natural velocity variation.
		delta := rand.Intn(variation*2+1) - variation
		e.Velocity += delta

		// Downbeat accent (beat 0 or in first quarter of bar).
		beatInBar := e.StartBeat - float64(int(e.StartBeat/4)*4)
		if beatInBar < 0.25 {
			e.Velocity += accentLvl
		}

		if e.Velocity < 1 {
			e.Velocity = 1
		}
		if e.Velocity > 127 {
			e.Velocity = 127
		}
	}

	return events
}
