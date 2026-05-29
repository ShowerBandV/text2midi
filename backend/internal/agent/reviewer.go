package agent

import (
	"fmt"
	"math"

	"github.com/ShowerBandV/text2midi/internal/llm"
	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ReviewReport holds the 6-dimension quality assessment.
type ReviewReport struct {
	Melody      float64 `json:"melody"`
	Harmony     float64 `json:"harmony"`
	Rhythm      float64 `json:"rhythm"`
	Structure   float64 `json:"structure"`
	Style       float64 `json:"style"`
	Arrangement float64 `json:"arrangement"`
	Total       float64 `json:"total"`
	Issues      []string `json:"issues"`
}

// ReviewerAgent evaluates music quality. Clef equivalent: clef-reviewer.
func ReviewerAgent(client *llm.Client, eventsByTrack map[string][]schema.NoteEvent, totalBars int) *ReviewReport {
	report := &ReviewReport{}

	// Melody: score based on pitch range, distinct notes, and note count.
	lead := eventsByTrack["lead"]
	if len(lead) < 4 {
		report.Issues = append(report.Issues, "too few lead notes")
		report.Melody = 2.0
		report.Total = 2.0
		return report
	}

	minP, maxP := 127, 0
	pitchSet := make(map[int]int)
	for _, ev := range lead {
		if ev.Pitch < minP {
			minP = ev.Pitch
		}
		if ev.Pitch > maxP {
			maxP = ev.Pitch
		}
		pitchSet[ev.Pitch]++
	}
	pitchRange := maxP - minP

	// Melody: scale 2-9, start from 5, adjust based on quality.
	report.Melody = 5.0
	if pitchRange >= 5 {
		report.Melody += 0.5 // some melodic shape
	}
	if pitchRange >= 10 {
		report.Melody += 1.0 // good melodic range
	}
	if pitchRange >= 18 {
		report.Melody += 0.5 // wide range (can be good or bad)
	}
	if pitchRange < 3 {
		report.Melody -= 2.0
		report.Issues = append(report.Issues, "pitch range too narrow (<3 semitones)")
	}
	if pitchRange > 36 {
		report.Melody -= 1.5
		report.Issues = append(report.Issues, "pitch range extremely wide (>3 octaves)")
	}
	// Note variety: more distinct pitches = better (up to a point).
	distinct := len(pitchSet)
	if distinct >= 6 {
		report.Melody += 0.5
	}
	if distinct >= 10 {
		report.Melody += 0.5
	}
	if distinct >= 15 {
		report.Melody += 0.5
	}
	if distinct < 3 {
		report.Melody -= 2.0
		report.Issues = append(report.Issues, "too few distinct pitches")
	}
	// Note count vs bars: should have reasonable density.
	notesPerBar := float64(len(lead)) / float64(totalBars)
	if notesPerBar >= 2 {
		report.Melody += 0.5
	}
	if notesPerBar >= 4 {
		report.Melody += 0.5
	}
	if notesPerBar < 1 {
		report.Melody -= 1.5
		report.Issues = append(report.Issues, "melody too sparse (<1 note/bar)")
	}
	if notesPerBar > 12 {
		report.Melody -= 0.5
		report.Issues = append(report.Issues, "melody possibly too dense (>12 notes/bar)")
	}
	report.Melody = math.Max(report.Melody, 1.0)
	report.Melody = math.Min(report.Melody, 9.5)

	// Harmony: check chord/pad variety.
	pad := eventsByTrack["pad"]
	if len(pad) > 0 {
		chordPCs := make(map[int]bool)
		for _, ev := range pad {
			chordPCs[ev.Pitch%12] = true
		}
		report.Harmony = 5.0
		pcCount := len(chordPCs)
		if pcCount >= 5 {
			report.Harmony += 0.5
		}
		if pcCount >= 7 {
			report.Harmony += 1.0
		}
		if pcCount >= 9 {
			report.Harmony += 0.5
		}
		if pcCount < 3 {
			report.Harmony -= 2.0
			report.Issues = append(report.Issues, "limited chord variety (<3 pitch classes)")
		}
		// Chord density: notes per bar.
		padPerBar := float64(len(pad)) / float64(totalBars)
		if padPerBar > 20 {
			report.Harmony -= 1.0
			report.Issues = append(report.Issues, "pad too dense (>20 notes/bar)")
		}
		report.Harmony = math.Max(report.Harmony, 2.0)
		report.Harmony = math.Min(report.Harmony, 9.0)
	} else {
		report.Harmony = 4.0 // no pad track at all
		report.Issues = append(report.Issues, "no pad/chord track found")
	}

	// Rhythm: check drum density, hit variety, velocity spread.
	drums := eventsByTrack["drums"]
	if len(drums) > 4 {
		report.Rhythm = 5.0
		drumTypes := make(map[int]bool)
		var velSum, velMin, velMax int
		velMin = 127
		for _, ev := range drums {
			drumTypes[ev.Pitch] = true
			velSum += ev.Velocity
			if ev.Velocity < velMin {
				velMin = ev.Velocity
			}
			if ev.Velocity > velMax {
				velMax = ev.Velocity
			}
		}
		// Drum type variety (kick/snare/hi-hat/ride/crash...).
		if len(drumTypes) >= 3 {
			report.Rhythm += 0.5
		}
		if len(drumTypes) >= 4 {
			report.Rhythm += 0.5
		}
		// Velocity spread: wider = more expressive.
		velSpread := velMax - velMin
		if velSpread >= 30 {
			report.Rhythm += 0.5
		}
		if velSpread >= 50 {
			report.Rhythm += 0.5
		}
		if velSpread < 10 {
			report.Rhythm -= 1.5
			report.Issues = append(report.Issues, "drum velocity too flat (no dynamics)")
		}
		// Drum events per bar.
		drumsPerBar := float64(len(drums)) / float64(totalBars)
		if drumsPerBar < 4 {
			report.Rhythm -= 1.0
			report.Issues = append(report.Issues, "drums too sparse")
		}
		report.Rhythm = math.Max(report.Rhythm, 2.0)
		report.Rhythm = math.Min(report.Rhythm, 9.0)
	} else {
		report.Rhythm = 3.0
		report.Issues = append(report.Issues, "very few drum hits")
	}

	// Structure: based on bar count and track diversity.
	report.Structure = 4.0
	if totalBars >= 8 {
		report.Structure += 1.0
	}
	if totalBars >= 16 {
		report.Structure += 1.5
	}
	if totalBars >= 24 {
		report.Structure += 1.0
	}
	trackCount := 0
	for _, evs := range eventsByTrack {
		if len(evs) > 0 {
			trackCount++
		}
	}
	if trackCount >= 4 {
		report.Structure += 0.5
	}
	report.Structure = math.Min(report.Structure, 8.5)

	// Style & Arrangement: default (we can't judge subjectively without LLM).
	report.Style = 6.0
	report.Arrangement = 6.0
	if trackCount >= 4 {
		report.Arrangement += 0.5
	}

	// Total: weighted average.
	report.Total = (report.Melody*0.3 + report.Harmony*0.2 + report.Rhythm*0.2 +
		report.Structure*0.1 + report.Style*0.1 + report.Arrangement*0.1)
	report.Total = math.Round(report.Total*10) / 10

	fmt.Printf("[Reviewer] melody=%.1f harm=%.1f rhythm=%.1f struct=%.1f total=%.1f issues=%d\n",
		report.Melody, report.Harmony, report.Rhythm, report.Structure, report.Total, len(report.Issues))
	return report
}

// ReviewWithLLM calls the LLM for subjective music quality assessment.
// Unlike ReviewerAgent (Go rules), this provides real musical critique.
func ReviewWithLLM(client *llm.Client, eventsByTrack map[string][]schema.NoteEvent, plan *schema.SongPlan) (*ReviewReport, error) {
	prompt := llm.BuildReviewerPrompt(eventsByTrack, plan)

	systemPrompt := `You are a professional music critic and game composer.
Evaluate the music across 6 dimensions on a 1-10 scale.
Be critical and specific — vague praise is not helpful.
If a dimension is weak, explain exactly what's wrong and how to fix it.`

	result, err := client.JSONWithTemp(systemPrompt, prompt, 0.5)
	if err != nil {
		return nil, fmt.Errorf("reviewer LLM: %w", err)
	}

	report := &ReviewReport{}
	if m, ok := result["melody"].(float64); ok {
		report.Melody = m
	}
	if h, ok := result["harmony"].(float64); ok {
		report.Harmony = h
	}
	if r, ok := result["rhythm"].(float64); ok {
		report.Rhythm = r
	}
	if s, ok := result["structure"].(float64); ok {
		report.Structure = s
	}
	if s, ok := result["style"].(float64); ok {
		report.Style = s
	}
	if a, ok := result["arrangement"].(float64); ok {
		report.Arrangement = a
	}
	if t, ok := result["total"].(float64); ok {
		report.Total = t
	} else {
		report.Total = (report.Melody*0.3 + report.Harmony*0.2 + report.Rhythm*0.2 +
			report.Structure*0.1 + report.Style*0.1 + report.Arrangement*0.1)
	}
	if issues, ok := result["issues"].([]any); ok {
		for _, iss := range issues {
			if s, ok := iss.(string); ok {
				report.Issues = append(report.Issues, s)
			}
		}
	}

	fmt.Printf("[Reviewer-LLM] melody=%.1f harm=%.1f rhythm=%.1f total=%.1f issues=%d\n",
		report.Melody, report.Harmony, report.Rhythm, report.Total, len(report.Issues))
	return report, nil
}

// OrchestratorAgent adds expression (CC/velocity variation). Clef equivalent: clef-orchestrator.
// Currently limited to velocity-based dynamics.
func OrchestratorAgent(eventsByTrack map[string][]schema.NoteEvent, totalBars int) {
	for trackID, events := range eventsByTrack {
		if trackID == "drums" { continue }
		for i := range events {
			bar := int(events[i].StartBeat) / 4
			// Crescendo in last third.
			if bar > totalBars*2/3 {
				events[i].Velocity = int(float64(events[i].Velocity) * 1.15)
				if events[i].Velocity > 127 { events[i].Velocity = 127 }
			}
			// Decrescendo in intro.
			if bar < 2 {
				events[i].Velocity = int(float64(events[i].Velocity) * 0.85)
			}
		}
	}
	fmt.Println("[Orchestrator] applied dynamics")
}
