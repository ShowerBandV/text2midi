import React, { useState, useEffect, useRef } from "react";
import { MidiTrack, InstrumentType } from "../types";
import { generateMidiFileBlobUrl, playNote } from "../utils/audio";
import {
  Download,
  Share2,
  Copy,
  FolderHeart,
  FileMusic,
  Maximize2,
  Play,
  Pause,
  Clock,
  Music,
  Gauge,
  Sliders,
  Check,
  Disc2
} from "lucide-react";

interface LibraryProps {
  tracks: MidiTrack[];
  setTracks: (tracks: MidiTrack[]) => void;
  activeTrackId: string | null;
  setActiveTrackId: (id: string | null) => void;
  onSelectTrackForEditor: (track: MidiTrack) => void;
}

export default function Library({
  tracks,
  setTracks,
  activeTrackId,
  setActiveTrackId,
  onSelectTrackForEditor
}: LibraryProps) {
  const activeTrack = tracks.find(t => t.id === activeTrackId) || tracks[0];
  const [copied, setCopied] = useState(false);
  const [liked, setLiked] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTimeStr, setCurrentTimeStr] = useState("00:00");
  
  const audioCtxRef = useRef<AudioContext | null>(null);
  const playTimerRef = useRef<number | null>(null);
  const playheadRef = useRef<number>(0);

  const triggerSearchRef = useRef<Set<string>>(new Set());

  // Copy link action callback
  const handleCopyLink = () => {
    navigator.clipboard.writeText(window.location.href);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // MIDI File download generator trigger
  const handleDownloadMidi = () => {
    if (!activeTrack) return;
    const downloadUrl = generateMidiFileBlobUrl(activeTrack.notes, activeTrack.metadata.tempo);
    const link = document.createElement("a");
    link.href = downloadUrl;
    link.download = `${activeTrack.metadata.title}.mid`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const lazyGetAudioContext = (): AudioContext => {
    if (!audioCtxRef.current) {
      audioCtxRef.current = new (window.AudioContext || (window as any).webkitAudioContext)();
    }
    if (audioCtxRef.current.state === "suspended") {
      audioCtxRef.current.resume();
    }
    return audioCtxRef.current;
  };

  // Preview composition sound toggle
  useEffect(() => {
    if (isPlaying && activeTrack) {
      const tempo = activeTrack.metadata.tempo;
      const notes = activeTrack.notes;
      const instrument = activeTrack.instrument;
      const globalVelocity = activeTrack.globalVelocity;

      const startTime = performance.now() / 1000 - playheadRef.current;
      const ctx = lazyGetAudioContext();

      const tick = () => {
        const now = performance.now() / 1000;
        const elapsed = now - startTime;
        const currentBeat = (elapsed * (tempo / 60)) % 16;
        
        // Update timer labels
        const minutes = Math.floor(elapsed / 60).toString().padStart(2, "0");
        const seconds = Math.floor(elapsed % 60).toString().padStart(2, "0");
        setCurrentTimeStr(`${minutes}:${seconds}`);

        // trigger note audio
        notes.forEach((note) => {
          if (
            note.time <= currentBeat &&
            note.time > currentBeat - 0.1 &&
            !triggerSearchRef.current.has(note.id)
          ) {
            triggerSearchRef.current.add(note.id);
            const durationSec = note.duration * (60 / tempo);
            playNote(ctx, note.pitch, durationSec, instrument, Math.min(note.velocity, globalVelocity));
          }
        });

        // Reset if we loop around the 16 beat sequence
        if (currentBeat < 0.1) {
          triggerSearchRef.current.clear();
        }

        playTimerRef.current = requestAnimationFrame(tick);
      };

      playTimerRef.current = requestAnimationFrame(tick);
    } else {
      if (playTimerRef.current) {
        cancelAnimationFrame(playTimerRef.current);
      }
    }

    return () => {
      if (playTimerRef.current) {
        cancelAnimationFrame(playTimerRef.current);
      }
    };
  }, [isPlaying, activeTrackId]);

  const togglePreviewPlay = () => {
    lazyGetAudioContext();
    if (isPlaying) {
      setIsPlaying(false);
    } else {
      triggerSearchRef.current.clear();
      setIsPlaying(true);
    }
  };

  if (!activeTrack) {
    return (
      <div className="flex flex-col items-center justify-center p-xl text-on-surface-variant font-mono">
        <Sliders className="w-12 h-12 text-primary animate-bounce mb-sm" />
        No tracks generated or saved. Return to Generate tab to create MIDI.
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-12 gap-xl select-none select-none">
      
      {/* Left Column: Tapedeck tape Cassette & meta */}
      <div className="md:col-span-7 flex flex-col gap-lg">
        <div className="glass-panel rounded-xl p-lg relative overflow-hidden group border border-white/5 shadow-2xl">
          {/* Neon atmospheric orb backgrounds */}
          <div className="absolute -top-24 -right-24 w-64 h-64 bg-primary/10 blur-[80px] rounded-full pointer-events-none" />
          
          <div className="relative z-10 flex flex-col md:flex-row gap-lg items-center">
            
            {/* Visual Tape Deco cassette block */}
            <div className="w-full md:w-64 h-84 bg-surface-container-high rounded-xl p-md border border-white/10 shadow-2xl flex flex-col transform transition-transform duration-500 group-hover:rotate-1">
              <div className="flex-1 rounded p-md flex flex-col justify-between border border-white/5 bg-black/55 relative overflow-hidden">
                <div className="flex justify-between items-start">
                  <span className="text-[10px] text-secondary font-bold tracking-widest bg-secondary/10 px-2 py-0.5 rounded-full">A-SIDE</span>
                  <Disc2 className={`w-5 h-5 text-primary ${isPlaying ? "animate-spin" : ""}`} style={{ animationDuration: "1.5s" }} />
                </div>
                
                {/* Title & Metadata details */}
                <div className="flex flex-col gap-xs mt-auto relative z-10">
                  <h2 className="font-display font-extrabold text-headline-lg-mobile text-primary leading-tight truncate">
                    {activeTrack.metadata.title}
                  </h2>
                  <p className="text-[10px] text-on-surface-variant tracking-wider uppercase font-medium">
                    SEED: {activeTrack.metadata.seed}
                  </p>
                </div>
                <div className="absolute inset-0 bg-[radial-gradient(circle_at_bottom_left,rgba(208,188,255,0.08),transparent_50%)]" />
              </div>

              {/* Tape Reels window */}
              <div className="h-24 flex items-center justify-center relative bg-surface-container-low rounded-b-lg border-t border-white/5 overflow-hidden">
                <div className="flex gap-lg opacity-60">
                  <div
                    className={`w-16 h-16 rounded-full border-2 border-dashed border-primary flex items-center justify-center ${isPlaying ? "animate-spin" : ""}`}
                    style={{ animationDuration: "6s" }}
                  >
                    <div className="w-8 h-8 rounded-full border border-white/15 bg-black/40" />
                  </div>
                  <div
                    className={`w-16 h-16 rounded-full border-2 border-dashed border-secondary flex items-center justify-center ${isPlaying ? "animate-spin" : ""}`}
                    style={{ animationDuration: "9s" }}
                  >
                    <div className="w-8 h-8 rounded-full border border-white/15 bg-black/40" />
                  </div>
                </div>
                <div className="absolute inset-x-0 h-4 bg-gradient-to-r from-transparent via-primary/5 to-transparent pointer-events-none" />
              </div>
            </div>

            {/* List key metadata blocks */}
            <div className="flex-1 flex flex-col gap-md w-full">
              <div className="space-y-sm">
                <span className="text-[10px] text-secondary uppercase tracking-widest font-bold">Sequence Metadata</span>
                
                <div className="grid grid-cols-2 gap-md">
                  <div className="bg-white/5 p-md rounded-lg border border-white/5 hover:bg-white/10 transition-colors">
                    <p className="text-[10px] text-on-surface-variant uppercase flex items-center gap-1">
                      <Clock className="w-3 h-3 text-secondary" /> Duration
                    </p>
                    <p className="font-display font-extrabold text-headline-lg-mobile text-on-surface">
                      {activeTrack.metadata.durationStr}
                    </p>
                  </div>
                  <div className="bg-white/5 p-md rounded-lg border border-white/5 hover:bg-white/10 transition-colors">
                    <p className="text-[10px] text-on-surface-variant uppercase flex items-center gap-1">
                      <Music className="w-3 h-3 text-primary" /> Key
                    </p>
                    <p className="font-display font-extrabold text-headline-lg-mobile text-on-surface">
                      {activeTrack.metadata.key} {activeTrack.metadata.scale}
                    </p>
                  </div>
                  <div className="bg-white/5 p-md rounded-lg border border-white/5 hover:bg-white/10 transition-colors">
                    <p className="text-[10px] text-on-surface-variant uppercase flex items-center gap-1">
                      <Gauge className="w-3 h-3 text-primary" /> Tempo
                    </p>
                    <p className="font-display font-extrabold text-headline-lg-mobile text-on-surface font-mono">
                      {activeTrack.metadata.tempo} <span className="text-xs font-normal opacity-60">BPM</span>
                    </p>
                  </div>
                  <div className="bg-white/5 p-md rounded-lg border border-white/5 hover:bg-white/10 transition-colors">
                    <p className="text-[10px] text-on-surface-variant uppercase flex items-center gap-1">
                      <Sliders className="w-3 h-3 text-secondary" /> Complexity
                    </p>
                    <p className="font-display font-extrabold text-headline-lg-mobile text-on-surface">
                      {activeTrack.metadata.complexity}
                    </p>
                  </div>
                </div>
              </div>

              {/* Pill Badge tags */}
              <div className="flex flex-wrap gap-2 items-center mt-sm">
                <div className="px-md py-1 bg-primary/15 rounded-full border border-primary/30 text-primary text-xs font-semibold">
                  {activeTrack.metadata.genre || "Cybernetic Jazz"}
                </div>
                <div className="px-md py-1 bg-secondary/15 rounded-full border border-secondary/30 text-secondary text-xs font-semibold">
                  Polyphonic
                </div>
              </div>
            </div>

          </div>
        </div>

        {/* Load track directly into Editor workspace CTA */}
        <div className="glass-panel p-lg rounded-xl flex items-center justify-between gap-xl">
          <div className="flex flex-col text-left">
            <h4 className="font-bold text-base text-white tracking-wide">Enter Composition Studio</h4>
            <p className="text-xs text-on-surface-variant mt-1 leading-relaxed">
              Open this MIDI track inside the Piano Roll editor to adjust musical scales, tempo steps, drag notes, or sequence new voices.
            </p>
          </div>
          <button 
            onClick={() => onSelectTrackForEditor(activeTrack)}
            className="flex-shrink-0 bg-white/5 hover:bg-white/10 text-white font-mono hover:text-primary border border-white/10 font-bold px-lg py-md rounded-lg transition-all active:scale-95 flex items-center gap-2 cursor-pointer text-xs uppercase tracking-wider"
          >
            <Maximize2 className="w-4 h-4" /> Load in Editor
          </button>
        </div>
      </div>

      {/* Right Column: Export, Save, and Social Sharing controls */}
      <div className="md:col-span-5 flex flex-col gap-lg">
        
        {/* Core Export Panel */}
        <div className="glass-panel rounded-xl p-lg flex flex-col gap-lg border border-white/5 shadow-2xl neon-glow">
          <h3 className="font-display font-semibold text-headline-lg text-on-surface select-none leading-none">Export &amp; Save</h3>
          
          <button
            onClick={handleDownloadMidi}
            className="group flex items-center justify-between bg-gradient-to-r from-primary to-secondary p-lg rounded-xl transition-all hover:scale-[1.02] active:scale-95 shadow-xl select-none cursor-pointer text-black"
          >
            <div className="flex flex-col items-start text-left">
              <span className="text-[10px] text-black/70 font-extrabold uppercase">Primary action</span>
              <span className="font-display font-extrabold text-headline-lg-mobile">Download .mid</span>
            </div>
            <Download className="w-8 h-8 group-hover:translate-y-1 transition-transform" />
          </button>

          {/* Secondary exporting functions */}
          <div className="flex flex-col gap-sm">
            <button className="flex items-center gap-md p-md rounded-xl border border-white/10 hover:bg-white/5 hover:border-secondary-container transition-all active:scale-95 text-left w-full cursor-pointer">
              <div className="w-11 h-11 bg-secondary-container/10 rounded-lg flex items-center justify-center text-secondary border border-secondary/20">
                <FileMusic className="w-5 h-5 animate-pulse" />
              </div>
              <div className="flex-grow select-none">
                <p className="font-bold text-sm text-on-surface">Export to WAV</p>
                <p className="text-[10px] text-on-surface-variant">High fildelity lossless offline wav render</p>
              </div>
            </button>

            <button className="flex items-center gap-md p-md rounded-xl border border-white/10 hover:bg-white/5 hover:border-primary-container transition-all active:scale-95 text-left w-full cursor-pointer">
              <div className="w-11 h-11 bg-primary-container/10 rounded-lg flex items-center justify-center text-primary border border-primary/20">
                <FolderHeart className="w-5 h-5" />
              </div>
              <div className="flex-grow select-none">
                <p className="font-bold text-sm text-on-surface">Save Archive</p>
                <p className="text-[10px] text-on-surface-variant">Store permanently within local MIDI library</p>
              </div>
            </button>
          </div>
        </div>

        {/* Share Creation Box */}
        <div className="glass-panel rounded-xl p-lg border border-white/5 shadow-2xl flex flex-col gap-sm">
          <div className="flex items-center justify-between mb-sm">
            <span className="text-[10px] text-on-surface-variant uppercase tracking-widest font-bold">Share creation</span>
            
            <div className="flex gap-2">
              <button 
                onClick={handleCopyLink}
                className="w-9 h-9 rounded-lg bg-surface-container-high border border-white/10 flex items-center justify-center hover:text-secondary hover:border-secondary transition-all active:scale-90 cursor-pointer"
                title="Copy share link"
              >
                {copied ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
              </button>
              <button 
                className="w-9 h-9 rounded-lg bg-surface-container-high border border-white/10 flex items-center justify-center hover:text-secondary hover:border-secondary transition-all active:scale-90 cursor-pointer"
                title="Share overlay"
              >
                <Share2 className="w-4 h-4" />
              </button>
            </div>
          </div>

          <div className="flex gap-md overflow-x-auto pb-1 mt-1">
            <div className="flex-shrink-0 w-11 h-11 rounded-full bg-blue-500/10 border border-blue-500/30 flex items-center justify-center cursor-pointer hover:bg-blue-500/25 text-blue-400 transition-colors active:scale-90">
              <span className="text-xs font-bold leading-none">@</span>
            </div>
            <div 
              onClick={() => setLiked(!liked)}
              className={`flex-shrink-0 w-11 h-11 rounded-full border flex items-center justify-center cursor-pointer transition-colors active:scale-90 ${
                liked 
                  ? "bg-pink-500/25 border-pink-500 text-pink-400 animate-pulse" 
                  : "bg-pink-500/10 border-pink-500/30 text-pink-400 hover:bg-pink-500/20"
              }`}
            >
              <span className="transform translate-y-[1px]">♥</span>
            </div>
            <div className="flex-shrink-0 w-11 h-11 rounded-full bg-white/5 border border-white/15 flex items-center justify-center cursor-pointer hover:bg-white/15 text-white transition-colors active:scale-90">
              <span className="text-xs font-mono font-bold leading-none">AI</span>
            </div>
          </div>
        </div>

      </div>

      {/* Bottom Composition waveforms simulated previews */}
      <section className="col-span-12 mt-md">
        <div className="glass-panel rounded-xl p-lg relative border border-white/5 shadow-2xl">
          <div className="flex justify-between items-center mb-md">
            <div className="flex items-center gap-md">
              <button
                onClick={togglePreviewPlay}
                className="w-12 h-12 rounded-full bg-primary text-black flex items-center justify-center shadow-lg hover:scale-110 active:scale-95 transition-transform cursor-pointer"
              >
                {isPlaying ? <Pause className="w-5 h-5 fill-current" /> : <Play className="w-5 h-5 fill-current ml-0.5" />}
              </button>
              <div className="text-left select-none">
                <h4 className="font-bold text-sm text-white">Preview AI Symphony</h4>
                <p className="text-[10px] text-on-surface-variant uppercase tracking-wider">Low-latency physical sound render</p>
              </div>
            </div>

            <div className="flex items-center gap-md text-[10px] text-on-surface-variant font-bold select-none">
              <span>{currentTimeStr} / 00:16</span>
              <div className="w-32 h-1 bg-white/10 rounded-full overflow-hidden">
                <div 
                  className="h-full bg-secondary transition-all"
                  style={{ width: isPlaying ? "65%" : "15%" }}
                />
              </div>
            </div>
          </div>

          {/* simulated visualizer bar arrays, dynamic size animation */}
          <div className="h-24 flex items-end justify-between gap-[2px] overflow-hidden">
            {Array.from({ length: 70 }).map((_, i) => {
              // Procedural heights randomized
              const randomHeight = isPlaying 
                ? `${Math.floor(Math.sin((i + Date.now()/1000) * 0.4) * 40) + 50}%` 
                : `${(Math.sin(i * 0.1) * 20 + 35)}%`;
              return (
                <div
                  key={`wave-${i}`}
                  style={{ height: randomHeight }}
                  className="flex-1 bg-gradient-to-t from-primary to-secondary/35 rounded-t-sm transition-all duration-300 opacity-60 hover:opacity-100"
                />
              );
            })}
          </div>
        </div>
      </section>

    </div>
  );
}
