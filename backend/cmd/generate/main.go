package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ShowerBandV/text2midi/internal/agent"
	"github.com/ShowerBandV/text2midi/internal/composer"
	"github.com/ShowerBandV/text2midi/internal/llm"
	"github.com/ShowerBandV/text2midi/internal/midi"
	"github.com/ShowerBandV/text2midi/internal/musicdna"
	"github.com/ShowerBandV/text2midi/internal/schema"
)

func main() {
	prompt := flag.String("prompt", "", "Music description")
	styleName := flag.String("style", "trap", "Style")
	bpm := flag.Int("bpm", 140, "BPM")
	flag.Int64("seed", 0, "Random seed (0=random)")
	flag.String("key", "C minor", "Key")
	bars := flag.Int("bars", 8, "Bars")
	out := flag.String("out", "./midi_output", "Output dir")
	local := flag.Bool("local", false, "Local mode (no API key, rule-based generation)")
	flag.Parse()

	if *prompt == "" && !*local {
		fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/generate/ --prompt \"...\"")
		fmt.Fprintln(os.Stderr, "       go run ./cmd/generate/ --local  (rule-based, no API key)")
		os.Exit(1)
	}

	llm.LoadDotEnv()

	// Local mode: skip LLM, use rule-based generation directly.
	if *local {
		fmt.Println("[Local mode] Generating without LLM...")
		ctx := composer.NewDefaultContext(*bars, *bpm).
			WithStyle(0.3, 0.6, 0.4, 0.3)
		ctx.Motif = []int{0, 2, 4, 3, 0}
		events := composer.ComposeSongWithContext(ctx)
		fmt.Printf("Generated %d tracks:\n", len(events))
		for name, evs := range events {
			fmt.Printf("  %s: %d events\n", name, len(evs))
		}
		return
	}

	if os.Getenv("OPENAI_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY required (or use --local for offline mode)")
		os.Exit(1)
	}

	client, err := llm.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Client: %v\n", err)
		os.Exit(1)
	}

	intentRes, err := agent.ParseIntent(client, *prompt, false, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Intent: %v\n", err)
		os.Exit(1)
	}

	plan, planRaw, err := agent.PlanSong(client, intentRes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Plan: %v\n", err)
		os.Exit(1)
	}
	// Copy feature vector from intent result to plan.
	intentMap, _ := intentRes["intent"].(map[string]any)
	if fvRaw, ok := intentMap["feature_vector"]; ok {
		if fvMap, ok := fvRaw.(map[string]any); ok {
			if d, ok := fvMap["darkness"]; ok {
				plan.FeatureVector.Darkness = toFloat(d)
			}
			if e, ok := fvMap["energy"]; ok {
				plan.FeatureVector.Energy = toFloat(e)
			}
			if a, ok := fvMap["acousticness"]; ok {
				plan.FeatureVector.Acousticness = toFloat(a)
			}
			if d, ok := fvMap["density"]; ok {
				plan.FeatureVector.Density = toFloat(d)
			}
			if r, ok := fvMap["rhythmic_complexity"]; ok {
				plan.FeatureVector.RhythmicComplexity = toFloat(r)
			}
			if t, ok := fvMap["tension"]; ok {
				plan.FeatureVector.Tension = toFloat(t)
			}
			if l, ok := fvMap["lo_fi"]; ok {
				plan.FeatureVector.LoFi = toFloat(l)
			}
		}
	}

	// Extract mood for composer personality.
	mood := "default"
	if styles, ok := intentMap["style"]; ok {
		if s, ok := styles.([]any); ok && len(s) > 0 {
			if ms, ok := s[0].(string); ok {
				mood = ms
			}
		}
	}

	// Pick composer personality.
	composerDNA := composer.PickComposer(*styleName, *prompt, mood)
	fmt.Printf("  Composer: %s", composerDNA.Name)
	// --- SongMemory: track motif and section info ---

	// Enable/disable post-processing based on DNA.
	if *bpm > 0 {
		plan.BPM = *bpm
	}
	if *bars > 0 {
		plan.TotalBars = *bars
	}

	arr, _, err := agent.PlanArrangement(client, intentRes, planRaw, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Arr: %v\n", err)
		os.Exit(1)
	}

	// --- Midra-style 4-generator pipeline ---



	}
	_ = templateHarmony
	chordStrs := []string{"C", "G", "Am", "F"}
	// --- MusicDNA Template Lookup ---
	templateLib := musicdna.NewTemplateDB("./templates")
	if templates, err := templateLib.FindByStyle(*styleName); err == nil && len(templates) > 0 {
		for _, t := range templates {
			if len(t.DNA.Harmony.Progression) > 1 {
				chordStrs = nil
				for _, cb := range t.DNA.Harmony.Progression {
					chordStrs = append(chordStrs, cb.Chord)
				}
				if len(chordStrs) > plan.TotalBars { chordStrs = chordStrs[:plan.TotalBars] }
				fmt.Printf("  Template: %s chords=%v
", t.Name, chordStrs)
				break
			}
		}
	}
	// Build style profile from template library.
	profile, profErr := musicdna.BuildStyleProfile(templateLib, *styleName)
	if profErr != nil {
		profile = &musicdna.StyleProfile{Name: "default"}
	}
	_ = profile

	evMap := make(map[string][]schema.NoteEvent)

	evMap["drums"] = composer.GenerateDrumsMidra(plan.TotalBars)
	evMap["bass"] = composer.GenerateBassMidra(chordStrs, plan.TotalBars)
	evMap["pad"] = composer.GenerateChordsMidra(chordStrs, plan.TotalBars)
	evMap["lead"] = composer.GenerateLeadMidra(plan.Key.Root, plan.Key.Mode, plan.TotalBars)
	fmt.Printf("  Generated: drums+bass+pad+lead\n")

	// Generate rhythm guitar power chords for distorted guitar tracks.
	for _, at := range arr.Tracks {
		if at.ID == "distorted_guitar" || at.ID == "rhythm_guitar" {
			if _, exists := evMap["rhythm_guitar"]; !exists {
			}
		}
	}

	tracks := make([]schema.TrackIR, 0)
	for _, at := range arr.Tracks {
		// Map arrangement track IDs to generator event keys.
		lookup := at.ID
		switch at.ID {
		case "distorted_guitar", "rhythm_guitar":
			lookup = "rhythm_guitar"
		case "piano":
			lookup = "chords"
		case "pad", "synth_pad", "warm_pad", "ambient_pad":
			lookup = "chords"
		case "strings", "string_ensemble", "rapid_strings", "string_pad":
			lookup = "chords"
		case "choir", "vocal":
			lookup = "lead"
		case "brass", "horn", "heroic_brass", "brass_ensemble", "orchestral_hits":
			lookup = "lead"
		case "lead_guitar", "guitar":
			lookup = "lead"
		case "guzheng", "pipa", "dizi", "harp":
			lookup = "lead"
		case "timpani", "percussion", "taiko", "taiko_drums", "driving_percussion":
			lookup = "drums"
		}
		ev := evMap[lookup]
		if ev == nil {
			ev = []schema.NoteEvent{}
		}
		tracks = append(tracks, schema.TrackIR{
			ID:      at.ID,
			Name:    at.ID,
			Channel: at.Channel,
			Program: at.Program,
			Volume:  100,
			Pan:     64,
			Enabled: true,
			Events:  ev,
		})
	}

	name := plan.Title
	outputPath := *out + "/" + name + ".mid"
	os.MkdirAll(*out, 0755)

	midiIR := schema.MidiIR{
		Meta: schema.Meta{
			TicksPerBeat: 480,
			BPM:          plan.BPM,
		},
		Tracks: tracks,
	}

	// --- Stem export ---
	os.MkdirAll(outputPath+"/../stems", 0755)
	composer.ExportStems(midiIR, outputPath+"/../stems", name, nil)

	result, err := midi.RenderMIDI(midiIR, outputPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Render: %v\n", err)
		os.Exit(1)
	}

	// --- MusicDNA extraction ---
	extractor := musicdna.NewExtractor()
	dna := extractor.Extract(evMap, plan.TotalBars, plan.Key.Root+" "+plan.Key.Mode)
	fmt.Println(dna.Print())

	fmt.Printf("  MIDI written: %s\n", result.OutputPath)
	fmt.Printf("  Tracks: %d | Notes: %d | Duration: %.1fs\n",
		result.TotalTracks, result.TotalNoteEvents, result.DurationSeconds)
	fmt.Println("  Done!")
}

// styleAwareMotif generates a motif based on musical feature vector.
// This ensures different styles use different interval patterns.
func styleAwareMotif(darkness, energy, tension, rhythmic float64, mode string) []int {
	switch {
	case energy > 0.7 && rhythmic > 0.5 && tension > 0.5:
		// Aggressive battle music: fourths, fifths, large intervals
		return []int{0, 5, 7, 5, 0, 7, 10, 5} // power chord arpeggios
	case energy > 0.6 && rhythmic < 0.4 && tension < 0.4:
		// Pop / upbeat: pentatonic, stepwise
		return []int{0, 2, 4, 5, 4, 2, 0, 2} // classic pop
	case darkness > 0.6 && energy < 0.4:
		// Dark ambient / sad: minor thirds, descending
		if mode == "minor" {
			return []int{0, 3, 2, 0, -2, 0, 2, 3} // melancholic fall
		}
		return []int{0, 3, 5, 3, 0, -2, 0, 2} // bluesy
	case tension > 0.6 && darkness > 0.5:
		// Tense / thriller: tritone, augmented
		return []int{0, 6, 3, 6, 0, 4, 2, 0} // diminished feel
	case rhythmic > 0.6 && tension > 0.4:
		// Hip-hop / trap: syncopated, repetitive
		return []int{0, 3, 5, 3, 5, 3, 0, 3} // loop-oriented
	default:
		// Default: pentatonic, balanced
		if mode == "minor" {
			return []int{0, 3, 5, 4, 0}
		}
		return []int{0, 2, 4, 3, 0}
	}
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	default:
		return 0
	}
}
