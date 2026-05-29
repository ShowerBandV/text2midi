// Package agent implements the three LLM agents that form the "brain" of the
// music generation pipeline. Ported from music_agent/agents/.
package agent

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ShowerBandV/text2midi/internal/llm"
	"github.com/ShowerBandV/text2midi/internal/schema"
	"github.com/ShowerBandV/text2midi/internal/style"
)

// ParseIntent calls the LLM to parse a user's text prompt into structured intent.
// Extracts feature_vector and merges with the selected style's default vector.
func ParseIntent(client *llm.Client, userPrompt string, enforceCoreTracks bool, maxDurationSeconds *int) (map[string]any, error) {
	prompt := llm.BuildIntentParserPrompt(userPrompt, enforceCoreTracks, maxDurationSeconds)
	systemPrompt := "You are a strict JSON generator. Return JSON only."

	result, err := client.JSON(systemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("intent_parser: %w", err)
	}

	// Validate the shape.
	if _, ok := result["intent"]; !ok {
		return nil, fmt.Errorf("intent_parser: response missing 'intent' key: %v", mapKeys(result))
	}

	intentMap, ok := result["intent"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("intent_parser: 'intent' is not an object")
	}

	// Extract primary style name for default vector merge.
	primaryStyle := ""
	if styles, ok := getList(intentMap, "style"); ok && len(styles) > 0 {
		if s, ok := styles[0].(string); ok {
			primaryStyle = s
		}
	}

	// Merge feature_vector: LLM output overrides style default.
	defaultVec := style.GetDefaultVector(primaryStyle)
	var mergedFV map[string]any
	if fv, ok := getMap(intentMap, "feature_vector"); ok {
		// Start from default, overlay LLM values.
		mergedFV = map[string]any{
			"darkness":            defaultVec.Darkness,
			"energy":              defaultVec.Energy,
			"acousticness":        defaultVec.Acousticness,
			"density":             defaultVec.Density,
			"rhythmic_complexity": defaultVec.RhythmicComplexity,
			"tension":             defaultVec.Tension,
			"lo_fi":               defaultVec.LoFi,
		}
		for k, v := range fv {
			if val, ok := v.(float64); ok {
				mergedFV[k] = val
			}
		}
	} else {
		mergedFV = map[string]any{
			"darkness":            defaultVec.Darkness,
			"energy":              defaultVec.Energy,
			"acousticness":        defaultVec.Acousticness,
			"density":             defaultVec.Density,
			"rhythmic_complexity": defaultVec.RhythmicComplexity,
			"tension":             defaultVec.Tension,
			"lo_fi":               defaultVec.LoFi,
		}
	}
	intentMap["feature_vector"] = mergedFV

	// Ensure _meta.
	result["_meta"] = map[string]any{"source": "llm"}

	// Log the parsed intent with feature vector.
	fmt.Printf("[Agent] intent_parser ->styles=%v  mood=%v  use_case=%q  duration=%.0fs  tracks=%v\n",
		intentMap["style"], intentMap["mood"], intentMap["use_case"],
		toFloat64(intentMap["duration_seconds"]), intentMap["requested_tracks"])
	fmt.Printf("[Agent] feature_vector ->darkness=%.2f  energy=%.2f  acousticness=%.2f  density=%.2f  rhythmic=%.2f  tension=%.2f  lofi=%.2f\n",
		toFloat64(mergedFV["darkness"]), toFloat64(mergedFV["energy"]), toFloat64(mergedFV["acousticness"]),
		toFloat64(mergedFV["density"]), toFloat64(mergedFV["rhythmic_complexity"]),
		toFloat64(mergedFV["tension"]), toFloat64(mergedFV["lo_fi"]))

	return result, nil
}

// PlanSong calls the LLM to generate a song plan (bpm, key, chord progression, etc.)
// from the parsed intent. Uses temperature 0.7 for creative variety.
func PlanSong(client *llm.Client, intentResult map[string]any) (*schema.SongPlan, map[string]any, error) {
	intentJSON := toJSON(intentResult)
	prompt := llm.BuildSongPlannerPrompt(intentJSON)
	systemPrompt := "You are a creative music composer. Return the song plan as strict JSON."

	result, err := client.JSONWithTemp(systemPrompt, prompt, 0.7)
	if err != nil {
		return nil, nil, fmt.Errorf("song_planner: %w", err)
	}

	// Validate shape.
	spRaw, ok := result["song_plan"]
	if !ok {
		return nil, nil, fmt.Errorf("song_planner: response missing 'song_plan' key")
	}
	spMap, ok := spRaw.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("song_planner: 'song_plan' is not an object")
	}

	plan, err := mapToSongPlan(spMap)
	if err != nil {
		return nil, nil, fmt.Errorf("song_planner: %w", err)
	}

	// Add _meta.
	result["_meta"] = map[string]any{"source": "llm"}

	fmt.Printf("[Agent] song_planner ->%s  %d bpm  %s %s  %d bars  %d chords\n",
		plan.Title, plan.BPM, plan.Key.Root, plan.Key.Mode, plan.TotalBars, len(plan.ChordProgression))

	return plan, result, nil
}

// PlanArrangement calls the LLM to generate an arrangement (track/instrument config)
// from the intent and song plan. Uses temperature 0.7 for creative variety.
func PlanArrangement(client *llm.Client, intentResult map[string]any, songPlanMap map[string]any, enforceCoreTracks bool) (*schema.Arrangement, map[string]any, error) {
	intentJSON := toJSON(intentResult)
	songPlanJSON := toJSON(songPlanMap)
	prompt := llm.BuildArrangementPlannerPrompt(intentJSON, songPlanJSON, enforceCoreTracks)
	systemPrompt := "You are a creative orchestrator. Return the arrangement as strict JSON."

	result, err := client.JSONWithTemp(systemPrompt, prompt, 0.7)
	if err != nil {
		return nil, nil, fmt.Errorf("arrangement_planner: %w", err)
	}

	// Navigate: result["arrangement"]["tracks"] is an object keyed by track ID.
	arrRaw, ok := result["arrangement"]
	if !ok {
		return nil, nil, fmt.Errorf("arrangement_planner: response missing 'arrangement' key: %v", mapKeys(result))
	}
	arrMap, ok := arrRaw.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("arrangement_planner: 'arrangement' is not an object")
	}
	tracksRaw, ok := arrMap["tracks"]
	if !ok {
		return nil, nil, fmt.Errorf("arrangement_planner: 'arrangement' missing 'tracks'")
	}
	tracksMap, ok := tracksRaw.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("arrangement_planner: 'tracks' is not an object")
	}

	arrangement := &schema.Arrangement{}
	for id, trRaw := range tracksMap {
		trMap, ok := trRaw.(map[string]any)
		if !ok {
			continue
		}
		track, err := mapToArrangementTrack(id, trMap)
		if err != nil {
			fmt.Printf("[Agent] arrangement_planner: skipping track %q: %v\n", id, err)
			continue
		}
		arrangement.Tracks = append(arrangement.Tracks, *track)
	}

	// Sort by channel for deterministic order.
	sort.Slice(arrangement.Tracks, func(i, j int) bool {
		return arrangement.Tracks[i].Channel < arrangement.Tracks[j].Channel
	})

	result["_meta"] = map[string]any{"source": "llm"}

	fmt.Printf("[Agent] arrangement_planner ->%d tracks: ", len(arrangement.Tracks))
	for _, t := range arrangement.Tracks {
		fmt.Printf("%s(ch%d) ", t.ID, t.Channel)
	}
	fmt.Println()

	return arrangement, result, nil
}

// ParseFeatureVectorFromIntent extracts the merged feature_vector from an intent result
// and returns a schema.FeatureVector. Returns a neutral default if not found.
func ParseFeatureVectorFromIntent(intentResult map[string]any) schema.FeatureVector {
	intentMap, ok := intentResult["intent"].(map[string]any)
	if !ok {
		return schema.FeatureVector{Darkness: 0.5, Energy: 0.5, Acousticness: 0.5, Density: 0.5, RhythmicComplexity: 0.5, Tension: 0.5, LoFi: 0.5}
	}
	fv, ok := intentMap["feature_vector"].(map[string]any)
	if !ok {
		return schema.FeatureVector{Darkness: 0.5, Energy: 0.5, Acousticness: 0.5, Density: 0.5, RhythmicComplexity: 0.5, Tension: 0.5, LoFi: 0.5}
	}
	return schema.FeatureVector{
		Darkness:           toFloat64(fv["darkness"]),
		Energy:             toFloat64(fv["energy"]),
		Acousticness:       toFloat64(fv["acousticness"]),
		Density:            toFloat64(fv["density"]),
		RhythmicComplexity: toFloat64(fv["rhythmic_complexity"]),
		Tension:            toFloat64(fv["tension"]),
		LoFi:               toFloat64(fv["lo_fi"]),
	}
}

// --- helper: map ->SongPlan ---

func mapToSongPlan(m map[string]any) (*schema.SongPlan, error) {
	p := &schema.SongPlan{
		Title:     getString(m, "title"),
		BPM:       getInt(m, "bpm"),
		TotalBars: getInt(m, "total_bars"),
		Loopable:  getBool(m, "loopable"),
	}

	// Time signature.
	if ts, ok := getMap(m, "time_signature"); ok {
		p.TimeSignature = schema.TimeSignature{
			Numerator:   getInt(ts, "numerator"),
			Denominator: getInt(ts, "denominator"),
		}
	} else {
		p.TimeSignature = schema.TimeSignature{Numerator: 4, Denominator: 4}
	}

	// Key.
	if k, ok := getMap(m, "key"); ok {
		p.Key = schema.Key{
			Root:  getString(k, "root"),
			Mode:  getString(k, "mode"),
			Scale: getString(k, "scale"),
		}
		if p.Key.Scale == "" {
			if p.Key.Mode == "minor" {
				p.Key.Scale = "natural_minor"
			} else {
				p.Key.Scale = "major"
			}
		}
	}

	// Estimated duration.
	if dur, ok := m["estimated_duration_seconds"]; ok {
		p.EstimatedDuration = toFloat64(dur)
	} else {
		beatsPerBar := p.TimeSignature.Numerator
		if beatsPerBar <= 0 {
			beatsPerBar = 4
		}
		totalBeats := float64(p.TotalBars * beatsPerBar)
		if p.BPM > 0 {
			p.EstimatedDuration = totalBeats * (60.0 / float64(p.BPM))
		}
	}

	// Chord progression.
	if cp, ok := getList(m, "chord_progression"); ok {
		for _, item := range cp {
			if itemMap, ok := item.(map[string]any); ok {
				p.ChordProgression = append(p.ChordProgression, schema.ChordChange{
					Bar:   getInt(itemMap, "bar"),
					Chord: getString(itemMap, "chord"),
				})
			}
		}
	}

	// Sections (optional per-section energy/density/register).
	if secs, ok := getList(m, "sections"); ok {
		for _, item := range secs {
			if sm, ok := item.(map[string]any); ok {
				sec := schema.SongSection{
					Name:     getString(sm, "name"),
					StartBar: getInt(sm, "start_bar"),
					Bars:     getInt(sm, "bars"),
					Energy:   toFloat64(sm["energy"]),
					Density:  toFloat64(sm["density"]),
					Register: getString(sm, "register"),
				}
				if sec.Bars <= 0 {
					sec.Bars = getInt(sm, "length_bars") // alternate key
				}
				if sec.Name != "" && sec.Bars > 0 {
					p.Sections = append(p.Sections, sec)
				}
			}
		}
	}

	// Validate.
	if p.TotalBars <= 0 {
		return nil, fmt.Errorf("invalid total_bars: %d", p.TotalBars)
	}
	if p.BPM <= 0 {
		return nil, fmt.Errorf("invalid bpm: %d", p.BPM)
	}
	if len(p.ChordProgression) == 0 {
		return nil, fmt.Errorf("empty chord_progression")
	}
	if p.Key.Root == "" {
		return nil, fmt.Errorf("missing key.root")
	}

	return p, nil
}

// --- helper: map ->ArrangementTrack ---

func mapToArrangementTrack(id string, m map[string]any) (*schema.ArrangementTrack, error) {
	t := &schema.ArrangementTrack{
		ID:                 id,
		Name:               getString(m, "name"),
		Role:               getString(m, "role"),
		Enabled:            true, // default: enabled. LLM can override with "enabled": false
		IsCoreTrack:        getBool(m, "is_core_track"),
		GenerationStrategy: getString(m, "generation_strategy"),
	}

	// MIDI sub-object.
	if midi, ok := getMap(m, "midi"); ok {
		t.Channel = getInt(midi, "channel")
		if prog, ok := midi["program"]; ok && prog != nil {
			switch v := prog.(type) {
			case float64:
				p := int(v)
				t.Program = &p
			}
		}
	} else {
		return nil, fmt.Errorf("missing 'midi'")
	}

	// Mix sub-object.
	if mix, ok := getMap(m, "mix"); ok {
		t.Volume = getInt(mix, "volume")
		t.Pan = getInt(mix, "pan")
	} else {
		t.Volume = 100
		t.Pan = 64
	}

	// Validation.
	if t.Channel < 0 || t.Channel > 15 {
		return nil, fmt.Errorf("invalid channel %d", t.Channel)
	}
	if t.Channel == 9 && t.Program != nil {
		// Drum channel should not have program.
		t.Program = nil
	}
	if t.Channel != 9 && t.ID == "drums" {
		t.Channel = 9
		t.Program = nil
	}

	return t, nil
}

// --- JSON / type helpers ---

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func mapKeys(m map[string]any) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getInt(m map[string]any, key string) int {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

func getMap(m map[string]any, key string) (map[string]any, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return nil, false
	}
	result, ok := v.(map[string]any)
	return result, ok
}

func getList(m map[string]any, key string) ([]any, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return nil, false
	}
	result, ok := v.([]any)
	return result, ok
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

// ─── Single-shot beat generation ────────────────────────────────────────

// BeatResponse is the JSON structure returned by the LLM beat prompt.
type BeatResponse struct {
	Style       string       `json:"style"`
	DrumPattern DrumPattern  `json:"drumPattern"`
	BassPattern []int        `json:"bassPattern"`
	MelodyNotes []MelodyNote `json:"melodyNotes"`
	Performance *Performance `json:"performance,omitempty"`
}

// DrumPattern holds 16-step sequences for kick, snare, and hihat.
type DrumPattern struct {
	Kick  []int `json:"kick"`
	Snare []int `json:"snare"`
	Hihat []int `json:"hihat"`
}

// MelodyNote is a single note in the melody.
type MelodyNote struct {
	Start        int    `json:"start"`
	Duration     int    `json:"duration"`
	Pitch        int    `json:"pitch"`
	Velocity     int    `json:"velocity,omitempty"`     // 1-127, varies for expression
	Articulation string `json:"articulation,omitempty"` // "legato", "staccato", "accent", "normal"
}

// Performance holds expression data applied across the whole track.
type Performance struct {
	ExpressionCurve []ExpressionPoint `json:"expressionCurve,omitempty"`
	SustainPedal    []SustainEvent    `json:"sustainPedal,omitempty"`
	PitchBend       []PitchBendPoint  `json:"pitchBend,omitempty"`
	GlobalDynamics  float64           `json:"globalDynamics,omitempty"` // 0.0-1.0
}

// PitchBendPoint defines a pitch bend value at a bar position.
type PitchBendPoint struct {
	Bar   float64 `json:"bar"`
	Value int     `json:"value"` // 0-16383, 8192=center (no bend)
}

// ExpressionPoint defines a CC11 expression value at a bar position.
type ExpressionPoint struct {
	Bar   float64 `json:"bar"`
	Value int     `json:"value"` // 0-127
}

// SustainEvent defines when sustain pedal turns on/off.
type SustainEvent struct {
	Bar float64 `json:"bar"`
	On  bool    `json:"on"`
}

// GenerateBeat calls the LLM once to generate a complete beat (drums, bass, melody)
// from a user prompt. Uses temperature 0.8 for creative variety.
func GenerateBeat(client *llm.Client, userPrompt, style, styleDesc, key string, bpm, bars int) (*schema.SongPlan, *schema.Arrangement, map[string][]schema.NoteEvent, error) {
	seed := fmt.Sprintf("%d", hash64(int64(bpm*bars+len(userPrompt))*31+time.Now().UnixNano()))
	prompt := llm.BuildBeatPrompt(style, styleDesc, userPrompt, bpm, bars, key, seed)
	systemPrompt := "You are a creative beat maker. Return the pattern as strict JSON."

	result, err := client.JSONWithTemp(systemPrompt, prompt, 0.8)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("generate_beat: %w", err)
	}

	// Marshal the raw result to bytes, then unmarshal into our struct.
	data, err := json.Marshal(result)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal beat result: %w", err)
	}

	var beat BeatResponse
	if err := json.Unmarshal(data, &beat); err != nil {
		return nil, nil, nil, fmt.Errorf("parse beat response: %w", err)
	}

	fmt.Printf("[Agent] beat ->style=%s  kick=%d hits  snare=%d hits  hihat=%d hits  bass=%d steps  melody=%d notes\n",
		beat.Style, sum(beat.DrumPattern.Kick), sum(beat.DrumPattern.Snare),
		sum(beat.DrumPattern.Hihat), countNonZero(beat.BassPattern), len(beat.MelodyNotes))

	// Build SongPlan from parameters.
	parsedKey, parsedMode := parseKeyMode(key)
	plan := &schema.SongPlan{
		Title: fmt.Sprintf("%s Beat", beat.Style),
		BPM:   bpm,
		TimeSignature: schema.TimeSignature{
			Numerator: 4, Denominator: 4,
		},
		Key: schema.Key{
			Root:  parsedKey,
			Mode:  parsedMode,
			Scale: "natural_minor",
		},
		TotalBars: bars,
		Loopable:  true,
	}

	// Build a simple chord progression based on the key.
	plan.ChordProgression = buildChordProgression(parsedKey, parsedMode, bars)

	// Build Arrangement.
	progBass := int(34) // Electric Bass
	progLead := int(80) // Lead (saw)
	arrangement := &schema.Arrangement{
		Tracks: []schema.ArrangementTrack{
			{ID: "drums", Name: "Drums", Role: "rhythm", Enabled: true,
				IsCoreTrack: true, GenerationStrategy: "direct",
				Channel: 9, Program: nil, Volume: 105, Pan: 64},
			{ID: "bass", Name: "Bass", Role: "bass", Enabled: true,
				IsCoreTrack: true, GenerationStrategy: "direct",
				Channel: 0, Program: &progBass, Volume: 100, Pan: 64},
			{ID: "lead", Name: "Lead", Role: "melody", Enabled: true,
				IsCoreTrack: true, GenerationStrategy: "direct",
				Channel: 2, Program: &progLead, Volume: 100, Pan: 64},
		},
	}

	// Convert drum pattern ->NoteEvents.
	eventsByTrack := make(map[string][]schema.NoteEvent)
	eventsByTrack["drums"] = drumPatternToEvents(beat.DrumPattern, bars)
	eventsByTrack["bass"] = bassPatternToEvents(beat.BassPattern, bars, parsedKey)
	eventsByTrack["lead"] = melodyNotesToEvents(beat.MelodyNotes, bars)

	return plan, arrangement, eventsByTrack, nil
}

// GeneratePatterns calls the LLM with the beat template (hip-hop 16-step prompt)
// to generate drum/bass/melody patterns, already knowing the key/bpm/bars from PlanSong.
// Uses temperature 0.8 and a time-based seed so each run produces different patterns.
// Dispatches to metal/rock/pop/hip-hop prompt based on the feature vector.
func GeneratePatterns(client *llm.Client, userPrompt, style, styleDesc, key string, bpm, bars int, fv schema.FeatureVector) (map[string][]schema.NoteEvent, []schema.CCEvent, []schema.PitchBendEvent, error) {
	seed := fmt.Sprintf("%d", hash64(int64(bpm*bars+len(userPrompt))*31+time.Now().UnixNano()))

	// Choose prompt template based on feature vector dimensions.
	// Order matters: more specific conditions checked first.
	//   Metal:  high darkness + high energy + high tension
	//   Pop:    low darkness + low tension (bright, consonant)
	//   Rock:   high energy + low rhythmic complexity
	//   Default: hip-hop / electronic
	var prompt string
	var systemPrompt string
	promptType := "hip-hop"

	switch {
	case fv.Darkness >= 0.7 && fv.Energy > 0.8 && fv.Tension >= 0.5:
		prompt = llm.BuildMetalPatternPrompt(style, styleDesc, userPrompt, bpm, bars, key, seed)
		systemPrompt = "You are a metal musician. Return the pattern as strict JSON."
		promptType = "metal"
	case fv.Energy > 0.4 && fv.Darkness < 0.4 && fv.Tension <= 0.3:
		prompt = llm.BuildPopPatternPrompt(style, styleDesc, userPrompt, bpm, bars, key, seed)
		systemPrompt = "You are a pop producer. Return the pattern as strict JSON."
		promptType = "pop"
	case fv.Energy > 0.6 && fv.RhythmicComplexity < 0.5:
		prompt = llm.BuildRockPatternPrompt(style, styleDesc, userPrompt, bpm, bars, key, seed)
		systemPrompt = "You are a rock musician. Return the pattern as strict JSON."
		promptType = "rock"
	default:
		prompt = llm.BuildBeatPrompt(style, styleDesc, userPrompt, bpm, bars, key, seed)
		systemPrompt = "You are a creative beat maker. Return the pattern as strict JSON."
	}

	result, err := client.JSONWithTemp(systemPrompt, prompt, 0.8)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("generate_patterns: %w", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal patterns: %w", err)
	}

	var beat BeatResponse
	if err := json.Unmarshal(data, &beat); err != nil {
		return nil, nil, nil, fmt.Errorf("parse patterns: %w", err)
	}

	fmt.Printf("[Agent] generate_patterns [%s] ->style=%s  kick=%d  snare=%d  hihat=%d  bass=%d steps  melody=%d notes\n",
		promptType, beat.Style, sum(beat.DrumPattern.Kick), sum(beat.DrumPattern.Snare),
		sum(beat.DrumPattern.Hihat), countNonZero(beat.BassPattern), len(beat.MelodyNotes))

	parsedKey, _ := parseKeyMode(key)

	eventsByTrack := make(map[string][]schema.NoteEvent)
	eventsByTrack["drums"] = drumPatternToEvents(beat.DrumPattern, bars)
	eventsByTrack["bass"] = bassPatternToEvents(beat.BassPattern, bars, parsedKey)
	eventsByTrack["lead"] = melodyNotesToEvents(beat.MelodyNotes, bars)

	// Apply performance expression (crescendo/fade-out/humanize/microtiming).
	ApplyPerformance(eventsByTrack, beat.Performance, bars, bpm, beat.Style)

	// Generate CC events + pitch bend.
	ccEvents := GenerateCCEvents(beat.Performance, bars, eventsByTrack)
	pbEvents := GeneratePitchBend(beat.Performance)

	return eventsByTrack, ccEvents, pbEvents, nil
}

// GenerateChordPad creates chord pad NoteEvents from a song plan's chord progression.
// Each chord plays the full triad across multiple octaves for a warm pad sound.
// Returns events keyed by "chords" added to evMap.
func GenerateChordPad(plan *schema.SongPlan, evMap map[string][]schema.NoteEvent) {
	if plan == nil || len(plan.ChordProgression) == 0 {
		return
	}
	var events []schema.NoteEvent
	beatsPerBar := 4

	for _, cp := range plan.ChordProgression {
		if cp.Bar >= plan.TotalBars {
			break
		}
		startBeat := float64(cp.Bar * beatsPerBar)

		// Extract chord root and quality.
		root, isMinor := parseChordRoot(cp.Chord)

		// Map root to MIDI pitches at octave 3 and 4.
		notes3 := chordPitches(root, isMinor, 3) // C3 range
		notes4 := chordPitches(root, isMinor, 4) // C4 range

		// Play all chord tones as a pad: long duration, moderate velocity.
		allPitches := append(notes3, notes4...)
		for _, pitch := range allPitches {
			if pitch >= 21 && pitch <= 108 {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: pitch,
					StartBeat:    startBeat,
					DurationBeat: float64(beatsPerBar) * 0.95, // almost full bar
					Velocity:     65 + (cp.Bar % 3)*5,          // slight variation
				})
			}
		}
	}

	evMap["chords"] = events
}

// parseChordRoot returns the root note name and whether the chord is minor.
func parseChordRoot(chord string) (string, bool) {
	if chord == "" {
		return "C", false
	}
	if chord[len(chord)-1] == 'm' {
		return chord[:len(chord)-1], true
	}
	return chord, false
}

// chordPitches returns the MIDI pitches for a major or minor triad at a given octave.
func chordPitches(root string, minor bool, octave int) []int {
	semi := map[string]int{
		"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3,
		"E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8,
		"Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11,
	}
	base := (octave + 1) * 12
	r, ok := semi[root]
	if !ok {
		r = 0
	}
	rootPitch := base + r
	third := 4
	if minor {
		third = 3
	}
	return []int{rootPitch, rootPitch + third, rootPitch + 7}
}

// GeneratePitchBend creates pitch bend events from a Performance object.
func GeneratePitchBend(perf *Performance) []schema.PitchBendEvent {
	if perf == nil || len(perf.PitchBend) == 0 {
		return nil
	}
	var events []schema.PitchBendEvent
	for _, pb := range perf.PitchBend {
		val := pb.Value
		if val < 0 {
			val = 0
		}
		if val > 16383 {
			val = 16383
		}
		events = append(events, schema.PitchBendEvent{Bar: pb.Bar, Value: val})
	}
	return events
}

// GenerateCCEvents creates CC events (expression curve + sustain pedal + vibrato) from a
// Performance object. Callers add these to TrackIR.CCEvents before rendering.
func GenerateCCEvents(perf *Performance, totalBars int, eventsByTrack map[string][]schema.NoteEvent) []schema.CCEvent {
	if perf == nil {
		// Even without performance data, add a simple expression curve.
		// Crescendo first half, diminuendo last quarter.
		var events []schema.CCEvent
		for bar := 0; bar < totalBars; bar++ {
			val := 60
			if bar < totalBars/2 {
				// Rise from 50 to 100.
				progress := float64(bar) / float64(totalBars/2)
				val = 50 + int(progress*50)
			} else {
				// Hold then fade.
				progress := float64(bar-totalBars/2) / float64(totalBars/2)
				val = 100 - int(progress*60)
				if val < 30 {
					val = 30
				}
			}
			events = append(events, schema.CCEvent{Bar: float64(bar), Controller: 11, Value: val})
		}
		return events
	}

	var events []schema.CCEvent

	// Expression curve ->CC11.
	if len(perf.ExpressionCurve) > 0 {
		for _, pt := range perf.ExpressionCurve {
			events = append(events, schema.CCEvent{
				Bar: pt.Bar, Controller: 11, Value: clampVal(pt.Value),
			})
		}
	} else {
		// Default expression: 80% throughout.
		for bar := 0; bar < totalBars; bar++ {
			events = append(events, schema.CCEvent{Bar: float64(bar), Controller: 11, Value: 100})
		}
	}

	// Sustain pedal events ->CC64.
	if len(perf.SustainPedal) > 0 {
		for _, sp := range perf.SustainPedal {
			val := 0
			if sp.On {
				val = 127
			}
			events = append(events, schema.CCEvent{Bar: sp.Bar, Controller: 64, Value: val})
		}
	}

	// Vibrato (CC1) on long notes.
	vibEvents := addVibratoCC(eventsByTrack, 480)
	events = append(events, vibEvents...)

	// Ritardando: extra expression decay in last 2 bars.
	ritEvents := addRitardandoEvents(totalBars)
	events = append(events, ritEvents...)

	return events
}

func clampVal(v int) int {
	if v < 0 {
		return 0
	}
	if v > 127 {
		return 127
	}
	return v
}

// newRand creates a deterministic random generator seeded per bar.
func newRand(seed int64) *randWrapper {
	return &randWrapper{seed: seed, pos: 0}
}

// randWrapper is a simple linear-congruential generator (no external deps).
type randWrapper struct {
	seed int64
	pos  int
}

func (r *randWrapper) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	v := int(hash64(r.seed + int64(r.pos)))
	r.pos++
	if v < 0 {
		v = -v
	}
	return v % n
}

func (r *randWrapper) Float64() float64 {
	return float64(r.Intn(1000000)) / 1000000.0
}

func hash64(x int64) int64 {
	// Simple 64-bit hash (using uint64 to avoid overflow on 32-bit archs).
	u := uint64(x)
	u = (u ^ (u >> 30)) * 0xbf58476d1ce4e5b9
	u = (u ^ (u >> 27)) * 0x94d049bb133111eb
	u = u ^ (u >> 31)
	return int64(u)
}

// drumPatternToEvents converts a 16-step drum pattern to NoteEvents, repeating for each bar
// with per-bar randomization (ghost notes, hit skipping, velocity jitter).
func drumPatternToEvents(dp DrumPattern, totalBars int) []schema.NoteEvent {
	stepDuration := 0.25
	var events []schema.NoteEvent

	for bar := 0; bar < totalBars; bar++ {
		rng := newRand(int64(bar)*2000 + 42)
		barStart := float64(bar) * 4.0

		for step := 0; step < 16; step++ {
			beatPos := barStart + float64(step)*stepDuration

			// Kick: 15% skip, 10% ghost note.
			playKick := step < len(dp.Kick) && dp.Kick[step] == 1
			if playKick && rng.Float64() < 0.15 {
				playKick = false
			}
			if !playKick && rng.Float64() < 0.10 && step > 0 {
				playKick = true
			}
			if playKick {
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 36, DrumName: "kick",
					StartBeat: beatPos, DurationBeat: 0.1,
					Velocity: clampVel(98 + int(rng.Intn(20))),
				})
			}

			// Snare: 10% skip, 5% flam.
			playSnare := step < len(dp.Snare) && dp.Snare[step] == 1
			if playSnare && rng.Float64() < 0.10 {
				playSnare = false
			}
			if playSnare {
				vel := clampVel(94 + int(rng.Intn(18)))
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: 38, DrumName: "snare",
					StartBeat: beatPos, DurationBeat: 0.1, Velocity: vel,
				})
				if rng.Float64() < 0.05 {
					events = append(events, schema.NoteEvent{
						Type: "note", Pitch: 38, DrumName: "snare",
						StartBeat: beatPos + 0.06, DurationBeat: 0.08,
						Velocity: clampVel(vel - 15),
					})
				}
			}

			// Hi-hat: varied velocity, 8% open hat.
			playHat := step < len(dp.Hihat) && dp.Hihat[step] == 1
			if playHat {
				hatPitch := 42
				hatVel := clampVel(70 + int(rng.Intn(30)))
				if rng.Float64() < 0.08 {
					hatPitch = 46
					hatVel = clampVel(hatVel + 10)
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: hatPitch, DrumName: "closed_hat",
					StartBeat: beatPos, DurationBeat: 0.08, Velocity: hatVel,
				})
			}
		}
	}
	return events
}

// bassPatternToEvents converts a 16-step bass pattern to NoteEvents with per-bar variation.
func bassPatternToEvents(pattern []int, totalBars int, keyRoot string) []schema.NoteEvent {
	rootSemitones := map[string]int{
		"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3,
		"E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8,
		"Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11,
	}
	semi, ok := rootSemitones[keyRoot]
	if !ok {
		semi = 0
	}
	basePitch := 36 + semi
	stepDuration := 0.25
	var events []schema.NoteEvent

	for bar := 0; bar < totalBars; bar++ {
		rng := newRand(int64(bar)*3000 + 42)
		barStart := float64(bar) * 4.0
		for step := 0; step < 16; step++ {
			beatPos := barStart + float64(step)*stepDuration
			offset := 0
			if step < len(pattern) {
				offset = pattern[step]
			}
			// 20% chance to add a passing tone (offset ±5 semitones).
			if offset == 0 && rng.Float64() < 0.20 {
				offset = []int{-5, -3, 2, 3, 5, 7}[rng.Intn(6)]
			}
			// 15% chance to octave-jump the bass.
			if rng.Float64() < 0.15 {
				offset += 12
			}
			pitch := clampNote(basePitch + offset)
			duration := stepDuration * (0.5 + rng.Float64()) // 0.5-1.5 steps
			vel := clampVel(88 + int(rng.Intn(20)))
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat: beatPos, DurationBeat: duration, Velocity: vel,
			})
		}
	}
	return events
}

// melodyNotesToEvents converts LLM melody notes to NoteEvents,
// repeating the base pattern across all bars with per-bar randomization
// (pitch, duration, velocity jitter, random rests, occasional extra notes).
func melodyNotesToEvents(notes []MelodyNote, totalBars int) []schema.NoteEvent {
	stepDuration := 0.25
	var events []schema.NoteEvent

	if len(notes) > 0 {
		for bar := 0; bar < totalBars; bar++ {
			barSeed := int64(bar)*1000 + 42
			rng := newRand(barSeed)
			barStart := float64(bar) * 4.0

			for _, n := range notes {
				// 10% chance to skip this note (random rest).
				if rng.Float64() < 0.10 {
					continue
				}

				startBeat := barStart + float64(n.Start)*stepDuration
				duration := float64(n.Duration) * stepDuration
				if duration < 0.1 {
					duration = 0.1
				}

				// Pitch: 70% stay same, 30% shift ±2 semitones (scale neighbor).
				pitch := clampNote(n.Pitch)
				if rng.Float64() < 0.30 {
					pitch = clampNote(pitch + int(rng.Intn(5)) - 2) // -2..+2
				}

				vel := n.Velocity
				if vel < 1 || vel > 127 {
					vel = 90
				}

				// Duration jitter: ±30%.
				durFactor := 0.7 + rng.Float64()*0.6 // 0.7-1.3
				actualDuration := duration * durFactor

				// Velocity jitter: ±12.
				actualVel := vel + int(rng.Intn(25)) - 12

				// Articulation-based adjustments.
				switch n.Articulation {
				case "staccato":
					actualDuration = actualDuration * 0.4
					actualVel = actualVel + 8
				case "accent":
					actualVel = actualVel + 15
				case "legato":
					actualDuration = actualDuration * 1.1
				}

				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: pitch,
					StartBeat: startBeat, DurationBeat: clampDuration(actualDuration),
					Velocity: clampVel(actualVel),
				})
			}

			// 15% chance to insert an extra grace note.
			if rng.Float64() < 0.15 && len(notes) > 0 {
				template := notes[rng.Intn(len(notes))]
				extraPitch := clampNote(template.Pitch + int(rng.Intn(7)) - 3)
				extraStart := barStart + float64(rng.Intn(16))*stepDuration
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: extraPitch,
					StartBeat: extraStart, DurationBeat: 0.25,
					Velocity: clampVel(60 + int(rng.Intn(40))),
				})
			}
		}
		return events
	}

	// Fallback placeholder.
	for bar := 0; bar < totalBars; bar++ {
		rng := newRand(int64(bar)*1000 + 42)
		events = append(events, schema.NoteEvent{
			Type: "note", Pitch: 72 + int(rng.Intn(5))-2,
			StartBeat: float64(bar) * 4.0,
			DurationBeat: 1.0 + rng.Float64()*2.0,
			Velocity: clampVel(70 + int(rng.Intn(30))),
		})
	}
	return events
}

// ApplyPerformance post-processes generated events with expression curves,
// crescendo/diminuendo, fade-out, velocity humanization, microtiming, and auto articulation.
// It modifies eventsByTrack in place. style is used for style-specific microtiming.
func ApplyPerformance(eventsByTrack map[string][]schema.NoteEvent, perf *Performance, totalBars, bpm int, style string) {
	if perf == nil {
		// Even without LLM data, apply all humanization layers.
		applyMicrotiming(eventsByTrack, style, totalBars)
		applyAutoArticulation(eventsByTrack)
		applyFadeOut(eventsByTrack, totalBars)
		applyGaussianVelocity(eventsByTrack, totalBars)
		return
	}

	// 1. Build per-bar expression multiplier.
	barExpr := make([]float64, totalBars)
	if len(perf.ExpressionCurve) > 0 {
		for bar := 0; bar < totalBars; bar++ {
			expr := interpolateExpression(perf.ExpressionCurve, float64(bar))
			barExpr[bar] = float64(expr) / 100.0
		}
	} else {
		for bar := 0; bar < totalBars; bar++ {
			barExpr[bar] = 1.0
		}
	}

	globalScale := perf.GlobalDynamics
	if globalScale <= 0 {
		globalScale = 1.0
	}

	// 2. Apply expression + dynamics to each event's velocity.
	for trackID := range eventsByTrack {
		for i := range eventsByTrack[trackID] {
			e := &eventsByTrack[trackID][i]
			bar := int(e.StartBeat) / 4
			if bar >= totalBars { bar = totalBars - 1 }
			if bar < 0 { bar = 0 }
			exprScale := barExpr[bar]
			if exprScale < 0.1 { exprScale = 0.1 }
			e.Velocity = clampVel(int(float64(e.Velocity) * exprScale * globalScale))
		}
	}

	// 3. Microtiming offset per style.
	applyMicrotiming(eventsByTrack, style, totalBars)

	// 4. Auto articulation: detect legato/staccato from intervals.
	applyAutoArticulation(eventsByTrack)

	// 5. Ending fade-out (last 25%).
	fadeBars := totalBars / 4
	if fadeBars < 1 { fadeBars = 1 }
	applyFadeOutTail(eventsByTrack, totalBars, fadeBars)

	// 6. Gaussian velocity jitter (instead of uniform ±3).
	applyGaussianVelocity(eventsByTrack, totalBars)
}

// --- performance helpers ---

// interpolateExpression finds the expression value at a given bar position
// by linear interpolation of the expression curve.
func interpolateExpression(curve []ExpressionPoint, bar float64) int {
	if len(curve) == 0 {
		return 100
	}
	if bar <= curve[0].Bar {
		return curve[0].Value
	}
	if bar >= curve[len(curve)-1].Bar {
		return curve[len(curve)-1].Value
	}
	for i := 0; i < len(curve)-1; i++ {
		if bar >= curve[i].Bar && bar < curve[i+1].Bar {
			t := (bar - curve[i].Bar) / (curve[i+1].Bar - curve[i].Bar)
			val := float64(curve[i].Value) + t*float64(curve[i+1].Value-curve[i].Value)
			return int(val)
		}
	}
	return 100
}

// applyFadeOut gradually lowers velocity in the last bars.
func applyFadeOut(eventsByTrack map[string][]schema.NoteEvent, totalBars int) {
	fadeStart := totalBars - totalBars/4
	if fadeStart < 1 {
		fadeStart = 1
	}
	applyFadeOutTail(eventsByTrack, totalBars, totalBars-fadeStart)
}

func applyFadeOutTail(eventsByTrack map[string][]schema.NoteEvent, totalBars, fadeBars int) {
	if fadeBars <= 0 {
		return
	}
	fadeStart := totalBars - fadeBars

	for trackID := range eventsByTrack {
		for i := range eventsByTrack[trackID] {
			e := &eventsByTrack[trackID][i]
			bar := int(e.StartBeat) / 4
			if bar >= fadeStart {
				// Linear fade from 1.0 to 0.15.
				progress := float64(bar-fadeStart) / float64(fadeBars)
				factor := 1.0 - progress*0.85
				if factor < 0.15 {
					factor = 0.15
				}
				e.Velocity = clampVel(int(float64(e.Velocity) * factor))
			}
		}
	}
}

// ApplySectionalDynamics modifies events based on song section structure.
// Uses the PlanSong sections (intro/verse/chorus/outro) to vary density,
// velocity, and add drum fills at transitions.
// sectionsRaw is songPlanMap["song_plan"]["sections"] as []any.
func ApplySectionalDynamics(eventsByTrack map[string][]schema.NoteEvent, songPlanRaw map[string]any, totalBars int) {
	// Extract sections from song plan.
	var sections []struct {
		ID         string  `json:"id"`
		StartBar   int     `json:"start_bar"`
		LengthBars int     `json:"length_bars"`
		Energy     float64 `json:"energy"`
	}

	spRaw, ok := songPlanRaw["song_plan"]
	if !ok {
		return
	}
	spMap, ok := spRaw.(map[string]any)
	if !ok {
		return
	}
	secsRaw, ok := spMap["sections"]
	if !ok {
		return
	}
	secsList, ok := secsRaw.([]any)
	if !ok || len(secsList) == 0 {
		return
	}

	for _, s := range secsList {
		sMap, ok := s.(map[string]any)
		if !ok {
			continue
		}
		var sec struct {
			ID         string  `json:"id"`
			StartBar   int     `json:"start_bar"`
			LengthBars int     `json:"length_bars"`
			Energy     float64 `json:"energy"`
		}
		if id, ok := sMap["id"].(string); ok {
			sec.ID = id
		}
		if sb, ok := sMap["start_bar"].(float64); ok {
			sec.StartBar = int(sb)
		}
		if lb, ok := sMap["length_bars"].(float64); ok {
			sec.LengthBars = int(lb)
		}
		if en, ok := sMap["energy"].(float64); ok {
			sec.Energy = en
		}
		sections = append(sections, sec)
	}
	if len(sections) == 0 {
		return
	}

	// Build per-bar energy map.
	barEnergy := make([]float64, totalBars)
	for i := range barEnergy {
		barEnergy[i] = 0.5 // default
	}
	for _, sec := range sections {
		for b := sec.StartBar; b < sec.StartBar+sec.LengthBars && b < totalBars; b++ {
			barEnergy[b] = sec.Energy
		}
	}

	// Find transition bars (last bar of each section before a new one).
	transitions := make(map[int]bool)
	for i := 1; i < len(sections); i++ {
		prevEnd := sections[i-1].StartBar + sections[i-1].LengthBars - 1
		if prevEnd >= 0 && prevEnd < totalBars {
			transitions[prevEnd] = true
		}
	}

	for trackID := range eventsByTrack {
		track := eventsByTrack[trackID]
		if len(track) == 0 {
			continue
		}

		// Phase 1: Scale velocity by section energy.
		for i := range track {
			bar := int(track[i].StartBeat) / 4
			if bar >= totalBars {
				bar = totalBars - 1
			}
			if bar < 0 {
				bar = 0
			}
			energy := barEnergy[bar]
			// Energy 0.3 = soft intro, 0.9 = loud chorus
			energyFactor := 0.4 + energy*0.8 // maps 0.3-0.9 ->0.64-1.12
			track[i].Velocity = clampVel(int(float64(track[i].Velocity) * energyFactor))
		}

		// Phase 2: Density variation --remove notes in low-energy sections.
		if trackID == "drums" {
			continue // drums handled separately
		}
		var filtered []schema.NoteEvent
		for i := range track {
			bar := int(track[i].StartBeat) / 4
			if bar >= totalBars {
				bar = totalBars - 1
			}
			if bar < 0 {
				bar = 0
			}
			energy := barEnergy[bar]
			// In low-energy sections, remove 30% of notes randomly.
			if energy < 0.5 {
				h := hash64(int64(i*100+bar)*37) % 10
				if h < 3 {
					continue // skip 30%
				}
			}
			filtered = append(filtered, track[i])
		}
		eventsByTrack[trackID] = filtered

		// Phase 3: Add extra emphasis notes at transitions (last bar before section change).
		if trackID != "drums" && trackID != "bass" {
			var withFills []schema.NoteEvent
			withFills = append(withFills, eventsByTrack[trackID]...)
			for bar := range transitions {
				if bar >= totalBars-1 {
					continue
				}
				// Add 2-3 extra notes in the transition bar.
				barStart := float64(bar) * 4.0
				for k := 0; k < 3; k++ {
					h := hash64(int64(bar*1000+k)*53)
					pos := float64(h%16) * 0.25
					pitchOff := int(h % 7)
					withFills = append(withFills, schema.NoteEvent{
						Type: "note", Pitch: clampNote(64 + pitchOff - 3),
						StartBeat: barStart + pos, DurationBeat: 0.25,
						Velocity: clampVel(90 + int(h%20)),
					})
				}
			}
			eventsByTrack[trackID] = withFills
		}
	}

	// Phase 4: Drum fills at transitions.
	if drums, ok := eventsByTrack["drums"]; ok {
		for bar := range transitions {
			if bar >= totalBars-1 {
				continue
			}
			barStart := float64(bar) * 4.0
			// Add extra drum hits: rapid kick+snare fill.
			for k := 0; k < 8; k++ {
				h := hash64(int64(bar*2000+k)*61)
				pos := float64(h%16) * 0.25
				pitch := 36 // kick
				if h%3 == 1 {
					pitch = 38 // snare
				} else if h%3 == 2 {
					pitch = 42 // hat
				}
				drums = append(drums, schema.NoteEvent{
					Type: "note", Pitch: pitch,
					StartBeat: barStart + pos, DurationBeat: 0.1,
					Velocity: clampVel(100 + int(h%20)),
				})
			}
		}
		eventsByTrack["drums"] = drums
	}
}

// applyGaussianVelocity replaces uniform jitter with a gaussian-like distribution.
// Uses a simple 2-step uniform approximation for the gaussian shape.
func applyGaussianVelocity(eventsByTrack map[string][]schema.NoteEvent, totalBars int) {
	for trackID := range eventsByTrack {
		for i := range eventsByTrack[trackID] {
			e := &eventsByTrack[trackID][i]
			bar := int(e.StartBeat) / 4
			if bar >= totalBars { bar = totalBars - 1 }
			if bar < 0 { bar = 0 }

			// Strong beat (beat 1) ->+10-20%
			beatPos := int(e.StartBeat*4) % 4
			beatBoost := 0
			switch beatPos {
			case 0: beatBoost = 12 // downbeat
			case 2: beatBoost = 5  // beat 3 (secondary strong)
			default: beatBoost = -3 // weak beats slightly softer
			}

			// Gaussian-like jitter: sum of two uniforms = ~normal distribution.
			jitter := 0
			for j := 0; j < 2; j++ {
				jitter += int(hash64(int64(i*1000+bar*100+j)) % 11) - 5
			}
			jitter = jitter / 2 // scale to ±5 range

			e.Velocity = clampVel(e.Velocity + beatBoost + jitter)
		}
	}
}

// applyMicrotiming offsets note start times per style for a more human feel.
func applyMicrotiming(eventsByTrack map[string][]schema.NoteEvent, style string, totalBars int) {
	// Determine groove style from the style name.
	hasSwing := style == "jazz" || style == "swing" || style == "lofi"
	isLaidback := style == "lofi" || style == "westCoast" || style == "r&b"
	isPushed := style == "drill" || style == "jerseyClub"

	for trackID := range eventsByTrack {
		if trackID == "drums" {
			continue // drums keep strict timing
		}
		for i := range eventsByTrack[trackID] {
			e := &eventsByTrack[trackID][i]
			step := int(e.StartBeat*4) % 2 // 0=on beat, 1=off beat

			var offset float64
			if hasSwing && step == 1 {
				// Swing: push offbeat 16th notes forward.
				offset = 0.03 + float64(hash64(int64(i)*37)%5)*0.004
			}
			if isLaidback && step == 0 {
				// Laidback: slightly delay strong beats.
				offset = 0.02 + float64(hash64(int64(i)*41)%4)*0.005
			}
			if isPushed && step == 1 {
				// Pushed: anticipate offbeats.
				offset = -(0.02 + float64(hash64(int64(i)*43)%4)*0.005)
			}

			e.StartBeat += offset
		}
	}
}

// applyAutoArticulation detects legato/staccato from note intervals and timing.
func applyAutoArticulation(eventsByTrack map[string][]schema.NoteEvent) {
	for trackID := range eventsByTrack {
		track := eventsByTrack[trackID]
		if len(track) < 2 {
			continue
		}

		for i := 0; i < len(track)-1; i++ {
			curr := &track[i]
			next := &track[i+1]
			gap := next.StartBeat - (curr.StartBeat + curr.DurationBeat)
			interval := absInt(next.Pitch - curr.Pitch)

			// Small interval (< 3 semitones) + small gap ->legato.
			if interval <= 3 && gap < 0.15 {
				curr.DurationBeat = next.StartBeat - curr.StartBeat + 0.02 // slight overlap
			}

			// Large gap (> 0.5 beats) ->staccato (shorten 70%).
			if gap > 0.5 {
				curr.DurationBeat = curr.DurationBeat * 0.7
			}

			// Large interval (> 7 semitones) + small gap ->accent next note.
			if interval > 7 && gap < 0.1 {
				next.Velocity = clampVel(next.Velocity + 8)
			}
		}
	}
}

// clampVel clamps velocity to 1--27.
func clampVel(v int) int {
	if v < 1 {
		return 1
	}
	if v > 127 {
		return 127
	}
	return v
}

// clampDuration ensures a note has minimum duration.
func clampDuration(d float64) float64 {
	if d < 0.05 {
		return 0.05
	}
	return d
}

// clampNote clamps pitch to 21--08.
func clampNote(p int) int {
	if p < 21 { return 21 }
	if p > 108 { return 108 }
	return p
}

func absInt(x int) int {
	if x < 0 { return -x }
	return x
}

// addVibratoCC inserts CC1 modulation events on long lead notes for vibrato effect.
func addVibratoCC(eventsByTrack map[string][]schema.NoteEvent, tpb int) []schema.CCEvent {
	var ccEvents []schema.CCEvent
	for trackID := range eventsByTrack {
		if trackID == "drums" || trackID == "bass" {
			continue
		}
		for _, e := range eventsByTrack[trackID] {
			durationMs := e.DurationBeat * 60000.0 / 120.0 // rough at 120 BPM
			if durationMs < 400 {
				continue // too short for vibrato
			}
			// Add 4 vibrato points per note.
			steps := 4
			for i := 0; i < steps; i++ {
				bar := e.StartBeat + e.DurationBeat*float64(i)/float64(steps)
				// Sine-like modulation: swing between 58-70 around center 64.
				phase := 2 * 3.14159 * float64(i) / float64(steps)
				value := 64 + int(8.0*sin(phase))
				ccEvents = append(ccEvents, schema.CCEvent{
					Bar: bar, Controller: 1, Value: clampVal(value),
				})
			}
		}
	}
	return ccEvents
}

// sin is a simple sine approximation for vibrato LFO.
func sin(x float64) float64 {
	// Taylor series: sin(x) >-x - x³/6 + x>-120
	x2 := x * x
	return x - x*x2/6.0 + x*x2*x2/120.0
}

// addRitardandoEvents slows tempo in the last bars by inserting
// gradually increasing tempo meta events (as bar-position annotations).
// Returns tempo curve points for the MIDI renderer (not yet implemented in render).
func addRitardandoEvents(totalBars int) []schema.CCEvent {
	// For now this is a placeholder: real ritardando requires tempo events
	// in the meta track, which needs renderer support.
	// We approximate by returning expression events that decay faster.
	if totalBars < 4 {
		return nil
	}
	var events []schema.CCEvent
	for bar := totalBars - 2; bar < totalBars; bar++ {
		expr := 100 - (bar-(totalBars-2))*30
		if expr < 20 {
			expr = 20
		}
		events = append(events, schema.CCEvent{
			Bar: float64(bar), Controller: 11, Value: clampVal(expr),
		})
	}
	return events
}

// parseKeyMode splits a key string like "C minor" into root ("C") and mode ("minor").
func parseKeyMode(key string) (string, string) {
	parts := splitN(key, " ", 2)
	root := parts[0]
	mode := "minor"
	if len(parts) > 1 {
		mode = parts[1]
	}
	return root, mode
}

// buildChordProgression creates a simple chord progression based on key.
func buildChordProgression(root, mode string, bars int) []schema.ChordChange {
	// In minor: i - VI - III - VII
	// In major: I - V - vi - IV
	var degrees []string
	if mode == "minor" || mode == "Minor" || mode == "m" {
		// i - bVI - bIII - bVII
		degrees = []string{
			root + "m",
			intervalChord(root, 8, true),  // bVI
			intervalChord(root, 3, true),  // bIII
			intervalChord(root, 10, true), // bVII
		}
	} else {
		degrees = []string{
			root,                              // I
			intervalChord(root, 7, false),     // V
			intervalChord(root, 9, false),     // vi
			intervalChord(root, 5, false),     // IV
		}
	}

	var prog []schema.ChordChange
	for bar := 0; bar < bars; bar++ {
		prog = append(prog, schema.ChordChange{
			Bar:   bar,
			Chord: degrees[bar%len(degrees)],
		})
	}
	return prog
}

// intervalChord returns the chord at a given semitone interval from root.
// isMinor=true means the chord should be minor.
func intervalChord(root string, semitones int, isMinor bool) string {
	noteToSemi := map[string]int{
		"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3,
		"E": 4, "F": 5, "F#": 6, "Gb": 6, "G": 7, "G#": 8,
		"Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11,
	}
	semiToNote := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

	rootSemi, ok := noteToSemi[root]
	if !ok {
		return "C"
	}
	targetSemi := (rootSemi + semitones) % 12
	targetNote := semiToNote[targetSemi]

	if isMinor {
		return targetNote + "m"
	}
	return targetNote
}

// --- helpers ---

func sum(s []int) int {
	total := 0
	for _, v := range s {
		total += v
	}
	return total
}

func countNonZero(s []int) int {
	count := 0
	for _, v := range s {
		if v != 0 {
			count++
		}
	}
	return count
}

// splitN splits a string by a delimiter, returning at most n parts.
func splitN(s, sep string, n int) []string {
	out := make([]string, 0, n)
	start := 0
	for i := 0; i < n-1 && start < len(s); i++ {
		idx := indexOf(s, sep, start)
		if idx < 0 {
			break
		}
		out = append(out, s[start:idx])
		start = idx + len(sep)
	}
	out = append(out, s[start:])
	return out
}

func indexOf(s, sep string, start int) int {
	if start >= len(s) {
		return -1
	}
	for i := start; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

// GenerateMelodyNotes calls the LLM to compose a full note sequence for a single instrument.
// Unlike GeneratePatterns (which uses a 16-step grid for hip-hop patterns),
// this allows arbitrary fractional beat positions for expressive, natural-sounding melodies.
//
// Parameters:
//   - instrument: "lead", "bass", "pipa", "guzheng", etc.
//   - chordProgJSON: JSON string of the chord progression array
//   - totalBeats: total duration in beats (bars * beatsPerBar)
func GenerateMelodyNotes(client *llm.Client, instrument, key, scale, styleDesc, featureVec, chordProgJSON string, bpm, totalBeats int) ([]schema.NoteEvent, error) {
	seed := fmt.Sprintf("%d", hash64(int64(bpm*totalBeats+len(instrument))*31+time.Now().UnixNano()))
	prompt := llm.BuildNoteSequencePrompt(instrument, key, scale, chordProgJSON, styleDesc, featureVec, bpm, totalBeats, seed)
	systemPrompt := "You are a professional composer. Return JSON only."

	result, err := client.JSONWithTemp(systemPrompt, prompt, 0.85)
	if err != nil {
		return nil, fmt.Errorf("generate_melody(%s): %w", instrument, err)
	}

	// Parse events array from result.
	eventsRaw, ok := result["events"]
	if !ok {
		// Try alternative key names.
		if ev, ok2 := result["notes"]; ok2 {
			eventsRaw = ev
		} else {
			return nil, fmt.Errorf("generate_melody(%s): response missing 'events' key", instrument)
		}
	}

	eventsList, ok := eventsRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("generate_melody(%s): 'events' is not an array", instrument)
	}

	var notes []schema.NoteEvent
	for _, item := range eventsList {
		ev, ok := item.(map[string]any)
		if !ok {
			continue
		}
		pitch := getInt(ev, "pitch")
		start := toFloat64(ev["start_beat"])
		dur := toFloat64(ev["duration_beat"])
		vel := getInt(ev, "velocity")
		art := getString(ev, "articulation")
		// Apply articulation to note parameters.
		switch art {
		case "staccato", "staccatissimo":
			dur = dur * 0.4
			vel += 5
		case "legato", "tenuto":
			dur = dur * 1.3
		case "accent", "marcato", "sfz":
			vel += 15
			if vel > 127 {
				vel = 127
			}
		case "pizzicato":

			dur = dur * 0.3
			vel -= 5
		case "tremolo":

			dur = dur * 0.15
		vel = vel - 10
			if vel < 1 {
				vel = 1
			}
		case "power_chord":

			vel += 10
			dur = dur * 1.2
		case "bend":

			vel += 5
		}
		if vel < 1 {
			vel = 80
		}
		if pitch < 21 || pitch > 108 {
			continue
		}
		if start < 0 {
			start = 0
		}
		if dur <= 0.05 {
			dur = 0.125
		}
		notes = append(notes, schema.NoteEvent{
			Type: "note", Pitch: pitch,
			StartBeat: start, DurationBeat: dur, Velocity: vel,
		})
	}

	fmt.Printf("[Agent] generate_melody(%s) ->%d notes\n", instrument, len(notes))
	return notes, nil
}

// ChordProgressionToJSON converts the plan's chord progression array to a JSON string.
func ChordProgressionToJSON(cp []schema.ChordChange) string {
	b, err := json.Marshal(cp)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// SummarizeMelody creates a compact text summary of a lead melody for the bass generator.
// Example: "73 notes, range G3-C5, key notes: G4(8x) A4(6x) C5(4x), energy peaks at bar 10-12"
func SummarizeMelody(notes []schema.NoteEvent) string {
	if len(notes) == 0 {
		return "no melody"
	}

	// Count pitch frequencies.
	pitchCount := make(map[int]int)
	minP, maxP := 127, 0
	totalVel := 0
	firstNotes := ""

	for i, n := range notes {
		pitchCount[n.Pitch]++
		if n.Pitch < minP {
			minP = n.Pitch
		}
		if n.Pitch > maxP {
			maxP = n.Pitch
		}
		totalVel += n.Velocity
		if i < 5 {
			if firstNotes != "" {
				firstNotes += ", "
			}
			firstNotes += fmt.Sprintf("%d@%.1f", n.Pitch, n.StartBeat)
		}
	}

	// Find top pitches.
	type pc struct {
		p int
		c int
	}
	var sorted []pc
	for p, c := range pitchCount {
		sorted = append(sorted, pc{p, c})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].c > sorted[j].c
	})

	topStr := ""
	for i, p := range sorted {
		if i >= 3 {
			break
		}
		if topStr != "" {
			topStr += ", "
		}
		topStr += fmt.Sprintf("%d(x%d)", p.p, p.c)
	}

	avgVel := totalVel / len(notes)
	return fmt.Sprintf("%d notes, range %d-%d, avg_vel=%d, top: [%s], start: %s",
		len(notes), minP, maxP, avgVel, topStr, firstNotes)
}

// GenerateBassFromMelody calls the LLM to compose a bass line that follows a lead melody.
// The bass locks with the melody's chord roots and strong beats.
func GenerateBassFromMelody(client *llm.Client, leadNotes []schema.NoteEvent,
	key, scale, styleDesc, featureVec, chordProgJSON string,
	bpm, totalBeats int) ([]schema.NoteEvent, error) {

	leadSummary := SummarizeMelody(leadNotes)
	seed := fmt.Sprintf("%d", hash64(int64(bpm*totalBeats+len(leadSummary))*31+time.Now().UnixNano()))
	prompt := llm.BuildBassFromMelodyPrompt(key, scale, chordProgJSON, styleDesc, featureVec, leadSummary, bpm, totalBeats, seed)
	systemPrompt := "You are a bassist. Return JSON only."

	result, err := client.JSONWithTemp(systemPrompt, prompt, 0.8)
	if err != nil {
		return nil, fmt.Errorf("generate_bass_from_melody: %w", err)
	}

	eventsRaw, ok := result["events"]
	if !ok {
		return nil, fmt.Errorf("generate_bass_from_melody: missing 'events' key")
	}

	eventsList, ok := eventsRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("generate_bass_from_melody: 'events' not an array")
	}

	var notes []schema.NoteEvent
	for _, item := range eventsList {
		ev, ok := item.(map[string]any)
		if !ok {
			continue
		}
		pitch := getInt(ev, "pitch")
		start := toFloat64(ev["start_beat"])
		dur := toFloat64(ev["duration_beat"])
		vel := getInt(ev, "velocity")
		art := getString(ev, "articulation")
		// Apply articulation to note parameters.
		switch art {
		case "staccato", "staccatissimo":
			dur = dur * 0.4
			vel += 5
		case "legato", "tenuto":
			dur = dur * 1.3
		case "accent", "marcato", "sfz":
			vel += 15
			if vel > 127 {
				vel = 127
			}
		case "pizzicato":

			dur = dur * 0.3
			vel -= 5
		case "tremolo":

			dur = dur * 0.15
		vel = vel - 10
			if vel < 1 {
				vel = 1
			}
		case "power_chord":

			vel += 10
			dur = dur * 1.2
		case "bend":

			vel += 5
		}
		if vel < 1 {
			vel = 80
		}
		if pitch < 21 || pitch > 108 {
			continue
		}
		if start < 0 {
			start = 0
		}
		if dur <= 0.05 {
			dur = 0.25
		}
		notes = append(notes, schema.NoteEvent{
			Type: "note", Pitch: pitch,
			StartBeat: start, DurationBeat: dur, Velocity: vel,
		})
	}

	fmt.Printf("[Agent] generate_bass_from_melody ->%d notes, following lead (%s)\n",
		len(notes), leadSummary)
	return notes, nil
}
