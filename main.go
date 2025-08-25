package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/gomidi/midi/v2/smf"
)

func main() {
	jsonOutput := flag.Bool("json", false, "Output MIDI information as JSON")
	exportDrums := flag.Bool("export-drums", false, "Export drum patterns from MIDI file")
	printTimeline := flag.Bool("timeline", false, "Print beat timeline from BEAT track")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <file>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	filename := flag.Arg(0)

	var sngFile *SngFile
	var midiFile *smf.SMF
	var err error

	ext := strings.ToLower(filepath.Ext(filename))

	if ext == ".sng" {
		sngFile, err = OpenSngFile(filename)

		if err != nil {
			log.Printf("Error opening SNG file: %v\n", err)
			os.Exit(1)
		}

		defer sngFile.Close()

		// load the midi file in it

		midiData, err := sngFile.ReadFile("notes.mid")

		if err != nil {
			log.Printf("No MIDI file found in SNG package\n")
		} else {
			midiFile, err = smf.ReadFrom(bytes.NewReader(midiData))
			if err != nil {
				log.Printf("Error reading MIDI data: %v\n", err)
			}
		}
	} else {
		// treat the file as a regular midi file

		file, err := os.Open(filename)
		if err != nil {
			log.Printf("Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		midiFile, err = smf.ReadFrom(file)
		if err != nil {
			log.Printf("Error reading MIDI file: %v\n", err)
			os.Exit(1)
		}

	}

	if *exportDrums {
		ExportDrumsFromMidi(midiFile, filename)
	} else if *printTimeline {
		printTimeline := func(midiFile *smf.SMF, filename string) {
			timeline, err := ExtractBeatTimeline(midiFile)
			if err != nil {
				log.Printf("Error extracting timeline: %v\n", err)
				return
			}
			fmt.Printf("Timeline for: %s\n", filename)
			fmt.Print(timeline.String())
		}
		printTimeline(midiFile, filename)
	} else {
		if sngFile != nil {
			printSngFile(sngFile, *jsonOutput)

			if *jsonOutput {
				return
			}
		}

		printMidiInfo(midiFile, filename, *jsonOutput)
	}
}

func printMidiInfo(smfData *smf.SMF, filename string, jsonOutput bool) {
	if jsonOutput {
		jsonData, err := json.MarshalIndent(smfData, "", "  ")
		if err != nil {
			log.Printf("Error marshaling to JSON: %v\n", err)
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

		var noteCount, ccCount, pgmCount int
		var firstTime, lastTime uint32
		var channels = make(map[uint8]bool)
		var instruments = make(map[uint8]string)

		firstTime = track[0].Delta
		lastTime = firstTime
		currentTime := firstTime

		for _, event := range track {
			currentTime += event.Delta
			lastTime = currentTime

			msg := event.Message

			var ch, key, vel uint8

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
		cleanLyrics := extractLyrics(track)
		if cleanLyrics != "" {
			fmt.Printf("  Lyrics: %s\n", cleanLyrics)
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

func printSngFile(sngFile *SngFile, jsonOutput bool) {
	if jsonOutput {
		output := map[string]interface{}{
			"header":   sngFile.Header,
			"metadata": sngFile.GetMetadata(),
			"files":    sngFile.Files,
		}
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			log.Printf("Error marshaling to JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
		return
	}

	fmt.Printf("Version: %d\n", sngFile.Header.Version)
	fmt.Println()

	metadata := sngFile.GetMetadata()
	if len(metadata) > 0 {
		fmt.Println("Metadata:")
		for key, value := range metadata {
			fmt.Printf("  %s: %s\n", key, value)
		}
		fmt.Println()
	}

	files := sngFile.ListFiles()
	fmt.Printf("Contains %d files:\n", len(files))
	for i, filename := range files {
		entry := sngFile.Files[i]
		fmt.Printf("  %s (%d bytes)\n", filename, entry.Size)
	}
	fmt.Println()
}
