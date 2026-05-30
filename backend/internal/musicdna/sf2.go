package musicdna

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// SF2Instrument holds key range data for a GM program number.
type SF2Instrument struct {
	Name     string `json:"name"`
	KeyRange [2]int `json:"key_range"`
}

// SF2Profile maps GM program numbers to instrument constraints.
type SF2Profile struct {
	Instruments map[string]SF2Instrument `json:"instruments"`
}

// LoadSF2Profile reads an SF2 profile JSON file.
func LoadSF2Profile(path string) (*SF2Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var profile SF2Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// ApplySF2Constraints clips all note pitches to the valid range for each track's program.
// Returns the number of notes that were clipped.
func ApplySF2Constraints(ir *schema.MidiIR, profile *SF2Profile) int {
	if profile == nil {
		return 0
	}
	clipped := 0
	for ti := range ir.Tracks {
		t := &ir.Tracks[ti]
		if t.Program == nil {
			continue
		}
		progKey := strconv.Itoa(*t.Program)
		inst, ok := profile.Instruments[progKey]
		if !ok {
			continue
		}
		for ei := range t.Events {
			if t.Events[ei].Type != "note" {
				continue
			}
			p := t.Events[ei].Pitch
			if p < inst.KeyRange[0] {
				t.Events[ei].Pitch = inst.KeyRange[0]
				clipped++
			}
			if p > inst.KeyRange[1] {
				t.Events[ei].Pitch = inst.KeyRange[1]
				clipped++
			}
		}
	}
	if clipped > 0 {
		fmt.Printf("  [SF2] clipped %d notes to instrument key ranges\n", clipped)
	}
	return clipped
}
