package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/yourname/text2midi/internal/agent"
	"github.com/yourname/text2midi/internal/composer"
	"github.com/yourname/text2midi/internal/generator"
	"github.com/yourname/text2midi/internal/mutation"
	"github.com/yourname/text2midi/internal/llm"
	"github.com/yourname/text2midi/internal/midi"
	"github.com/yourname/text2midi/internal/schema"
)

func main() {
	prompt := flag.String("prompt", "", "Music description")
	styleName := flag.String("style", "trap", "Style")
	bpm := flag.Int("bpm", 140, "BPM")
	flag.String("key", "C minor", "Key")
	bars := flag.Int("bars", 8, "Bars")
	out := flag.String("out", "./midi_output", "Output dir")
	flag.Parse()

	if *prompt == "" {
		fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/generate/ --prompt \"...\"")
		os.Exit(1)
	}

	llm.LoadDotEnv()
	if os.Getenv("OPENAI_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY required")
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
			if d, ok := fvMap["darkness"]; ok { plan.FeatureVector.Darkness = toFloat(d) }
			if e, ok := fvMap["energy"]; ok { plan.FeatureVector.Energy = toFloat(e) }
			if a, ok := fvMap["acousticness"]; ok { plan.FeatureVector.Acousticness = toFloat(a) }
			if d, ok := fvMap["density"]; ok { plan.FeatureVector.Density = toFloat(d) }
			if r, ok := fvMap["rhythmic_complexity"]; ok { plan.FeatureVector.RhythmicComplexity = toFloat(r) }
			if t, ok := fvMap["tension"]; ok { plan.FeatureVector.Tension = toFloat(t) }
			if l, ok := fvMap["lo_fi"]; ok { plan.FeatureVector.LoFi = toFloat(l) }
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
	songMem := composer.NewSongMemory()


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

	sd := fmt.Sprintf("%s beat, %d BPM", *styleName, *bpm)
	evMap, _, _, err := agent.GeneratePatterns(client, *prompt, *styleName, sd,
		plan.Key.Root+" "+plan.Key.Mode, plan.BPM, plan.TotalBars, plan.FeatureVector)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Patterns: %v\n", err)
		os.Exit(1)
	}

	totalBeats := plan.TotalBars * 4
	cpJSON := agent.ChordProgressionToJSON(plan.ChordProgression)
	desc := fmt.Sprintf(`{"style":"%s"}`, *styleName)
	fvBytes, _ := json.Marshal(plan.FeatureVector)

	ln, err := agent.GenerateMelodyNotes(client, "lead", plan.Key.Root, plan.Key.Scale,
		desc, string(fvBytes), cpJSON, plan.BPM, totalBeats)
	if err == nil && len(ln) > 0 {
		// Style-aware: metal=minimal processing, pop=full processing
		isMetal := plan.FeatureVector.Darkness >= 0.7 && plan.FeatureVector.Energy > 0.8
		isHighEnergy := isMetal || (plan.FeatureVector.Energy > 0.7 && plan.FeatureVector.RhythmicComplexity < 0.5 && plan.FeatureVector.Darkness > 0.3)

		// Apply MelodyGrammar: scale mask, interval limiter, gravity, phrasing.
		grammar := composer.NewMelodyGrammar(plan.Key.Root, plan.Key.Mode)
		ln = grammar.ApplyAll(ln, plan.TotalBars)

		if isHighEnergy {
			// Metal: lead guitar = intro riff (bars 0-1) + solo (last 2 bars).
			// Remove lead notes outside those sections.
			introEnd := 2 * 4.0    // first 2 bars
			soloStart := float64(plan.TotalBars-2) * 4.0
			var filtered []schema.NoteEvent
			for _, n := range ln {
				if n.StartBeat < introEnd || n.StartBeat >= soloStart {
					filtered = append(filtered, n)
				}
			}
			ln = filtered
			// Lengthen notes for sustain.
			for i := range ln {
				if ln[i].DurationBeat < 0.5 {
					ln[i].DurationBeat = 0.5
				}
			}
			fmt.Printf("  Lead: intro+solo only (%d notes)\n", len(ln))
		} else {
			ln = composer.NewMotifExtractor().ApplyMotifDevelopment(ln, plan.TotalBars)
			ln = composer.ApplyRegisterExpansion(ln, plan.TotalBars)
			ln = composer.ApplySyncopation(ln)
			ln = composer.ApplyCallResponse(ln)
			ln = composer.ApplyAnacrusis(ln, plan.TotalBars)
		}
		gs := composer.DefaultGroove(plan.FeatureVector.Energy, "mpc58")
		ln = composer.ApplyGroove(ln, gs)
		evMap["lead"] = ln
		fmt.Printf("  Lead: %d notes (metal=%v)", len(ln), isMetal)
	songMem.LearnMotif(ln)
	}

	if lead, ok := evMap["lead"]; ok && len(lead) > 0 {
		bn, err := agent.GenerateBassFromMelody(client, lead, plan.Key.Root, plan.Key.Scale,
			desc, string(fvBytes), cpJSON, plan.BPM, totalBeats)
		if err == nil && len(bn) > 0 {
			evMap["bass"] = bn
			fmt.Printf("  Bass: %d notes\n", len(bn))
		}
	}

	// --- Section transitions ---
	if plan.TotalBars > 4 {
		sectionNames := []string{"intro", "verse", "chorus", "bridge", "outro"}
		energies := composer.BuildSectionProfile(sectionNames)
		barStarts := make([]int, len(sectionNames))
		for i := range sectionNames {
			barStarts[i] = i * (plan.TotalBars / len(sectionNames))
		}
		composer.ApplyAllTransitions(evMap, energies, barStarts, plan.BPM)
	}

	// --- Drum density: style+energy adaptive ---
	if drums, ok := evMap["drums"]; ok {
		evMap["drums"] = composer.AdjustDrumDensity(drums, plan.FeatureVector.Energy, plan.TotalBars, *styleName)
	}

	// --- Dynamic layering (instrument count by energy) ---
	if plan.TotalBars > 4 {
		sectionNames := []string{"intro", "verse", "chorus", "bridge", "outro"}
		energies := composer.BuildSectionProfile(sectionNames)
		barStarts := make([]int, len(sectionNames))
		for i := range sectionNames {
			barStarts[i] = i * (plan.TotalBars / len(sectionNames))
		}
		composer.ApplyLayeredDynamics(evMap, energies, barStarts)
	}

	// --- Texture layer ---
	texType := composer.SelectTexture(0.5)
	composer.GenerateTexture(evMap, texType, plan.Key.Root, plan.TotalBars, plan.BPM, plan.FeatureVector.Energy)

	// --- Voice crossing fix (DNA-controlled) ---
	if composerDNA.AllowVoiceCrossing {
		composer.FixVoiceCrossing(evMap)
	}

	// --- Creative Chaos ---
	mutSeed := time.Now().UnixNano()
	cc := mutation.DefaultChaos(composerDNA.Chaos, plan.FeatureVector.Energy, mutSeed)
	for id, evs := range evMap {
		evMap[id] = mutation.ApplyChaos(evs, cc, id)
	}
	fmt.Printf("  Chaos applied (%.1f)", composerDNA.Chaos)

	agent.GenerateChordPad(plan, evMap)

	// Generate rhythm guitar power chords for distorted guitar tracks.
	for _, at := range arr.Tracks {
		if at.ID == "distorted_guitar" || at.ID == "rhythm_guitar" {
			if _, exists := evMap["rhythm_guitar"]; !exists {
				rg := generator.GenerateRhythmGuitar(*plan, at)
				if len(rg) > 0 {
					evMap["rhythm_guitar"] = rg
					fmt.Printf("  Rhythm guitar: %d power chords\n", len(rg))
				}
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
			BPM:         plan.BPM,
		},
		Tracks: tracks,
	}

	// --- Stem export disabled ---

	result, err := midi.RenderMIDI(midiIR, outputPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Render: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  MIDI written: %s\n", result.OutputPath)
	fmt.Printf("  Tracks: %d | Notes: %d | Duration: %.1fs\n",
		result.TotalTracks, result.TotalNoteEvents, result.DurationSeconds)
	fmt.Println("  Done!")
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
