package llm

import (
	"strings"
	"testing"
)

func TestBuildIntentParserPrompt_Basic(t *testing.T) {
	prompt := BuildIntentParserPrompt("a dark cyberpunk track", true, nil)

	if !strings.Contains(prompt, "a dark cyberpunk track") {
		t.Error("prompt should contain user input")
	}
	if !strings.Contains(prompt, "requested_tracks must include at least") {
		t.Error("prompt should enforce core tracks")
	}
	if !strings.Contains(prompt, "default to 30") {
		t.Error("prompt should mention default duration")
	}
	if !strings.Contains(prompt, "Return JSON only") {
		t.Error("prompt should demand JSON output")
	}
}

func TestBuildIntentParserPrompt_NoCoreEnforcement(t *testing.T) {
	prompt := BuildIntentParserPrompt("ambient pad", false, nil)

	if strings.Contains(prompt, "requested_tracks must include at least") {
		t.Error("should NOT enforce core tracks when false")
	}
	if !strings.Contains(prompt, "do not force fixed instrument sets") {
		t.Error("should say not to force fixed instruments")
	}
}

func TestBuildIntentParserPrompt_MaxDuration(t *testing.T) {
	maxDur := 60
	prompt := BuildIntentParserPrompt("test", true, &maxDur)

	if !strings.Contains(prompt, "<= 60") {
		t.Error("prompt should include duration cap")
	}
}

func TestBuildSongPlannerPrompt_ContainsFields(t *testing.T) {
	intentJSON := `{"intent": {"style": ["cyberpunk"], "tempo_preference": "fast"}}`
	prompt := BuildSongPlannerPrompt(intentJSON)

	required := []string{"song_plan", "bpm", "time_signature", "key", "chord_progression", "total_bars"}
	for _, field := range required {
		if !strings.Contains(prompt, field) {
			t.Errorf("prompt should contain %q", field)
		}
	}
	if !strings.Contains(prompt, "natural_minor") {
		t.Error("prompt should mention scale constraint")
	}
}

func TestBuildArrangementPlannerPrompt_ContainsFields(t *testing.T) {
	intentJSON := `{"intent": {"style": ["cyberpunk"]}}`
	songPlanJSON := `{"song_plan": {"bpm": 140, "key": {"root": "D"}}}`
	prompt := BuildArrangementPlannerPrompt(intentJSON, songPlanJSON, true)

	required := []string{"arrangement", "tracks", "drums", "channel", "program"}
	for _, field := range required {
		if !strings.Contains(prompt, field) {
			t.Errorf("prompt should contain %q", field)
		}
	}
	if !strings.Contains(prompt, "channel=9") {
		t.Error("prompt should mention drum channel 9 rule")
	}
}

func TestBuildArrangementPlannerPrompt_NoCore(t *testing.T) {
	intentJSON := `{}`
	songPlanJSON := `{}`
	prompt := BuildArrangementPlannerPrompt(intentJSON, songPlanJSON, false)

	if strings.Contains(prompt, "must include core tracks") {
		t.Error("should NOT enforce core tracks when false")
	}
}

func TestBuildTrackNoteGeneratorPrompt_ContainsConstraints(t *testing.T) {
	songPlan := `{"total_bars": 8, "time_signature": {"numerator": 4}}`
	track := `{"id": "lead", "role": "melody"}`
	prompt := BuildTrackNoteGeneratorPrompt(songPlan, track)

	constraints := []string{"pitch", "velocity", "start_beat", "duration_beat", "events"}
	for _, c := range constraints {
		if !strings.Contains(prompt, c) {
			t.Errorf("prompt should contain %q", c)
		}
	}
}

func TestStripMarkdownFences(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`{"key": "value"}`, `{"key": "value"}`},
		{"```json\n{\"key\": \"value\"}\n```", "{\"key\": \"value\"}"},
		{"```\n{\"key\": \"value\"}\n```", "{\"key\": \"value\"}"},
		{"  {\"key\": \"value\"}  ", "{\"key\": \"value\"}"},
	}
	for _, tt := range tests {
		got := StripMarkdownFences(tt.input)
		if got != tt.want {
			t.Errorf("StripMarkdownFences(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestKnowledgeEmbedded(t *testing.T) {
	// Verify key knowledge templates are non-empty.
	if len(chordKnowledge) < 500 {
		t.Error("chordKnowledge too short")
	}
	if len(instrumentKnowledge) < 500 {
		t.Error("instrumentKnowledge too short")
	}
	if len(noteKnowledge) < 500 {
		t.Error("noteKnowledge too short")
	}

	// Check for critical content in chord knowledge.
	checks := []string{"bright_pop", "dark_loop", "epic_minor", "boss_battle", "i - VI - VII - V"}
	for _, c := range checks {
		if !strings.Contains(chordKnowledge, c) {
			t.Errorf("chordKnowledge missing %q", c)
		}
	}

	// Check instruments.
	if !strings.Contains(instrumentKnowledge, "Electric Bass(finger)=34") {
		t.Error("instrumentKnowledge missing program numbers")
	}
	if !strings.Contains(instrumentKnowledge, "Lead(saw)=80") {
		t.Error("instrumentKnowledge missing lead program")
	}
}
