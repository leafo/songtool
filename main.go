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
	exportGmDrums := flag.Bool("export-gm-drums", false, "Export drum patterns to General MIDI file")
	exportGmVocals := flag.Bool("export-gm-vocals", false, "Export vocal melody to General MIDI file")
	exportGm := flag.Bool("export-gm", false, "Export both drums and vocals to single General MIDI file")
	printTimeline := flag.Bool("timeline", false, "Print beat timeline from BEAT track")
	exportToneLib := flag.Bool("export-tonelib-xml", false, "Export to ToneLib the_song.dat XML format")
	createToneLibSong := flag.Bool("export-tonelib-song", false, "Create complete ToneLib .song file (ZIP archive)")
	filterTrack := flag.String("filter-track", "", "Filter to show only tracks whose name contains this string (case-insensitive)")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <file> [output]\n", os.Args[0])
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

	if *exportGmDrums || *exportGmVocals || *exportGm {
		outputFile := flag.Arg(1)
		if outputFile == "" {
			if *exportGmDrums {
				outputFile = "gm_drums.mid"
			} else if *exportGmVocals {
				outputFile = "gm_vocals.mid"
			} else if *exportGm {
				outputFile = "gm_complete.mid"
			}
		}

		file, err := os.Create(outputFile)
		if err != nil {
			log.Printf("Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		exporter := NewGeneralMidiExporter()
		err = exporter.SetupTimingTrack(midiFile)
		if err != nil {
			log.Printf("Error setting up timing track: %v\n", err)
			os.Exit(1)
		}

		if *exportGmDrums || *exportGm {
			err = exporter.AddDrumTracks(midiFile)
			if err != nil {
				log.Printf("Error adding drum tracks: %v\n", err)
				os.Exit(1)
			}
		}

		if *exportGmVocals || *exportGm {
			err = exporter.AddVocalTracks(midiFile)
			if err != nil {
				log.Printf("Error adding vocal tracks: %v\n", err)
				os.Exit(1)
			}
		}

		err = exporter.WriteTo(file)
		if err != nil {
			log.Printf("Error writing MIDI file: %v\n", err)
			os.Exit(1)
		}

		var exportType string
		if *exportGmDrums && !*exportGmVocals {
			exportType = "GM Drums"
		} else if *exportGmVocals && !*exportGmDrums {
			exportType = "GM Vocals"
		} else {
			exportType = "Complete GM"
		}

		fmt.Printf("%s exported to: %s\n", exportType, outputFile)
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
	} else if *exportToneLib {
		exportToToneLib(midiFile, sngFile, filename)
	} else if *createToneLibSong {
		outputFile := flag.Arg(1)
		if outputFile == "" {
			outputFile = "output.song"
		}
		createToneLibSongFile(midiFile, sngFile, outputFile)
	} else {
		if sngFile != nil {
			printSngFile(sngFile, *jsonOutput)

			if *jsonOutput {
				return
			}
		}

		printMidiInfo(midiFile, filename, *jsonOutput, *filterTrack)
	}
}

func printMidiInfo(smfData *smf.SMF, filename string, jsonOutput bool, filterTrack string) {
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

		// Apply track filtering if specified
		if filterTrack != "" {
			if trackName == "" || !strings.Contains(strings.ToLower(trackName), strings.ToLower(filterTrack)) {
				continue
			}
		}

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

		// Event type counters
		eventCounts := make(map[string]int)
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

			// Get the message type and use it as the event name
			msgType := msg.Type()
			eventTypeName := msgType.String()
			eventCounts[eventTypeName]++

			// Still need to extract channel and instrument info for other parts of the output
			var ch, key, vel uint8
			if msg.GetNoteOn(&ch, &key, &vel) {
				channels[ch] = true
			} else if msg.GetNoteOff(&ch, &key, &vel) {
				channels[ch] = true
			} else if msg.GetControlChange(&ch, &key, &vel) {
				channels[ch] = true
			} else if msg.GetProgramChange(&ch, &vel) {
				channels[ch] = true
				instruments[ch] = getGMInstrument(vel)
			} else if msg.GetPitchBend(&ch, nil, nil) {
				channels[ch] = true
			} else if msg.GetPolyAfterTouch(&ch, &key, &vel) {
				channels[ch] = true
			} else if msg.GetAfterTouch(&ch, &vel) {
				channels[ch] = true
			}
		}

		duration := lastTime - firstTime
		var durationMs float64
		if tf, ok := smfData.TimeFormat.(smf.MetricTicks); ok {
			durationMs = float64(duration) / float64(tf) * 500 // Assuming 120 BPM
		}

		fmt.Printf("  Duration: %d ticks (%.2f seconds @ 120 BPM)\n", duration, durationMs/1000)

		// Print event counts by type
		if len(eventCounts) > 0 {
			fmt.Println("  Event counts by type:")
			for eventType, count := range eventCounts {
				fmt.Printf("    %s: %d\n", eventType, count)
			}
		}
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

		// If filtering is active, show detailed event information
		if filterTrack != "" {
			fmt.Println("  Detailed Events:")
			printTrackEvents(track)
		}

		fmt.Println()
	}
}

func printTrackEvents(track smf.Track) {
	var currentTime uint32 = 0

	for eventIndex, event := range track {
		currentTime += event.Delta
		msg := event.Message
		msgType := msg.Type()

		fmt.Printf("    [%d] Tick: %d, Event: %s", eventIndex, currentTime, msgType.String())

		// Extract specific event data for common event types
		var ch, key, vel uint8
		var pitchValue int16

		if msg.GetNoteOn(&ch, &key, &vel) {
			fmt.Printf("(ch=%d, key=%d, vel=%d)", ch, key, vel)
		} else if msg.GetNoteOff(&ch, &key, &vel) {
			fmt.Printf("(ch=%d, key=%d, vel=%d)", ch, key, vel)
		} else if msg.GetControlChange(&ch, &key, &vel) {
			fmt.Printf("(ch=%d, cc=%d, val=%d)", ch, key, vel)
		} else if msg.GetProgramChange(&ch, &vel) {
			fmt.Printf("(ch=%d, program=%d)", ch, vel)
		} else if msg.GetPitchBend(&ch, &pitchValue, nil) {
			fmt.Printf("(ch=%d, value=%d)", ch, pitchValue)
		} else if msg.GetPolyAfterTouch(&ch, &key, &vel) {
			fmt.Printf("(ch=%d, key=%d, pressure=%d)", ch, key, vel)
		} else if msg.GetAfterTouch(&ch, &vel) {
			fmt.Printf("(ch=%d, pressure=%d)", ch, vel)
		} else {
			// For meta events and other types, try to extract text/data
			var text string
			if msg.GetMetaTrackName(&text) {
				fmt.Printf("(\"%s\")", text)
			} else if msg.GetMetaText(&text) {
				fmt.Printf("(\"%s\")", text)
			} else if msg.GetMetaLyric(&text) {
				fmt.Printf("(\"%s\")", text)
			} else if msg.GetMetaMarker(&text) {
				fmt.Printf("(\"%s\")", text)
			} else {
				var tempo float64
				var num, denom uint8
				if msg.GetMetaTempo(&tempo) {
					fmt.Printf("(%.1f BPM)", tempo)
				} else if msg.GetMetaTimeSig(&num, &denom, nil, nil) {
					fmt.Printf("(%d/%d)", num, 1<<denom)
				}
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

// exportToToneLib exports MIDI/SNG data to ToneLib the_song.dat XML format
func exportToToneLib(midiFile *smf.SMF, sngFile *SngFile, filename string) {
	err := ConvertToToneLib(midiFile, sngFile, "")
	if err != nil {
		log.Printf("Error exporting to ToneLib: %v\n", err)
		return
	}
}

// createToneLibSongFile creates a complete ToneLib .song ZIP archive
func createToneLibSongFile(midiFile *smf.SMF, sngFile *SngFile, outputFile string) {
	fmt.Printf("Creating ToneLib song file: %s\n", outputFile)

	err := CreateToneLibSongFile(midiFile, sngFile, outputFile)
	if err != nil {
		log.Printf("Error creating ToneLib song file: %v\n", err)
		return
	}

	fmt.Printf("Successfully created ToneLib song file: %s\n", outputFile)
}
