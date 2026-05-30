import React, { useState, useEffect, useRef, useCallback } from "react";
import { MidiTrack, InstrumentType, MidiNote } from "../types";
import { generateMidiFileBlobUrl, playNote } from "../utils/audio";
import { getDownloadUrl, generateMidi } from "../utils/api";
import { detectChords, notesToDNEEvents, pitchToMidi, DetectedChord } from "../utils/chords";
import * as api from "../utils/api";
import { useT } from "../i18n";
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
  Trash2,
  Guitar,
  WandSparkles,
  Dna,
  ChevronDown,
  ChevronRight,
  RefreshCw
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
  const t = useT();
  const activeTrack = tracks.find(t => t.id === activeTrackId) || tracks[0];
  const [copied, setCopied] = useState(false);
  const [liked, setLiked] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTimeStr, setCurrentTimeStr] = useState("00:00");
  
  const audioCtxRef = useRef<AudioContext | null>(null);
  const playTimerRef = useRef<number | null>(null);
  const playheadRef = useRef<number>(0);

  const triggerSearchRef = useRef<Set<string>>(new Set());

  // ─── Feature panel states ────────────────────────────────────────
  const [chords, setChords] = useState<DetectedChord[]>([]);
  const [showChords, setShowChords] = useState(false);
  const [dnaResult, setDnaResult] = useState<any>(null);
  const [showDna, setShowDna] = useState(false);
  const [regenerating, setRegenerating] = useState(false);
  const [loadingChords, setLoadingChords] = useState(false);
  const [loadingDna, setLoadingDna] = useState(false);

  const runChordDetection = useCallback(() => {
    if (chords.length > 0) { setShowChords(!showChords); return; }
    setLoadingChords(true);
    // Small delay so UI isn't janky
    setTimeout(() => {
      const detected = detectChords(activeTrack?.notes || [], 8);
      setChords(detected);
      setShowChords(true);
      setLoadingChords(false);
    }, 100);
  }, [activeTrack, chords, showChords]);

  const runDNAAnalysis = useCallback(async () => {
    if (dnaResult) { setShowDna(!showDna); return; }
    if (!activeTrack) return;
    setLoadingDna(true);
    try {
      const events = notesToDNEEvents(activeTrack.notes);
      const res = await api.extractDNA({
        events_by_track: { main: events },
        total_bars: Math.max(1, Math.ceil(
          (activeTrack.notes.reduce((max, n) => Math.max(max, n.time + n.duration), 0)) / 4
        )),
        key: `${activeTrack.metadata.key} ${activeTrack.metadata.scale}`,
      });
      setDnaResult(res);
      setShowDna(true);
    } catch (err) {
      console.error("DNA analysis failed:", err);
    } finally {
      setLoadingDna(false);
    }
  }, [activeTrack, dnaResult, showDna]);

  const handleRegenerate = async () => {
    if (!activeTrack || regenerating) return;
    setRegenerating(true);
    try {
      const res = await generateMidi({
        prompt: activeTrack.metadata.title,
        style: activeTrack.metadata.genre?.toLowerCase() || "pop",
        bpm: activeTrack.metadata.tempo,
        key: `${activeTrack.metadata.key} ${activeTrack.metadata.scale}`,
        bars: 8,
        tier: "free",
      });
      if (res.fileId) {
        const newTrack: MidiTrack = {
          id: "track-" + Date.now(),
          notes: activeTrack.notes,
          metadata: {
            title: (res.fileName || res.meta?.output_path?.split("/").pop()?.replace(".mid", "") || activeTrack.metadata.title) + " (variation)",
            seed: Math.floor(Math.random() * 9000000) + 1000000,
            tempo: activeTrack.metadata.tempo,
            key: activeTrack.metadata.key,
            scale: activeTrack.metadata.scale,
            complexity: activeTrack.metadata.complexity,
            genre: activeTrack.metadata.genre || "Generated",
            durationStr: res.durationSeconds
              ? Math.floor(res.durationSeconds / 60) + ":" + String(Math.floor(res.durationSeconds % 60)).padStart(2, "0")
              : activeTrack.metadata.durationStr,
          },
          instrument: activeTrack.instrument,
          globalVelocity: activeTrack.globalVelocity,
          createdAt: new Date().toISOString(),
          fileId: res.fileId,
        };
        setTracks([newTrack, ...tracks]);
        setActiveTrackId(newTrack.id);
      }
    } catch (err) {
      console.error("Regenerate failed:", err);
    } finally {
      setRegenerating(false);
    }
  };

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
        <p className="text-sm font-medium mb-1">{t("lib.empty")}</p>
        <p className="text-xs opacity-60">{t("lib.emptyHint")}</p>
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
            {t("lib.title")} <span className="text-primary">({tracks.length})</span>
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
      <div className="flex-1 min-w-0 flex flex-col gap-4 overflow-y-auto">

        {/* Track info + metadata */}
        <div className="glass-panel rounded-xl p-4 relative overflow-hidden group border border-white/5 shadow-lg">
          <div className="absolute -top-24 -right-24 w-64 h-64 bg-primary/10 blur-[80px] rounded-full pointer-events-none" />
          
          <div className="relative z-10 flex flex-col md:flex-row gap-4 items-start">
            
            {/* Mini cassette visual */}
            <div className="w-full md:w-48 shrink-0 bg-surface-container-high rounded-xl p-2 border border-white/10 shadow-lg">
              <div className="rounded-lg p-3 flex flex-col justify-between border border-white/5 bg-black/55 relative overflow-hidden min-h-[7rem]">
                <div className="flex justify-between items-start">
                  <span className="text-[9px] text-secondary font-bold tracking-widest bg-secondary/10 px-1.5 py-0.5 rounded-full">{t("misc.aSide")}</span>
                  <Disc2 className={`w-4 h-4 text-primary ${isPlaying ? 'animate-spin' : ''}`} style={{ animationDuration: '1.5s' }} />
                </div>
                <div className="mt-auto">
                  <h2 className="font-display font-extrabold text-lg text-primary leading-tight truncate">
                    {activeTrack.metadata.title}
                  </h2>
                  <p className="text-[9px] text-on-surface-variant tracking-wider uppercase font-medium mt-0.5">
                    {t("meta.seed")}: {activeTrack.metadata.seed}
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
            <div className="flex-1 grid grid-cols-2 gap-2 w-full">
              {[
                { icon: Clock, label: t('meta.duration'), value: activeTrack.metadata.durationStr, color: 'text-secondary' },
                { icon: Music, label: t('meta.key'), value: `${activeTrack.metadata.key} ${activeTrack.metadata.scale}`, color: 'text-primary' },
                { icon: Gauge, label: t('meta.tempo'), value: `${activeTrack.metadata.tempo} BPM`, color: 'text-primary' },
                { icon: Sliders, label: t('meta.complexity'), value: activeTrack.metadata.complexity, color: 'text-secondary' },
              ].map((item) => (
                <div key={item.label} className="bg-white/5 p-2.5 rounded-lg border border-white/5">
                  <p className="text-[9px] text-on-surface-variant uppercase flex items-center gap-1 mb-0.5">
                    <item.icon className={`w-3 h-3 ${item.color}`} /> {item.label}
                  </p>
                  <p className="font-display font-extrabold text-sm text-on-surface">
                    {item.value}
                  </p>
                </div>
              ))}
              <div className="col-span-2 flex flex-wrap gap-2 mt-1">
                <span className="px-3 py-1 bg-primary/15 rounded-full border border-primary/30 text-primary text-xs font-semibold">
                  {activeTrack.metadata.genre || t('misc.generated')}
                </span>
                <span className="px-3 py-1 bg-secondary/15 rounded-full border border-secondary/30 text-secondary text-xs font-semibold">
                  {t("misc." + activeTrack.instrument as any) || activeTrack.instrument}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Preview player */}
        <div className="glass-panel rounded-xl p-4 border border-white/5 shadow-lg">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-3">
              <button
                onClick={togglePreviewPlay}
                className="w-9 h-9 rounded-full bg-primary text-black flex items-center justify-center shadow-lg hover:scale-110 active:scale-95 transition-transform cursor-pointer"
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

        {/* Export + Share row */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Export */}
          <div className="glass-panel rounded-xl p-4 flex flex-col gap-3 border border-white/5 shadow-lg">
            <h3 className="font-display font-semibold text-sm text-on-surface">{t("lib.export")}</h3>
            <button
              onClick={activeTrack?.fileId ? handleDownloadFromServer : handleDownloadMidi}
              className="flex items-center justify-between bg-gradient-to-r from-primary to-secondary p-3 rounded-xl hover:brightness-110 active:scale-[0.98] transition-all cursor-pointer text-black"
            >
              <div>
                <span className="text-[9px] text-black/70 font-extrabold uppercase block">
                  {activeTrack?.fileId ? t("lib.downloadServer") : t("lib.downloadLocal")}
                </span>
                <span className="font-display font-extrabold text-sm">{t("lib.download")}</span>
              </div>
              <Download className="w-5 h-5" />
            </button>
            {activeTrack?.fileId && (
              <button
                onClick={handleDownloadFromServer}
                className="flex items-center gap-3 p-2.5 rounded-xl border border-white/10 hover:bg-white/5 transition-all cursor-pointer text-left"
              >
                <Server className="w-4 h-4 text-secondary" />
                <div>
                  <p className="text-xs font-bold text-on-surface">{t("lib.fromServer")}</p>
                  <p className="text-[9px] text-on-surface-variant">{t("lib.fromServerDesc")}</p>
                </div>
              </button>
            )}
          </div>

          {/* Share */}
          <div className="glass-panel rounded-xl p-4 border border-white/5 shadow-lg">
            <div className="flex items-center justify-between mb-2">
              <h3 className="font-display font-semibold text-sm text-on-surface">{t("lib.share")}</h3>
              <button
                onClick={handleCopyLink}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-container-high border border-white/10 hover:text-secondary hover:border-secondary transition-all cursor-pointer text-xs"
              >
                {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                {copied ? 'Copied!' : 'Copy Link'}
              </button>
            </div>
            <div className="flex gap-3">
              <div className="w-8 h-8 rounded-full bg-blue-500/10 border border-blue-500/30 flex items-center justify-center cursor-pointer hover:bg-blue-500/25 text-blue-400 transition-colors">
                <span className="text-xs font-bold">@</span>
              </div>
              <div
                onClick={() => setLiked(!liked)}
                className={`w-8 h-8 rounded-full border flex items-center justify-center cursor-pointer transition-colors ${
                  liked
                    ? 'bg-pink-500/25 border-pink-500 text-pink-400 animate-pulse'
                    : 'bg-pink-500/10 border-pink-500/30 text-pink-400 hover:bg-pink-500/20'
                }`}
              >
                <span className="translate-y-[0.5px]">♥</span>
              </div>
              <div className="w-8 h-8 rounded-full bg-white/5 border border-white/15 flex items-center justify-center cursor-pointer hover:bg-white/15 text-white transition-colors">
                <span className="text-xs font-mono font-bold">AI</span>
              </div>
            </div>
          </div>
        </div>

        {/* ─── Feature: Chord Detection ────────────────────────────── */}
        <div className="glass-panel rounded-xl border border-white/5 shadow-lg overflow-hidden">
          <button
            onClick={runChordDetection}
            className="w-full flex items-center justify-between p-4 hover:bg-white/5 transition-all cursor-pointer text-left"
          >
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-secondary/10 flex items-center justify-center text-secondary">
                <Guitar className="w-4 h-4" />
              </div>
              <div>
                <p className="text-sm font-bold text-white">Chord Detection</p>
                <p className="text-[9px] text-on-surface-variant">Detect chords from MIDI notes</p>
              </div>
            </div>
            {loadingChords ? (
              <RefreshCw className="w-4 h-4 text-on-surface-variant animate-spin" />
            ) : showChords ? (
              <ChevronDown className="w-4 h-4 text-on-surface-variant" />
            ) : (
              <ChevronRight className="w-4 h-4 text-on-surface-variant" />
            )}
          </button>

          {showChords && chords.length > 0 && (
            <div className="px-4 pb-4">
              <div className="flex flex-wrap gap-1.5">
                {chords.map((c, i) => (
                  <div
                    key={i}
                    className={`px-2.5 py-1 rounded-lg text-xs font-mono font-bold border ${
                      c.confidence > 0.5
                        ? "bg-primary/10 border-primary/30 text-primary"
                        : "bg-white/5 border-white/10 text-on-surface-variant"
                    }`}
                    title={`Bar ${c.bar}: ${c.notes.join(", ")}`}
                  >
                    {c.name}
                    <span className="text-[9px] opacity-50 ml-1">bar {c.bar}</span>
                  </div>
                ))}
              </div>
              {chords.length === 0 && (
                <p className="text-xs text-on-surface-variant">No chord detected in this track.</p>
              )}
            </div>
          )}
        </div>

        {/* ─── Feature: AI Regenerate ──────────────────────────────── */}
        <div className="glass-panel rounded-xl p-4 border border-white/5 shadow-lg">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                <WandSparkles className="w-4 h-4" />
              </div>
              <div>
                <p className="text-sm font-bold text-white">AI Regenerate</p>
                <p className="text-[9px] text-on-surface-variant">Create a variation of this track</p>
              </div>
            </div>
            <button
              onClick={handleRegenerate}
              disabled={regenerating}
              className="bg-gradient-to-r from-primary to-secondary text-black text-xs font-bold px-4 py-2 rounded-xl hover:brightness-110 active:scale-95 transition-all cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
            >
              {regenerating ? (
                <>
                  <RefreshCw className="w-3.5 h-3.5 animate-spin" /> Generating...
                </>
              ) : (
                <>
                  <WandSparkles className="w-3.5 h-3.5" /> Regenerate
                </>
              )}
            </button>
          </div>
        </div>

        {/* ─── Feature: DNA Analysis ───────────────────────────────── */}
        <div className="glass-panel rounded-xl border border-white/5 shadow-lg overflow-hidden">
          <button
            onClick={runDNAAnalysis}
            className="w-full flex items-center justify-between p-4 hover:bg-white/5 transition-all cursor-pointer text-left"
          >
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-tertiary/10 flex items-center justify-center text-tertiary">
                <Dna className="w-4 h-4" />
              </div>
              <div>
                <p className="text-sm font-bold text-white">DNA Analysis</p>
                <p className="text-[9px] text-on-surface-variant">MusicDNA structure & quality</p>
              </div>
            </div>
            {loadingDna ? (
              <RefreshCw className="w-4 h-4 text-on-surface-variant animate-spin" />
            ) : showDna ? (
              <ChevronDown className="w-4 h-4 text-on-surface-variant" />
            ) : (
              <ChevronRight className="w-4 h-4 text-on-surface-variant" />
            )}
          </button>

          {showDna && dnaResult && (
            <div className="px-4 pb-4">
              {/* Quality score */}
              <div className="flex items-center gap-2 mb-3">
                <span className="text-[10px] text-on-surface-variant uppercase font-semibold">Quality:</span>
                <div className="flex-1 h-1.5 bg-white/10 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-red-500 via-yellow-500 to-green-500 rounded-full transition-all"
                    style={{ width: `${Math.round((dnaResult.quality || 0) * 100)}%` }}
                  />
                </div>
                <span className="text-xs font-mono text-secondary font-bold">
                  {Math.round((dnaResult.quality || 0) * 100)}%
                </span>
              </div>

              {/* DNA dimensions */}
              {dnaResult.dna && (
                <div className="grid grid-cols-2 gap-2">
                  {[
                    { label: t("dna.structure"), key: "structure", color: "text-primary" },
                    { label: t("dna.harmony"), key: "harmony", color: "text-secondary" },
                    { label: t("dna.motif"), key: "motif", color: "text-tertiary" },
                    { label: t("dna.rhythm"), key: "rhythm", color: "text-primary" },
                    { label: t("dna.texture"), key: "texture", color: "text-secondary" },
                    { label: t("dna.dynamics"), key: "dynamics", color: "text-tertiary" },
                    { label: t("dna.emotion"), key: "emotion", color: "text-primary" },
                  ].map((dim) => {
                    const val = dnaResult.dna[dim.key];
                    if (!val) return null;
                    // Find a numeric sub-field
                    const numVal = typeof val === "object"
                      ? Object.values(val).find(v => typeof v === "number") as number | undefined
                      : undefined;
                    return (
                      <div key={dim.key} className="bg-white/5 rounded-lg p-2.5">
                        <p className={`text-[9px] uppercase font-semibold ${dim.color}`}>{dim.label}</p>
                        {numVal !== undefined && (
                          <div className="flex items-center gap-1.5 mt-1">
                            <div className="flex-1 h-1 bg-white/10 rounded-full overflow-hidden">
                              <div
                                className="h-full bg-current rounded-full"
                                style={{ width: `${Math.min(100, Math.round(numVal * 100))}%` }}
                              />
                            </div>
                            <span className="text-[10px] font-mono text-on-surface-variant">
                              {Math.round(numVal * 100)}%
                            </span>
                          </div>
                        )}
                        {numVal === undefined && (
                          <p className="text-[10px] font-mono text-on-surface-variant mt-0.5 truncate">
                            {JSON.stringify(val).slice(0, 30)}
                          </p>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>

      </div>

      {/* ─── Buy Me a Coffee (hidden) ────────────────────────────── */}
      {/* <BuyMeCoffee /> */}

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
