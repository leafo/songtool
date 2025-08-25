package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/gomidi/midi/v2/smf"
)

func main() {
	var jsonOutput bool
	var exportDrums bool
	var filename string

	// Parse command line arguments
	args := os.Args[1:]
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "--export-drums":
			exportDrums = true
		default:
			if filename == "" {
				filename = arg
			} else {
				fmt.Fprintf(os.Stderr, "Usage: %s [--json] [--export-drums] <file>\n", os.Args[0])
				os.Exit(1)
			}
		}
	}

	if filename == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s [--json] [--export-drums] <file>\n", os.Args[0])
		os.Exit(1)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".sng" {
		handleSngFile(filename, jsonOutput, exportDrums)
		return
	}

	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	smfData, err := smf.ReadFrom(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading MIDI file: %v\n", err)
		os.Exit(1)
	}

	if exportDrums {
		ExportDrumsFromMidi(smfData, filename)
	} else {
		printMidiInfo(smfData, filename, jsonOutput)
	}
}

func printMidiInfo(smfData *smf.SMF, filename string, jsonOutput bool) {
	if jsonOutput {
		jsonData, err := json.MarshalIndent(smfData, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling to JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
		return
	}

	fmt.Printf("MIDI File: %s\n", filename)
	fmt.Printf("Format: %d\n", smfData.Format())
	if tf, ok := smfData.TimeFormat.(smf.MetricTicks); ok {
		fmt.Printf("Ticks per quarter note: %d\n", tf)
	} else {
		fmt.Printf("Time format: %v\n", smfData.TimeFormat)
	}
	fmt.Printf("Number of tracks: %d\n", len(smfData.Tracks))
	fmt.Println()

	for i, track := range smfData.Tracks {
		trackName := getTrackName(track)
		if trackName != "" {
			fmt.Printf("Track %d: %s\n", i, trackName)
		} else {
			fmt.Printf("Track %d:\n", i)
		}
		fmt.Printf("  Number of events: %d\n", len(track))

		if len(track) == 0 {
			fmt.Println("  (empty track)")
			continue
		}

		var noteCount, ccCount, pgmCount, lyricCount int
		var firstTime, lastTime uint32
		var channels = make(map[uint8]bool)
		var instruments = make(map[uint8]string)
		var lyrics []string

		firstTime = track[0].Delta
		lastTime = firstTime
		currentTime := firstTime

		for _, event := range track {
			currentTime += event.Delta
			lastTime = currentTime

			msg := event.Message

			var ch, key, vel uint8
			var lyric, text string
			if msg.GetNoteOn(&ch, &key, &vel) {
				noteCount++
				channels[ch] = true
			} else if msg.GetNoteOff(&ch, &key, &vel) {
				channels[ch] = true
			} else if msg.GetControlChange(&ch, &key, &vel) {
				ccCount++
				channels[ch] = true
			} else if msg.GetProgramChange(&ch, &vel) {
				pgmCount++
				channels[ch] = true
				instruments[ch] = getGMInstrument(vel)
			} else if msg.GetMetaLyric(&lyric) {
				lyricCount++
				lyrics = append(lyrics, lyric)
			} else if msg.GetMetaText(&text) {
				// Skip bracketed animation markers, look for actual lyrics
				if len(text) > 0 && text[0] != '[' {
					lyricCount++
					lyrics = append(lyrics, text)
				}
			}
		}

		duration := lastTime - firstTime
		var durationMs float64
		if tf, ok := smfData.TimeFormat.(smf.MetricTicks); ok {
			durationMs = float64(duration) / float64(tf) * 500 // Assuming 120 BPM
		}

		fmt.Printf("  Duration: %d ticks (%.2f seconds @ 120 BPM)\n", duration, durationMs/1000)
		fmt.Printf("  Note events: %d\n", noteCount)
		fmt.Printf("  Control change events: %d\n", ccCount)
		fmt.Printf("  Program change events: %d\n", pgmCount)
		if lyricCount > 0 {
			fmt.Printf("  Lyric events: %d\n", lyricCount)
			if len(lyrics) > 0 {
				cleanLyrics := parseRockBandLyrics(lyrics)
				fmt.Printf("  Lyrics: %s\n", cleanLyrics)
			}
		}

		if len(channels) > 0 {
			fmt.Printf("  Channels used: ")
			first := true
			for ch := range channels {
				if !first {
					fmt.Printf(", ")
				}
				fmt.Printf("%d", ch)
				first = false
			}
			fmt.Println()
		}

		if len(instruments) > 0 {
			fmt.Println("  Instruments:")
			for ch, inst := range instruments {
				fmt.Printf("    Channel %d: %s\n", ch, inst)
			}
		}

		fmt.Println()
	}
}

func getTrackName(track smf.Track) string {
	for _, event := range track {
		msg := event.Message

		var trackName string
		if msg.GetMetaTrackName(&trackName) {
			return trackName
		}

		var text string
		if msg.GetMetaText(&text) {
			return text
		}
	}
	return ""
}

func parseRockBandLyrics(rawLyrics []string) string {
	var result []string
	var currentWord strings.Builder

	for _, lyric := range rawLyrics {
		if lyric == "" {
			continue
		}

		// Skip if it's just a "+" (syllable continuation marker)
		if lyric == "+" {
			continue
		}

		// Clean up the lyric text
		cleaned := lyric

		// Remove non-pitched markers (#, ^) and range dividers (%)
		cleaned = strings.TrimSuffix(cleaned, "#")
		cleaned = strings.TrimSuffix(cleaned, "^")
		cleaned = strings.TrimSuffix(cleaned, "%")

		// Handle actual hyphens (= becomes -)
		cleaned = strings.ReplaceAll(cleaned, "=", "-")

		// Check if this syllable continues with "+"
		isSlideNote := strings.HasSuffix(cleaned, "+")
		if isSlideNote {
			cleaned = strings.TrimSuffix(cleaned, "+")
			cleaned = strings.TrimSpace(cleaned)
		}

		// Check if this is a syllable continuation (starts with hyphen after cleaning markers)
		isSyllableContinuation := strings.HasSuffix(cleaned, "-")
		if isSyllableContinuation {
			cleaned = strings.TrimSuffix(cleaned, "-")
			cleaned = strings.TrimSpace(cleaned)
		}

		// Add to current word
		currentWord.WriteString(cleaned)

		// If this syllable doesn't continue to next (no trailing hyphen), complete the word
		if !isSyllableContinuation && !isSlideNote {
			word := currentWord.String()
			if word != "" {
				result = append(result, word)
			}
			currentWord.Reset()
		}
	}

	// Handle any remaining word
	if currentWord.Len() > 0 {
		word := currentWord.String()
		if word != "" {
			result = append(result, word)
		}
	}

	return strings.Join(result, " ")
}

func handleSngFile(filename string, jsonOutput bool, exportDrums bool) {
	sng, err := OpenSngFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening SNG file: %v\n", err)
		os.Exit(1)
	}
	defer sng.Close()

	if jsonOutput {
		output := map[string]interface{}{
			"header":   sng.Header,
			"metadata": sng.GetMetadata(),
			"files":    sng.Files,
		}
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling to JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
		return
	}

	fmt.Printf("SNG File: %s\n", filename)
	fmt.Printf("Version: %d\n", sng.Header.Version)
	fmt.Println()

	metadata := sng.GetMetadata()
	if len(metadata) > 0 {
		fmt.Println("Metadata:")
		for key, value := range metadata {
			fmt.Printf("  %s: %s\n", key, value)
		}
		fmt.Println()
	}

	files := sng.ListFiles()
	fmt.Printf("Contains %d files:\n", len(files))
	for i, filename := range files {
		entry := sng.Files[i]
		fmt.Printf("  %s (%d bytes)\n", filename, entry.Size)
	}
	fmt.Println()

	// Try to read and parse the MIDI file
	midiData, err := sng.ReadFile("notes.mid")
	if err != nil {
		fmt.Printf("No MIDI file found in SNG package\n")
		return
	}

	smfData, err := smf.ReadFrom(bytes.NewReader(midiData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading MIDI data: %v\n", err)
		return
	}

	if exportDrums {
		ExportDrumsFromMidi(smfData, "notes.mid (from SNG)")
	} else {
		printMidiInfo(smfData, "notes.mid (from SNG)", jsonOutput)
	}
}
