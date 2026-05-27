# text2midi — Backend

AI-powered MIDI generation engine. Go backend with CLI and HTTP API.

See [root README](../README.md) for quick start and project overview.

## Backend-specific docs

- `agent.md` — LLM Agent pipeline design
- `design.md` — Architecture decisions
- `midi.md` — MIDI format implementation notes
- `httpcode.md` — API design notes

## Directory

```
backend/
├── cmd/
│   ├── generate/     CLI generator
│   └── server/       HTTP API server
├── internal/
│   ├── agent/        LLM agent chain
│   ├── composer/     Post-processing (groove, motif, structure, etc.)
│   ├── generator/    Rule-based generators (bass, chords, drums, lead)
│   ├── harmony/      Harmonic constraint engine
│   ├── llm/          Prompt templates + LLM client
│   ├── midi/         Native SMF Type 1 writer
│   ├── music/        Music theory utilities
│   ├── mutation/     Creative chaos engine
│   ├── schema/       Core data types
│   ├── store/        File store for generated MIDI
│   └── style/        Style database (40+ genres)
├── go.mod
└── generated/        Server-side MIDI output
```
