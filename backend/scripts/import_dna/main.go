// Command import_dna imports MIDI files into the MusicDNA template library.
// Usage: go run ./scripts/import_dna/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ShowerBandV/text2midi/internal/midi"
	"github.com/ShowerBandV/text2midi/internal/musicdna"
)

func main() {
	// Source: Jay Chou MIDI files
	srcDir := "../DNA/jaychou"
	if len(os.Args) > 1 {
		srcDir = os.Args[1]
	}

	// Target: template library
	targetDir := "./templates"
	if len(os.Args) > 2 {
		targetDir = os.Args[2]
	}

	templateLib := musicdna.NewTemplateDB(targetDir)

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read dir %s: %v\n", srcDir, err)
		os.Exit(1)
	}

	imported := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".mid") &&
			!strings.HasSuffix(strings.ToLower(entry.Name()), ".midi") {
			continue
		}

		path := filepath.Join(srcDir, entry.Name())
		fmt.Printf("Processing: %s\n", entry.Name())

		events, totalBars, err := midi.ReadMIDIFile(path)
		if err != nil {
			fmt.Printf("  skip: %v\n", err)
			continue
		}

		if len(events) == 0 {
			fmt.Printf("  skip: no events\n")
			continue
		}

		// Determine key (we don't parse key signature from MIDI, default to C major).
		// The extractor will detect harmony from the notes.
		key := "C major"

		// Name: remove .mid extension
		name := strings.TrimSuffix(entry.Name(), ".mid")
		name = strings.TrimSuffix(name, ".midi")

		_, err = templateLib.FromMIDI(events, totalBars, key, "jaychou", name, path)
		if err != nil {
			fmt.Printf("  save error: %v\n", err)
			continue
		}
		imported++
	}

	fmt.Printf("\nDone: %d/%d files imported to %s\n", imported, len(entries), targetDir)
}
