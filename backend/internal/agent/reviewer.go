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

	// Compute objective metrics.
	lead := eventsByTrack["lead"]
	if len(lead) < 8 {
		report.Issues = append(report.Issues, "too few lead notes")
		report.Total = 3.0
		return report
	}

	// Melody: check pitch range and repetition.
	minP, maxP := 127, 0
	pitchSet := make(map[int]int)
	for _, ev := range lead {
		if ev.Pitch < minP { minP = ev.Pitch }
		if ev.Pitch > maxP { maxP = ev.Pitch }
		pitchSet[ev.Pitch]++
	}
	pitchRange := maxP - minP
	report.Melody = 5.0
	if pitchRange < 7 { report.Melody -= 1.5; report.Issues = append(report.Issues, "pitch range too narrow") }
	if pitchRange > 30 { report.Melody -= 1.0; report.Issues = append(report.Issues, "pitch range too wide") }
	if len(pitchSet) < 5 { report.Melody -= 1.5; report.Issues = append(report.Issues, "too few distinct pitches") }
	report.Melody = math.Max(report.Melody, 2.0)
	report.Melody = math.Min(report.Melody, 10.0)

	// Harmony: check chord variety.
	pad := eventsByTrack["pad"]
	chordNotes := make(map[int]bool)
	for _, ev := range pad {
		chordNotes[ev.Pitch%12] = true
	}
	report.Harmony = 5.0
	if len(chordNotes) < 4 { report.Harmony -= 1.5; report.Issues = append(report.Issues, "limited chord variety") }
	if len(chordNotes) > 8 { report.Harmony += 1.0 }
	report.Harmony = math.Min(report.Harmony, 10.0)

	// Rhythm: check drum density and variation.
	drums := eventsByTrack["drums"]
	if len(drums) > 10 {
		velSeen := make(map[int]bool)
		for _, ev := range drums {
			velSeen[ev.Velocity] = true
		}
		report.Rhythm = 5.0 + float64(len(velSeen))*0.5
		report.Rhythm = math.Min(report.Rhythm, 10.0)
	} else {
		report.Rhythm = 4.0
	}

	// Structure: check section count.
	report.Structure = 6.0
	if totalBars >= 8 { report.Structure += 1.0 }
	if totalBars >= 16 { report.Structure += 1.0 }

	// Style & Arrangement: default.
	report.Style = 7.0
	report.Arrangement = 7.0

	// Total.
	report.Total = (report.Melody*0.3 + report.Harmony*0.2 + report.Rhythm*0.2 +
		report.Structure*0.1 + report.Style*0.1 + report.Arrangement*0.1)

	fmt.Printf("[Reviewer] melody=%.1f harm=%.1f rhythm=%.1f struct=%.1f total=%.1f issues=%d\n",
		report.Melody, report.Harmony, report.Rhythm, report.Structure, report.Total, len(report.Issues))
	return report
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
