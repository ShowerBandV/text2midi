// Package midi — SMF Type 0/1 reader with proper note duration/chord parsing.
package midi

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

// TrackEvent is a raw MIDI event with absolute tick position.
type TrackEvent struct {
	Tick     int
	Type     byte // 0x90=note_on, 0x80=note_off, 0xFF=meta
	Channel  byte
	Note     byte
	Velocity byte
	MetaType byte
	MetaData []byte
}

// ParsedNote is a note with computed duration.
type ParsedNote struct {
	Pitch      int
	StartTick  int
	Duration   int // in ticks
	Velocity   int
	Channel    int
	TrackIndex int
}

// ReadMIDIFile parses a .mid file and returns parsed notes + metadata.
func ReadMIDIFile(path string) ([]ParsedNote, int, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, "", fmt.Errorf("read %s: %w", path, err)
	}

	if string(data[0:4]) != "MThd" {
		return nil, 0, "", fmt.Errorf("not a MIDI file")
	}

	format := binary.BigEndian.Uint16(data[8:10])
	numTracks := binary.BigEndian.Uint16(data[10:12])
	ticksPerBeat := int(binary.BigEndian.Uint16(data[12:14]))

	if format > 1 {
		return nil, 0, "", fmt.Errorf("unsupported SMF format: %d", format)
	}

	// Read all tracks into raw events.
	type rawTrack struct {
		events []TrackEvent
		name   string
	}
	var tracks []rawTrack
	offset := 14

	for t := uint16(0); t < numTracks && offset+8 < len(data); t++ {
		if string(data[offset:offset+4]) != "MTrk" {
			break
		}
		trackLen := int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		trackEnd := offset + trackLen
		if trackEnd > len(data) {
			trackEnd = len(data)
		}

		var events []TrackEvent
		tick := 0
		trackName := fmt.Sprintf("track_%d", t)
		var lastStatus byte

		for offset < trackEnd {
			delta, consumed := readVarLenBytes(data[offset:])
			offset += consumed
			tick += delta

			if offset >= trackEnd {
				break
			}

			status := data[offset]

			if status == 0xFF {
				// Meta event.
				offset++
				metaType := data[offset]
				offset++
				lenVal, consumed := readVarLenBytes(data[offset:])
				offset += consumed
				metaData := data[offset : offset+lenVal]
				offset += lenVal

				if metaType == 0x03 { // Track name
					trackName = string(metaData)
				}
				events = append(events, TrackEvent{
					Tick: tick, Type: 0xFF, MetaType: metaType, MetaData: metaData,
				})
				continue
			}

			var eventType, channel byte
			if status >= 0x80 {
				eventType = status & 0xF0
				channel = status & 0x0F
				lastStatus = status
				offset++
			} else {
				// Running status.
				eventType = lastStatus & 0xF0
				channel = lastStatus & 0x0F
			}

			switch eventType {
			case 0x90: // Note On
				if offset+1 < trackEnd {
					note := data[offset]
					vel := data[offset+1]
					offset += 2
					events = append(events, TrackEvent{
						Tick: tick, Type: 0x90, Channel: channel,
						Note: note, Velocity: vel,
					})
				}
			case 0x80: // Note Off
				if offset+1 < trackEnd {
					note := data[offset]
					_ = data[offset+1]
					offset += 2
					events = append(events, TrackEvent{
						Tick: tick, Type: 0x80, Channel: channel,
						Note: note,
					})
				}
			default:
				// Skip other events.
				switch eventType {
				case 0xC0, 0xD0:
					offset++
				case 0xB0, 0xE0:
					offset += 2
				default:
					offset++
				}
			}
		}

		tracks = append(tracks, rawTrack{events: events, name: trackName})
	}

	// Convert raw events to parsed notes (matching note-on with note-off for proper duration).
	type noteKey struct {
		pitch   int
		channel int
	}

	var allNotes []ParsedNote
	totalTicks := 0

	for ti, track := range tracks {
		activeNotes := make(map[noteKey]int) // key → start tick
		for _, ev := range track.events {
			if ev.Type == 0x90 && ev.Velocity > 0 {
				key := noteKey{pitch: int(ev.Note), channel: int(ev.Channel)}
				activeNotes[key] = ev.Tick
			} else if ev.Type == 0x80 || (ev.Type == 0x90 && ev.Velocity == 0) {
				key := noteKey{pitch: int(ev.Note), channel: int(ev.Channel)}
				if startTick, ok := activeNotes[key]; ok {
					duration := ev.Tick - startTick
					if duration <= 0 {
						duration = ticksPerBeat / 4
					}
					allNotes = append(allNotes, ParsedNote{
						Pitch:      int(ev.Note),
						StartTick:  startTick,
						Duration:   duration,
						Velocity:   80,
						Channel:    int(ev.Channel),
						TrackIndex: ti,
					})
					delete(activeNotes, key)
				}
			}
			if ev.Tick > totalTicks {
				totalTicks = ev.Tick
			}
		}
	}

	sort.Slice(allNotes, func(i, j int) bool {
		return allNotes[i].StartTick < allNotes[j].StartTick
	})

	totalBars := totalTicks / (ticksPerBeat * 4)
	if totalBars < 1 {
		totalBars = 4
	}

	fmt.Printf("[MIDIReader] %d tracks, %d notes, %d ticks/beat, %d bars from %s\n",
		len(tracks), len(allNotes), ticksPerBeat, totalBars, path)
	return allNotes, totalBars, "", nil
}

// ConvertTicksToBeats converts parsed notes to beat-based NoteEvents.
func ConvertTicksToBeats(notes []ParsedNote, ticksPerBeat int) []BeatNote {
	var result []BeatNote
	for _, n := range notes {
		result = append(result, BeatNote{
			Pitch:      n.Pitch,
			StartBeat:  float64(n.StartTick) / float64(ticksPerBeat),
			Duration:   float64(n.Duration) / float64(ticksPerBeat),
			Velocity:   n.Velocity,
			Channel:    n.Channel,
			TrackIndex: n.TrackIndex,
		})
	}
	return result
}

// BeatNote is a note with beat-based timing.
type BeatNote struct {
	Pitch      int
	StartBeat  float64
	Duration   float64
	Velocity   int
	Channel    int
	TrackIndex int
}

func readVarLenBytes(data []byte) (int, int) {
	value := 0
	for i := 0; i < len(data) && i < 4; i++ {
		value = (value << 7) | int(data[i]&0x7F)
		if data[i]&0x80 == 0 {
			return value, i + 1
		}
	}
	return value, len(data)
}
