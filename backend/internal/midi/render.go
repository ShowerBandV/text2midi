// Package midi implements a Standard MIDI File (SMF) Type 1 writer.
// Ported from music_agent/core/mido_renderer.py --but implements the binary
// MIDI format directly instead of relying on the mido library.
package midi

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// RenderResult contains metadata about the rendered MIDI file.
type RenderResult struct {
	OutputPath      string  `json:"output_path"`
	TicksPerBeat    int     `json:"ticks_per_beat"`
	TotalTracks     int     `json:"total_tracks"`
	TotalNoteEvents int     `json:"total_note_events"`
	DurationSeconds float64 `json:"duration_seconds"`
}

// RenderMIDI converts a MidiIR to a .mid file on disk.
// Ported from music_agent/core/mido_renderer.py::render_midi.
func RenderMIDI(mid schema.MidiIR, outputPath string, selectedTracks []string) (*RenderResult, error) {
	// Filter tracks.
	allTracks := filterEnabled(mid.Tracks)
	var tracks []schema.TrackIR
	if selectedTracks == nil {
		tracks = allTracks
	} else {
		selectedSet := make(map[string]bool, len(selectedTracks))
		for _, id := range selectedTracks {
			selectedSet[id] = true
		}
		trackMap := make(map[string]schema.TrackIR, len(allTracks))
		for _, t := range allTracks {
			trackMap[t.ID] = t
		}
		for _, id := range selectedTracks {
			if _, ok := trackMap[id]; !ok {
				return nil, fmt.Errorf("selected track %q not found or disabled", id)
			}
		}
		for _, t := range allTracks {
			if selectedSet[t.ID] {
				tracks = append(tracks, t)
			}
		}
	}

	if len(tracks) == 0 {
		return nil, fmt.Errorf("no tracks to render")
	}

	tpb := mid.Meta.TicksPerBeat
	if tpb <= 0 {
		tpb = 480
	}

	// We'll build the file in memory: header + track chunks.
	var fileBytes []byte

	// --- Header chunk ---
	nTracks := uint16(1 + len(tracks)) // meta track + instrument tracks
	header := buildHeader(uint16(1), nTracks, uint16(tpb))
	fileBytes = append(fileBytes, header...)

	// --- Meta track (tempo + time signature) ---
	bpm := mid.Meta.BPM
	if bpm <= 0 {
		bpm = 120
	}
	tempoUS := bpmToTempo(bpm) // microseconds per quarter note
	ts := mid.Meta.TimeSignature

	var metaTrackBytes []byte
	// Set Tempo meta event
	metaTrackBytes = append(metaTrackBytes, encodeVarLen(0)...) // delta=0
	metaTrackBytes = append(metaTrackBytes, 0xFF, 0x51, 0x03)
	metaTrackBytes = append(metaTrackBytes, byte(tempoUS>>16), byte(tempoUS>>8), byte(tempoUS))

	// Time Signature meta event
	metaTrackBytes = append(metaTrackBytes, encodeVarLen(0)...) // delta=0
	metaTrackBytes = append(metaTrackBytes, 0xFF, 0x58, 0x04)
	metaTrackBytes = append(metaTrackBytes, byte(ts.Numerator), byte(ts.Denominator),
		24, 8) // clocks_per_click=24, 32nd_notes_per_beat=8

	// End of Track
	metaTrackBytes = append(metaTrackBytes, encodeVarLen(0)...)
	metaTrackBytes = append(metaTrackBytes, 0xFF, 0x2F, 0x00)

	fileBytes = append(fileBytes, buildTrackChunk(metaTrackBytes)...)

	// --- Instrument tracks ---
	totalNoteEvents := 0
	for _, tr := range tracks {
		var trackBytes []byte

		// Track name meta event
		trackBytes = append(trackBytes, encodeVarLen(0)...)
		nameData := []byte(tr.Name)
		trackBytes = append(trackBytes, 0xFF, 0x03, byte(len(nameData)))
		trackBytes = append(trackBytes, nameData...)

		// Program change (if not drum channel 9)
		if tr.Program != nil && tr.Channel != 9 {
			trackBytes = append(trackBytes, encodeVarLen(0)...)
			trackBytes = append(trackBytes, 0xC0|byte(tr.Channel), byte(*tr.Program))
		}

		// Volume (CC7)
		trackBytes = append(trackBytes, encodeVarLen(0)...)
		trackBytes = append(trackBytes, 0xB0|byte(tr.Channel), 0x07, byte(clamp(tr.Volume, 0, 127)))

		// Pan (CC10)
		trackBytes = append(trackBytes, encodeVarLen(0)...)
		trackBytes = append(trackBytes, 0xB0|byte(tr.Channel), 0x0A, byte(clamp(tr.Pan, 0, 127)))

		// Note events: convert absolute beats ->ticks ->delta-time events.
		var absEvents []absEvent
		for _, ev := range tr.Events {
			totalNoteEvents++
			startTick := beatToTick(ev.StartBeat, tpb)
			durTick := beatToTick(ev.DurationBeat, tpb)
			if durTick < 1 {
				durTick = 1
			}
			pitch := clamp(ev.Pitch, 0, 127)
			vel := clamp(ev.Velocity, 1, 127)

			absEvents = append(absEvents, absEvent{
				tick: startTick, isNoteOn: true,
				pitch: byte(pitch), velocity: byte(vel), channel: byte(tr.Channel),
			})
			absEvents = append(absEvents, absEvent{
				tick: startTick + durTick, isNoteOn: false,
				pitch: byte(pitch), velocity: 0, channel: byte(tr.Channel),
			})
		}

		// Sort by tick; note-offs before note-ons at same tick.
		sortAbsEvents(absEvents)

		// Convert to delta-time MIDI events.
		lastTick := 0
		for _, ae := range absEvents {
			delta := ae.tick - lastTick
			lastTick = ae.tick

			trackBytes = append(trackBytes, encodeVarLen(delta)...)
			if ae.isNoteOn {
				trackBytes = append(trackBytes, 0x90|ae.channel, ae.pitch, ae.velocity)
			} else {
				trackBytes = append(trackBytes, 0x80|ae.channel, ae.pitch, ae.velocity)
			}
		}

		// Insert pitch bend events.
		for _, pb := range tr.PitchBendEvents {
			pbTick := beatToTick(pb.Bar*4.0, tpb)
			delta := pbTick - lastTick
			if delta < 0 {
				delta = 0
			}
			lastTick = pbTick
			// Pitch bend: 0xE0 | channel, LS 7 bits, MS 7 bits
			val := pb.Value
			if val < 0 {
				val = 0
			}
			if val > 16383 {
				val = 16383
			}
			lsb := byte(val & 0x7F)
			msb := byte((val >> 7) & 0x7F)
			trackBytes = append(trackBytes, encodeVarLen(delta)...)
			trackBytes = append(trackBytes, 0xE0|byte(tr.Channel), lsb, msb)
		}

		// Insert CC events (expression, sustain, etc.).
		for _, cc := range tr.CCEvents {
			ccTick := beatToTick(cc.Bar*4.0, tpb)
			delta := ccTick - lastTick
			if delta < 0 {
				delta = 0
			}
			lastTick = ccTick
			trackBytes = append(trackBytes, encodeVarLen(delta)...)
			trackBytes = append(trackBytes, 0xB0|byte(tr.Channel), byte(cc.Controller), byte(clamp(cc.Value, 0, 127)))
		}

		// End of Track
		trackBytes = append(trackBytes, encodeVarLen(0)...)
		trackBytes = append(trackBytes, 0xFF, 0x2F, 0x00)

		fileBytes = append(fileBytes, buildTrackChunk(trackBytes)...)
	}

	// Write to disk.
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(outputPath, fileBytes, 0644); err != nil {
		return nil, fmt.Errorf("write midi file: %w", err)
	}

	totalBeats := float64(mid.Meta.TotalBars * mid.Meta.BeatsPerBar)
	if totalBeats <= 0 {
		totalBeats = float64(mid.Meta.TotalBars * 4)
	}
	durationSec := totalBeats * (60.0 / float64(bpm))

	return &RenderResult{
		OutputPath:      outputPath,
		TicksPerBeat:    tpb,
		TotalTracks:     len(tracks),
		TotalNoteEvents: totalNoteEvents,
		DurationSeconds: durationSec,
	}, nil
}

// --- helpers ---

// buildHeader builds a MIDI file header chunk.
func buildHeader(format, nTracks, division uint16) []byte {
	hdr := make([]byte, 14)
	copy(hdr[0:4], []byte("MThd"))
	binary.BigEndian.PutUint32(hdr[4:8], 6)      // chunk length
	binary.BigEndian.PutUint16(hdr[8:10], format)  // 0=single, 1=multi
	binary.BigEndian.PutUint16(hdr[10:12], nTracks)
	binary.BigEndian.PutUint16(hdr[12:14], division)
	return hdr
}

// buildTrackChunk wraps event bytes in an MTrk chunk with length prefix.
func buildTrackChunk(data []byte) []byte {
	chunk := make([]byte, 8+len(data))
	copy(chunk[0:4], []byte("MTrk"))
	binary.BigEndian.PutUint32(chunk[4:8], uint32(len(data)))
	copy(chunk[8:], data)
	return chunk
}

// encodeVarLen encodes an integer as MIDI variable-length (7-bit) encoding.
func encodeVarLen(value int) []byte {
	if value < 0 {
		value = 0
	}
	// Count how many bytes needed.
	n := value
	var count int
	for {
		count++
		n >>= 7
		if n == 0 {
			break
		}
	}
	buf := make([]byte, count)
	for i := range buf {
		b := byte(value >> (7 * (count - 1 - i)) & 0x7F)
		if i < count-1 {
			b |= 0x80
		}
		buf[i] = b
	}
	return buf
}

// bpmToTempo converts BPM to microseconds per quarter note.
func bpmToTempo(bpm int) int {
	if bpm <= 0 {
		bpm = 120
	}
	return int(math.Round(60_000_000.0 / float64(bpm)))
}

// beatToTick converts beats to ticks at the given resolution.
func beatToTick(beat float64, ticksPerBeat int) int {
	return int(math.Round(beat * float64(ticksPerBeat)))
}

// clamp restricts a value to [lo, hi].
func clamp(val, lo, hi int) int {
	if val < lo {
		return lo
	}
	if val > hi {
		return hi
	}
	return val
}

// filterEnabled returns only enabled tracks.
func filterEnabled(tracks []schema.TrackIR) []schema.TrackIR {
	var out []schema.TrackIR
	for _, t := range tracks {
		if t.Enabled {
			out = append(out, t)
		}
	}
	return out
}

// sortAbsEvents sorts absolute events by tick, with note-offs before note-ons at the same tick.
func sortAbsEvents(events []absEvent) {
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			swap := false
			if events[j].tick < events[i].tick {
				swap = true
			} else if events[j].tick == events[i].tick && !events[j].isNoteOn && events[i].isNoteOn {
				swap = true
			}
			if swap {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
}

type absEvent struct {
	tick     int
	isNoteOn bool
	pitch    byte
	velocity byte
	channel  byte
}
