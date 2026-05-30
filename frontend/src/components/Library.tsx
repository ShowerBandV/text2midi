import React, { useState, useEffect, useRef } from "react";
import { MidiTrack, InstrumentType } from "../types";
import { generateMidiFileBlobUrl, playNote } from "../utils/audio";
import { getDownloadUrl } from "../utils/api";
import {
  Download,
  Copy,
  FolderHeart,
  FileMusic,
  Play,
  Pause,
  Clock,
  Music,
  Gauge,
  Sliders,
  Check,
  Disc2,
  Server,
  Coffee,
  Trash2
} from "lucide-react";

interface LibraryProps {
  tracks: MidiTrack[];
  setTracks: (tracks: MidiTrack[]) => void;
  activeTrackId: string | null;
  setActiveTrackId: (id: string | null) => void;

}

export default function Library({
  tracks,
  setTracks,
  activeTrackId,
  setActiveTrackId,
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

  // MIDI File download generator trigger (client-side)
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

  // Server-side download (when fileId is available from backend)
  const handleDownloadFromServer = () => {
    if (!activeTrack?.fileId) return;
    const url = getDownloadUrl(activeTrack.fileId);
    const link = document.createElement("a");
    link.href = url;
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

  // ─── Empty state ──────────────────────────────────────────────────
  if (tracks.length === 0) {
    return (
      <div className="h-full flex flex-col items-center justify-center p-xl text-on-surface-variant font-mono">
        <div className="w-16 h-16 rounded-full bg-surface-container-high flex items-center justify-center mb-4">
          <Music className="w-7 h-7 text-primary" />
        </div>
        <p className="text-sm font-medium mb-1">No tracks yet</p>
        <p className="text-xs opacity-60">Go to Generate tab to create your first MIDI.</p>
      </div>
    );
  }

  if (!activeTrack) {
    // Auto-select first track
    setActiveTrackId(tracks[0].id);
    return null;
  }

  return (
    <div className="flex h-full gap-lg select-none">

      {/* ─── Left: Track List ─────────────────────────────────────── */}
      <aside className="w-72 min-w-[16rem] flex flex-col gap-3 overflow-y-auto flex-shrink-0">
        <div className="flex items-center justify-between px-1">
          <h3 className="text-xs text-on-surface-variant uppercase tracking-widest font-semibold">
            Library <span className="text-primary">({tracks.length})</span>
          </h3>
        </div>

        <div className="flex flex-col gap-2">
          {tracks.map((track) => {
            const isActive = track.id === activeTrackId;
            return (
              <button
                key={track.id}
                onClick={() => {
                  setActiveTrackId(track.id);
                  setIsPlaying(false);
                }}
                className={`w-full text-left rounded-xl p-3 border transition-all cursor-pointer ${
                  isActive
                    ? 'bg-primary/10 border-primary/40 shadow-lg'
                    : 'bg-surface-container-low border-white/5 hover:bg-white/5 hover:border-white/20'
                }`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0 flex-1">
                    <p className={`text-sm font-bold truncate ${isActive ? 'text-primary' : 'text-white'}`}>
                      {track.metadata.title}
                    </p>
                    <p className="text-[10px] text-on-surface-variant font-mono mt-1">
                      {track.metadata.tempo} BPM · {track.metadata.key} {track.metadata.scale}
                    </p>
                  </div>
                  <Disc2 className={`w-4 h-4 flex-shrink-0 mt-0.5 ${isActive ? 'text-secondary' : 'text-on-surface-variant/30'}`} />
                </div>
                <div className="flex items-center gap-2 mt-2">
                  <span className="text-[9px] px-2 py-0.5 rounded-full bg-white/5 text-on-surface-variant font-mono">
                    {track.metadata.durationStr}
                  </span>
                  <span className="text-[9px] px-2 py-0.5 rounded-full bg-white/5 text-on-surface-variant font-mono">
                    {track.instrument}
                  </span>
                </div>
                <p className="text-[9px] text-on-surface-variant/50 font-mono mt-1.5">
                  {new Date(track.createdAt).toLocaleDateString()}
                </p>
              </button>
            );
          })}
        </div>
      </aside>

      {/* ─── Right: Detail View ────────────────────────────────────── */}
      <div className="flex-1 min-w-0 flex flex-col gap-lg overflow-y-auto">

        {/* Track info + metadata */}
        <div className="glass-panel rounded-xl p-lg relative overflow-hidden group border border-white/5 shadow-2xl">
          <div className="absolute -top-24 -right-24 w-64 h-64 bg-primary/10 blur-[80px] rounded-full pointer-events-none" />
          
          <div className="relative z-10 flex flex-col md:flex-row gap-lg items-start">
            
            {/* Mini cassette visual */}
            <div className="w-full md:w-56 shrink-0 bg-surface-container-high rounded-xl p-3 border border-white/10 shadow-lg">
              <div className="rounded-lg p-3 flex flex-col justify-between border border-white/5 bg-black/55 relative overflow-hidden min-h-[7rem]">
                <div className="flex justify-between items-start">
                  <span className="text-[9px] text-secondary font-bold tracking-widest bg-secondary/10 px-1.5 py-0.5 rounded-full">A-SIDE</span>
                  <Disc2 className={`w-4 h-4 text-primary ${isPlaying ? 'animate-spin' : ''}`} style={{ animationDuration: '1.5s' }} />
                </div>
                <div className="mt-auto">
                  <h2 className="font-display font-extrabold text-lg text-primary leading-tight truncate">
                    {activeTrack.metadata.title}
                  </h2>
                  <p className="text-[9px] text-on-surface-variant tracking-wider uppercase font-medium mt-0.5">
                    SEED: {activeTrack.metadata.seed}
                  </p>
                </div>
                <div className="absolute inset-0 bg-[radial-gradient(circle_at_bottom_left,rgba(208,188,255,0.08),transparent_50%)]" />
              </div>
              <div className="h-14 flex items-center justify-center bg-surface-container-low rounded-b-lg border-t border-white/5 overflow-hidden">
                <div className="flex gap-3 opacity-40">
                  <div className={`w-9 h-9 rounded-full border-2 border-dashed border-primary flex items-center justify-center ${isPlaying ? 'animate-spin' : ''}`} style={{ animationDuration: '6s' }}>
                    <div className="w-4 h-4 rounded-full border border-white/15 bg-black/40" />
                  </div>
                  <div className={`w-9 h-9 rounded-full border-2 border-dashed border-secondary flex items-center justify-center ${isPlaying ? 'animate-spin' : ''}`} style={{ animationDuration: '9s' }}>
                    <div className="w-4 h-4 rounded-full border border-white/15 bg-black/40" />
                  </div>
                </div>
              </div>
            </div>

            {/* Metadata grid */}
            <div className="flex-1 grid grid-cols-2 gap-3 w-full">
              {[
                { icon: Clock, label: 'Duration', value: activeTrack.metadata.durationStr, color: 'text-secondary' },
                { icon: Music, label: 'Key', value: `${activeTrack.metadata.key} ${activeTrack.metadata.scale}`, color: 'text-primary' },
                { icon: Gauge, label: 'Tempo', value: `${activeTrack.metadata.tempo} BPM`, color: 'text-primary' },
                { icon: Sliders, label: 'Complexity', value: activeTrack.metadata.complexity, color: 'text-secondary' },
              ].map((item) => (
                <div key={item.label} className="bg-white/5 p-3 rounded-lg border border-white/5 hover:bg-white/10 transition-colors">
                  <p className="text-[9px] text-on-surface-variant uppercase flex items-center gap-1 mb-1">
                    <item.icon className={`w-3 h-3 ${item.color}`} /> {item.label}
                  </p>
                  <p className="font-display font-extrabold text-base text-on-surface">
                    {item.value}
                  </p>
                </div>
              ))}
              <div className="col-span-2 flex flex-wrap gap-2 mt-1">
                <span className="px-3 py-1 bg-primary/15 rounded-full border border-primary/30 text-primary text-xs font-semibold">
                  {activeTrack.metadata.genre || 'Generated'}
                </span>
                <span className="px-3 py-1 bg-secondary/15 rounded-full border border-secondary/30 text-secondary text-xs font-semibold">
                  {activeTrack.instrument}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Export + Share row */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-lg">
          {/* Export */}
          <div className="glass-panel rounded-xl p-lg flex flex-col gap-3 border border-white/5 shadow-lg">
            <h3 className="font-display font-semibold text-base text-on-surface">Export</h3>
            <button
              onClick={activeTrack?.fileId ? handleDownloadFromServer : handleDownloadMidi}
              className="flex items-center justify-between bg-gradient-to-r from-primary to-secondary p-4 rounded-xl hover:brightness-110 active:scale-[0.98] transition-all cursor-pointer text-black"
            >
              <div>
                <span className="text-[9px] text-black/70 font-extrabold uppercase block">
                  {activeTrack?.fileId ? 'Download from server' : 'Download locally'}
                </span>
                <span className="font-display font-extrabold text-base">Download .mid</span>
              </div>
              <Download className="w-6 h-6" />
            </button>
            {activeTrack?.fileId && (
              <button
                onClick={handleDownloadFromServer}
                className="flex items-center gap-3 p-3 rounded-xl border border-white/10 hover:bg-white/5 transition-all cursor-pointer text-left"
              >
                <Server className="w-4 h-4 text-secondary" />
                <div>
                  <p className="text-xs font-bold text-on-surface">From Server</p>
                  <p className="text-[9px] text-on-surface-variant">Original backend file</p>
                </div>
              </button>
            )}
          </div>

          {/* Share */}
          <div className="glass-panel rounded-xl p-lg border border-white/5 shadow-lg">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-display font-semibold text-base text-on-surface">Share</h3>
              <button
                onClick={handleCopyLink}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-container-high border border-white/10 hover:text-secondary hover:border-secondary transition-all cursor-pointer text-xs"
              >
                {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                {copied ? 'Copied!' : 'Copy Link'}
              </button>
            </div>
            <div className="flex gap-3">
              <div className="w-9 h-9 rounded-full bg-blue-500/10 border border-blue-500/30 flex items-center justify-center cursor-pointer hover:bg-blue-500/25 text-blue-400 transition-colors">
                <span className="text-xs font-bold">@</span>
              </div>
              <div
                onClick={() => setLiked(!liked)}
                className={`w-9 h-9 rounded-full border flex items-center justify-center cursor-pointer transition-colors ${
                  liked
                    ? 'bg-pink-500/25 border-pink-500 text-pink-400 animate-pulse'
                    : 'bg-pink-500/10 border-pink-500/30 text-pink-400 hover:bg-pink-500/20'
                }`}
              >
                <span className="translate-y-[0.5px]">♥</span>
              </div>
              <div className="w-9 h-9 rounded-full bg-white/5 border border-white/15 flex items-center justify-center cursor-pointer hover:bg-white/15 text-white transition-colors">
                <span className="text-xs font-mono font-bold">AI</span>
              </div>
            </div>
          </div>
        </div>

        {/* Preview player */}
        <div className="glass-panel rounded-xl p-lg border border-white/5 shadow-lg">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-3">
              <button
                onClick={togglePreviewPlay}
                className="w-10 h-10 rounded-full bg-primary text-black flex items-center justify-center shadow-lg hover:scale-110 active:scale-95 transition-transform cursor-pointer"
              >
                {isPlaying ? <Pause className="w-4 h-4 fill-current" /> : <Play className="w-4 h-4 fill-current ml-0.5" />}
              </button>
              <div>
                <p className="text-sm font-bold text-white">Preview</p>
                <p className="text-[9px] text-on-surface-variant">Browser audio render</p>
              </div>
            </div>
            <div className="flex items-center gap-3 text-[9px] text-on-surface-variant font-bold">
              <span>{currentTimeStr} / 00:16</span>
              <div className="w-24 h-1 bg-white/10 rounded-full overflow-hidden">
                <div className="h-full bg-secondary transition-all" style={{ width: isPlaying ? '65%' : '15%' }} />
              </div>
            </div>
          </div>
          <div className="h-20 flex items-end justify-between gap-[2px] overflow-hidden">
            {Array.from({ length: 60 }).map((_, i) => {
              const h = isPlaying
                ? `${Math.floor(Math.sin((i + Date.now() / 1000) * 0.4) * 40) + 50}%`
                : `${(Math.sin(i * 0.1) * 20 + 35)}%`;
              return (
                <div
                  key={i}
                  style={{ height: h }}
                  className="flex-1 bg-gradient-to-t from-primary to-secondary/35 rounded-t-sm transition-all duration-300 opacity-60"
                />
              );
            })}
          </div>
        </div>
      </div>

      {/* ─── Buy Me a Coffee ──────────────────────────────────────── */}
      <BuyMeCoffee />

    </div>
  );
}

// ─── Buy Me a Coffee floating widget ──────────────────────────────

function BuyMeCoffee() {
  const [open, setOpen] = useState(false);

  return (
    <div className="fixed left-6 top-1/2 -translate-y-1/2 z-50 flex items-start gap-3">
      {/* Float button */}
      <button
        onClick={() => setOpen(!open)}
        className="w-11 h-11 rounded-full bg-gradient-to-r from-amber-500 to-amber-600 text-white shadow-xl hover:scale-110 active:scale-95 transition-all cursor-pointer flex items-center justify-center flex-shrink-0"
        title="Buy me a coffee"
      >
        <Coffee className="w-5 h-5" />
      </button>

      {/* QR code panel — opens to the right */}
      {open && (
        <div className="glass-panel rounded-2xl p-4 border border-white/10 shadow-2xl flex flex-col items-center gap-2">
          <div className="w-36 h-36 bg-white rounded-xl flex items-center justify-center p-2">
            <img
              src="https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=https%3A%2F%2Fwww.buymeacoffee.com%2Ftext2midi"
              alt="Buy Me a Coffee QR code"
              className="w-full h-full"
            />
          </div>
          <span className="text-xs text-on-surface-variant font-mono text-center leading-relaxed">
            Buy me a coffee ☕
          </span>
        </div>
      )}
    </div>
  );
}
