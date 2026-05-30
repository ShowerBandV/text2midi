export interface MidiNote {
  id: string;
  pitch: string; // e.g. "C4", "Eb4", "D#5"
  time: number;  // Beat/Step offset (0, 0.5, 1, 1.5...)
  duration: number; // Beat/Step duration (0.5, 1.0, 2.0...)
  velocity: number; // MIDI velocity (0-127)
}

export type InstrumentType = "piano" | "synth" | "strings";

export interface MidiMetadata {
  title: string;
  seed: number;
  tempo: number;
  key: string;     // e.g. "C", "Eb"
  scale: string;   // e.g. "Major", "Minor", "Phrygian"
  complexity: "Low" | "Medium" | "High";
  genre: string;   // e.g. "Cybernetic Jazz", "Synthwave", "Ambient"
  durationStr: string; // e.g. "03:42"
}

export interface MidiTrack {
  id: string;
  notes: MidiNote[];
  metadata: MidiMetadata;
  instrument: InstrumentType;
  globalVelocity: number;
  createdAt: string;
  fileId?: string; // Go backend file ID for download
}

export interface PlaybackState {
  isPlaying: boolean;
  tempo: number;
  currentBeat: number;
  loopEndBeat: number;
}

// ─── Auth ───────────────────────────────────────────────────────────

export interface User {
  id: number;
  username: string;
  created_at: string;
}

export interface AuthState {
  user: User | null;
  token: string | null;
}

export interface InfoResponse {
  styles: string[];
  tiers: Record<string, number>;
}
