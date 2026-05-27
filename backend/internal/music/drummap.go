package music

// DrumMap maps drum names to MIDI pitches (General MIDI standard).
// Ported from music_agent/core/drum_map.py.
//
// In General MIDI, channel 10 (index 9) is always drums.
// These are the pitch numbers on that channel.
var DrumMap = map[string]int{
	"kick":       36,
	"snare":      38,
	"closed_hat": 42,
	"open_hat":   46,
	"crash":      49,
	"ride":       51,
	"low_tom":    45,
	"mid_tom":    47,
	"high_tom":   50,
}
