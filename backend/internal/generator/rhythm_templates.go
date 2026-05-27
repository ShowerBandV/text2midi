// Package generator --Rhythm template library.
// Pre-validated drum patterns indexed by tag and style.
// Templates are 16-step arrays (16th notes, one bar).
package generator

// DrumTemplate holds a single-bar drum pattern.
type DrumTemplate struct {
	Name   string
	Tags   []string // style tags: "rock", "metal", "pop", "hiphop", "fill"
	Kick   [16]int
	Snare  [16]int
	Hihat  [16]int
}

// GetTemplatesByTag returns all templates matching a tag.
func GetTemplatesByTag(tag string) []DrumTemplate {
	var result []DrumTemplate
	for _, t := range rhythmTemplates {
		for _, tg := range t.Tags {
			if tg == tag {
				result = append(result, t)
				break
			}
		}
	}
	return result
}

// GetTemplatesByTags returns templates matching ALL given tags.
func GetTemplatesByTags(tags []string) []DrumTemplate {
	var result []DrumTemplate
	for _, t := range rhythmTemplates {
		match := true
		for _, wanted := range tags {
			found := false
			for _, tg := range t.Tags {
				if tg == wanted {
					found = true
					break
				}
			}
			if !found {
				match = false
				break
			}
		}
		if match {
			result = append(result, t)
		}
	}
	return result
}

// rhythmTemplates is the master template database.
var rhythmTemplates = []DrumTemplate{
	// ─── Rock ────────────────────────────────────────────────────────
	{
		Name: "rock_basic",
		Tags: []string{"rock", "basic", "4/4"},
		Kick:   [16]int{1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Hihat:  [16]int{1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0},
	},
	{
		Name: "rock_driving",
		Tags: []string{"rock", "driving", "4/4"},
		Kick:   [16]int{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 0, 1, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Hihat:  [16]int{1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0},
	},
	{
		Name: "rock_fill",
		Tags: []string{"rock", "fill", "transition"},
		Kick:   [16]int{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 1, 0, 1},
		Snare:  [16]int{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Hihat:  [16]int{1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1},
	},

	// ─── Metal ──────────────────────────────────────────────────────
	{
		Name: "metal_basic",
		Tags: []string{"metal", "blast", "4/4"},
		Kick:   [16]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		Snare:  [16]int{0, 0, 0, 0, 1, 1, 1, 1, 0, 0, 0, 0, 1, 1, 1, 1},
		Hihat:  [16]int{1, 0, 1, 0, 1, 1, 1, 0, 1, 0, 1, 0, 1, 1, 1, 0},
	},
	{
		Name: "metal_groove",
		Tags: []string{"metal", "groove", "4/4"},
		Kick:   [16]int{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 1, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Hihat:  [16]int{1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0},
	},

	// ─── Pop ────────────────────────────────────────────────────────
	{
		Name: "pop_fourfloor",
		Tags: []string{"pop", "four_on_floor", "4/4"},
		Kick:   [16]int{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Hihat:  [16]int{1, 0, 1, 0, 1, 1, 1, 0, 1, 0, 1, 0, 1, 1, 1, 0},
	},
	{
		Name: "pop_dance",
		Tags: []string{"pop", "dance", "4/4"},
		Kick:   [16]int{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 0, 1, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Hihat:  [16]int{1, 0, 1, 1, 1, 0, 1, 1, 1, 0, 1, 1, 1, 0, 1, 1},
	},

	// ─── Hip-Hop ────────────────────────────────────────────────────
	{
		Name: "hiphop_basic",
		Tags: []string{"hiphop", "basic", "4/4"},
		Kick:   [16]int{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0},
		Hihat:  [16]int{1, 0, 1, 0, 1, 0, 1, 1, 1, 0, 1, 0, 1, 0, 1, 0},
	},
	{
		Name: "hiphop_trap",
		Tags: []string{"hiphop", "trap", "4/4"},
		Kick:   [16]int{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0},
		Hihat:  [16]int{1, 1, 0, 1, 1, 1, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1},
	},
	{
		Name: "hiphop_boom_bap",
		Tags: []string{"hiphop", "boom_bap", "4/4"},
		Kick:   [16]int{1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Hihat:  [16]int{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
	},

	// ─── R&B / Soul ─────────────────────────────────────────────────
	{
		Name: "rnb_classic",
		Tags: []string{"rnb", "soul", "groove"},
		Kick:   [16]int{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Hihat:  [16]int{0, 0, 1, 0, 1, 0, 1, 0, 0, 0, 1, 0, 1, 0, 1, 0},
	},
	{
		Name: "rnb_modern",
		Tags: []string{"rnb", "modern", "trap_soul"},
		Kick:   [16]int{1, 0, 0, 0, 1, 0, 1, 0, 0, 0, 1, 0, 1, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0},
		Hihat:  [16]int{1, 0, 1, 1, 0, 1, 1, 1, 1, 0, 1, 1, 0, 1, 1, 1},
	},

	// ─── Funk ───────────────────────────────────────────────────────
	{
		Name: "funk_groove",
		Tags: []string{"funk", "groove", "16th"},
		Kick:   [16]int{1, 0, 0, 0, 1, 1, 0, 0, 0, 1, 0, 0, 1, 1, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Hihat:  [16]int{1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0},
	},

	// ─── Ambient / Cinematic ────────────────────────────────────────
	{
		Name: "ambient_pulse",
		Tags: []string{"ambient", "cinematic", "slow"},
		Kick:   [16]int{1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Hihat:  [16]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	{
		Name: "cinematic_taiko",
		Tags: []string{"cinematic", "epic", "taiko"},
		Kick:   [16]int{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		Hihat:  [16]int{1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0},
	},

	// ─── Latin / World ──────────────────────────────────────────────
	{
		Name: "latin_salsa",
		Tags: []string{"latin", "salsa", "clave"},
		Kick:   [16]int{1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 0},
		Snare:  [16]int{0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 1, 0},
		Hihat:  [16]int{1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0},
	},
}

// GetTemplate returns a template by name, or nil.
func GetTemplate(name string) *DrumTemplate {
	for _, t := range rhythmTemplates {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// SuggestTemplate scores and returns the best template for given style tags.
func SuggestTemplate(styles []string) *DrumTemplate {
	// Count matches per template.
	type scored struct {
		tpl  DrumTemplate
		score int
	}
	var scoredTpls []scored

	for _, t := range rhythmTemplates {
		s := 0
		for _, st := range styles {
			for _, tg := range t.Tags {
				if tg == st {
					s++
				}
			}
		}
		if s > 0 {
			scoredTpls = append(scoredTpls, scored{tpl: t, score: s})
		}
	}

	if len(scoredTpls) == 0 {
		return GetTemplate("rock_basic")
	}

	// Return highest-scored.
	best := scoredTpls[0]
	for _, s := range scoredTpls[1:] {
		if s.score > best.score {
			best = s
		}
	}
	return &best.tpl
}
