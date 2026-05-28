// Package style — style template system for DNA library integration.
package style

import (
	"github.com/ShowerBandV/text2midi/internal/musicdna"
)

// ToDNATemplate converts a style.Info + MusicDNA into a DNATemplate
// suitable for saving to the DNA library.
// This bridges the style database with the DNA template system:
// every style can produce a baseline template for generation.
func ToDNATemplate(name string, info Info, dna *musicdna.MusicDNA) *musicdna.DNATemplate {
	tmpl := &musicdna.DNATemplate{
		Name:    name,
		Style:   name,
		Quality: 0.5,
		Source:  "style_database",
	}

	if dna != nil {
		tmpl.DNA = *dna
		tmpl.Quality = musicdna.ScoreTemplate(dna)
	}

	return tmpl
}

// RegisterAllStyles creates DNATemplates for all available styles
// and saves them to the given library directory.
// Each template uses the style's default feature vector as the base DNA.
func RegisterAllStyles(libDir string) (int, error) {
	lib := musicdna.NewLibrary(libDir)
	styles := All()
	count := 0

	for name, info := range styles {
		// Build a minimal DNA from the style's default feature vector.
		fv := info.DefaultVector
		dna := &musicdna.MusicDNA{
			Emotion: musicdna.EmotionDNA{
				Energy:     fv.Energy,
				Tension:    fv.Tension,
				Brightness: 1.0 - fv.Darkness,
				Warmth:     1.0 - fv.Darkness,
				Stability:  0.5,
				Confidence: 0.3,
			},
			Rhythm: musicdna.RhythmDNA{
				Density:     fv.Density,
				Syncopation: fv.RhythmicComplexity,
				Variety:     fv.RhythmicComplexity,
				Confidence:  0.3,
			},
			Dynamics: musicdna.DynamicsDNA{
				DynamicRange: fv.Energy,
				AvgVelocity:  fv.Energy * 0.8,
				Confidence:   0.3,
			},
		}

		tmpl := ToDNATemplate(name, info, dna)
		if err := lib.Save(tmpl); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}
