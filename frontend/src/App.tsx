import React, { useState, useEffect } from "react";
import Navbar from "./components/Navbar";
import Library from "./components/Library";
import { MidiNote, InstrumentType, MidiTrack, MidiMetadata, User, InfoResponse } from "./types";
import {
  Sparkles,
  Music,
  Brain,
  Sliders,
  Layers,
  Mic,
  Volume2
} from "lucide-react";
import * as api from "./utils/api";
import { useT } from "./i18n";

// Pre-defined initial template tracks to make the application fully functional out of the box!
const INITIAL_TRACKS: MidiTrack[] = [
  {
    id: "track-cyberpunk",
    instrument: "synth",
    globalVelocity: 95,
    createdAt: new Date().toISOString(),
    metadata: {
      title: "Cyberpunk_Lullaby_v2",
      seed: 8829103,
      tempo: 128,
      key: "C",
      scale: "Minor",
      complexity: "High",
      genre: "Cybernetic Jazz",
      durationStr: "03:42"
    },
    notes: [
      { id: "pre-1", pitch: "C5", time: 0, duration: 2.0, velocity: 90 },
      { id: "pre-2", pitch: "Eb4", time: 1.0, duration: 1.5, velocity: 85 },
      { id: "pre-3", pitch: "G4", time: 2.0, duration: 2.0, velocity: 95 },
      { id: "pre-4", pitch: "Bb4", time: 3.5, duration: 1.5, velocity: 80 },
      { id: "pre-5", pitch: "C5", time: 5.0, duration: 3.0, velocity: 100 },
      { id: "pre-6", pitch: "G4", time: 8.0, duration: 2.0, velocity: 90 },
      { id: "pre-7", pitch: "F4", time: 10.0, duration: 1.5, velocity: 85 },
      { id: "pre-8", pitch: "Eb4", time: 12.0, duration: 2.5, velocity: 95 }
    ]
  },
  {
    id: "track-jazz",
    instrument: "piano",
    globalVelocity: 85,
    createdAt: new Date().toISOString(),
    metadata: {
      title: "Melancholic_Jazz_Suite",
      seed: 7421935,
      tempo: 120,
      key: "Eb",
      scale: "Minor",
      complexity: "Medium",
      genre: "Classic Acoustic Jazz",
      durationStr: "04:15"
    },
    notes: [
      { id: "jazz-1", pitch: "Eb4", time: 0, duration: 0.5, velocity: 80 },
      { id: "jazz-2", pitch: "Gb4", time: 0.5, duration: 0.5, velocity: 75 },
      { id: "jazz-3", pitch: "Bb4", time: 1.0, duration: 1.0, velocity: 85 },
      { id: "jazz-4", pitch: "Db5", time: 2.0, duration: 0.5, velocity: 70 },
      { id: "jazz-5", pitch: "F4", time: 2.5, duration: 1.5, velocity: 75 },
      { id: "jazz-6", pitch: "Ab4", time: 4.0, duration: 2.0, velocity: 80 },
      { id: "jazz-7", pitch: "C4", time: 6.0, duration: 0.5, velocity: 85 },
      { id: "jazz-8", pitch: "Eb4", time: 7.0, duration: 3.0, velocity: 90 }
    ]
  },
  {
    id: "track-cinematic",
    instrument: "strings",
    globalVelocity: 75,
    createdAt: new Date().toISOString(),
    metadata: {
      title: "Neon_Dreams_Suite",
      seed: 1290458,
      tempo: 90,
      key: "G",
      scale: "Major",
      complexity: "High",
      genre: "Cinematic Ambient",
      durationStr: "02:50"
    },
    notes: [
      { id: "strings-1", pitch: "G4", time: 0, duration: 3.5, velocity: 70 },
      { id: "strings-2", pitch: "B4", time: 0, duration: 3.5, velocity: 65 },
      { id: "strings-3", pitch: "D5", time: 0.5, duration: 3.0, velocity: 75 },
      { id: "strings-4", pitch: "C4", time: 4.0, duration: 4.0, velocity: 80 },
      { id: "strings-5", pitch: "E4", time: 4.0, duration: 4.0, velocity: 70 },
      { id: "strings-6", pitch: "G4", time: 4.5, duration: 3.5, velocity: 75 },
      { id: "strings-7", pitch: "D4", time: 8.0, duration: 6.0, velocity: 85 }
    ]
  },
  {
    id: "track-lofi",
    instrument: "piano",
    globalVelocity: 70,
    createdAt: new Date(Date.now() - 86400000).toISOString(),
    metadata: {
      title: "Chill_Lofi_Study_Beats",
      seed: 4419023,
      tempo: 85,
      key: "A",
      scale: "Minor",
      complexity: "Low",
      genre: "Lo-fi Hip Hop",
      durationStr: "02:22"
    },
    notes: []
  },
  {
    id: "track-electro",
    instrument: "synth",
    globalVelocity: 100,
    createdAt: new Date(Date.now() - 172800000).toISOString(),
    metadata: {
      title: "Electro_Surge_Bass",
      seed: 5512341,
      tempo: 140,
      key: "F",
      scale: "Minor",
      complexity: "High",
      genre: "Electronic Dance",
      durationStr: "04:08"
    },
    notes: []
  },
  {
    id: "track-ambient",
    instrument: "strings",
    globalVelocity: 65,
    createdAt: new Date(Date.now() - 259200000).toISOString(),
    metadata: {
      title: "Deep_Ambient_Pad",
      seed: 6678901,
      tempo: 70,
      key: "D",
      scale: "Major",
      complexity: "Medium",
      genre: "Ambient Drone",
      durationStr: "06:30"
    },
    notes: []
  },
  {
    id: "track-funk",
    instrument: "piano",
    globalVelocity: 88,
    createdAt: new Date(Date.now() - 345600000).toISOString(),
    metadata: {
      title: "Funky_Soul_Groove",
      seed: 7723456,
      tempo: 110,
      key: "Eb",
      scale: "Major",
      complexity: "Medium",
      genre: "Funk Soul",
      durationStr: "03:55"
    },
    notes: []
  },
  {
    id: "track-orchestral",
    instrument: "strings",
    globalVelocity: 92,
    createdAt: new Date(Date.now() - 432000000).toISOString(),
    metadata: {
      title: "Epic_Orchestral_Rise",
      seed: 8834567,
      tempo: 130,
      key: "C",
      scale: "Minor",
      complexity: "High",
      genre: "Orchestral",
      durationStr: "05:12"
    },
    notes: []
  },
  {
    id: "track-house",
    instrument: "synth",
    globalVelocity: 98,
    createdAt: new Date(Date.now() - 518400000).toISOString(),
    metadata: {
      title: "Deep_House_Morning",
      seed: 9945678,
      tempo: 124,
      key: "G",
      scale: "Major",
      complexity: "Medium",
      genre: "Deep House",
      durationStr: "05:45"
    },
    notes: []
  },
  {
    id: "track-blues",
    instrument: "piano",
    globalVelocity: 78,
    createdAt: new Date(Date.now() - 604800000).toISOString(),
    metadata: {
      title: "Midnight_Blues_Impro",
      seed: 1012345,
      tempo: 95,
      key: "A",
      scale: "Major",
      complexity: "Medium",
      genre: "Blues",
      durationStr: "03:38"
    },
    notes: []
  },
  {
    id: "track-trap",
    instrument: "synth",
    globalVelocity: 100,
    createdAt: new Date(Date.now() - 691200000).toISOString(),
    metadata: {
      title: "Dark_Trap_808",
      seed: 1123456,
      tempo: 150,
      key: "D",
      scale: "Minor",
      complexity: "High",
      genre: "Trap",
      durationStr: "03:20"
    },
    notes: []
  },
  {
    id: "track-acoustic",
    instrument: "piano",
    globalVelocity: 72,
    createdAt: new Date(Date.now() - 777600000).toISOString(),
    metadata: {
      title: "Acoustic_Ballad_Rev",
      seed: 1234567,
      tempo: 78,
      key: "C",
      scale: "Major",
      complexity: "Low",
      genre: "Acoustic Ballad",
      durationStr: "04:02"
    },
    notes: []
  },
  {
    id: "track-synthwave",
    instrument: "synth",
    globalVelocity: 90,
    createdAt: new Date(Date.now() - 864000000).toISOString(),
    metadata: {
      title: "Retro_Synthwave_84",
      seed: 1345678,
      tempo: 128,
      key: "B",
      scale: "Minor",
      complexity: "Medium",
      genre: "Synthwave",
      durationStr: "04:30"
    },
    notes: []
  },
  {
    id: "track-jazz2",
    instrument: "piano",
    globalVelocity: 82,
    createdAt: new Date(Date.now() - 950400000).toISOString(),
    metadata: {
      title: "Smoky_Jazz_Club",
      seed: 1456789,
      tempo: 105,
      key: "Ab",
      scale: "Major",
      complexity: "High",
      genre: "Jazz Fusion",
      durationStr: "05:20"
    },
    notes: []
  },
  {
    id: "track-cinematic2",
    instrument: "strings",
    globalVelocity: 85,
    createdAt: new Date(Date.now() - 1036800000).toISOString(),
    metadata: {
      title: "Cinematic_Climax",
      seed: 1567890,
      tempo: 100,
      key: "E",
      scale: "Minor",
      complexity: "High",
      genre: "Cinematic",
      durationStr: "04:45"
    },
    notes: []
  }
];

export default function App() {
  const t = useT();
  const [activeTab, setActiveTab] = useState<"generate" | "library">("generate");
  const [prompt, setPrompt] = useState("");

  // ─── Auth state ───────────────────────────────────────────────────
  const [user, setUser] = useState<User | null>(null);

  // Restore session from localStorage
  useEffect(() => {
    const token = api.getToken();
    if (token) {
      // Try to fetch prefs as a session validation
      api.getPrefs()
        .then(() => {
          // Token is valid — we'd need user info; for now just indicate logged in
          // The backend doesn't expose a /me endpoint, so store minimal user info
          setUser({ id: 0, username: 'User', created_at: '' });
        })
        .catch(() => {
          api.clearToken();
          setUser(null);
        });
    }
  }, []);

  const handleLogin = async (username: string, password: string) => {
    const res = await api.login(username, password);
    api.setToken(res.token);
    setUser(res.user);
    return res;
  };

  const handleRegister = async (username: string, password: string) => {
    const res = await api.register(username, password);
    api.setToken(res.token);
    setUser(res.user);
    return res;
  };

  const handleLogout = async () => {
    try { await api.logout(); } catch { /* ignore */ }
    api.clearToken();
    setUser(null);
  };

  // ─── Info (styles/tiers) ──────────────────────────────────────────
  const [info, setInfo] = useState<InfoResponse | null>(null);

  useEffect(() => {
    api.getInfo()
      .then(setInfo)
      .catch(() => {/* backend might not be running */});
  }, []);

  // Synthesizer Parameter State variables (defaults to values of first tracks)
  const [tempo, setTempo] = useState(128);
  const [rootKey, setRootKey] = useState("C");
  const [scaleType, setScaleType] = useState("Minor");
  const [instrument, setInstrument] = useState<InstrumentType>("synth");
  const [globalVelocity, setGlobalVelocity] = useState(95);

  // Active track / notes compilation state
  const [tracks, setTracks] = useState<MidiTrack[]>(INITIAL_TRACKS);
  const [activeTrackId, setActiveTrackId] = useState<string | null>("track-cyberpunk");
  const [notes, setNotes] = useState<MidiNote[]>(INITIAL_TRACKS[0].notes);

  // Load history from backend on mount
  useEffect(() => {
    api.listFiles()
      .then((records) => {
        if (Array.isArray(records) && records.length > 0) {
          const historyTracks: MidiTrack[] = records.map((r) => ({
            id: r.id || 'track-' + Math.random(),
            notes: [],
            metadata: {
              title: r.file_name || 'Untitled',
              seed: Math.floor(Math.random() * 9000000) + 1000000,
              tempo: r.render_meta?.ticks_per_beat ? 120 : 120,
              key: 'C',
              scale: 'Major',
              complexity: 'Medium',
              genre: 'Generated',
              durationStr: r.render_meta?.duration_seconds
                ? Math.floor(r.render_meta.duration_seconds / 60) + ':' + String(Math.floor(r.render_meta.duration_seconds % 60)).padStart(2, '0')
                : '00:00'
            },
            instrument: 'piano' as InstrumentType,
            globalVelocity: 80,
            createdAt: r.created_at || new Date().toISOString(),
            fileId: r.id
          }));
          setTracks(prev => {
            const existing = new Set(prev.map(t => t.id));
            const newTracks = historyTracks.filter((t: MidiTrack) => !existing.has(t.id));
            return [...newTracks, ...prev];
          });
        }
      })
      .catch(() => {});
  }, []);

  // AI loading metrics state
  const [isGenerating, setIsGenerating] = useState(false);
  const [generationStep, setGenerationStep] = useState(0);

  const loadingMessages = [
    t("load.analyzing"),
    t("load.querying"),
    t("load.chords"),
    t("load.beats"),
    t("load.velocity"),
    t("load.synthesizing")
  ];

  // Map updates of notes context directly in active track object
  useEffect(() => {
    if (activeTrackId) {
      setTracks((prevTracks) =>
        prevTracks.map((t) =>
          t.id === activeTrackId
            ? { ...t, notes, metadata: { ...t.metadata, tempo, key: rootKey, scale: scaleType }, instrument, globalVelocity }
            : t
        )
      );
    }
  }, [notes, tempo, rootKey, scaleType, instrument, globalVelocity, activeTrackId]);

  // Synchronize loading message ticker
  useEffect(() => {
    let timer: any;
    if (isGenerating) {
      timer = setInterval(() => {
        setGenerationStep((p) => (p + 1) % loadingMessages.length);
      }, 1000);
    } else {
      setGenerationStep(0);
    }
    return () => clearInterval(timer);
  }, [isGenerating]);


  // ─── Auth modal control ─────────────────────────────────────────
  const [showAuthModal, setShowAuthModal] = useState(false);

  // Preset prompts clicking handles
  const handleApplyPresetGroup = (pText: string, tBpm: number, rKey: string, sScale: string) => {
    setPrompt(pText);
    setTempo(tBpm);
    setRootKey(rKey);
    setScaleType(sScale);
  };

  // Triggers API Call to backend Full-stack server
  const handleGenerateMidi = async () => {
    // Require login
    if (!user) {
      setShowAuthModal(true);
      return;
    }
    setIsGenerating(true);
    try {
      // Use dynamic styles from backend info if available
      const styleName = info?.styles?.[0] || "pop";
      const response = await api.generateMidi({
        prompt,
        style: styleName,
        bpm: tempo,
        key: rootKey + " " + scaleType.toLowerCase(),
        bars: 8,
        tier: "free"
      });

      if (response && response.fileId) {
        const newTrack: MidiTrack = {
          id: "track-" + Date.now(),
          notes: [],
          metadata: {
            title: response.fileName || response.meta?.output_path?.split("/").pop()?.replace(".mid", "") || "Generated Track",
            seed: Math.floor(Math.random() * 9000000) + 1000000,
            tempo: tempo,
            key: rootKey,
            scale: scaleType,
            complexity: "Medium",
            genre: prompt?.split(" ").slice(0, 2).join(" ") || "Generated",
            durationStr: response.durationSeconds ? Math.floor(response.durationSeconds / 60) + ":" + String(Math.floor(response.durationSeconds % 60)).padStart(2, "0") : "03:00"
          },
          instrument,
          globalVelocity,
          createdAt: new Date().toISOString(),
          fileId: response.fileId
        };

        setTracks([newTrack, ...tracks]);
        setActiveTrackId(newTrack.id);
        setActiveTab("library");
      }
    } catch (err) {
      console.error("Midi generation failure:", err);
    } finally {
      setIsGenerating(false);
    }
  };

  return (
    <div className="h-screen bg-[#121212] flex flex-col font-sans text-on-background overflow-hidden gap-2 Selection:bg-primary-container Selection:text-on-primary-container">
      {/* Dynamic atmospheric glowing canvas nodes */}
      <div className="absolute inset-x-0 top-0 h-[600px] hero-gradient pointer-events-none -z-10" />
      <div className="absolute top-[-10%] right-[-5%] w-[500px] h-[500px] bg-primary/8 blur-[130px] rounded-full -z-10 pointer-events-none" />
      <div className="absolute bottom-[20%] left-[-5%] w-[400px] h-[400px] bg-secondary/8 blur-[110px] rounded-full -z-10 pointer-events-none" />

      {/* Shared Navigation Header */}
      <Navbar
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        user={user}
        onLogin={handleLogin}
        onRegister={handleRegister}
        onLogout={handleLogout}
        showAuthModal={showAuthModal}
        setShowAuthModal={setShowAuthModal}
      />

      {/* Main Container screen sections */}
      <main className="flex-1 overflow-hidden min-h-0">
        {/* Loading overlay view block */}
        {isGenerating && (
          <div className="fixed inset-0 z-50 bg-[#0e0e0e]/90 backdrop-blur-md flex flex-col items-center justify-center p-lg">
            <div className="w-full flex flex-col items-center text-center gap-md">
              <div className="relative flex items-center justify-center">
                <div className="w-16 h-16 rounded-full border-4 border-dashed border-primary animate-spin" style={{ animationDuration: "3s" }} />
                <Brain className="w-6 h-6 text-primary absolute animate-pulse" />
              </div>
              <div className="space-y-sm mt-3">
                <h3 className="font-display font-extrabold text-headline-lg-mobile text-white uppercase tracking-wider">
                  MidiMind Orchestrator
                </h3>
                <p className="text-xs text-secondary animate-pulse">
                  {loadingMessages[generationStep]}
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Dynamic Route views Render Switch */}
        <div className="w-full px-lg h-full overflow-hidden">
          {activeTab === "generate" && (
            <div className="flex flex-col gap-14 h-full overflow-y-auto">
              
              {/* Giant Display header and subtitle */}
              <div className="flex flex-col items-center text-center gap-lg mt-8 select-none">
                <h1 className="font-display font-extrabold text-display-lg tracking-tight text-white leading-tight">
                  Turn Words Into <span className="text-primary italic">Music</span>
                </h1>
                <p className="text-on-surface-variant text-sm md:text-base leading-relaxed font-medium">
                  Describe the sound you hear in your head — a moody synth arpeggio, a jazz piano ballad, a cinematic orchestral swell — and our engine crafts a production-ready MIDI file in seconds.
                </p>
              </div>

              {/* Main Generation Card Overlay interface */}
              <div className="w-full mx-auto glass-panel p-lg rounded-2xl neon-glow">
                <div className="flex flex-col gap-md">
                  <div className="relative">
                    <textarea
                      value={prompt}
                      onChange={(e) => setPrompt(e.target.value)}
                      placeholder={t("gen.placeholder")}
                      className="w-full h-40 bg-surface-container-low border border-white/5 rounded-xl p-lg text-sm md:text-md text-white font-medium focus:ring-1 focus:ring-primary focus:outline-none transition-all resize-none placeholder:opacity-30"
                    />
                    
                    {/* mic overlay icon details */}
                    <div className="absolute bottom-4 right-4 flex items-center gap-sm opacity-60 select-none">
                      <Mic className="w-4 h-4 text-on-surface-variant" />
                      <span className="text-[10px] font-mono tracking-widest uppercase text-on-surface-variant">Voice Active</span>
                    </div>
                  </div>

                  {/* Preset prompt capsules clicking */}
                  <div className="flex flex-col md:flex-row justify-between items-center gap-md">
                    <div className="flex flex-wrap gap-2 justify-center">
                      <button
                        onClick={() => handleApplyPresetGroup("A moody neon arpeggio synth line in C minor", 128, "C", "Minor")}
                        className="bg-surface-container-highest hover:bg-white/10 px-md py-1 rounded-full text-[10px] font-mono font-bold text-primary flex items-center gap-1 cursor-pointer transition-colors"
                      >
                        <Sparkles className="w-3 h-3" /> {t("gen.presetCyberpunk")}
                      </button>
                      <button
                        onClick={() => handleApplyPresetGroup("Smooth acoustic jazz piano progression Eb Minor", 120, "Eb", "Minor")}
                        className="bg-surface-container-highest hover:bg-white/10 px-md py-1 rounded-full text-[10px] font-mono font-bold text-secondary flex items-center gap-1 cursor-pointer transition-colors"
                      >
                        <Music className="w-3 h-3" /> {t("gen.presetJazz")}
                      </button>
                      <button
                        onClick={() => handleApplyPresetGroup("Elevated epic violin string pads swell", 90, "G", "Major")}
                        className="bg-surface-container-highest hover:bg-white/10 px-md py-1 rounded-full text-[10px] font-mono font-bold text-tertiary flex items-center gap-1 cursor-pointer transition-colors"
                      >
                        <Volume2 className="w-3 h-3" /> {t("gen.presetString")}
                      </button>
                    </div>

                    <button
                      onClick={handleGenerateMidi}
                      className="w-full md:w-auto bg-gradient-to-r from-primary-container to-secondary-container hover:brightness-110 active:scale-95 text-on-primary-container px-xl py-md rounded-xl font-bold font-display flex items-center justify-center gap-2 cursor-pointer shadow-xl transition-all"
                    >
                      {t("gen.generate")}
                      <Sparkles className="w-5 h-5" />
                    </button>
                  </div>
                </div>
              </div>

              {/* Backend info: available styles & tier limits */}
              {info && (
                <div className="flex flex-wrap items-center gap-3 text-[10px] text-on-surface-variant font-mono justify-center">
                  <span className="flex items-center gap-1">
                    <span className="w-1.5 h-1.5 rounded-full bg-secondary" />
                    {t("info.styles")}: {info.styles?.join(', ') || 'N/A'}
                  </span>
                  <span className="opacity-30">|</span>
                  <span className="flex items-center gap-1">
                    <span className="w-1.5 h-1.5 rounded-full bg-primary" />
                    {t("info.tiers")}: {Object.entries(info.tiers || {}).map(([tier, b]) => `${tier} (${b} ${t("info.bars")})`).join(' · ')}
                  </span>
                </div>
              )}

              {/* Bento Grid layouts */}
              <div className="mx-auto grid grid-cols-1 md:grid-cols-12 gap-lg mt-xl select-none select-none">
                
                {/* Feature 1: AI Composition */}
                <div className="md:col-span-8 glass-panel p-xl rounded-xl group hover:border-primary/40 transition-all overflow-hidden relative border border-white/5">
                  <div className="relative z-10 text-left">
                    <div className="w-12 h-12 bg-primary/10 rounded-lg flex items-center justify-center mb-md border border-primary/20 text-primary">
                      <Brain className="w-6 h-6" />
                    </div>
                    <h3 className="font-display font-extrabold text-headline-lg-mobile md:text-headline-lg text-white mb-sm">
                      {t("feat.engine")}
                    </h3>
                    <p className="text-on-surface-variant text-xs md:text-sm leading-relaxed">
                      {t("feat.engineDesc")}
                    </p>
                  </div>
                  <div className="absolute right-[-20px] bottom-[-20px] opacity-[0.03] group-hover:opacity-[0.07] transition-all transform group-hover:scale-105 duration-700 pointer-events-none">
                    <Brain className="w-[180px] h-[180px] text-white" />
                  </div>
                </div>

                {/* Feature 2: Customizable */}
                <div className="md:col-span-4 glass-panel p-xl rounded-xl group hover:border-secondary/40 transition-all border border-white/5">
                  <div className="w-12 h-12 bg-secondary/10 rounded-lg flex items-center justify-center mb-md border border-secondary/20 text-secondary">
                    <Sliders className="w-6 h-6" />
                  </div>
                  <h3 className="font-display font-extrabold text-headline-lg-mobile md:text-headline-lg text-white mb-sm text-left">
                    {t("feat.llm")}
                  </h3>
                  <p className="text-on-surface-variant text-xs md:text-sm text-left leading-relaxed">
                    {t("feat.llmDesc")}
                  </p>
                </div>

                {/* Feature 3: Multi-Track */}
                <div className="md:col-span-4 glass-panel p-xl rounded-xl group hover:border-tertiary/40 transition-all border border-white/5">
                  <div className="w-12 h-12 bg-tertiary/10 rounded-lg flex items-center justify-center mb-md border border-tertiary/20 text-tertiary">
                    <Layers className="w-6 h-6" />
                  </div>
                  <h3 className="font-display font-extrabold text-headline-lg-mobile md:text-headline-lg text-white mb-sm text-left">
                    {t("feat.arrangement")}
                  </h3>
                  <p className="text-on-surface-variant text-xs md:text-sm text-left leading-relaxed">
                    {t("feat.arrangementDesc")}
                  </p>
                </div>

                {/* Feature 4: Cloud Engine (Hotlink graphic embed) */}
                <div className="md:col-span-8 glass-panel rounded-xl overflow-hidden relative min-h-[300px] border border-white/5">
                  <img
                    className="absolute inset-0 w-full h-full object-cover opacity-40 mix-blend-overlay pointer-none select-none pointer-events-none"
                    referrerPolicy="no-referrer"
                    alt="MIDI Interface mockup curved laptop glow colors"
                    src="https://lh3.googleusercontent.com/aida-public/AB6AXuC0hmf3JRzY-9bxkLMN1hioGZLfDyiYvY3i-ib36fAKrv_S0T-CU4_tZRLmHv2thFbRmIawCdNFJ0y7iuysQvx2L9KCqZx8ca7o8ZUmPmpScMX7pxxozv9Qr0gBrAFIfHzI3hVhKKwyiCa6doFQkVdLvcByij3Teku4zdKLEbeCu-sd76zGf2ta5Km3yWZ7HOdKwR61iXixT70mic_6SMsbgsUf19Azmn5_XViJrbqLveWqYEJFIbf4QbopIEwiUqW01i_b1HsSwwOK"
                  />
                  <div className="absolute inset-0 bg-gradient-to-r from-surface-container-lowest via-surface-container-lowest/80 to-transparent p-xl flex flex-col justify-end text-left">
                    <h3 className="font-display font-extrabold text-headline-lg-mobile md:text-headline-lg text-white mb-sm">
                      {t("feat.realtime")}
                    </h3>
                    <p className="text-on-surface-variant text-xs md:text-sm leading-relaxed">
                      {t("feat.realtimeDesc")}
                    </p>
                  </div>
                </div>
              </div>

              {/* Dynamic waveform laser footer banner */}
              <section className="flex flex-col items-center text-center mt-6">
                <div className="w-full h-24 mb-6">
                  <svg fill="none" height="60" viewBox="0 0 1000 60" width="100%" xmlns="http://www.w3.org/2000/svg">
                    <path className="animated-wave" d="M0 30 Q250 0 500 30 T1000 30" stroke="url(#wave-gradient-1)" strokeWidth="2" />
                    <path className="animated-wave animate-delay-1" d="M0 30 Q250 60 500 30 T1000 30" stroke="url(#wave-gradient-2)" strokeWidth="2" style={{ animationDelay: "-1.5s" }} />
                    <defs>
                      <linearGradient id="wave-gradient-1" x1="0" x2="1000" y1="30" y2="30" gradientUnits="userSpaceOnUse">
                        <stop stopColor="#d0bcff" />
                        <stop offset="1" stopColor="#5de6ff" />
                      </linearGradient>
                      <linearGradient id="wave-gradient-2" x1="0" x2="1000" y1="30" y2="30" gradientUnits="userSpaceOnUse">
                        <stop stopColor="#5de6ff" />
                        <stop offset="1" stopColor="#ffafd3" />
                      </linearGradient>
                    </defs>
                  </svg>
                </div>
                <h2 className="font-display font-black text-headline-lg-mobile md:text-headline-lg text-white leading-none">
                  Elevate Your Sound
                </h2>
                <p className="text-on-surface-variant text-xs mt-3">
                  {t("footer.tagline")}
                </p>
              </section>

            </div>
          )}

          

          {activeTab === "library" && (
            <div className="mt-4 h-[calc(100%-1rem)]">
              <Library
                tracks={tracks}
                setTracks={setTracks}
                activeTrackId={activeTrackId}
                setActiveTrackId={setActiveTrackId}

              />
            </div>
          )}
        </div>
      </main>

      {/* Shared site Footer */}
      
    

      {/* Footer */}
      <footer className="w-full py-lg mt-auto bg-surface-container-lowest border-t border-white/5 select-none">
        <div className="flex flex-col md:flex-row justify-between items-center px-lg w-full gap-md text-xs">
          <div className="flex flex-col items-center md:items-start gap-1">
            <div className="font-display font-extrabold text-base bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent">
              MidiMind AI
            </div>
            <p className="font-mono text-[10px] text-on-surface-variant opacity-80">
              © 2026 MidiMind AI. All rights reserved.
            </p>
          </div>
          <div className="flex flex-wrap justify-center gap-lg font-mono text-[10px] text-on-surface-variant font-bold uppercase tracking-wider">
            <a className="hover:text-primary transition-colors" href="#terms">Terms</a>
            <a className="hover:text-primary transition-colors" href="#privacy">Privacy</a>
            <a className="hover:text-primary transition-colors" href="#api">API docs</a>
            <a className="hover:text-primary transition-colors" href="#discord">Discord</a>
            <a className="hover:text-primary transition-colors" href="#twitter">Twitter / X</a>
          </div>
        </div>
      </footer>
    </div>
  );
}
