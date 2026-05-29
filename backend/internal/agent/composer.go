// Package agent — LLM music composition agents.
// Each agent specializes in one musical dimension, similar to Clef's multi-agent approach.
// Agents generate notes via LLM, not Go algorithms.
package agent

import (
	"fmt"

	"github.com/ShowerBandV/text2midi/internal/llm"
	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ComposerAgent generates lead melody. Clef equivalent: clef-composer.
func ComposerAgent(client *llm.Client, keyRoot, keyMode string, bpm, totalBeats int, chordProgJSON string) ([]schema.NoteEvent, error) {
	seed := fmt.Sprintf("%d", bpm*totalBeats)
	prompt := llm.BuildNoteSequencePrompt("lead", keyRoot, keyMode, chordProgJSON,
		"{style:pop}", "{dark:0.3,en:0.5,ten:0.3,rhy:0.4}", bpm, totalBeats, seed)
	result, err := client.JSONWithTemp("You are a professional lead melody composer. Return strict JSON.", prompt, 0.85)
	if err != nil {
		return nil, err
	}
	return parseNoteEvents(result)
}

// HarmonistAgent generates chord progressions and pad voicings. Clef equivalent: clef-harmonist.
func HarmonistAgent(client *llm.Client, keyRoot, keyMode string, bpm, totalBeats int, chordProgJSON string) ([]schema.NoteEvent, error) {
	seed := fmt.Sprintf("%d", bpm*totalBeats+1)
	prompt := llm.BuildNoteSequencePrompt("pad", keyRoot, keyMode, chordProgJSON,
		"{style:pop}", "{dark:0.3,en:0.4,ten:0.2,rhy:0.3}", bpm, totalBeats, seed)
	result, err := client.JSONWithTemp("You are a professional harmony arranger. Write chords/pad notes. Return strict JSON.", prompt, 0.8)
	if err != nil {
		return nil, err
	}
	return parseNoteEvents(result)
}

// RhythmistAgent generates bass and drums. Clef equivalent: clef-rhythmist.
func RhythmistAgent(client *llm.Client, keyRoot, keyMode string, bpm, totalBeats int, chordProgJSON, role string) ([]schema.NoteEvent, error) {
	seed := fmt.Sprintf("%d", bpm*totalBeats+2)
	prompt := llm.BuildNoteSequencePrompt(role, keyRoot, keyMode, chordProgJSON,
		"{style:pop}", "{dark:0.3,en:0.5,ten:0.3,rhy:0.5}", bpm, totalBeats, seed)
	sysPrompt := "You are a professional bass player. Write bass lines. Return strict JSON."
	if role == "drums" {
		sysPrompt = "You are a professional drummer. Write drum patterns. Return strict JSON."
	}
	result, err := client.JSONWithTemp(sysPrompt, prompt, 0.8)
	if err != nil {
		return nil, err
	}
	return parseNoteEvents(result)
}

// parseNoteEvents extracts "events" from LLM JSON response.
func parseNoteEvents(result map[string]any) ([]schema.NoteEvent, error) {
	rawEvents, ok := result["events"]
	if !ok {
		return nil, fmt.Errorf("missing 'events' key in LLM response")
	}
	evList, ok := rawEvents.([]any)
	if !ok {
		return nil, fmt.Errorf("'events' is not a list")
	}
	var events []schema.NoteEvent
	for _, item := range evList {
		evMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ev := schema.NoteEvent{Type: "note"}
		if p, ok := evMap["pitch"].(float64); ok {
			ev.Pitch = int(p)
		}
		if s, ok := evMap["start_beat"].(float64); ok {
			ev.StartBeat = s
		}
		if d, ok := evMap["duration_beat"].(float64); ok {
			ev.DurationBeat = d
		}
		if v, ok := evMap["velocity"].(float64); ok {
			ev.Velocity = int(v)
		}
		events = append(events, ev)
	}
	return events, nil
}

func init() {
	_ = fmt.Sprintf
}
