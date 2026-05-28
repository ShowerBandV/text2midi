// Package midi — SMF Type 0/1 reader.
// Parses .mid files into NoteEvents for DNA extraction and template building.
package midi

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ReadMIDIFile parses a .mid file and returns NoteEvents grouped by track.
// Returns track names and events per track.
func ReadMIDIFile(path string) (map[string][]schema.NoteEvent, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("read %s: %w", path, err)
	}

	if len(data) < 14 {
		return nil, 0, fmt.Errorf("file too small: %d bytes", len(data))
	}

	// Parse header.
	if string(data[0:4]) != "MThd" {
		return nil, 0, fmt.Errorf("not a MIDI file: missing MThd header")
	}
	headerLen := binary.BigEndian.Uint32(data[4:8])
	if headerLen < 6 {
		return nil, 0, fmt.Errorf("header too short")
	}
	format := binary.BigEndian.Uint16(data[8:10])
	numTracks := binary.BigEndian.Uint16(data[10:12])
	// ticksPerQuarterNote is at data[12:14], we don't need it for now

	if format > 1 {
		return nil, 0, fmt.Errorf("unsupported SMF format: %d", format)
	}

	result := make(map[string][]schema.NoteEvent)
	offset := 14
	tick := 0
	totalBars := 0

	for t := uint16(0); t < numTracks; t++ {
		if offset+8 > len(data) {
			break
		}
		if string(data[offset:offset+4]) != "MTrk" {
			return nil, 0, fmt.Errorf("expected MTrk at track %d", t)
		}
		trackLen := int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8

		trackName := fmt.Sprintf("track_%d", t)
		var events []schema.NoteEvent
		tick = 0
		trackEnd := offset + trackLen
		if trackEnd > len(data) {
			trackEnd = len(data)
		}

		for offset < trackEnd {
			// Read delta time (variable length).
			delta, consumed := readVarLen(data[offset:])
			offset += consumed
			tick += delta

			if offset >= trackEnd {
				break
			}

			status := data[offset]

			// Meta event (0xFF)
			if status == 0xFF {
				offset++
				if offset >= trackEnd {
					break
				}
				metaType := data[offset]
				offset++
				metaLen, consumed := readVarLen(data[offset:])
				offset += consumed
				metaData := data[offset : offset+int(metaLen)]
				offset += int(metaLen)

				switch metaType {
				case 0x03: // Track name
					trackName = string(metaData)
				case 0x51: // Tempo
					if len(metaData) >= 3 {
						microsecPerBeat := int(metaData[0])<<16 | int(metaData[1])<<8 | int(metaData[2])
						_ = microsecPerBeat // 60000000 / microsecPerBeat = BPM
					}
				}
				continue
			}

			// Running status: if status < 0x80, use previous status byte.
			channel := byte(0)
			eventType := status
			if status >= 0x80 {
				channel = status & 0x0F
				eventType = status & 0xF0
				offset++
			} else {
				// Running status — reuse previous event type
				channel = prevStatus & 0x0F
				eventType = prevStatus & 0xF0
			}
			_ = channel

			switch eventType {
			case 0x90: // Note On
				if offset+1 >= trackEnd {
					break
				}
				pitch := int(data[offset])
				velocity := int(data[offset+1])
				offset += 2
				if velocity > 0 {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: pitch,
						StartBeat:    float64(tick) / 480.0,
						DurationBeat: 0.25,
						Velocity:     velocity,
					})
				}
				prevStatus = status

			case 0x80: // Note Off
				if offset+1 >= trackEnd {
					break
				}
				pitch := int(data[offset])
				_ = pitch
				offset += 2
				prevStatus = status

			default:
				// Skip other events (CC, pitch bend, etc.)
				switch eventType {
				case 0xB0, 0xC0, 0xD0, 0xE0:
					offset++
					if eventType != 0xC0 && eventType != 0xD0 {
						offset++
					}
				default:
					// Unknown, skip one byte
					offset++
				}
				prevStatus = status
			}
		}

		// Calculate approximate bar count from ticks.
		if len(events) > 0 {
			maxBeat := 0.0
			for _, ev := range events {
				if ev.StartBeat > maxBeat {
					maxBeat = ev.StartBeat
				}
			}
			bars := int(maxBeat)/4 + 1
			if bars > totalBars {
				totalBars = bars
			}
		}

		if len(events) > 0 {
			// Deduplicate notes at the same position (keep highest velocity).
			type posKey struct {
				pitch     int
				startBeat float64
			}
			dedup := make(map[posKey]schema.NoteEvent)
			for _, ev := range events {
				key := posKey{pitch: ev.Pitch, startBeat: ev.StartBeat}
				if existing, ok := dedup[key]; !ok || ev.Velocity > existing.Velocity {
					dedup[key] = ev
				}
			}
			events = make([]schema.NoteEvent, 0, len(dedup))
			for _, ev := range dedup {
				events = append(events, ev)
			}
			sort.Slice(events, func(i, j int) bool {
				return events[i].StartBeat < events[j].StartBeat
			})
			result[trackName] = events
		}
	}

	if totalBars < 4 {
		totalBars = 4
	}

	fmt.Printf("[MIDIReader] parsed %d tracks, %d bars from %s\n", len(result), totalBars, path)
	return result, totalBars, nil
}

// readVarLen reads a MIDI variable-length value.
func readVarLen(data []byte) (int, int) {
	value := 0
	for i := 0; i < len(data); i++ {
		value = (value << 7) | int(data[i]&0x7F)
		if data[i]&0x80 == 0 {
			return value, i + 1
		}
	}
	return value, len(data)
}

var prevStatus byte = 0
