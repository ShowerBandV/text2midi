// Web Audio API Pitch converter & synthesizers

const notesMap: Record<string, number> = {
  "C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3, "E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8, "Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11
};

export function pitchToFrequency(pitch: string): number {
  if (!pitch) return 440;
  
  // Parse note and octave
  const notePart = pitch.slice(0, -1);
  const octavePart = pitch.slice(-1);
  const octave = parseInt(octavePart, 10);
  
  if (isNaN(octave)) return 440;
  
  const semitone = notesMap[notePart];
  if (semitone === undefined) return 440;
  
  // MIDI number calculation (C4 = index 60)
  const midiNumber = 12 * (octave + 1) + semitone;
  // Standard frequency conversion formula
  return 440 * Math.pow(2, (midiNumber - 69) / 12);
}

export function playNote(
  audioCtx: AudioContext,
  pitch: string,
  durationSec: number,
  instrument: "piano" | "synth" | "strings",
  velocity: number = 85
) {
  const osc1 = audioCtx.createOscillator();
  const osc2 = audioCtx.createOscillator();
  const gainNode = audioCtx.createGain();
  const filter = audioCtx.createBiquadFilter();

  const freq = pitchToFrequency(pitch);
  const volumeMultiplier = (velocity / 127) * 0.35; // Map velocity to sensible gain limits

  const now = audioCtx.currentTime;

  if (instrument === "piano") {
    // Grand Piano synthesis curve:
    // Triangle wave + subtle sine sub-harmonic, rapid decay
    osc1.type = "triangle";
    osc1.frequency.value = freq;

    osc2.type = "sine";
    osc2.frequency.value = freq * 2; // Octave overtone
    
    // Low pass filter to warm up the sound
    filter.type = "lowpass";
    filter.frequency.setValueAtTime(1400, now);
    filter.frequency.exponentialRampToValueAtTime(300, now + durationSec);

    // ADSR Envelope
    gainNode.gain.setValueAtTime(0, now);
    gainNode.gain.linearRampToValueAtTime(volumeMultiplier, now + 0.005); // Rapid attack
    gainNode.gain.exponentialRampToValueAtTime(volumeMultiplier * 0.3, now + 0.15); // Decay
    gainNode.gain.exponentialRampToValueAtTime(0.0001, now + durationSec); // Release

    osc1.connect(filter);
    osc2.connect(filter);
    filter.connect(gainNode);
    gainNode.connect(audioCtx.destination);

    osc1.start(now);
    osc2.start(now);
    osc1.stop(now + durationSec + 0.1);
    osc2.stop(now + durationSec + 0.1);

  } else if (instrument === "synth") {
    // Aggressive detuned dual sawtooth waves with sweep filter
    osc1.type = "sawtooth";
    osc1.frequency.value = freq - 2; // detune -2Hz

    osc2.type = "sawtooth";
    osc2.frequency.value = freq + 2; // detune +2Hz

    // Hot sweeping lowpass resonant filter
    filter.type = "lowpass";
    filter.Q.value = 8; // resonance
    filter.frequency.setValueAtTime(freq * 4, now);
    filter.frequency.exponentialRampToValueAtTime(freq * 1.2, now + durationSec);

    // ADSR Envelope
    gainNode.gain.setValueAtTime(0, now);
    gainNode.gain.linearRampToValueAtTime(volumeMultiplier * 0.8, now + 0.02); // Sharp punchy attack
    gainNode.gain.linearRampToValueAtTime(volumeMultiplier * 0.5, now + 0.1); // Decay
    gainNode.gain.setValueAtTime(volumeMultiplier * 0.5, now + durationSec - 0.05); // Sustain
    gainNode.gain.exponentialRampToValueAtTime(0.0001, now + durationSec); // Quick release

    osc1.connect(filter);
    osc2.connect(filter);
    filter.connect(gainNode);
    gainNode.connect(audioCtx.destination);

    osc1.start(now);
    osc2.start(now);
    osc1.stop(now + durationSec);
    osc2.stop(now + durationSec);

  } else if (instrument === "strings") {
    // Warm detuned sawtooth/triangle slow bow strings
    osc1.type = "sawtooth";
    osc1.frequency.value = freq;

    osc2.type = "triangle";
    osc2.frequency.value = freq * 0.5; // sub-bass warm drone

    filter.type = "lowpass";
    filter.frequency.setValueAtTime(900, now);
    filter.frequency.exponentialRampToValueAtTime(500, now + durationSec * 0.8);

    // Warm string ADSR (Long bow attack and smooth release bleed)
    gainNode.gain.setValueAtTime(0, now);
    gainNode.gain.linearRampToValueAtTime(volumeMultiplier * 0.8, now + 0.15); // Stately slow attack
    gainNode.gain.setValueAtTime(volumeMultiplier * 0.8, now + durationSec - 0.15); // Long sustain
    gainNode.gain.linearRampToValueAtTime(0.0001, now + durationSec + 0.25); // Very soft string release bleed

    osc1.connect(filter);
    osc2.connect(filter);
    filter.connect(gainNode);
    gainNode.connect(audioCtx.destination);

    osc1.start(now);
    osc2.start(now);
    osc1.stop(now + durationSec + 0.3);
    osc2.stop(now + durationSec + 0.3);
  }
}

// Convert track content into standard playable MIDI file bytes and download URL
export function generateMidiFileBlobUrl(notes: any[], tempo: number): string {
  // We'll compose a standard single-track MIDI format level 0 file in memory!
  // This is a sensational display of deep musical utility.
  
  const headerChunk = [
    0x4d, 0x54, 0x68, 0x64, // "MThd" label
    0x00, 0x00, 0x00, 0x06, // Chunk size (6 bytes)
    0x00, 0x00,             // Format type (0 = single track)
    0x00, 0x01,             // Number of tracks (1)
    0x01, 0xe0              // ticks per quarter note divisor (480 ticks)
  ];

  const ticksPerQuarter = 480;
  
  // Sort notes by onset time
  const events: Array<{ tick: number; type: "on" | "off"; pitch: string; velocity: number }> = [];
  notes.forEach((note) => {
    const startTick = Math.round(note.time * ticksPerQuarter);
    const endTick = Math.round((note.time + note.duration) * ticksPerQuarter);
    events.push({ tick: startTick, type: "on", pitch: note.pitch, velocity: note.velocity });
    events.push({ tick: endTick, type: "off", pitch: note.pitch, velocity: note.velocity });
  });

  // Sort events by tick time
  events.sort((a, b) => a.tick - b.tick);

  const translatePitchToMidiNum = (pitch: string): number => {
    const notePart = pitch.slice(0, -1);
    const octave = parseInt(pitch.slice(-1), 10);
    const semitone = notesMap[notePart] || 0;
    return 12 * (octave + 1) + semitone;
  };

  const trackEventsBytes: number[] = [];
  
  // Add microsecond per quarter tempo metadata event
  // tempo bpm to microseconds per beat: 60,000,000 / tempo
  const microsecPerBeat = Math.round(60000000 / tempo);
  const t1 = (microsecPerBeat >> 16) & 0xff;
  const t2 = (microsecPerBeat >> 8) & 0xff;
  const t3 = microsecPerBeat & 0xff;

  // Add tempo meta event (Delta time 0, meta event FF, type 51, length 03)
  trackEventsBytes.push(0x00, 0xff, 0x51, 0x03, t1, t2, t3);

  // Helper to push variable-length quantity integer delta times
  const pushVarLen = (value: number) => {
    let buffer = value & 0x7f;
    while ((value >>= 7) > 0) {
      buffer <<= 8;
      buffer |= (value & 0x7f) | 0x80;
    }
    while (true) {
      trackEventsBytes.push(buffer & 0xff);
      if (buffer & 0x80) {
        buffer >>= 8;
      } else {
        break;
      }
    }
  };

  let currentTick = 0;
  events.forEach((evt) => {
    const deltaTicks = evt.tick - currentTick;
    pushVarLen(deltaTicks);
    currentTick = evt.tick;

    const midiNum = translatePitchToMidiNum(evt.pitch);
    const velocityByte = evt.velocity & 0x7f;

    if (evt.type === "on") {
      trackEventsBytes.push(0x90, midiNum, velocityByte); // Note On channel 0
    } else {
      trackEventsBytes.push(0x80, midiNum, 0x00); // Note Off channel 0
    }
  });

  // End of track meta-event (Delta 0, FF 2F 00)
  trackEventsBytes.push(0x00, 0xff, 0x2f, 0x00);

  const trackLength = trackEventsBytes.length;
  const trackChunkHeader = [
    0x4d, 0x54, 0x72, 0x6b, // "MTrk" label
    (trackLength >> 24) & 0xff,
    (trackLength >> 16) & 0xff,
    (trackLength >> 8) & 0xff,
    trackLength & 0xff
  ];

  const midiBytes = new Uint8Array([
    ...headerChunk,
    ...trackChunkHeader,
    ...trackEventsBytes
  ]);

  const blob = new Blob([midiBytes], { type: "audio/midi" });
  return URL.createObjectURL(blob);
}
