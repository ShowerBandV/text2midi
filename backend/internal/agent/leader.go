package agent

import (
	"fmt"
)

// LeaderTask represents a task for one agent.
type LeaderTask struct {
	Agent       string `json:"agent"`
	Instruction string `json:"instruction"`
	DependsOn   string `json:"depends_on,omitempty"`
}

// LeaderPlan contains tasks and iteration status.
type LeaderPlan struct {
	Tasks             []LeaderTask `json:"tasks"`
	IterationComplete bool         `json:"iteration_complete"`
	Reasoning         string       `json:"reasoning"`
	Round             int          `json:"round"`
}

// LeaderAgent analyzes review and decides next actions. Clef equivalent: clef-leader.
// Returns true if iteration should continue.
func LeaderAgent(report *ReviewReport, maxRounds int) *LeaderPlan {
	plan := &LeaderPlan{
		Round:             maxRounds,
		IterationComplete: true,
	}

	if report == nil {
		plan.Reasoning = "no review report available"
		return plan
	}

	// Terminate conditions (matching Clef):
	// - Total ≥ 7.5
	// - No single dimension < 6.0
	// - Melody ≥ 7.0
	if report.Total >= 7.5 && report.Melody >= 7.0 &&
		report.Harmony >= 6.0 && report.Rhythm >= 6.0 &&
		report.Structure >= 6.0 && report.Style >= 6.0 {
		plan.Reasoning = fmt.Sprintf("all criteria met (total=%.1f, melody=%.1f)", report.Total, report.Melody)
		return plan
	}

	// Build tasks for failing dimensions.
	if report.Melody < 7.0 {
		plan.Tasks = append(plan.Tasks, LeaderTask{
			Agent:       "composer",
			Instruction: fmt.Sprintf("improve melody (current score=%.1f). Increase pitch range, add more repetition/development.", report.Melody),
		})
	}
	if report.Harmony < 6.0 {
		plan.Tasks = append(plan.Tasks, LeaderTask{
			Agent:       "harmonist",
			Instruction: fmt.Sprintf("improve chord variety (current score=%.1f). Use more distinct chord types.", report.Harmony),
		})
	}
	if report.Rhythm < 6.0 {
		plan.Tasks = append(plan.Tasks, LeaderTask{
			Agent:       "rhythmist",
			Instruction: fmt.Sprintf("improve drum variation (current score=%.1f). Add velocity variety.", report.Rhythm),
		})
	}

	if len(plan.Tasks) > 0 {
		plan.IterationComplete = false
		plan.Reasoning = fmt.Sprintf("%d dimensions below threshold", len(plan.Tasks))
	}

	fmt.Printf("[Leader] tasks=%d complete=%t reason=%s\n",
		len(plan.Tasks), plan.IterationComplete, plan.Reasoning)
	return plan
}
