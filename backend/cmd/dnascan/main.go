// Command dnascan scans MIDI output directories and builds a DNA library.
//
// Usage:
//   go run ./cmd/dnascan/                                --scan directories
//   go run ./cmd/dnascan/ --dir ./generated               --scan specific directory
//   go run ./cmd/dnascan/ --lib ./dna_library --rebuild   --rebuild library
//   go run ./cmd/dnascan/ --register-styles              --register all built-in styles
//   go run ./cmd/dnascan/ --generate house               --generate from DNA template
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ShowerBandV/text2midi/internal/composer"
	"github.com/ShowerBandV/text2midi/internal/musicdna"
	"github.com/ShowerBandV/text2midi/internal/store"
	"github.com/ShowerBandV/text2midi/internal/style"
)

func main() {
	dir := flag.String("dir", "", "Directory to scan")
	libDir := flag.String("lib", "./dna_library", "DNA library directory")
	rebuild := flag.Bool("rebuild", false, "Rebuild library from scratch")
	generate := flag.String("generate", "", "Generate MIDI from a DNA template name")
	registerStyles := flag.Bool("register-styles", false, "Register all built-in styles as DNA templates")
	flag.Parse()

	if *registerStyles {
		count, err := style.RegisterAllStyles(*libDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Registered %d style templates in %s\n", count, *libDir)
		return
	}

	if *generate != "" {
		lib := musicdna.NewLibrary(*libDir)
		tmpl, err := lib.Load(*generate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Template %q not found: %v\n", *generate, err)
			os.Exit(1)
		}
		ctx := composer.GenerateContextFromDNA(&tmpl.DNA, 0, 0)
		events := composer.ComposeSongWithContext(ctx)
		fmt.Printf("Generated %d tracks from DNA template %q\n", len(events), *generate)
		for name, evs := range events {
			fmt.Printf("  %s: %d events\n", name, len(evs))
		}
		return
	}

	lib := musicdna.NewLibrary(*libDir)
	if *rebuild {
		fmt.Printf("Rebuilding library at %s...\n", *libDir)
		os.RemoveAll(*libDir)
	}

	var scanDirs []string
	if *dir != "" {
		scanDirs = append(scanDirs, *dir)
	} else {
		for _, d := range []string{"./generated", "./midi_output"} {
			if info, err := os.Stat(d); err == nil && info.IsDir() {
				scanDirs = append(scanDirs, d)
			}
		}
	}

	if len(scanDirs) == 0 {
		fmt.Println("No directories found. Use --dir, --register-styles, or --generate.")
		os.Exit(1)
	}

	var totalFound, totalImported int
	for _, scanDir := range scanDirs {
		fmt.Printf("\nScanning %s...\n", scanDir)
		entries, err := os.ReadDir(scanDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			id := entry.Name()
			metaPath := filepath.Join(scanDir, id, "meta.json")
			dnaPath := filepath.Join(scanDir, id, "dna.json")

			metaData, err := os.ReadFile(metaPath)
			if err != nil {
				dnaData, err := os.ReadFile(dnaPath)
				if err != nil {
					continue
				}
				dna, err := musicdna.FromJSON(dnaData)
				if err != nil {
					continue
				}
				totalFound++
				if !dnaExists(lib, id) {
					saveDNA(lib, id, "unknown", dna)
					totalImported++
				}
				continue
			}

			var record store.FileRecord
			if err := json.Unmarshal(metaData, &record); err != nil {
				continue
			}
			totalFound++
			if dnaExists(lib, id) {
				continue
			}
			dnaData, err := os.ReadFile(dnaPath)
			if err != nil {
				continue
			}
			dna, err := musicdna.FromJSON(dnaData)
			if err != nil {
				continue
			}
			saveDNA(lib, id, record.FileName, dna)
			totalImported++
		}
	}

	fmt.Printf("\nDone: %d found, %d imported to %s\n", totalFound, totalImported, *libDir)
	if totalImported > 0 {
		listLibrary(lib)
	}
}

func dnaExists(lib *musicdna.Library, id string) bool {
	_, err := lib.Load(id)
	return err == nil
}

func saveDNA(lib *musicdna.Library, id, name string, dna *musicdna.MusicDNA) {
	quality := musicdna.ScoreTemplate(dna)
	tmpl := &musicdna.DNATemplate{
		Name:    id,
		Style:   extractStyle(dna),
		DNA:     *dna,
		Quality: quality,
		Source:  name,
	}
	if err := lib.Save(tmpl); err != nil {
		fmt.Fprintf(os.Stderr, "  Failed to save %s: %v\n", id, err)
		return
	}
	fmt.Printf("  Imported: %s (quality=%.2f)\n", id, quality)
}

func extractStyle(dna *musicdna.MusicDNA) string {
	if dna == nil {
		return "unknown"
	}
	key := dna.Harmony.Key
	if key != "" {
		return key
	}
	return "unknown"
}

func listLibrary(lib *musicdna.Library) {
	templates, err := lib.List("")
	if err != nil {
		return
	}
	fmt.Printf("\nDNA Library (%d templates):\n", len(templates))
	for _, t := range templates {
		fmt.Printf("  %-30s quality=%.2f style=%s\n", t.Name, t.Quality, t.Style)
	}
}
