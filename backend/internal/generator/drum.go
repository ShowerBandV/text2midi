package generator

import (
	"math/rand"
	"sort"

	"github.com/ShowerBandV/text2midi/internal/music"
	"github.com/ShowerBandV/text2midi/internal/schema"
)

// GenerateDrums generates drum pattern events.
// Feature vector influences: Energy ->velocity/hits, RhythmicComplexity ->syncopation, LoFi ->groove.
func GenerateDrums(plan schema.SongPlan, track schema.ArrangementTrack) []schema.NoteEvent {
	totalBars := plan.TotalBars
	fv := plan.FeatureVector

	// Energy ->baseline velocity and additional kick hits.
	energyVel := int(fv.Energy * 30) // 0-30 extra velocity
	kickExtra := int(fv.Energy * 2)  // 0-2 extra kick hits

	// RhythmicComplexity ->syncopation density (1-3 extra offbeat kicks).
	syncLevel := 1 + int(fv.RhythmicComplexity*2)

	// LoFi ->timing jitter (0.0-0.05 beats) and velocity randomness.
	lofiJitter := fv.LoFi * 0.05
	lofiVelRange := 5 + int(fv.LoFi*20)

	// Hi-hat 8th-note motif.
	motifLen := 8
	hatMotif := make([]int, motifLen)
	// Density controls hi-hat density: low = only on strong 8ths, high = more.
	hatThreshold := 0.3 + fv.Density*0.5
	for i := range hatMotif {
		if rand.Float64() < hatThreshold {
			hatMotif[i] = 1
		} else {
			hatMotif[i] = 0
		}
	}
	hatMotif[0] = 1
	hatMotif[motifLen-1] = rand.Intn(2)

	// Kick motif: pick from candidate positions based on complexity.
	kickCandidates := []float64{0.0, 0.75, 1.5, 2.0, 2.75, 3.5}
	kickMotif := make([]float64, kickExtra+syncLevel)
	for i := range kickMotif {
		kickMotif[i] = kickCandidates[rand.Intn(len(kickCandidates))]
	}

	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0

		// Kick drum: beats 0, 2 + motif hits.
		kickBeats := []float64{0.0, 2.0}
		kickBeats = append(kickBeats, kickMotif...)
		sort.Float64s(kickBeats)
		seen := map[float64]bool{}
		for _, b := range kickBeats {
			if seen[b] {
				continue
			}
			seen[b] = true
			jitter := 0.0
			if lofiJitter > 0 {
				jitter = (rand.Float64() - 0.5) * 2 * lofiJitter
			}
			vel := 80 + energyVel + rand.Intn(10+lofiVelRange)
			if vel > 127 {
				vel = 127
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: music.DrumMap["kick"],
				DrumName: "kick", StartBeat: base + b + jitter,
				DurationBeat: 0.1, Velocity: vel,
			})
		}

		// Snare: beats 1 and 3 (with possible syncopation offset).
		snareOff := 0.0
		if fv.RhythmicComplexity > 0.6 && rand.Float64() < 0.2 {
			snareOff = 0.25 // swing feel
		}
		for _, b := range []float64{1.0 + snareOff, 3.0} {
			jitter := 0.0
			if lofiJitter > 0 {
				jitter = (rand.Float64() - 0.5) * 2 * lofiJitter
			}
			vel := 80 + energyVel + rand.Intn(10+lofiVelRange)
			if vel > 127 {
				vel = 127
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: music.DrumMap["snare"],
				DrumName: "snare", StartBeat: base + b + jitter,
				DurationBeat: 0.1, Velocity: vel,
			})
		}

		// Hi-hat: 8th notes with motif gating.
		for i := 0; i < 8; i++ {
			if hatMotif[i%motifLen] == 0 {
				continue
			}
			jitter := 0.0
			if lofiJitter > 0 {
				jitter = (rand.Float64() - 0.5) * 2 * lofiJitter
			}
			vel := 55 + int(fv.Energy*30) + rand.Intn(15+lofiVelRange)
			if vel > 127 {
				vel = 127
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: music.DrumMap["closed_hat"],
				DrumName: "closed_hat", StartBeat: base + float64(i)*0.5 + jitter,
				DurationBeat: 0.1, Velocity: vel,
			})
		}

		// ── Crash cymbal at section boundaries ─────────────────────
		// Section boundaries: bar 0 (start), every barsPerSection bar (e.g. 4, 8, 12).
		// Crash on beat 1 (baseline offset) for maximum impact.
		barsPerSection := 4
		isSectionStart := bar%barsPerSection == 0
		// Also crash on the very first bar and on half-way points.
		addCrash := isSectionStart
		// Add extra crashes for high energy.
		if fv.Energy > 0.7 && bar%2 == 0 {
			addCrash = true
		}

		if addCrash {
			crashBeat := float64(0) // beat 0 (start of bar)
			crashVel := 100 + int(fv.Energy*27)
			if crashVel > 127 {
				crashVel = 127
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: music.DrumMap["crash"],
				DrumName: "crash", StartBeat: base + crashBeat,
				DurationBeat: 0.5, Velocity: crashVel,
			})
		}

		// ── Tom fill at section turnarounds ─────────────────────────
		// In the last 2 beats of a section-ending bar, add tom hits.
		isLastBarOfSection := bar > 0 && (bar+1)%barsPerSection == 0
		if isLastBarOfSection && fv.Energy > 0.4 {
			// Tom fill: 4 rapid hits descending.
			tomPitches := []int{music.DrumMap["high_tom"], music.DrumMap["mid_tom"],
				music.DrumMap["low_tom"], music.DrumMap["kick"]}
			for ti, tp := range tomPitches {
				tomBeat := 2.5 + float64(ti)*0.25 // beats 2.5, 2.75, 3.0, 3.25
				tomVel := 80 + int(fv.Energy*30) + rand.Intn(15)
				if tomVel > 127 {
					tomVel = 127
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: tp,
					DrumName: "tom_fill", StartBeat: base + tomBeat,
					DurationBeat: 0.08, Velocity: tomVel,
				})
			}
		}
	}
	return events
}
