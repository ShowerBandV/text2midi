/**
 * Chord detection from MIDI notes.
 * Groups notes by bar, identifies pitch-class sets, maps to chord names.
 */

import { MidiNote } from "../types";

// ─── Pitch helpers ────────────────────────────────────────────────

const NOTE_TO_SEMITONE: Record<string, number> = {
  "C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3,
  "E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8,
  "Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11,
};

export function pitchToMidi(pitch: string): number {
  const note = pitch.slice(0, -1);
  const octave = parseInt(pitch.slice(-1), 10);
  const semitone = NOTE_TO_SEMITONE[note];
  if (semitone === undefined || isNaN(octave)) return 60;
  return 12 * (octave + 1) + semitone;
}

function pitchClass(midi: number): number {
  return midi % 12;
}

const NOTE_NAMES = ["C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"];

function pcName(pc: number): string {
  return NOTE_NAMES[pc % 12];
}

// ─── Chord templates (pitch-class sets) ──────────────────────────

interface ChordTemplate {
  name: string;
  intervals: number[]; // semitone intervals from root
  quality: "major" | "minor" | "dim" | "aug" | "7th" | "sus";
}

const CHORD_TEMPLATES: ChordTemplate[] = [
  { name: "",            intervals: [0, 4, 7],       quality: "major" },
  { name: "m",           intervals: [0, 3, 7],       quality: "minor" },
  { name: "dim",         intervals: [0, 3, 6],       quality: "dim" },
  { name: "aug",         intervals: [0, 4, 8],       quality: "aug" },
  { name: "maj7",        intervals: [0, 4, 7, 11],   quality: "7th" },
  { name: "7",           intervals: [0, 4, 7, 10],   quality: "7th" },
  { name: "m7",          intervals: [0, 3, 7, 10],   quality: "7th" },
  { name: "dim7",        intervals: [0, 3, 6, 9],    quality: "7th" },
  { name: "m7b5",        intervals: [0, 3, 6, 10],   quality: "7th" },
  { name: "sus2",        intervals: [0, 2, 7],       quality: "sus" },
  { name: "sus4",        intervals: [0, 5, 7],       quality: "sus" },
  { name: "6",           intervals: [0, 4, 7, 9],    quality: "major" },
  { name: "m6",          intervals: [0, 3, 7, 9],    quality: "minor" },
  { name: "9",           intervals: [0, 4, 7, 10, 14], quality: "7th" },
  { name: "m9",          intervals: [0, 3, 7, 10, 14], quality: "7th" },
  { name: "maj9",        intervals: [0, 4, 7, 11, 14], quality: "7th" },
  { name: "add9",        intervals: [0, 4, 7, 14],   quality: "7th" },
];

// ─── Detection ────────────────────────────────────────────────────

export interface DetectedChord {
  bar: number;
  beat: number;
  root: string;       // e.g. "C", "Eb"
  quality: string;    // e.g. "", "m", "7", "maj7"
  name: string;       // e.g. "C", "Ebmaj7"
  notes: string[];    // pitch names in the chord
  confidence: number; // 0-1
}

/**
 * Normalise a pitch-class set to its lowest rotation (for matching).
 */
function normaliseSet(pcs: number[]): number[] {
  if (pcs.length === 0) return [];
  const sorted = [...new Set(pcs)].sort((a, b) => a - b);
  // Try each rotation, pick the one with smallest first interval
  let best = sorted;
  for (let i = 1; i < sorted.length; i++) {
    const rot = [...sorted.slice(i), ...sorted.slice(0, i).map(x => x + 12)];
    const normalised = rot.map(x => x - rot[0]);
    const bestNorm = best.map(x => x - best[0]);
    if (normalised.length !== bestNorm.length) continue;
    let smaller = false;
    for (let j = 0; j < normalised.length; j++) {
      if (normalised[j] < bestNorm[j]) { smaller = true; break; }
      if (normalised[j] > bestNorm[j]) break;
    }
    if (smaller) best = rot;
  }
  return best.map(x => x % 12);
}

/**
 * Match a pitch-class set against chord templates.
 */
function matchChord(pcs: number[]): { root: number; quality: string; name: string } | null {
  const set = normaliseSet(pcs);
  if (set.length < 2) return null;

  // Try each possible root
  for (let root = 0; root < 12; root++) {
    const relative = set.map(p => (p - root + 12) % 12).sort((a, b) => a - b);
    for (const tmpl of CHORD_TEMPLATES) {
      const expected = tmpl.intervals.map(i => i % 12).sort((a, b) => a - b);
      if (relative.length === expected.length && relative.every((v, i) => v === expected[i])) {
        return {
          root,
          quality: tmpl.name,
          name: pcName(root) + tmpl.name,
        };
      }
    }
  }
  return null;
}

/**
 * Detect chords from a list of MIDI notes.
 * Groups by bar-sized windows, detecting the chord in each window.
 */
export function detectChords(notes: MidiNote[], bars: number = 8): DetectedChord[] {
  if (notes.length === 0) return [];

  const maxBeat = Math.max(...notes.map(n => n.time + n.duration));
  const numBars = Math.max(bars, Math.ceil(maxBeat / 4));

  const result: DetectedChord[] = [];

  for (let bar = 0; bar < numBars; bar++) {
    const barStart = bar * 4;
    const barEnd = (bar + 1) * 4;

    // Collect all notes that sound in this bar
    const soundingNotes = notes.filter(n => n.time < barEnd && n.time + n.duration > barStart);
    if (soundingNotes.length === 0) continue;

    const pcs = [...new Set(soundingNotes.map(n => pitchClass(pitchToMidi(n.pitch))))];

    const match = matchChord(pcs);
    if (match) {
      result.push({
        bar: bar + 1,
        beat: barStart,
        root: pcName(match.root),
        quality: match.quality,
        name: match.name,
        notes: [...new Set(soundingNotes.map(n => n.pitch))],
        confidence: 0.8,
      });
    } else {
      // Fallback: just list the pitch classes
      result.push({
        bar: bar + 1,
        beat: barStart,
        root: pcName(pcs[0]),
        quality: "",
        name: pcs.map(p => pcName(p)).join("/"),
        notes: [...new Set(soundingNotes.map(n => n.pitch))],
        confidence: 0.3,
      });
    }
  }

  return result;
}

/**
 * Convert MidiNote[] to the format expected by /api/dna/extract.
 */
export function notesToDNEEvents(
  notes: MidiNote[],
): Array<{ pitch: number; start_beat: number; duration_beat: number; velocity: number }> {
  return notes.map(n => ({
    pitch: pitchToMidi(n.pitch),
    start_beat: n.time,
    duration_beat: n.duration,
    velocity: n.velocity,
  }));
}
