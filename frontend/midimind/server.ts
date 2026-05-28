import express from "express";
import path from "path";
import dotenv from "dotenv";
import { createServer as createViteServer } from "vite";
import { GoogleGenAI, Type } from "@google/genai";

// Load environment variables
dotenv.config();

const app = express();
app.use(express.json({ limit: "5mb" }));

const PORT = 3000;

// Lazy initialization of Gemini SDK as strictly specified for stability
let aiInstance: GoogleGenAI | null = null;
function getGeminiClient(): GoogleGenAI {
  const apiKey = process.env.GEMINI_API_KEY;
  if (!apiKey || apiKey === "MY_GEMINI_API_KEY" || apiKey.trim() === "") {
    throw new Error("GEMINI_API_KEY environment variable is missing or empty.");
  }
  if (!aiInstance) {
    aiInstance = new GoogleGenAI({
      apiKey,
      httpOptions: {
        headers: {
          "User-Agent": "aistudio-build",
        },
      },
    });
  }
  return aiInstance;
}

// Procedural musical generator for bulletproof fallback
function generateProceduralMidi(
  prompt: string,
  key: string = "C",
  scale: string = "Minor",
  tempo: number = 120,
  complexity: string = "High"
) {
  const scaleIntervals: Record<string, number[]> = {
    Major: [0, 2, 4, 5, 7, 9, 11],
    Minor: [0, 2, 3, 5, 7, 8, 10],
    Phrygian: [0, 1, 3, 5, 7, 8, 10],
    Dorian: [0, 2, 3, 5, 7, 9, 10],
  };

  const noteNames = ["C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"];
  const keyRootIndex = Math.max(0, noteNames.indexOf(key));
  const intervals = scaleIntervals[scale] || scaleIntervals["Minor"];

  // Helper to convert scale degrees to pitch string
  const getPitch = (degree: number, octave: number): string => {
    const scaleNote = intervals[degree % intervals.length];
    const rawNoteIndex = (keyRootIndex + scaleNote + Math.floor(degree / intervals.length) * 12) % 12;
    const addedOctave = Math.floor((keyRootIndex + scaleNote + Math.floor(degree / intervals.length) * 12) / 12);
    return `${noteNames[rawNoteIndex]}${octave + addedOctave}`;
  };

  const notes: Array<{ id: string; pitch: string; time: number; duration: number; velocity: number }> = [];
  const seed = Math.floor(Math.random() * 9000000) + 1000000;

  // Generate 4 bars of beats (16 beats total, steps of 0.5)
  // Let's make an interesting musical phrase based on prompt keywords
  const lowerPrompt = prompt.toLowerCase();
  let patternType = "arpeggio";
  if (lowerPrompt.includes("chord") || lowerPrompt.includes("lush") || lowerPrompt.includes("ambient")) {
    patternType = "chords";
  } else if (lowerPrompt.includes("bass") || lowerPrompt.includes("groove") || lowerPrompt.includes("techno")) {
    patternType = "bassline";
  }

  if (patternType === "chords") {
    // Generate beautiful chord pads on beats 0, 4, 8, 12
    for (let bar = 0; bar < 4; bar++) {
      const beatOffset = bar * 4;
      const chordDegrees = bar === 0 ? [0, 2, 4] : bar === 1 ? [3, 5, 7] : bar === 2 ? [4, 6, 8] : [1, 3, 5];
      chordDegrees.forEach((degree) => {
        notes.push({
          id: `note-pad-${bar}-${degree}`,
          pitch: getPitch(degree, 4),
          time: beatOffset,
          duration: 3.5,
          velocity: 75,
        });
        // Add lower root bass note
        if (degree === chordDegrees[0]) {
          notes.push({
            id: `note-bass-${bar}`,
            pitch: getPitch(degree - 7, 3),
            time: beatOffset,
            duration: 3.8,
            velocity: 85,
          });
        }
      });
    }
  } else if (patternType === "bassline") {
    // Generate bouncy, synchronized bass groove
    for (let step = 0; step < 32; step++) {
      const time = step * 0.5;
      // Skip some steps for groove synco
      if (step % 4 === 1 || step % 8 === 6 || step % 7 === 3) continue;

      const scaleDegree = step % 8 === 0 ? 0 : step % 8 === 4 ? 3 : step % 8 === 6 ? 4 : 2;
      notes.push({
        id: `note-bassline-${step}`,
        pitch: getPitch(scaleDegree - 7, 3),
        time,
        duration: step % 2 === 0 ? 0.4 : 0.2,
        velocity: 95 + Math.floor(Math.random() * 20),
      });
    }
  } else {
    // Elegant arpeggio (jazz/melancholic/default)
    const arpeggioPattern = [0, 2, 4, 7, 9, 7, 4, 2];
    for (let step = 0; step < 32; step++) {
      const time = step * 0.5;
      const isRest = complexity === "Low" ? step % 2 !== 0 : step % 4 === 3;
      if (isRest) continue;

      const patternIndex = step % arpeggioPattern.length;
      let degree = arpeggioPattern[patternIndex];
      // Adapt scale degree slightly per bar
      if (step >= 8 && step < 16) degree += 1;
      if (step >= 16 && step < 24) degree += 3;
      if (step >= 24) degree -= 1;

      notes.push({
        id: `note-arp-${step}`,
        pitch: getPitch(degree, 4),
        time,
        duration: 0.4,
        velocity: 80 + Math.floor(Math.sin(step) * 15),
      });
    }
  }

  return {
    notes,
    metadata: {
      title: prompt.trim()
        ? prompt.split(" ").slice(0, 3).join("_").replace(/[^a-zA-Z0-9_]/g, "") + "_v" + (Math.floor(Math.random() * 8) + 1)
        : "Procedural_Symphony",
      seed,
      tempo,
      key,
      scale,
      complexity: complexity as "Low" | "Medium" | "High",
      genre: patternType === "chords" ? "Ambient Cinematic" : patternType === "bassline" ? "Deep Electro Groove" : "Cybernetic Jazz",
      durationStr: "03:42",
    },
  };
}

// REST route for MIDI Generation
app.post("/api/generate", async (req, res) => {
  const { prompt, key = "C", scale = "Minor", tempo = 120, complexity = "High", instrument = "piano" } = req.body;
  const safePrompt = prompt || "A melancholic jazz piano solo in C minor...";

  console.log(`[MidiMind Server] MIDI Generation requested with prompt: "${safePrompt}" in ${key} ${scale} at ${tempo} BPM`);

  try {
    const ai = getGeminiClient();

    const systemInstruction = `You are MidiMind AI, an expert digital music composer and generative MIDI sequencer.
Your task is to generate and play a highly musical sequences of MIDI notes that strictly fit the specified keys, scales, and tempo.

Return a JSON object containing:
1. "notes": An array of note events representing a beat-synchronized sequence.
   Each note event MUST contain:
   - "pitch" (STRING, e.g., "C4", "Eb4", "D4", "G4", "Ab4", "C5", "Bb4", "F3", etc. Specify correct chromatic pitches matching the requested root key and scale). Include notes in logical octaves like 3, 4, 5.
   - "time" (NUMBER, specified in decimal beats, e.g., 0.0, 0.5, 1.0, 1.5, 2.0. Notes should align well on 0.5 or 0.25 beat intervals. Total duration must span up to 16.0 or 32.0 beats, which equates to 4 or 8 bars).
   - "duration" (NUMBER, specified in decimal beats, e.g. 0.25, 0.5, 1.0, 2.0).
   - "velocity" (INTEGER, range 1 to 127).
2. "metadata":
   - "title" (STRING, a cool tech-sounding title, e.g., "Cyberpunk_Lullaby_v2" or "Neon_Nights_Suite").
   - "genre" (STRING, determined musical genre, e.g., "Ambient Synth", "Phrygian Techno", "Cinematic Lofi").
   - "seed" (INTEGER, a random 7 digit integer).
   - "tempo" (INTEGER, match or intelligently adapt the requested tempo).
   - "key" (STRING, the root note of the scale).
   - "scale" (STRING, the exact scale selected).
   - "complexity" (STRING, "Low", "Medium", or "High").
   - "durationStr" (STRING, formatted length string, e.g. "03:42").

Ensure the generated pitch values strictly correspond to the chosen Scale Scale.
For ${key} ${scale}:
- C Major scale tones are: C, D, E, F, G, A, B.
- C Minor scale tones are: C, D, Eb, F, G, Ab, Bb.
Ensure excellent rhythm, harmony, and phrasing that matches the user's prompt mood completely (e.g. moody, fast, bright, melancholic, cybernetic).`;

    const response = await ai.models.generateContent({
      model: "gemini-3.5-flash",
      contents: `Generate a gorgeous, rhythmically solid sequence for the prompt: "${safePrompt}" using Key: "${key}", Scale: "${scale}", Tempo: ${tempo} BPM, Complexity: "${complexity}"`,
      config: {
        systemInstruction,
        temperature: 0.85,
        responseMimeType: "application/json",
        responseSchema: {
          type: Type.OBJECT,
          required: ["notes", "metadata"],
          properties: {
            notes: {
              type: Type.ARRAY,
              description: "Array of generated MIDI note details.",
              items: {
                type: Type.OBJECT,
                required: ["pitch", "time", "duration", "velocity"],
                properties: {
                  pitch: { type: Type.STRING, description: "Pitch name, e.g. C4 or Eb4" },
                  time: { type: Type.NUMBER, description: "Start time of the note in beats" },
                  duration: { type: Type.NUMBER, description: "Duration of the note in beats" },
                  velocity: { type: Type.INTEGER, description: "Note velocity tracker (1-127)" },
                },
              },
            },
            metadata: {
              type: Type.OBJECT,
              required: ["title", "seed", "tempo", "key", "scale", "complexity", "genre", "durationStr"],
              properties: {
                title: { type: Type.STRING },
                genre: { type: Type.STRING },
                seed: { type: Type.INTEGER },
                tempo: { type: Type.INTEGER },
                key: { type: Type.STRING },
                scale: { type: Type.STRING },
                complexity: { type: Type.STRING },
                durationStr: { type: Type.STRING },
              },
            },
          },
        },
      },
    });

    const parsedData = JSON.parse(response.text || "{}");
    // Ensure all notes are marked with unique ID
    if (parsedData && Array.isArray(parsedData.notes)) {
      parsedData.notes = parsedData.notes.map((note: any, index: number) => ({
        ...note,
        id: note.id || `ai-note-${index}-${note.pitch}`,
      }));
      res.json(parsedData);
    } else {
      throw new Error("Invalid response format received from model.");
    }
  } catch (error: any) {
    console.error("[MidiMind Server] Gemini generation failed, executing procedural fallback:", error.message);
    // Graceful fallback prevents blank display or client crash
    const fallbackMidi = generateProceduralMidi(safePrompt, key, scale, tempo, complexity);
    res.json(fallbackMidi);
  }
});

// Enable Vite dev server in development or serve static build in production
async function startServer() {
  if (process.env.NODE_ENV !== "production") {
    const vite = await createViteServer({
      server: { middlewareMode: true },
      appType: "spa",
    });
    app.use(vite.middlewares);
  } else {
    const distPath = path.join(process.cwd(), "dist");
    app.use(express.static(distPath));
    app.get("*", (req, res) => {
      res.sendFile(path.join(distPath, "index.html"));
    });
  }

  app.listen(PORT, "0.0.0.0", () => {
    console.log(`[MidiMind Server] Full-stack application online on http://0.0.0.0:${PORT}`);
  });
}

startServer();
