package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ShowerBandV/text2midi/internal/agent"
	"github.com/ShowerBandV/text2midi/internal/composer"
	"github.com/ShowerBandV/text2midi/internal/llm"
	"github.com/ShowerBandV/text2midi/internal/musicdna"
	"github.com/ShowerBandV/text2midi/internal/midi"
	"github.com/ShowerBandV/text2midi/internal/schema"
	"github.com/ShowerBandV/text2midi/internal/validator"
)

func main() {
	prompt := flag.String("prompt", "", "Music description")
	styleName := flag.String("style", "trap", "Style")
	bpm := flag.Int("bpm", 140, "BPM")
	key := flag.String("key", "C minor", "Key")
	bars := flag.Int("bars", 8, "Bars")
	out := flag.String("out", "./midi_output", "Output dir")
	local := flag.Bool("local", false, "Local mode (no API key, rule-based generation)")
	refine := flag.Bool("refine", false, "Enable LLM-based reviewer + iterative refinement (costs extra tokens)")
	dryRun := flag.Bool("dry-run", false, "Stop after plan stage — print plan summary and exit (LLM mode only)")
	resume := flag.Bool("resume", false, "Resume from last checkpoint after failure (LLM mode only)")
	pentatonic := flag.Bool("pentatonic", false, "Use pentatonic scale + Chinese ornamentation for lead melody")
	flatVel := flag.Int("flat-vel", 100, "Force all note velocities to this value (0=disabled)")
	validate := flag.Bool("validate", false, "Run music21-style validation + auto-fix measure durations")
	seed := flag.Int64("seed", 0, "Random seed (0=random per run, otherwise deterministic)")
	loopable := flag.Bool("loopable", false, "Make outro connect seamlessly to intro for game loop")
	progression := flag.String("progression", "", "Chord progression: warm/dark/hopeful/epic/tense/bright (overrides style default)")
	mode := flag.String("mode", "", "Scale mode: dorian/phrygian/lydian/mixolydian (overrides style default)")
	sf2 := flag.String("sf2", "", "SF2 profile path for instrument key range constraints")
	flag.Parse()

	if *prompt == "" && !*local {
		fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/generate/ --prompt \"...\"")
		fmt.Fprintln(os.Stderr, "       go run ./cmd/generate/ --local  (rule-based, no API key)")
		os.Exit(1)
	}

	llm.LoadDotEnv()

	// Local mode: skip LLM, use rule-based generation directly.
	if *local {
		composer.SetGlobalSeed(*seed)
		runLocal(*prompt, *styleName, *bpm, *bars, *key, *out, *dryRun, *pentatonic, *flatVel, *validate, *loopable, *progression, *mode, *sf2)
		return
	}
	composer.SetGlobalSeed(*seed)

	if os.Getenv("OPENAI_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY required (or use --local for offline mode)")
		os.Exit(1)
	}

	client, err := llm.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Client: %v\n", err)
		os.Exit(1)
	}

	// ── Checkpoint setup ─────────────────────────────────────────
	projectDir := filepath.Join(*out, ".projects", sanitizeName(*prompt))
	if *resume {
		fmt.Printf("  [Resume] loading from %s\n", projectDir)
	}

	intentRes, err := agent.ParseIntent(client, *prompt, false, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Intent: %v\n", err)
		os.Exit(1)
	}
	saveStage(projectDir, "01_intent.json", intentRes)

	plan, planRaw, err := agent.PlanSong(client, intentRes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Plan: %v\n", err)
		os.Exit(1)
	}
	saveStage(projectDir, "02_song_plan.json", planRaw)
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

	arr, arrRaw, err := agent.PlanArrangement(client, intentRes, planRaw, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Arr: %v\n", err)
		os.Exit(1)
	}
	saveStage(projectDir, "03_arrangement.json", arrRaw)

	// --- Dry-run: print plan and exit ---
	if *dryRun {
		fmt.Println("\n  ═══ Plan Summary ═══")
		fmt.Printf("  Title:       %s\n", plan.Title)
		fmt.Printf("  Key:         %s %s\n", plan.Key.Root, plan.Key.Mode)
		fmt.Printf("  BPM:         %d\n", plan.BPM)
		fmt.Printf("  Bars:        %d\n", plan.TotalBars)
		fmt.Printf("  Duration:    %.1fs\n", plan.EstimatedDuration)
		fmt.Printf("  Loopable:    %t\n", plan.Loopable)
		if len(plan.Sections) > 0 {
			fmt.Println("\n  Sections:")
			for _, sec := range plan.Sections {
				fmt.Printf("    %-10s bars %2d-%2d  energy=%.1f  density=%.1f  register=%s\n",
					sec.Name, sec.StartBar, sec.StartBar+sec.Bars, sec.Energy, sec.Density, sec.Register)
			}
		}
		fmt.Println("\n  Chord progression:")
		for _, c := range plan.ChordProgression {
			fmt.Printf("    bar %2d: %s\n", c.Bar, c.Chord)
		}
		fmt.Printf("\n  Arrangement: %d tracks\n", len(arr.Tracks))
		for _, t := range arr.Tracks {
			fmt.Printf("    %-12s ch%-2d  program=%-3d  role=%s\n", t.ID, t.Channel, ptrVal(t.Program), t.Role)
		}
		fmt.Println("\n  [dry-run] Plan OK. Remove --dry-run to generate MIDI.")
		return
	}

	// --- LLM Agent pipeline (Clef-style) ---
	evMap := make(map[string][]schema.NoteEvent)
	var evMu sync.Mutex

	// --- LLM Agents: only generate tracks that exist in the Arrangement ---
	// Determine which tracks are needed by mapping arrangement track IDs to event keys.
	needLead, needPad, needBass, needDrums := false, false, false, false
	for _, at := range arr.Tracks {
		key := lookupEventKey(at.ID)
		switch key {
		case "lead":
			needLead = true
		case "pad":
			needPad = true
		case "bass":
			needBass = true
		case "drums":
			needDrums = true
		}
	}
	// Fallback: if arrangement is empty, generate all core tracks.
	if !needLead && !needPad && !needBass && !needDrums {
		needLead, needPad, needBass, needDrums = true, true, true, true
	}

	cpJSON := agent.ChordProgressionToJSON(plan.ChordProgression)
	totalBeats := plan.TotalBars * 4
	taskCount := 0
	if needLead {
		taskCount++
	}
	if needPad {
		taskCount++
	}
	if needBass {
		taskCount++
	}
	if needDrums {
		taskCount++
	}

	var wg sync.WaitGroup
	wg.Add(taskCount)

	if needLead {
		go func() {
			defer wg.Done()
			ln, err := agent.ComposerAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON)
			evMu.Lock()
			if err == nil {
				evMap["lead"] = ln
			}
			fmt.Printf("  [Composer] lead: %d notes (err=%v)\n", len(ln), err)
			evMu.Unlock()
		}()
	}
	if needBass {
		go func() {
			defer wg.Done()
			// Use rule-based bass for guitar-driven styles (punk/metal/rock).
			styleForDrums := determineDrumStyle(intentMap)
			if styleForDrums != "" && (styleForDrums == "punk" || styleForDrums == "metal" || styleForDrums == "rock") {
				chordNames := chordsFromPlan(plan)
				bn := composer.GenerateBassStyled(styleForDrums, chordNames, plan.TotalBars)
				evMu.Lock()
				evMap["bass"] = bn
				evMu.Unlock()
				fmt.Printf("  [Rhythmist] bass: %d notes (rule-based, style=%s)\n", len(bn), styleForDrums)
			} else {
				bn, err := agent.RhythmistAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON, "bass")
				evMu.Lock()
				if err == nil {
					evMap["bass"] = bn
				}
				fmt.Printf("  [Rhythmist] bass: %d notes (LLM, err=%v)\n", len(bn), err)
				evMu.Unlock()
			}
		}()
	}
	if needPad {
		go func() {
			defer wg.Done()
			// Use rule-based power chords for guitar-driven styles.
			styleForDrums := determineDrumStyle(intentMap)
			if styleForDrums != "" && (styleForDrums == "punk" || styleForDrums == "metal" || styleForDrums == "rock") {
				chordNames := chordsFromPlan(plan)
				pn := composer.GenerateChordsStyled(styleForDrums, chordNames, plan.TotalBars, 0)
				evMu.Lock()
				evMap["pad"] = pn
				evMu.Unlock()
				fmt.Printf("  [Harmonist] chords: %d notes (rule-based power chords, style=%s)\n", len(pn), styleForDrums)
			} else {
				pn, err := agent.HarmonistAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON)
				evMu.Lock()
				if err == nil {
					evMap["pad"] = pn
				}
				fmt.Printf("  [Harmonist] pad: %d notes (LLM, err=%v)\n", len(pn), err)
				evMu.Unlock()
			}
		}()
	}
	if needDrums {
		go func() {
			defer wg.Done()
			// Use rule-based style drums when available (saves 1 LLM call).
			styleForDrums := determineDrumStyle(intentMap)
			if styleForDrums != "" {
				bars := plan.TotalBars
				energy := plan.FeatureVector.Energy
				dn := composer.GenerateDrumsStyled(styleForDrums, bars, energy)
				evMu.Lock()
				evMap["drums"] = dn
				evMu.Unlock()
				fmt.Printf("  [Rhythmist] drums: %d notes (rule-based, style=%s)\n", len(dn), styleForDrums)
			} else {
				dn, err := agent.RhythmistAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON, "drums")
				evMu.Lock()
				if err == nil {
					evMap["drums"] = dn
				}
				fmt.Printf("  [Rhythmist] drums: %d notes (LLM, err=%v)\n", len(dn), err)
				evMu.Unlock()
			}
		}()
	}

	wg.Wait()
	fmt.Printf("  Generated: %d tracks (LLM agents, parallel)\n", taskCount)
	saveStage(projectDir, "04_track_events.json", evMap)

	// --- Orchestrator: add dynamics ---
	agent.OrchestratorAgent(evMap, plan.TotalBars)

	// --- Refinement (opt-in with --refine) ---
	if *refine {
		maxRounds := 2
		if v := os.Getenv("LLM_MAX_REFINE_ROUNDS"); v != "" {
			fmt.Sscanf(v, "%d", &maxRounds)
		}
		for round := 0; round < maxRounds; round++ {
			// Snapshot before iteration — can rollback if review fails.
			snapshot := snapshotEvMap(evMap)

			report, err := agent.ReviewWithLLM(client, evMap, plan)
			if err != nil {
				fmt.Printf("  [Reviewer] LLM review failed: %v — rolling back\n", err)
				evMap = snapshot
				break
			}
			fmt.Printf("  [Reviewer] round %d: total=%.1f melody=%.1f harm=%.1f rhythm=%.1f\n",
				round+1, report.Total, report.Melody, report.Harmony, report.Rhythm)
			for _, issue := range report.Issues {
				fmt.Printf("    - %s\n", issue)
			}

			leaderPlan := agent.LeaderAgent(report, round+1)
			if leaderPlan.IterationComplete {
				fmt.Printf("  [Leader] iteration complete (round %d)\n", round+1)
				break
			}

			// Separate tasks with and without dependencies.
			var independent, dependent []agent.LeaderTask
			for _, task := range leaderPlan.Tasks {
				if task.DependsOn == "" {
					independent = append(independent, task)
				} else {
					dependent = append(dependent, task)
				}
			}

			// Run independent tasks in parallel.
			if len(independent) > 0 {
				var tWg sync.WaitGroup
				tWg.Add(len(independent))
				for _, task := range independent {
					task := task
					go func() {
						defer tWg.Done()
						fmt.Printf("  [Leader] executing: %s — %s\n", task.Agent, task.Instruction)
						switch task.Agent {
						case "composer":
							if ln, err := agent.ComposerAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON); err == nil {
								evMu.Lock()
								evMap["lead"] = ln
								evMu.Unlock()
							}
						case "harmonist":
							if pn, err := agent.HarmonistAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON); err == nil {
								evMu.Lock()
								evMap["pad"] = pn
								evMu.Unlock()
							}
						case "rhythmist":
							if bn, err := agent.RhythmistAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON, "bass"); err == nil {
								evMu.Lock()
								evMap["bass"] = bn
								evMu.Unlock()
							}
							if dn, err := agent.RhythmistAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON, "drums"); err == nil {
								evMu.Lock()
								evMap["drums"] = dn
								evMu.Unlock()
							}
						}
					}()
				}
				tWg.Wait()
			}

			// Run dependent tasks sequentially (they depend on previous results).
			for _, task := range dependent {
				fmt.Printf("  [Leader] executing: %s — %s (depends on %s)\n", task.Agent, task.Instruction, task.DependsOn)
				switch task.Agent {
				case "composer":
					if ln, err := agent.ComposerAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON); err == nil {
						evMap["lead"] = ln
					}
				case "harmonist":
					if pn, err := agent.HarmonistAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON); err == nil {
						evMap["pad"] = pn
					}
				case "rhythmist":
					if bn, err := agent.RhythmistAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON, "bass"); err == nil {
						evMap["bass"] = bn
					}
					if dn, err := agent.RhythmistAgent(client, plan.Key.Root, plan.Key.Mode, plan.BPM, totalBeats, cpJSON, "drums"); err == nil {
						evMap["drums"] = dn
					}
				}
			}
		}
	} else {
		// Quick Go rule-based check (informational only, no iteration).
		report := agent.ReviewerAgent(client, evMap, plan.TotalBars)
		fmt.Printf("  [Review] melody=%.1f harm=%.1f rhythm=%.1f struct=%.1f total=%.1f (use --refine for LLM iteration)\n",
			report.Melody, report.Harmony, report.Rhythm, report.Structure, report.Total)
	}

	// Generate rhythm guitar power chords for distorted guitar tracks.
	for _, at := range arr.Tracks {
		if at.ID == "distorted_guitar" || at.ID == "rhythm_guitar" {
			if _, exists := evMap["rhythm_guitar"]; !exists {
			}
		}
	}

	tracks := make([]schema.TrackIR, 0)
	for _, at := range arr.Tracks {
		// Map arrangement track IDs to the event keys that agents actually produce.
		// Agents write to: "lead", "pad", "bass", "drums", "chords".
		lookup := lookupEventKey(at.ID)
		ev := evMap[lookup]
		if ev == nil {
			// Fallback: try other likely keys.
			for _, fallback := range fallbackKeys(at.ID) {
				if ev = evMap[fallback]; ev != nil {
					lookup = fallback
					break
				}
			}
		}
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
		fmt.Printf("  Track %s → events[%s] (%d notes)\n", at.ID, lookup, len(ev))
	}

	name := plan.Title
	outputPath := *out + "/" + name + ".mid"
	os.MkdirAll(*out, 0755)

	// ── Flat velocity (MUST be before midiIR construction) ────
	if *flatVel > 0 {
		flattenVelocities(evMap, *flatVel)
	}

	midiIR := schema.MidiIR{
		Meta: schema.Meta{
			TicksPerBeat:  480,
			BPM:           plan.BPM,
			TotalBars:     plan.TotalBars,
			BeatsPerBar:   plan.TimeSignature.Numerator,
			TimeSignature: plan.TimeSignature,
		},
		Tracks: tracks,
	}

	// --- Stem export ---
	os.MkdirAll(outputPath+"/../stems", 0755)
	composer.ExportStems(midiIR, outputPath+"/../stems", name, nil)

	if err := validateMidiIR(midiIR); err != nil {
		fmt.Fprintf(os.Stderr, "Validation FAILED: %v\n", err)
		os.Exit(1)
	}

	// ── Full validator (opt-in with --validate) ──────────────────
	if *validate {
		report := validator.Validate(midiIR, plan.TotalBars, true)
		fmt.Print(validator.FormatReport(report))
		if !report.Passed {
			fmt.Fprintf(os.Stderr, "Validation FAILED with %d errors\n", len(report.Errors))
			os.Exit(1)
		}
	}

	// ── SF2 key range constraints ─────────────────────────────
	if *sf2 != "" {
		if p, e := musicdna.LoadSF2Profile(*sf2); e == nil {
			musicdna.ApplySF2Constraints(&midiIR, p)
		}
	}

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

	// ── Token usage & cost ─────────────────────────────────────
	usage := client.TotalUsage()
	if usage.Calls > 0 {
		fmt.Printf("\n  ═══ Token Usage ═══\n")
		fmt.Printf("  Calls:       %d\n", usage.Calls)
		fmt.Printf("  Input:       %d tokens\n", usage.InputTokens)
		fmt.Printf("  Output:      %d tokens\n", usage.OutputTokens)
		fmt.Printf("  Total:       %d tokens\n", usage.TotalTokens)
		fmt.Printf("  Model:       %s\n", usage.Model)
		fmt.Printf("  Cost:        $%.4f USD  (≈ ¥%.4f CNY)\n", usage.CostUSD, usage.CostCNY)
	}

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

// lookupEventKey maps an arrangement track ID to the primary event key.
// Agents (ComposerAgent, HarmonistAgent, RhythmistAgent) write to: lead, pad, bass, drums.
// Local mode writes to: lead, chords, bass, drums.
func lookupEventKey(id string) string {
	switch id {
	// ── Melodic tracks → "lead" ──
	case "lead", "melody", "lead_guitar", "guitar",
		"piano", "choir", "vocal", "synth_lead",
		"brass", "horn", "heroic_brass", "brass_ensemble", "orchestral_hits",
		"guzheng", "pipa", "dizi", "harp":
		return "lead"

	// ── Harmonic / pad tracks → "pad" (LLM agents) or "chords" (local / fallback) ──
	case "pad", "synth_pad", "warm_pad", "ambient_pad",
		"strings", "string_ensemble", "rapid_strings", "string_pad",
		"chords", "keys":
		return "pad"

	// ── Bass → "bass" ──
	case "bass", "sub_bass", "808":
		return "bass"

	// ── Drums / percussion → "drums" ──
	case "drums", "percussion", "timpani", "taiko", "taiko_drums", "driving_percussion":
		return "drums"

	// ── Guitar (rhythm) → "rhythm_guitar" ──
	case "distorted_guitar", "rhythm_guitar":
		return "rhythm_guitar"

	// ── FX / SFX → "pad" (closest texture match) ──
	case "fx", "sfx", "rain_fx", "noise", "texture":
		return "pad"

	default:
		return id // direct lookup
	}
}

// fallbackKeys returns secondary event keys to try if the primary lookup is empty.
func fallbackKeys(id string) []string {
	// If primary maps to "pad" but evMap only has "chords", try chords.
	// If primary maps to "lead" but evMap only has "pad", try pad.
	primary := lookupEventKey(id)
	switch primary {
	case "pad":
		return []string{"chords", "lead", "bass"}
	case "chords":
		return []string{"pad", "lead"}
	case "lead":
		return []string{"pad", "chords", "bass"}
	case "bass":
		return []string{"drums", "lead"}
	case "drums":
		return []string{"bass", "pad"}
	case "rhythm_guitar":
		return []string{"chords", "pad", "lead"}
	default:
		// Unknown ID — try common keys in order.
		return []string{"lead", "pad", "bass", "drums", "chords"}
	}
}

// validateMidiIR checks the MIDI IR for common issues before rendering.
// Returns nil if OK, or an error describing the first problem found.
func validateMidiIR(midiIR schema.MidiIR) error {
	if len(midiIR.Tracks) == 0 {
		return fmt.Errorf("no tracks in MIDI IR — generation produced nothing")
	}
	totalNotes := 0
	for _, t := range midiIR.Tracks {
		if !t.Enabled {
			continue
		}
		for _, ev := range t.Events {
			if ev.Type != "note" {
				continue
			}
			if ev.Pitch < 0 || ev.Pitch > 127 {
				return fmt.Errorf("track %q: note pitch %d out of MIDI range [0-127]", t.ID, ev.Pitch)
			}
			if ev.DurationBeat <= 0 {
				return fmt.Errorf("track %q: note at beat %.2f has duration %.2f ≤ 0", t.ID, ev.StartBeat, ev.DurationBeat)
			}
			totalNotes++
		}
	}
	if totalNotes == 0 {
		return fmt.Errorf("zero notes across all tracks — generation produced empty output")
	}
	fmt.Printf("  [Validate] OK: %d tracks, %d notes\n", len(midiIR.Tracks), totalNotes)
	return nil
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

// ─── Local (rule-based) generation ───────────────────────────────────

// runLocal generates MIDI entirely via Go rule-based engines, no LLM.
// Designed for offline use or quick iteration.
func runLocal(prompt, styleName string, bpm, bars int, key, out string, dryRun bool, pentatonic bool, flatVel int, runValidate bool, loopable bool, progression string, mode string, sf2Path string) {
	fmt.Println("[Local mode] Generating without LLM...")

	// ── Parse key ────────────────────────────────────────────────
	keyRoot := "C"
	keyMode := "major"
	if k := key; k != "" {
		if idx := strings.IndexByte(k, ' '); idx > 0 {
			keyRoot = k[:idx]
			keyMode = k[idx+1:]
		} else {
			keyRoot = k
		}
	}
	// Normalize mode.
	switch keyMode {
	case "m", "minor", "Minor", "natural_minor":
		keyMode = "minor"
	default:
		keyMode = "major"
	}

	// ── Style profile ────────────────────────────────────────────
	// Pick feature vector + defaults based on style name.
	darkness, energy, rhythmic, tension, defBPM, defBars, chordStyle := styleProfile(styleName)
	if bars <= 0 {
		bars = defBars
	}
	if bpm <= 0 {
		bpm = defBPM
	}

	fmt.Printf("  Style: %s | %s %s | %d bpm | %d bars\n", styleName, keyRoot, keyMode, bpm, bars)
	fmt.Printf("  Feature: dark=%.2f energy=%.2f rhythmic=%.2f tension=%.2f\n",
		darkness, energy, rhythmic, tension)

	// ── Chord progression ────────────────────────────────────────
	chords := progForStyle(keyRoot, keyMode, bars, chordStyle)
	if progression != "" {
		chords = progTemplate(keyRoot, keyMode, bars, progression)
	}

	// ── Dry-run: print plan and exit ─────────────────────────────
	if dryRun {
		layout := trackLayout(chordStyle)
		fmt.Println("\n  ═══ Local Plan Summary ═══")
		fmt.Printf("  Style:       %s\n", styleName)
		fmt.Printf("  Key:         %s %s\n", keyRoot, keyMode)
		fmt.Printf("  BPM:         %d\n", bpm)
		fmt.Printf("  Bars:        %d\n", bars)
		fmt.Printf("  Feature:     dark=%.2f energy=%.2f rhythmic=%.2f tension=%.2f\n",
			darkness, energy, rhythmic, tension)
		fmt.Println("\n  Chord progression:")
		for i := 0; i < min(len(chords), 8); i++ {
			fmt.Printf("    bar %d: %s\n", i, chords[i])
		}
		if len(chords) > 8 {
			fmt.Printf("    ... (repeats, %d total)\n", len(chords))
		}
		fmt.Println("\n  Track layout:")
		if layout.drums {
			fmt.Println("    Drums (ch9)")
		}
		if layout.bass {
			fmt.Printf("    %s (ch%d, prog=%d)\n", "Bass", 0, layout.bassProg)
		}
		ch := 1
		if layout.rhythm {
			fmt.Printf("    %s (ch%d, prog=%d)\n", layout.rhythmName, ch, layout.rhythmProg)
			ch++
		}
		if layout.lead {
			fmt.Printf("    %s (ch%d, prog=%d)\n", layout.leadName, ch, layout.leadProg)
			ch++
		}
		if layout.counter {
			fmt.Printf("    %s (ch%d, prog=%d)\n", layout.counterName, ch, layout.counterProg)
		}
		fmt.Println("\n  [dry-run] Plan OK. Remove --dry-run to generate MIDI.")
		return
	}

	// ── Generate tracks (parallel goroutines) ────────────────────
	evMap := make(map[string][]schema.NoteEvent)
	var evMu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(4)

	// Pre-compute lead parameters.
	stepProb := 0.55 + (1.0-tension)*0.3
	velMin := 45 + int(darkness*20)
	velMax := 75 + int(energy*40)
	if velMax > 115 {
		velMax = 115
	}
	secDensity := buildSectionDensity(bars, energy)
	secRegister := buildSectionRegister(bars, energy)
	blockRatio := 0.3 + energy*0.4

	// Override key mode if --mode is set.
	leadKeyMode := keyMode
	if mode != "" {
		leadKeyMode = mode
	}

	go func() {
		defer wg.Done()
		d := composer.GenerateDrumsStyled(chordStyle, bars, energy)
		evMu.Lock()
		evMap["drums"] = d
		fmt.Printf("  Drums: %d events (style=%s)\n", len(d), chordStyle)
		evMu.Unlock()
	}()

	go func() {
		defer wg.Done()
		b := composer.GenerateBassStyled(chordStyle, chords, bars)
		evMu.Lock()
		evMap["bass"] = b
		fmt.Printf("  Bass: %d events (style=%s)\n", len(b), chordStyle)
		evMu.Unlock()
	}()

	go func() {
		defer wg.Done()
		c := composer.GenerateChordsStyled(chordStyle, chords, bars, blockRatio)
		evMu.Lock()
		evMap["chords"] = c
		fmt.Printf("  Chords: %d events (style=%s)\n", len(c), chordStyle)
		evMu.Unlock()
	}()

	go func() {
		defer wg.Done()
		var l []schema.NoteEvent
		switch {
		case chordStyle == "metal":
			l = composer.GenerateLeadMetal(keyRoot, bars, energy)
		case chordStyle == "rock":
			l = composer.GenerateLeadRock(keyRoot, bars, energy)
		case chordStyle == "punk":
			l = composer.GenerateLeadPunk(keyRoot, bars, energy)
		case chordStyle == "pop" || chordStyle == "rpg" || chordStyle == "healing" || chordStyle == "victory":
			l = composer.GeneratePianoLegend(keyRoot, leadKeyMode, bars, chords)
		default:
			l = composer.GenerateLeadMidra(keyRoot, leadKeyMode, bars, stepProb, velMin, velMax, secDensity, secRegister, pentatonic)
		}
		evMu.Lock()
		evMap["lead"] = l
		fmt.Printf("  Lead (raw): %d events\n", len(l))
		evMu.Unlock()
	}()

	wg.Wait()

	// Apply melody grammar for musicality.
	grammar := composer.NewMelodyGrammar(keyRoot, keyMode)
	evMap["lead"] = grammar.ApplyAll(evMap["lead"], bars)
	fmt.Printf("  Lead (grammar): %d events\n", len(evMap["lead"]))

	// ── Build MIDI IR ────────────────────────────────────────────
	// Track layout is style-driven: punk uses 4-piece, emo uses full ensemble, etc.
	layout := trackLayout(chordStyle)
	trackList := make([]schema.TrackIR, 0, 6)
	ch := 0 // next available MIDI channel (9 = drums)

	// Drums always on ch9.
	if layout.drums {
		trackList = append(trackList, schema.TrackIR{
			ID: "drums", Name: "Drums", Channel: 9, Program: nil,
			Volume: 100, Pan: 64, Enabled: true, Events: evMap["drums"],
		})
	}

	// Bass.
	if layout.bass && ch < 9 {
		trackList = append(trackList, schema.TrackIR{
			ID: "bass", Name: "Bass", Channel: ch, Program: intPtr(layout.bassProg),
			Volume: 100, Pan: 64, Enabled: true, Events: evMap["bass"],
		})
		ch++
	}

	// Rhythm / chords track (pad or rhythm guitar depending on style).
	if layout.rhythm && ch < 9 {
		trackList = append(trackList, schema.TrackIR{
			ID: "rhythm", Name: layout.rhythmName, Channel: ch, Program: intPtr(layout.rhythmProg),
			Volume: layout.rhythmVol, Pan: 64, Enabled: true, Events: evMap["chords"],
		})
		ch++
	}

	// Lead melody.
	if layout.lead && ch < 9 {
		trackList = append(trackList, schema.TrackIR{
			ID: "lead", Name: layout.leadName, Channel: ch, Program: intPtr(layout.leadProg),
			Volume: 100, Pan: 64, Enabled: true, Events: evMap["lead"],
		})
		ch++
	}

	// Counter-melody / texture layer (strings, etc.).
	if layout.counter && ch < 9 {
		var counter []schema.NoteEvent
		if chordStyle == "pop" || chordStyle == "rpg" || chordStyle == "healing" {
			counter = composer.GenerateStringsLayered(evMap["lead"], bars)
		} else if chordStyle == "metal" {
			counter = composer.GenerateTwinHarmony(evMap["lead"], bars)
		} else {
			counter = composer.GenerateCounterMelody(evMap["lead"], bars)
		}
		if len(counter) > 0 {
			trackList = append(trackList, schema.TrackIR{
				ID: "counter", Name: layout.counterName, Channel: ch, Program: intPtr(layout.counterProg),
				Volume: layout.counterVol, Pan: 72, Enabled: true, Events: counter,
			})
			fmt.Printf("  %s (counter): %d events\n", layout.counterName, len(counter))
		}
	}

	// ── Apply section dynamics ───────────────────────────────────
	structure := composer.SelectStructure(energy, "rpg")
	composer.ApplyStructure(evMap, structure, bars)

	// ── Loopable: mirror first bar into last bar ────────────────
	if loopable {
		makeLoopable(evMap, bars)
	}

	// ── Flat velocity (MUST be last, after all post-processing) ──
	if flatVel > 0 {
		flattenVelocities(evMap, flatVel)
	}

	// ── SF2 profile loading ───────────────────────────────────
	var sf2Profile *musicdna.SF2Profile
	if sf2Path != "" {
		sf2Profile, _ = musicdna.LoadSF2Profile(sf2Path)
	}

	// ── Render MIDI ──────────────────────────────────────────────
	midiIR := schema.MidiIR{
		Meta: schema.Meta{
			TicksPerBeat: 480,
			BPM:          bpm,
			TotalBars:    bars,
			BeatsPerBar:  4,
			TimeSignature: schema.TimeSignature{Numerator: 4, Denominator: 4},
		},
		Tracks: trackList,
	}

	os.MkdirAll(out, 0755)
	name := "RPG_Casual"
	if prompt != "" {
		name = sanitizeName(prompt)
	}
	outputPath := out + "/" + name + ".mid"

	if err := validateMidiIR(midiIR); err != nil {
		fmt.Fprintf(os.Stderr, "Validation FAILED: %v\n", err)
		os.Exit(1)
	}

	// ── Full validator (opt-in with --validate) ──────────────────
	if runValidate {
		report := validator.Validate(midiIR, bars, true)
		fmt.Print(validator.FormatReport(report))
		if !report.Passed {
			fmt.Fprintf(os.Stderr, "Validation FAILED with %d errors\n", len(report.Errors))
			os.Exit(1)
		}
	}

	// ── SF2 key range constraints ─────────────────────────────
	if sf2Profile != nil {
		musicdna.ApplySF2Constraints(&midiIR, sf2Profile)
	}

	result, err := midi.RenderMIDI(midiIR, outputPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Render: %v\n", err)
		os.Exit(1)
	}

	// ── MusicDNA ─────────────────────────────────────────────────
	extractor := musicdna.NewExtractor()
	dna := extractor.Extract(evMap, bars, keyRoot+" "+keyMode)
	fmt.Println(dna.Print())

	fmt.Printf("\n  MIDI written: %s\n", result.OutputPath)
	fmt.Printf("  Tracks: %d | Notes: %d | Duration: %.1fs\n",
		result.TotalTracks, result.TotalNoteEvents, result.DurationSeconds)
	fmt.Println("  Done!")
}

// styleProfile returns feature vector + defaults for a named style.
// "rpg" / "emo" / "trap" / "pop" / "rock" / "metal" / "ambient".
func styleProfile(style string) (darkness, energy, rhythmic, tension float64, defBPM, defBars int, chordStyle string) {
	s := strings.ToLower(style)
	switch {
	case strings.Contains(s, "healing") || strings.Contains(s, "cozy") || strings.Contains(s, "chill"):
		return 0.05, 0.10, 0.05, 0.05, 60, 48, "healing" // ~3:12 at 60bpm
	case strings.Contains(s, "tension") || strings.Contains(s, "dungeon") || strings.Contains(s, "stealth") || strings.Contains(s, "ominous"):
		return 0.75, 0.22, 0.35, 0.70, 80, 32, "tension" // ~1:36 at 80bpm
	case strings.Contains(s, "victory") || strings.Contains(s, "fanfare") || strings.Contains(s, "triumph"):
		return 0.08, 0.82, 0.60, 0.10, 130, 24, "victory" // ~0:44 at 130bpm
	case strings.Contains(s, "jazz") || strings.Contains(s, "swing"):
		return 0.30, 0.40, 0.60, 0.25, 120, 60, "jazz" // ~2:00 at 120bpm
	case strings.Contains(s, "funk"):
		return 0.20, 0.65, 0.80, 0.20, 100, 50, "funk" // ~2:00 at 100bpm
	case strings.Contains(s, "emo") || strings.Contains(s, "sad") || strings.Contains(s, "melancholy"):
		return 0.75, 0.32, 0.22, 0.52, 72, 36, "emo" // ~2:00 at 72bpm
	case strings.Contains(s, "trap") || strings.Contains(s, "hip"):
		return 0.55, 0.55, 0.65, 0.40, 140, 70, "trap" // ~2:00 at 140bpm
	case strings.Contains(s, "metal") || strings.Contains(s, "heavy"):
		return 0.80, 0.85, 0.55, 0.65, 160, 72, "metal" // 72 bars ≈ 1:48 at 160bpm
	case strings.Contains(s, "rock"):
		return 0.45, 0.70, 0.35, 0.40, 130, 65, "rock" // ~2:00 at 130bpm
	case strings.Contains(s, "pop"):
		return 0.25, 0.60, 0.35, 0.25, 110, 64, "pop" // ~2:19 at 110bpm
	case strings.Contains(s, "punk"):
		return 0.35, 0.78, 0.50, 0.35, 170, 85, "punk" // ~2:00 at 170bpm
	case strings.Contains(s, "ambient") || strings.Contains(s, "atmo"):
		return 0.30, 0.18, 0.10, 0.15, 60, 30, "ambient" // ~2:00 at 60bpm
	default: // rpg / casual / game
		return 0.20, 0.45, 0.30, 0.15, 100, 48, "rpg" // ~1:55 at 100bpm
	}
}

// progForStyle returns a chord progression for a given style and key.
func progForStyle(root, mode string, totalBars int, chordStyle string) []string {
	if mode == "minor" {
		switch chordStyle {
		case "emo":
			// i - iv - VII - III (emo descending, melancholic)
			base := []string{root + "m", intervalChord(root, 5), intervalChord(root, 10), intervalChord(root, 3)}
			return repeatChords(base, totalBars)
		case "ambient", "healing":
			// i - VII - i - VI (static, floating)
			base := []string{root + "m", intervalChord(root, 10), root + "m", intervalChord(root, 8)}
			return repeatChords(base, totalBars)
		case "trap":
			// i - bVI - bVII - i (dark trap loop).
			base := []string{root + "m", intervalChord(root, 8), intervalChord(root, 10), root + "m"}
			return repeatChords(base, totalBars)
		case "tension":
			base := []string{root + "m", intervalChord(root, 1), root + "m", root + "m"}
			return repeatChords(base, totalBars)
		case "jazz":
			base := []string{intervalChord(root, 2) + "7", fifthOf(root) + "7", root + "maj7", root + "maj7"}
			return repeatChords(base, totalBars)
		case "funk":
			base := []string{root + "7", intervalChord(root, 5) + "7", root + "7", root + "7"}
			return repeatChords(base, totalBars)
		case "metal":
			// i - bVI - bVII - i (metal: dark, chromatic, not the pop minor loop).
			base := []string{root + "m", intervalChord(root, 8), intervalChord(root, 10), root + "m"}
			return repeatChords(base, totalBars)
		default:
			// i - bVI - bIII - bVII (classic minor loop)
			base := []string{root + "m", intervalChord(root, 8), intervalChord(root, 3), intervalChord(root, 10)}
			return repeatChords(base, totalBars)
		}
	}
	// Major
	switch chordStyle {
	case "pop":
		// Rich pop: I - vi7 - IVmaj7 - V7 (John Legend colors).
		base := []string{root + "maj7", relativeMinor(root) + "7", fourthOf(root) + "maj7", fifthOf(root) + "7"}
		return repeatChords(base, totalBars)
	case "rock":
		// I - IV - V - IV
		base := []string{root, fourthOf(root), fifthOf(root), fourthOf(root)}
		return repeatChords(base, totalBars)
	case "ambient", "healing":
		// I - IV - I - vi (peaceful float)
		base := []string{root, fourthOf(root), root, relativeMinor(root)}
		return repeatChords(base, totalBars)
	case "victory":
		// I - V - vi - IV (uplifting, triumphant).
		base := []string{root, fifthOf(root), relativeMinor(root), fourthOf(root)}
		return repeatChords(base, totalBars)
	default: // rpg, casual
		// I - V - vi - IV (peaceful RPG town)
		base := []string{root, fifthOf(root), relativeMinor(root), fourthOf(root)}
		return repeatChords(base, totalBars)
	}
}

// progTemplate returns a chord progression from a named emotional template.
func progTemplate(root, mode string, totalBars int, name string) []string {
	type template struct {
		degrees []int // semitone offsets from root
	}
	templates := map[string]template{
		"warm":    {[]int{0, 9, 4, 7}},     // I-vi-IV-V
		"dark":    {[]int{0, 8, 10, 0}},    // i-bVI-bVII-i
		"hopeful": {[]int{5, 0, 7, 4}},     // IV-I-V-vi
		"epic":    {[]int{0, 8, 3, 10}},    // i-bVI-bIII-bVII
		"tense":   {[]int{0, 1, 0, 0}},     // i-bII-i
		"bright":  {[]int{0, 7, 9, 5}},     // I-V-vi-IV
	}
	t, ok := templates[name]
	if !ok {
		return nil
	}
	base := make([]string, len(t.degrees))
	for i, d := range t.degrees {
		base[i] = intervalChord(root, d)
		if mode == "minor" && (name == "warm" || name == "hopeful" || name == "bright") {
			// minor mode: convert major progressions to relative minor
			base[i] = intervalChord(root, d)
		}
	}
	return repeatChords(base, totalBars)
}

func repeatChords(base []string, totalBars int) []string {
	// Midra generators consume one chord per bar.
	out := make([]string, totalBars)
	for bar := 0; bar < totalBars; bar++ {
		out[bar] = base[bar%len(base)]
	}
	return out
}

// fifthOf returns the V chord root (7 semitones up).
func fifthOf(root string) string {
	return transposeNote(root, 7)
}

// fourthOf returns the IV chord root (5 semitones up).
func fourthOf(root string) string {
	return transposeNote(root, 5)
}

// relativeMinor returns the vi chord root (9 semitones up = relative minor).
func relativeMinor(root string) string {
	r := transposeNote(root, 9)
	return r + "m"
}

// intervalChord transposes root by semitones and returns the note name.
func intervalChord(root string, semitones int) string {
	return transposeNote(root, semitones)
}

var noteOrder = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

func transposeNote(root string, semitones int) string {
	for i, n := range noteOrder {
		if n == root {
			return noteOrder[(i+semitones)%12]
		}
	}
	// Fuzzy: try without sharp.
	for i, n := range noteOrder {
		if strings.EqualFold(n, root) || strings.HasPrefix(n, root) {
			return noteOrder[(i+semitones)%12]
		}
	}
	return root
}

// buildSectionDensity returns per-bar density based on section energy arc.
// Intro sparse → verse moderate → chorus peak → outro fade.
func buildSectionDensity(totalBars int, energy float64) []float64 {
	curve := make([]float64, totalBars)
	third := totalBars / 3
	if third < 4 {
		third = 4
	}
	for bar := 0; bar < totalBars; bar++ {
		switch {
		case bar < 4:
			curve[bar] = 0.25 + energy*0.1 // intro: sparse
		case bar < third:
			curve[bar] = 0.35 + energy*0.3 // verse: moderate
		case bar < third*2:
			curve[bar] = 0.55 + energy*0.45 // chorus: peak
		case bar < totalBars-4:
			curve[bar] = 0.40 + energy*0.3 // bridge
		default:
			curve[bar] = 0.20 + energy*0.15 // outro: fade
		}
	}
	return curve
}

// buildSectionRegister returns per-bar octave shifts based on section energy.
// Intro=low octave, chorus=high octave.
func buildSectionRegister(totalBars int, energy float64) []int {
	reg := make([]int, totalBars)
	third := totalBars / 3
	if third < 4 {
		third = 4
	}
	for bar := 0; bar < totalBars; bar++ {
		switch {
		case bar < 4:
			reg[bar] = 4 // intro: low register
		case bar < third:
			reg[bar] = 5 // verse: mid
		case bar < third*2:
			reg[bar] = 6 // chorus: high
		case bar < totalBars-4:
			reg[bar] = 5 // bridge
		default:
			reg[bar] = 4 // outro: back to low
		}
	}
	return reg
}

// determineDrumStyle checks the parsed intent for style keywords that map to drum styles.
// Returns "" if no match — caller falls back to LLM drum generation.
func determineDrumStyle(intentMap map[string]any) string {
	// Check style list first.
	hasOrchestral := false
	if styles, ok := intentMap["style"]; ok {
		if sl, ok := styles.([]any); ok {
			for _, s := range sl {
				if str, ok := s.(string); ok {
					low := strings.ToLower(str)
					// Orchestral/epic/cinematic → don't force guitar styles.
					if strings.Contains(low, "orchestral") || strings.Contains(low, "epic") || strings.Contains(low, "cinematic") || strings.Contains(low, "symphonic") {
						hasOrchestral = true
						continue
					}
					switch {
					case strings.Contains(low, "metal") || strings.Contains(low, "heavy"):
						return "metal"
					case strings.Contains(low, "punk"):
						return "punk"
					case strings.Contains(low, "emo") || strings.Contains(low, "sad") || strings.Contains(low, "melancholy"):
						return "emo"
					case strings.Contains(low, "rock"):
						return "rock"
					}
				}
			}
		}
	}
	if hasOrchestral {
		return "" // let LLM handle orchestral arrangement
	}
	// Check mood — only apply if no orchestral override.
	if moods, ok := intentMap["mood"]; ok {
		if ml, ok := moods.([]any); ok {
			for _, m := range ml {
				if str, ok := m.(string); ok {
					low := strings.ToLower(str)
					if strings.Contains(low, "aggressive") || strings.Contains(low, "angry") {
						if !hasOrchestral {
							return "metal"
						}
					}
					if strings.Contains(low, "sad") || strings.Contains(low, "melancholic") {
						return "emo"
					}
				}
			}
		}
	}
	return "" // fallback to LLM
}

// chordsFromPlan extracts chord names from a SongPlan as a per-bar slice.
func chordsFromPlan(plan *schema.SongPlan) []string {
	chords := make([]string, plan.TotalBars)
	for i := range chords {
		chords[i] = "C" // default
	}
	for _, cc := range plan.ChordProgression {
		if cc.Bar >= 0 && cc.Bar < plan.TotalBars {
			chords[cc.Bar] = cc.Chord
			// Fill forward until next chord change.
			for j := cc.Bar + 1; j < plan.TotalBars; j++ {
				hasNext := false
				for _, nc := range plan.ChordProgression {
					if nc.Bar == j {
						hasNext = true
						break
					}
				}
				if hasNext {
					break
				}
				chords[j] = cc.Chord
			}
		}
	}
	return chords
}

// applyOrchestrationCurve fades instruments in/out per section.
// Intro: only 1-2 instruments. Verse: add bass. Chorus: full band.
// Simulates real arrangement build-up by reducing velocity of "inactive" tracks.
func applyOrchestrationCurve(evMap map[string][]schema.NoteEvent, totalBars int, style string) {
	// Per-style track activation schedule: which bars each track plays at full volume.
	type trackEntry struct {
		key          string
		startBar     int  // bar at which this track enters
		soloIntro    bool // if true, this track plays solo in intro
	}
	var schedule []trackEntry
	switch style {
	case "punk", "metal", "rock":
		// Guitar-driven: intro = rhythm guitar solo → verse + drums → chorus + bass + lead.
		schedule = []trackEntry{
			{key: "chords", startBar: 0, soloIntro: true},   // rhythm guitar from start
			{key: "drums", startBar: 4},                       // drums enter verse
			{key: "bass", startBar: 8},                        // bass enters chorus
			{key: "lead", startBar: 8},                        // lead enters chorus
		}
	case "emo", "rpg", "pop":
		// Piano-driven: intro = piano solo → verse + pad + bass → chorus + drums + strings.
		schedule = []trackEntry{
			{key: "lead", startBar: 0, soloIntro: true},
			{key: "chords", startBar: 2},
			{key: "bass", startBar: 4},
			{key: "drums", startBar: 8},
		}
	default:
		// Default: everything from bar 0 (no subtraction).
		return
	}

	// Build a set of which tracks are active at which bar.
	for _, entry := range schedule {
		evs := evMap[entry.key]
		if evs == nil {
			continue
		}
		for i := range evs {
			bar := int(evs[i].StartBeat) / 4
			if bar < entry.startBar {
				if entry.soloIntro && bar < entry.startBar && bar < 4 {
					// Solo instrument in intro: keep full volume.
				} else {
					// Not yet entered: reduce to near-silent.
					evs[i].Velocity = int(float64(evs[i].Velocity) * 0.1)
				}
			} else if bar < entry.startBar+2 {
				// Fade in over 2 bars.
				fadeProgress := float64(bar-entry.startBar) / 2.0
				evs[i].Velocity = int(float64(evs[i].Velocity) * (0.3 + fadeProgress*0.7))
			}
			// Clamp.
			if evs[i].Velocity < 4 {
				evs[i].Velocity = 4
			}
			if evs[i].Velocity > 127 {
				evs[i].Velocity = 127
			}
		}
	}

	// Print the schedule.
	fmt.Printf("  [Orch] arrangement arc: ")
	for _, e := range schedule {
		fmt.Printf("%s@%d ", e.key, e.startBar)
	}
	fmt.Println()
}

// flattenVelocities sets all note velocities to a fixed value.
// makeLoopable copies the first bar's events to the last bar, shifting their timing,
// so the outro seamlessly connects back to the intro for game looping.
func makeLoopable(evMap map[string][]schema.NoteEvent, totalBars int) {
	firstBarEnd := 4.0
	lastBarStart := float64(totalBars-1) * 4.0
	for key, evs := range evMap {
		var firstBarEvents []schema.NoteEvent
		for _, ev := range evs {
			if ev.StartBeat < firstBarEnd {
				firstBarEvents = append(firstBarEvents, ev)
			}
		}
		// Remove existing last-bar events.
		var filtered []schema.NoteEvent
		for _, ev := range evs {
			if ev.StartBeat < lastBarStart {
				filtered = append(filtered, ev)
			}
		}
		// Clone first-bar events to last bar.
		for _, ev := range firstBarEvents {
			clone := ev
			clone.StartBeat = lastBarStart + (ev.StartBeat - 0)
			filtered = append(filtered, clone)
		}
		evMap[key] = filtered
	}
	fmt.Println("  [Loopable] last bar mirrors first bar for seamless loop")
}

// snapshotEvMap creates a deep copy of the events map for rollback.
func snapshotEvMap(evMap map[string][]schema.NoteEvent) map[string][]schema.NoteEvent {
	snap := make(map[string][]schema.NoteEvent, len(evMap))
	for k, evs := range evMap {
		clone := make([]schema.NoteEvent, len(evs))
		copy(clone, evs)
		snap[k] = clone
	}
	return snap
}

func flattenVelocities(evMap map[string][]schema.NoteEvent, vel int) {
	for _, evs := range evMap {
		for i := range evs {
			evs[i].Velocity = vel
		}
	}
	fmt.Printf("  [FlatVel] all notes → velocity %d\n", vel)
}

func sanitizeName(s string) string {
	// Replace problematic filename chars with underscores.
	s = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, s)
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

func intPtr(v int) *int { return &v }

func ptrVal(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// saveStage writes a pipeline stage result as JSON to the project directory.
func saveStage(projectDir, filename string, data any) {
	dir := projectDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return // non-fatal
	}
	path := filepath.Join(dir, filename)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return
	}
	fmt.Printf("  [Checkpoint] saved %s\n", path)
}

// trackLayoutConfig defines which tracks to include and their MIDI programs.
// Each style picks a layout, e.g. punk = drums+bass+guitar+guitar (4-piece, no pad/strings).
type trackLayoutConfig struct {
	drums, bass, rhythm, lead, counter                        bool
	rhythmName, leadName, counterName                         string
	rhythmProg, leadProg, counterProg, bassProg               int
	rhythmVol, counterVol                                     int
}

// trackLayout returns the instrument layout for a given style.
func trackLayout(style string) trackLayoutConfig {
	switch style {
	case "punk":
		// Classic punk 4-piece: drums + bass + rhythm guitar + lead guitar. No pad, no strings.
		return trackLayoutConfig{
			drums: true, bass: true, rhythm: true, lead: true, counter: false,
			bassProg: 34, // Electric Bass (pick)
			rhythmName: "Rhythm Gtr", rhythmProg: 30, rhythmVol: 95, // Distortion Guitar
			leadName: "Lead Gtr", leadProg: 29, // Overdrive Guitar
		}
	case "metal":
		// Metal: drums + bass + rhythm guitar + lead guitar + harmony guitar (twin lead).
		return trackLayoutConfig{
			drums: true, bass: true, rhythm: true, lead: true, counter: true,
			bassProg: 34,
			rhythmName: "Rhythm Gtr", rhythmProg: 30, rhythmVol: 100,
			leadName: "Lead Gtr", leadProg: 30,
			counterName: "Harmony Gtr", counterProg: 29, counterVol: 95, // Overdrive for harmony
		}
	case "rock":
		// Rock: drums + bass + rhythm guitar + lead guitar. No pad, no strings.
		return trackLayoutConfig{
			drums: true, bass: true, rhythm: true, lead: true, counter: false,
			bassProg: 33, // Electric Bass (finger)
			rhythmName: "Rhythm Gtr", rhythmProg: 29, rhythmVol: 90, // Overdrive
			leadName: "Lead Gtr", leadProg: 29, // Overdrive Guitar
		}
	case "trap":
		// Trap: drums + 808 bass + pad + synth lead. No strings.
		return trackLayoutConfig{
			drums: true, bass: true, rhythm: true, lead: true, counter: false,
			bassProg: 39, // Synth Bass
			rhythmName: "Pad", rhythmProg: 90, rhythmVol: 85, // New Age Pad
			leadName: "Synth Lead", leadProg: 81, // Lead (square)
		}
	case "ambient", "healing":
		// Healing: no drums. Piano + warm pad + strings. Peaceful, for cozy games.
		return trackLayoutConfig{
			drums: false, bass: true, rhythm: true, lead: true, counter: true,
			bassProg: 34, // Electric Bass (finger)
			rhythmName: "Warm Pad", rhythmProg: 91, rhythmVol: 50, // Pad (warm)
			leadName: "Piano", leadProg: 1, // Acoustic Grand
			counterName: "Strings", counterProg: 49, // String Ensemble 2
		}
	case "tension":
		// Tension: no drums. Deep bass + dark pad + low strings. Ominous dungeon.
		return trackLayoutConfig{
			drums: false, bass: true, rhythm: true, lead: false, counter: true,
			bassProg: 39, // Synth Bass 1 (deep, menacing)
			rhythmName: "Dark Pad", rhythmProg: 97, rhythmVol: 60, // FX (crystal)
			counterName: "Low Strings", counterProg: 42, // Cello
		}
	case "victory":
		return trackLayoutConfig{
			drums: true, bass: true, rhythm: true, lead: true, counter: true,
			bassProg: 34,
			rhythmName: "Brass", rhythmProg: 62, rhythmVol: 100,
			leadName: "Trumpet", leadProg: 57,
			counterName: "Strings", counterProg: 49,
		}
	case "jazz":
		return trackLayoutConfig{
			drums: true, bass: true, rhythm: true, lead: true, counter: false,
			bassProg: 33, // Electric Bass (finger)
			rhythmName: "Piano", rhythmProg: 1, rhythmVol: 80, // Acoustic Grand
			leadName: "Sax", leadProg: 67, // Tenor Sax
		}
	case "funk":
		return trackLayoutConfig{
			drums: true, bass: true, rhythm: true, lead: true, counter: true,
			bassProg: 37, // Slap Bass
			rhythmName: "Guitar", rhythmProg: 28, rhythmVol: 90, // Muted Guitar
			leadName: "Brass", leadProg: 62, // Brass Section
			counterName: "Clav", counterProg: 8, // Clavinet
		}
	case "emo":
		// Emo: drums + bass + dark pad + piano lead + strings counter.
		return trackLayoutConfig{
			drums: true, bass: true, rhythm: true, lead: true, counter: true,
			bassProg: 33,
			rhythmName: "Pad", rhythmProg: 91, rhythmVol: 80, // Polysynth (dark)
			leadName: "Piano", leadProg: 0,
			counterName: "Strings", counterProg: 49, counterVol: 65, // String Ensemble
		}
	default: // rpg, pop
		// Full ensemble: drums + bass + warm pad + piano lead + strings counter.
		return trackLayoutConfig{
			drums: true, bass: true, rhythm: true, lead: true, counter: true,
			bassProg: 33,
			rhythmName: "Pad", rhythmProg: 89, rhythmVol: 85, // Warm Pad
			leadName: "Piano", leadProg: 0,
			counterName: "Strings", counterProg: 49, counterVol: 70,
		}
	}
}
