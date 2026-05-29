import React, { useRef, useState, useEffect } from "react";
import { MidiNote, InstrumentType, PlaybackState } from "../types";
import { playNote } from "../utils/audio";
import { Play, Pause, RotateCcw, Plus, Minus, CheckCircle, Scale, Volume2, Waves, Piano, Sliders, Music } from "lucide-react";

interface PianoRollProps {
  notes: MidiNote[];
  setNotes: (notes: MidiNote[]) => void;
  tempo: number;
  setTempo: (tempo: number) => void;
  rootKey: string;
  setRootKey: (key: string) => void;
  scaleType: string;
  setScaleType: (scale: string) => void;
  instrument: InstrumentType;
  setInstrument: (inst: InstrumentType) => void;
  globalVelocity: number;
  setGlobalVelocity: (vel: number) => void;
  title: string;
}

// Fixed note rows on our key matrix (C3 up to C5)
const PIANO_ROWS = [
  "C6", "B5", "A#5", "A5", "G#5", "G5", "F#5", "F5", "E5", "D#5", "D5", "C#5", "C5",
  "B4", "A#4", "A4", "G#4", "G4", "F#4", "F4", "E4", "D#4", "D4", "C#4", "C4",
  "B3", "A#3", "A3", "G#3", "G3", "F#3", "F3", "E3", "D#3", "D3", "C#3", "C3",
  "B2", "A#2", "A2", "G#2", "G2", "F#2", "F2", "E2", "D#2", "D2", "C#2", "C2"
];

// Determine if row index matches a black key on piano keyboard
const isBlackKey = (pitch: string): boolean => {
  return pitch.includes("#") || pitch.includes("b");
};

export default function PianoRoll({
  notes,
  setNotes,
  tempo,
  setTempo,
  rootKey,
  setRootKey,
  scaleType,
  setScaleType,
  instrument,
  setInstrument,
  globalVelocity,
  setGlobalVelocity,
  title
}: PianoRollProps) {
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentBeat, setCurrentBeat] = useState(0);
  const [zoomLevel, setZoomLevel] = useState(1); // multiplier for beat columns widths

  const audioCtxRef = useRef<AudioContext | null>(null);
const pianoKeysRef = useRef<HTMLDivElement | null>(null);
const pianoCanvasRef = useRef<HTMLDivElement | null>(null);
const scrollSyncLock = useRef(false);
  const animationFrameRef = useRef<number | null>(null);
  const startTimeRef = useRef<number>(0);
  const pausedOffsetRef = useRef<number>(0);
  const triggeredNotesRef = useRef<Set<string>>(new Set());
const beatRef = useRef(0);
const playheadRef = useRef<HTMLDivElement | null>(null);

  // Bidirectional scroll sync between pitch keys and grid canvas
const handleGridScroll = () => {
  if (scrollSyncLock.current) return;
  scrollSyncLock.current = true;
  if (pianoKeysRef.current && pianoCanvasRef.current) {
    pianoKeysRef.current.scrollTop = pianoCanvasRef.current.scrollTop;
  }
  scrollSyncLock.current = false;
};
// Sync grid when pitch keys scroll natively
const handleKeysScroll = () => {
  if (scrollSyncLock.current) return;
  scrollSyncLock.current = true;
  if (pianoKeysRef.current && pianoCanvasRef.current) {
    pianoCanvasRef.current.scrollTop = pianoKeysRef.current.scrollTop;
  }
  scrollSyncLock.current = false;
};

const COLUMN_WIDTH = 40 * zoomLevel; // pixels per half beat
  const ROW_HEIGHT = 40; // pixels per piano key note
  const TOTAL_BEATS = 128; // 32 bars * 4 beats
  const GRID_COLUMNS = TOTAL_BEATS * 2; // half-beat grid snap steps

  // Lazy initialize AudioContext on user click to prevent browser constraints
  const getAudioContext = (): AudioContext => {
    if (!audioCtxRef.current) {
      audioCtxRef.current = new (window.AudioContext || (window as any).webkitAudioContext)();
    }
    if (audioCtxRef.current.state === "suspended") {
      audioCtxRef.current.resume();
    }
    return audioCtxRef.current;
  };

  // Optimized playback: direct DOM playhead, no React re-render per frame
  useEffect(() => {
    if (isPlaying) {
      startTimeRef.current = performance.now() / 1000 - pausedOffsetRef.current;
      let lastDisplayBeat = -1;
      
      const tick = () => {
        const nowSec = performance.now() / 1000;
        const elapsedSec = nowSec - startTimeRef.current;
        const beatsPerSec = tempo / 60;
        const rawBeat = elapsedSec * beatsPerSec;
        const loopBeat = rawBeat % TOTAL_BEATS;

        beatRef.current = loopBeat;

        // Reset trigger memory on loop
        if (loopBeat < 1 && lastDisplayBeat > TOTAL_BEATS - 2) {
          triggeredNotesRef.current.clear();
        }

        // Move playhead via direct DOM (no React re-render)
        if (playheadRef.current) {
          const playheadX = (loopBeat / 0.5) * COLUMN_WIDTH;
          playheadRef.current.style.left = playheadX + 'px';
        }

        // Only update React state when beat display changes (not every frame)
        const displayBeat = Math.round(loopBeat * 10) / 10;
        if (Math.abs(displayBeat - lastDisplayBeat) > 0.05) {
          lastDisplayBeat = displayBeat;
          setCurrentBeat(loopBeat);
        }

        // Sound Engine - use beatRef for accurate timing
        const beatSec = 60 / tempo;
        const currentBeatAccurate = beatRef.current;
        
        notes.forEach((note) => {
          if (note.time >= TOTAL_BEATS) return;
          const triggerKey = `${note.pitch}-${note.time}`;
          const triggered = triggeredNotesRef.current.has(triggerKey);
          const timeDiff = Math.abs(currentBeatAccurate - note.time);
          const tolerance = Math.max(0.05, beatSec * 0.3);
          
          if (!triggered && timeDiff < tolerance) {
            triggeredNotesRef.current.add(triggerKey);
            playNote(
              getAudioContext(),
              note.pitch,
              note.duration * beatSec,
              instrument,
              Math.min(note.velocity, globalVelocity)
            );
          }
        });

        animationFrameRef.current = requestAnimationFrame(tick);
      };

      animationFrameRef.current = requestAnimationFrame(tick);
    } else {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
      pausedOffsetRef.current = beatRef.current * (60 / tempo);
    }

    return () => {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
    };
  }, [isPlaying, notes, tempo, instrument, globalVelocity]);

  const togglePlay = () => {
    getAudioContext(); // Resume context if suspended
    setIsPlaying(!isPlaying);
  };

  const resetTimeline = () => {
    setIsPlaying(false);
    setCurrentBeat(0);
    pausedOffsetRef.current = 0;
    triggeredNotesRef.current.clear();
  };

  // Grid interaction: Drag or Draw Note
  // We allow rapid drawing of notes with standard click of grid squares!
  const handleGridClick = (pitch: string, stepIndex: number) => {
    const clickTime = stepIndex * 0.5; // snaps to half beats

    // Check if there is already a note on this pitch at this exact snap step
    const existingIndex = notes.findIndex(n => n.pitch === pitch && Math.abs(n.time - clickTime) < 0.15);

    if (existingIndex !== -1) {
      // If note exists, click deletes it (toggle mode is standard for clean DAW UX)
      const updatedNotes = [...notes];
      updatedNotes.splice(existingIndex, 1);
      setNotes(updatedNotes);
    } else {
      // Create a gorgeous new note
      const newNote: MidiNote = {
        id: `note-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
        pitch,
        time: clickTime,
        duration: 0.5, // length is half-beat snap by default
        velocity: globalVelocity
      };

      // Sound feedback when clicked
      const ctx = getAudioContext();
      playNote(ctx, pitch, 0.25, instrument, globalVelocity);

      setNotes([...notes, newNote]);
    }
  };

  // Floating zoom utility triggers
  const zoomIn = () => setZoomLevel(prev => Math.min(2.5, prev + 0.15));
  const zoomOut = () => setZoomLevel(prev => Math.max(0.6, prev - 0.15));

  return (
    <div className="flex flex-col flex-grow overflow-hidden h-full">
      {/* Upper Status Banner */}
      <div className="bg-surface-container-low px-lg py-xs flex items-center justify-between border-b border-white/5 select-none text-xs">
        <div className="flex items-center gap-sm">
          <span className="w-2.5 h-2.5 rounded-full bg-secondary animate-pulse" />
          <span className="font-mono text-secondary tracking-wider uppercase font-semibold">
            {isPlaying ? "Sync active - Playing sequence" : "Idle - Editing Mode"}
          </span>
        </div>
        <div className="font-mono text-on-surface-variant opacity-60">
          Source file: {title || "Untitled"}.mid
        </div>
      </div>

      {/* Primary Toolbar section */}
      <div className="bg-surface px-lg py-sm flex flex-wrap items-center justify-between border-b border-white/10 gap-md">
        <div className="flex items-center gap-lg">
          {/* Transport panel controls */}
          <div className="flex items-center gap-2 glass-panel p-1 rounded-xl">
            <button
              onClick={resetTimeline}
              title="Reset playhead"
              className="p-2 hover:text-white text-on-surface-variant transition-colors cursor-pointer active:scale-90"
            >
              <RotateCcw className="w-4 h-4" />
            </button>
            <button
              onClick={togglePlay}
              className={`p-2 rounded-lg cursor-pointer active:scale-95 transition-all flex items-center ${
                isPlaying 
                  ? "bg-secondary text-black shadow-lg"
                  : "bg-primary text-black"
              }`}
            >
              {isPlaying ? <Pause className="w-5 h-5 fill-current" /> : <Play className="w-5 h-5 fill-current" />}
            </button>
          </div>

          <div className="flex items-center gap-md">
            {/* Realtime Tempo Knob input */}
            <div className="flex flex-col select-none flex-shrink-0">
              <span className="text-[10px] text-on-surface-variant uppercase tracking-widest font-semibold">Tempo</span>
              <div className="flex items-center gap-1">
                <input
                  type="number"
                  min="50"
                  max="240"
                  value={tempo}
                  onChange={(e) => setTempo(Math.max(50, Math.min(240, parseInt(e.target.value) || 120)))}
                  className="bg-transparent border-none text-center font-bold text-headline-lg-mobile text-primary focus:ring-0 p-0 w-14 font-mono focus:outline-none"
                />
                <span className="text-[10px] text-on-surface-variant">BPM</span>
              </div>
            </div>
            <div className="h-8 w-[1px] bg-white/10 mx-1" />
            
            {/* Time signature label */}
            <div className="flex flex-col select-none">
              <span className="text-[10px] text-on-surface-variant uppercase tracking-widest font-semibold">Time Sig</span>
              <div className="flex items-center">
                <span className="text-headline-lg-mobile font-mono font-bold text-primary">4/4</span>
              </div>
            </div>
          </div>
        </div>

        <div className="text-xs font-medium text-on-surface-variant">
          Click grid cells to <span className="text-primary font-bold">Draw</span> or <span className="text-secondary font-bold">Delete</span> MIDI notes.
        </div>
      </div>

      <div className="flex flex-grow overflow-hidden">
        {/* Sidebar Controls panel */}
        <aside className="w-96 min-w-[24rem] bg-surface-container-lowest border-r border-white/10 overflow-hidden p-lg flex flex-col gap-lg select-none">
          {/* Tone instrument matrix */}
          <div className="flex-shrink-0">
            <h3 className="text-[10px] text-on-surface-variant uppercase tracking-widest font-semibold gap-1.5 flex items-center mb-sm">
              <span>Instrument</span>
            </h3>
            <div className="flex flex-col gap-sm">
              {[
                { type: "piano", label: "Grand Piano", desc: "Triangle core string hammers", icon: "piano" },
                { type: "synth", label: "Analog Synth", desc: "Acid resonant cutoff sweep", icon: "settings_input_component" },
                { type: "strings", label: "Cinematic Strings", desc: "De-tuned legato violin sweep", icon: "piano" }
              ].map((inst) => (
                <div
                  key={inst.type}
                  onClick={() => setInstrument(inst.type as InstrumentType)}
                  className={`flex items-center justify-between p-md rounded-xl cursor-pointer transition-all border ${
                    instrument === inst.type
                      ? "bg-primary/15 border-primary text-primary active-neon-glow"
                      : "glass-panel hover:bg-white/5 border-transparent text-on-surface-variant"
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <span className="text-lg">{(() => { switch(inst.icon) { case "Piano": return <Piano className="w-5 h-5" />; case "Sliders": return <Sliders className="w-5 h-5" />; case "Music": return <Music className="w-5 h-5" />; default: return <Piano className="w-5 h-5" />; } })()}</span>
                    <div className="flex flex-col text-left">
                      <span className="font-bold text-sm tracking-wide">{inst.label}</span>
                      <span className="text-[9px] opacity-60 font-mono">{inst.desc}</span>
                    </div>
                  </div>
                  {instrument === inst.type && <CheckCircle className="w-4 h-4 text-primary" />}
                </div>
              ))}
            </div>
          </div>

          {/* Scale Setup card overlay */}
          <div className="glass-panel p-md rounded-xl border border-white/5 flex flex-col gap-md">
            <h3 className="text-[10px] text-on-surface-variant uppercase tracking-widest font-semibold flex items-center gap-1.5">
              <Scale className="w-3.5 h-3.5" />
              <span>Key & Scale Selection</span>
            </h3>
            <div className="grid grid-cols-2 gap-sm">
              <div className="flex flex-col flex-shrink-0">
                <label className="text-[9px] text-on-surface-variant uppercase font-mono mb-1">Root Key</label>
                <select
                  value={rootKey}
                  onChange={(e) => setRootKey(e.target.value)}
                  className="bg-surface-container-high border border-white/10 rounded-lg py-1 px-2 font-mono text-on-surface text-xs focus:ring-1 focus:ring-primary focus:border-primary focus:outline-none"
                >
                  {["C", "C#", "D", "Eb", "E", "F", "F#", "G", "Ab", "A", "Bb", "B"].map((key) => (
                    <option key={key} value={key}>{key}</option>
                  ))}
                </select>
              </div>

              <div className="flex flex-col">
                <label className="text-[9px] text-on-surface-variant uppercase font-mono mb-1">Scale Scale</label>
                <select
                  value={scaleType}
                  onChange={(e) => setScaleType(e.target.value)}
                  className="bg-surface-container-high border border-white/10 rounded-lg py-1 px-2 font-mono text-on-surface text-xs focus:ring-1 focus:ring-primary focus:border-primary focus:outline-none"
                >
                  {["Major", "Minor", "Phrygian", "Dorian"].map((scale) => (
                    <option key={scale} value={scale}>{scale}</option>
                  ))}
                </select>
              </div>
            </div>
          </div>

          {/* Core Velocity Tracker Slider */}
          <div>
            <div className="flex justify-between items-center mb-2">
              <h3 className="text-[10px] text-on-surface-variant uppercase tracking-widest font-semibold flex items-center gap-1.5">
                <Volume2 className="w-3.5 h-3.5" />
                <span>Global Velocity</span>
              </h3>
              <span className="text-xs text-secondary font-bold">{globalVelocity}</span>
            </div>
            <input
              type="range"
              min="1"
              max="127"
              value={globalVelocity}
              onChange={(e) => setGlobalVelocity(parseInt(e.target.value))}
              className="w-full h-1.5 bg-surface-container-high rounded-lg appearance-none cursor-pointer accent-secondary"
            />
            <div className="flex justify-between mt-1 text-[9px] text-on-surface-variant font-mono uppercase tracking-wider">
              <span>Soft</span>
              <span>Accented</span>
            </div>
          </div>

          {/* Visualizing spectrum simulation */}
          <div className="flex-grow flex flex-col justify-end">
            <div className="h-28 rounded-xl bg-surface-container relative overflow-hidden flex items-end justify-center gap-1 p-2">
              <div className="w-2.5 bg-primary rounded-t-full transition-all duration-300" style={{ height: isPlaying ? "45%" : "12%" }} />
              <div className="w-2.5 bg-primary/80 rounded-t-full transition-all duration-300" style={{ height: isPlaying ? "75%" : "18%" }} />
              <div className="w-2.5 bg-secondary rounded-t-full transition-all duration-200" style={{ height: isPlaying ? "90%" : "8%" }} />
              <div className="w-2.5 bg-secondary/80 rounded-t-full transition-all duration-400" style={{ height: isPlaying ? "50%" : "25%" }} />
              <div className="w-2.5 bg-primary rounded-t-full transition-all duration-300" style={{ height: isPlaying ? "35%" : "10%" }} />
              <div className="w-2.5 bg-primary/60 rounded-t-full transition-all duration-150" style={{ height: isPlaying ? "85%" : "15%" }} />
              <div className="w-2.5 bg-secondary/60 rounded-t-full transition-all duration-300" style={{ height: isPlaying ? "60%" : "6%" }} />
              <div className="absolute inset-0 bg-gradient-to-t from-surface-container via-transparent to-transparent pointer-events-none" />
            </div>
            <span className="text-center text-[10px] text-on-surface-variant uppercase tracking-widest font-semibold mt-1 flex items-center justify-center gap-1">
              <Waves className="w-3 h-3 text-secondary" />
              Real-time Spectrum
            </span>
          </div>
        </aside>

        {/* Piano Roll Area Grid */}
        <section className="flex-grow flex overflow-hidden relative bg-[#0a0a0a]">
          {/* Vertical C3-C5 Piano Keys Controller Column */}
          <div ref={pianoKeysRef} onScroll={handleKeysScroll} className="w-16 bg-surface-container-low flex flex-col border-r border-white/10 overflow-y-auto scrollbar-none flex-shrink-0" id="piano-keys">
            {PIANO_ROWS.map((pitch) => (
              <div
                key={pitch}
                onClick={() => {
                  // Direct tap auditory preview
                  const ctx = getAudioContext();
                  playNote(ctx, pitch, 0.35, instrument, globalVelocity);
                }}
                className={`h-[40px] flex-shrink-0 border-b border-white/5 flex items-center justify-end px-2 group hover:bg-white/10 transition-all select-none cursor-pointer ${
                  isBlackKey(pitch) ? "bg-black/40 text-on-surface" : "bg-white/5 text-on-surface"
                }`}
              >
                <span className="text-[9px] text-on-surface-variant font-mono font-medium opacity-40 group-hover:opacity-100 transition-opacity">
                  {pitch}
                </span>
              </div>
            ))}
          </div>

          {/* Playhead & Grid scroll container */}
          <div ref={pianoCanvasRef} onScroll={handleGridScroll} className="flex-grow overflow-auto relative min-h-0" style={{backgroundSize: `${COLUMN_WIDTH}px ${ROW_HEIGHT}px`, backgroundImage: `linear-gradient(to right, rgba(255,255,255,0.04) 1px, transparent 1px), linear-gradient(to bottom, rgba(255,255,255,0.04) 1px, transparent 1px)`}} id="piano-roll-canvas">
            
            {/* Playhead Line Indicator Component */}
            <div
              ref={playheadRef} className="absolute top-0 bottom-0 w-[2px] bg-secondary z-30 shadow-[0_0_12px_rgba(93,230,255,0.8)] pointer-events-none "
            />

            {/* Note events coordinate render layer */}
            <div className="relative" style={{ height: `${PIANO_ROWS.length * ROW_HEIGHT}px`, width: `${GRID_COLUMNS * COLUMN_WIDTH}px` }}>
              
              {/* Grid cell matrix trigger elements */}
              {PIANO_ROWS.map((pitch, rowIdx) => {
                const topVal = rowIdx * ROW_HEIGHT;
                return Array.from({ length: GRID_COLUMNS }).map((_, colIdx) => {
                  const leftVal = colIdx * COLUMN_WIDTH;
                  return (
                    <div
                      key={`grid-${pitch}-${colIdx}`}
                      onClick={() => handleGridClick(pitch, colIdx)}
                      className="absolute border-r border-b border-white/5 cursor-pointer hover:bg-white/5 transition-colors"
                      style={{
                        top: `${topVal}px`,
                        left: `${leftVal}px`,
                        width: `${COLUMN_WIDTH}px`,
                        height: `${ROW_HEIGHT}px`
                      }}
                    />
                  );
                });
              })}

              {/* Render loaded note rectangular blocks */}
              {notes.map((note) => {
                const rowIdx = PIANO_ROWS.indexOf(note.pitch);
                if (rowIdx === -1) return null; // out of visible register range

                const topVal = rowIdx * ROW_HEIGHT + 4; // micro offset
                const leftVal = (note.time / 0.5) * COLUMN_WIDTH + 1;
                const widthVal = (note.duration / 0.5) * COLUMN_WIDTH - 2;

                const isBlack = isBlackKey(note.pitch);

                return (
                  <div
                    key={note.id}
                    onClick={(e) => {
                      e.stopPropagation(); // don't redraw
                      // Delete note instantly of clicking on it
                      setNotes(notes.filter((n) => n.id !== note.id));
                    }}
                    style={{
                      top: `${topVal}px`,
                      left: `${leftVal}px`,
                      height: `${ROW_HEIGHT - 8}px`,
                      width: `${widthVal}px`
                    }}
                    className={`absolute rounded-sm z-20 cursor-pointer flex items-center justify-between px-2 font-mono font-bold hover:brightness-125 select-none transition-all duration-200 border md:text-[8px] text-[7px] text-black ${
                      instrument === "synth"
                        ? "bg-gradient-to-r from-secondary to-secondary/70 border-secondary/50 shadow-[0_0_12px_rgba(93,230,255,0.4)]"
                        : instrument === "strings"
                        ? "bg-gradient-to-r from-tertiary to-tertiary/70 border-tertiary/50 shadow-[0_0_12px_rgba(255,175,211,0.4)]"
                        : "bg-gradient-to-r from-primary to-primary/70 border-primary/50 shadow-[0_0_12px_rgba(208,188,255,0.4)]"
                    }`}
                  >
                    <span>{note.pitch}</span>
                    <span className="opacity-60">{note.velocity}</span>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Floating Zoom Action Tools */}
          <div className="absolute bottom-lg right-lg flex flex-col gap-sm z-40">
            <button
              onClick={zoomIn}
              title="Zoom in columns"
              className="w-10 h-10 glass-panel flex items-center justify-center rounded-lg hover:bg-white/10 hover:text-white transition-all cursor-pointer active:scale-90 text-on-surface-variant"
            >
              <Plus className="w-5 h-5" />
            </button>
            <button
              onClick={zoomOut}
              title="Zoom out columns"
              className="w-10 h-10 glass-panel flex items-center justify-center rounded-lg hover:bg-white/10 hover:text-white transition-all cursor-pointer active:scale-90 text-on-surface-variant"
            >
              <Minus className="w-5 h-5" />
            </button>
          </div>
        </section>
      </div>
    </div>
  );
}
