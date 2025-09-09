package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gitlab.com/gomidi/midi/v2/smf"
)

func main() {
	jsonOutput := flag.Bool("json", false, "Output information as JSON (supported with: default analysis, --timeline)")
	exportGmDrums := flag.Bool("export-gm-drums", false, "Export drum patterns to General MIDI file")
	exportGmVocals := flag.Bool("export-gm-vocals", false, "Export vocal melody to General MIDI file")
	exportGmBass := flag.Bool("export-gm-bass", false, "Export pro bass to General MIDI file")
	exportGm := flag.Bool("export-gm", false, "Export drums, vocals, and bass to single General MIDI file")
	printTimeline := flag.Bool("timeline", false, "Print beat timeline from BEAT track")
	exportToneLib := flag.Bool("export-tonelib-xml", false, "Export to ToneLib the_song.dat XML format")
	createToneLibSong := flag.Bool("export-tonelib-song", false, "Create complete ToneLib .song file (ZIP archive)")
	filterTrack := flag.String("filter-track", "", "Filter to show only tracks whose name contains this string (case-insensitive)")
	extractFile := flag.String("extract-file", "", "Extract and print contents of specified file from SNG package to stdout")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <file> [output]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	filename := flag.Arg(0)

	var song SongInterface
	var sngFile *SngFile     // Keep for SNG-specific operations
	var midiFile *smf.SMF    // Keep for MIDI-specific operations
	var chartFile *ChartFile // Keep for chart-specific operations
	var err error

	ext := strings.ToLower(filepath.Ext(filename))

	if ext == ".sng" {
		sngFile, err = OpenSngFile(filename)
		if err != nil {
			log.Printf("Error opening SNG file: %v\n", err)
			os.Exit(1)
		}
		defer sngFile.Close()
		song = sngFile

		// Also try to load individual files for legacy operations
		midiData, midiErr := sngFile.ReadFile("notes.mid")
		if midiErr == nil {
			midiFile, err = smf.ReadFrom(bytes.NewReader(midiData))
			if err != nil {
				log.Printf("Error reading MIDI data: %v\n", err)
			}
		}

		chartData, chartErr := sngFile.ReadFile("notes.chart")
		if chartErr == nil {
			chartFile, err = ParseChartFile(bytes.NewReader(chartData))
			if err != nil {
				log.Printf("Error reading chart data: %v\n", err)
			} else {
				chartFile.Filename = "notes.chart"
			}
		}

		if midiErr != nil && chartErr != nil {
			log.Printf("No MIDI or chart file found in SNG package\n")
		}
	} else if ext == ".chart" {
		chartFile, err = OpenChartFile(filename)
		if err != nil {
			log.Printf("Error opening chart file: %v\n", err)
			os.Exit(1)
		}
		song = chartFile
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
		song = &MidiFile{SMF: midiFile}
	}

	if *exportGmDrums || *exportGmVocals || *exportGmBass || *exportGm {
		if midiFile == nil && chartFile == nil {
			log.Printf("No MIDI or Chart data available for export\n")
			os.Exit(1)
		}
		outputFile := flag.Arg(1)
		if outputFile == "" {
			if *exportGmDrums {
				outputFile = "gm_drums.mid"
			} else if *exportGmVocals {
				outputFile = "gm_vocals.mid"
			} else if *exportGmBass {
				outputFile = "gm_bass.mid"
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

		// Setup timing track from available source
		if midiFile != nil {
			err = exporter.SetupTimingTrack(midiFile)
			if err != nil {
				log.Printf("Error setting up timing track from MIDI: %v\n", err)
				os.Exit(1)
			}
		} else if chartFile != nil {
			err = exporter.SetupTimingTrackFromChart(chartFile)
			if err != nil {
				log.Printf("Error setting up timing track from Chart: %v\n", err)
				os.Exit(1)
			}
		}

		if *exportGmDrums || *exportGm {
			if midiFile != nil {
				err = exporter.AddDrumTracks(midiFile)
				if err != nil {
					log.Printf("Error adding drum tracks from MIDI: %v\n", err)
					os.Exit(1)
				}
			} else if chartFile != nil {
				err = exporter.AddChartDrumTracks(chartFile)
				if err != nil {
					log.Printf("Error adding drum tracks from Chart: %v\n", err)
					os.Exit(1)
				}
			}
		}

		if *exportGmVocals || *exportGm {
			if midiFile != nil {
				err = exporter.AddVocalTracks(midiFile)
				if err != nil {
					log.Printf("Error adding vocal tracks: %v\n", err)
					os.Exit(1)
				}
			} else {
				log.Printf("Warning: Vocal export not supported for Chart files (Chart files contain no melodic data)")
			}
		}

		if *exportGmBass || *exportGm {
			if midiFile != nil {
				err = exporter.AddBassTracks(midiFile)
				if err != nil {
					log.Printf("Error adding bass tracks: %v\n", err)
					os.Exit(1)
				}
			} else {
				log.Printf("Warning: Bass export not supported for Chart files (Chart files contain no melodic data)")
			}
		}

		err = exporter.WriteTo(file)
		if err != nil {
			log.Printf("Error writing MIDI file: %v\n", err)
			os.Exit(1)
		}

		var exportType string
		if *exportGmDrums && !*exportGmVocals && !*exportGmBass {
			exportType = "GM Drums"
		} else if *exportGmVocals && !*exportGmDrums && !*exportGmBass {
			exportType = "GM Vocals"
		} else if *exportGmBass && !*exportGmDrums && !*exportGmVocals {
			exportType = "GM Bass"
		} else {
			exportType = "Complete GM"
		}

		fmt.Printf("%s exported to: %s\n", exportType, outputFile)
	} else if *printTimeline {
		timeline, err := song.GetTimeline()
		if err != nil {
			log.Printf("Error extracting timeline: %v\n", err)
			os.Exit(1)
		}

		if *jsonOutput {
			jsonData, err := json.MarshalIndent(timeline, "", "  ")
			if err != nil {
				log.Printf("Error marshaling timeline to JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(jsonData))
		} else {
			fmt.Printf("Timeline for: %s\n", filename)
			fmt.Print(timeline.String())
		}
	} else if *exportToneLib {
		exportToToneLib(song, filename)
	} else if *createToneLibSong {
		outputFile := flag.Arg(1)
		if outputFile == "" {
			outputFile = "output.song"
		}
		createToneLibSongFile(song, outputFile)
	} else if *extractFile != "" {
		if sngFile == nil {
			log.Printf("File extraction only supported for SNG files\n")
			os.Exit(1)
		}
		extractFileFromSng(sngFile, *extractFile)
	} else {
		if sngFile != nil {
			printSngFile(sngFile, *jsonOutput)

			if *jsonOutput {
				return
			}
		}

		if chartFile != nil {
			printChartInfo(chartFile, *jsonOutput, *filterTrack)
			return
		}

		if midiFile == nil {
			log.Printf("No valid chart or MIDI data found in file: %s\n", filename)
			os.Exit(1)
		}

		printMidiInfo(midiFile, filename, *jsonOutput, *filterTrack)
	}
}

func printMidiInfo(smfData *smf.SMF, filename string, jsonOutput bool, filterTrack string) {
	if smfData == nil {
		if jsonOutput {
			fmt.Println("null")
		} else {
			fmt.Printf("No MIDI data available\n")
		}
		return
	}

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

// exportToToneLib exports song data to ToneLib the_song.dat XML format
func exportToToneLib(song SongInterface, filename string) {
	var writer io.Writer
	outputFile := flag.Arg(1)
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			log.Printf("Error creating output file: %v\n", err)
			return
		}
		defer file.Close()
		writer = file
	} else {
		writer = os.Stdout
	}

	err := WriteToneLibXMLTo(writer, song)
	if err != nil {
		log.Printf("Error exporting to ToneLib: %v\n", err)
		return
	}
}

// createToneLibSongFile creates a complete ToneLib .song ZIP archive
func createToneLibSongFile(song SongInterface, outputFile string) {
	fmt.Printf("Creating ToneLib song file: %s\n", outputFile)

	file, err := os.Create(outputFile)
	if err != nil {
		log.Printf("Error creating output file: %v\n", err)
		return
	}
	defer file.Close()

	err = WriteToneLibSongTo(file, song)
	if err != nil {
		log.Printf("Error creating ToneLib song file: %v\n", err)
		return
	}

	fmt.Printf("Successfully created ToneLib song file: %s\n", outputFile)
}

// extractFileFromSng extracts and prints the contents of a file from an SNG package
func extractFileFromSng(sngFile *SngFile, filename string) {
	data, err := sngFile.ReadFile(filename)
	if err != nil {
		log.Printf("Error reading file '%s' from SNG package: %v\n", filename, err)
		os.Exit(1)
	}

	// Check if output file is specified as second argument
	outputFile := flag.Arg(1)
	if outputFile != "" {
		// Write to specified output file
		file, err := os.Create(outputFile)
		if err != nil {
			log.Printf("Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		_, err = file.Write(data)
		if err != nil {
			log.Printf("Error writing to output file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Extracted '%s' to: %s\n", filename, outputFile)
	} else {
		// Write to stdout
		_, err = os.Stdout.Write(data)
		if err != nil {
			log.Printf("Error writing to stdout: %v\n", err)
			os.Exit(1)
		}
	}
}

func printChartInfo(chart *ChartFile, jsonOutput bool, filterTrack string) {
	if jsonOutput {
		jsonData, err := json.MarshalIndent(chart, "", "  ")
		if err != nil {
			log.Printf("Error marshaling chart to JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
		return
	}

	fmt.Printf("Chart File: %s\n", chart.Filename)
	if chart.Song.Name != "" {
		fmt.Printf("Title: %s\n", chart.Song.Name)
	}
	if chart.Song.Artist != "" {
		fmt.Printf("Artist: %s\n", chart.Song.Artist)
	}
	if chart.Song.Album != "" {
		fmt.Printf("Album: %s\n", chart.Song.Album)
	}
	if chart.Song.Charter != "" {
		fmt.Printf("Charter: %s\n", chart.Song.Charter)
	}
	if chart.Song.Year != "" {
		fmt.Printf("Year: %s\n", chart.Song.Year)
	}
	if chart.Song.Genre != "" {
		fmt.Printf("Genre: %s\n", chart.Song.Genre)
	}
	fmt.Printf("Resolution: %d ticks per quarter note\n", chart.Song.Resolution)
	fmt.Printf("Offset: %d\n", chart.Song.Offset)
	if chart.Song.MusicStream != "" {
		fmt.Printf("Audio: %s\n", chart.Song.MusicStream)
	}
	fmt.Println()

	// Sync track info
	fmt.Printf("Sync Track:\n")
	fmt.Printf("  BPM changes: %d\n", len(chart.SyncTrack.BPMEvents))
	if len(chart.SyncTrack.BPMEvents) > 0 {
		firstBPM := chart.SyncTrack.BPMEvents[0]
		fmt.Printf("  Starting BPM: %.1f\n", float64(firstBPM.BPM)/1000.0)
	}
	fmt.Printf("  Time signature changes: %d\n", len(chart.SyncTrack.TimeSigEvents))
	fmt.Printf("  Anchor events: %d\n", len(chart.SyncTrack.AnchorEvents))
	fmt.Println()

	// Events info
	fmt.Printf("Global Events: %d\n", len(chart.Events.GlobalEvents))

	// Count lyrics and sections
	var lyricCount, sectionCount int
	for _, event := range chart.Events.GlobalEvents {
		if strings.HasPrefix(event.Text, "lyric ") {
			lyricCount++
		} else if strings.HasPrefix(event.Text, "section ") {
			sectionCount++
		}
	}
	if lyricCount > 0 {
		fmt.Printf("  Lyrics: %d\n", lyricCount)
	}
	if sectionCount > 0 {
		fmt.Printf("  Sections: %d\n", sectionCount)
	}
	fmt.Println()

	// Track info
	fmt.Printf("Tracks: %d\n", len(chart.Tracks))
	for trackName, track := range chart.Tracks {
		// Apply track filtering if specified
		if filterTrack != "" {
			if !strings.Contains(strings.ToLower(trackName), strings.ToLower(filterTrack)) {
				continue
			}
		}

		fmt.Printf("  %s:\n", trackName)
		fmt.Printf("    Notes: %d\n", len(track.Notes))
		fmt.Printf("    Specials: %d\n", len(track.Specials))
		fmt.Printf("    Track Events: %d\n", len(track.TrackEvents))

		if len(track.Notes) > 0 {
			firstNote := track.Notes[0]
			lastNote := track.Notes[len(track.Notes)-1]
			duration := lastNote.Tick - firstNote.Tick
			durationSeconds := calculateTickDuration(chart, firstNote.Tick, lastNote.Tick)
			fmt.Printf("    Duration: %d ticks (%.2f seconds)\n", duration, durationSeconds)

			// Count notes by fret
			fretCounts := make(map[uint8]int)
			sustainCount := 0
			for _, note := range track.Notes {
				fretCounts[note.Fret]++
				if note.Sustain > 0 {
					sustainCount++
				}
			}

			fmt.Printf("    Notes by fret: ")
			for fret := uint8(0); fret <= 7; fret++ {
				if count, exists := fretCounts[fret]; exists && count > 0 {
					fmt.Printf("F%d:%d ", fret, count)
				}
			}
			fmt.Println()

			if sustainCount > 0 {
				fmt.Printf("    Sustained notes: %d\n", sustainCount)
			}
		}

		// Show special events breakdown
		if len(track.Specials) > 0 {
			specialTypes := make(map[uint8]int)
			for _, special := range track.Specials {
				specialTypes[special.Type]++
			}
			fmt.Printf("    Special types: ")
			for sType, count := range specialTypes {
				var typeName string
				switch sType {
				case 2:
					typeName = "Starpower"
				case 64:
					typeName = "DrumFill"
				case 65:
					typeName = "DrumRoll"
				case 66:
					typeName = "DrumRollSpecial"
				default:
					typeName = fmt.Sprintf("S%d", sType)
				}
				fmt.Printf("%s:%d ", typeName, count)
			}
			fmt.Println()
		}

		// If filtering is active, show detailed event information
		if filterTrack != "" {
			fmt.Printf("    Detailed Events:\n")
			printChartTrackEvents(&track)
		}

		fmt.Println()
	}
}

func calculateTickDuration(chart *ChartFile, startTick, endTick uint32) float64 {
	if chart.Song.Resolution == 0 {
		return 0
	}

	totalSeconds := 0.0
	currentTick := startTick

	for _, bpmEvent := range chart.SyncTrack.BPMEvents {
		if bpmEvent.Tick <= startTick {
			continue
		}

		nextTick := bpmEvent.Tick
		if nextTick > endTick {
			nextTick = endTick
		}

		if currentTick < nextTick {
			bpm := chart.GetBPMAtTick(currentTick)
			ticksInSegment := nextTick - currentTick
			secondsInSegment := float64(ticksInSegment) / float64(chart.Song.Resolution) * 60.0 / bpm
			totalSeconds += secondsInSegment
		}

		currentTick = nextTick
		if currentTick >= endTick {
			break
		}
	}

	// Handle remaining ticks if any
	if currentTick < endTick {
		bpm := chart.GetBPMAtTick(currentTick)
		ticksInSegment := endTick - currentTick
		secondsInSegment := float64(ticksInSegment) / float64(chart.Song.Resolution) * 60.0 / bpm
		totalSeconds += secondsInSegment
	}

	return totalSeconds
}

func printChartTrackEvents(track *TrackSection) {
	// Combine all events and sort by tick
	type eventInfo struct {
		tick uint32
		text string
	}

	var allEvents []eventInfo

	for _, note := range track.Notes {
		fretName := fmt.Sprintf("Fret %d", note.Fret)
		if note.Fret == 7 {
			fretName = "Open"
		}
		text := fmt.Sprintf("Note %s", fretName)
		if note.Sustain > 0 {
			text += fmt.Sprintf(" (sustain: %d)", note.Sustain)
		}
		allEvents = append(allEvents, eventInfo{note.Tick, text})
	}

	for _, special := range track.Specials {
		var typeName string
		switch special.Type {
		case 2:
			typeName = "Starpower"
		case 64:
			typeName = "Drum Fill"
		case 65:
			typeName = "Drum Roll"
		case 66:
			typeName = "Special Drum Roll"
		default:
			typeName = fmt.Sprintf("Special %d", special.Type)
		}
		text := fmt.Sprintf("%s (length: %d)", typeName, special.Length)
		allEvents = append(allEvents, eventInfo{special.Tick, text})
	}

	for _, trackEvent := range track.TrackEvents {
		allEvents = append(allEvents, eventInfo{trackEvent.Tick, fmt.Sprintf("Event: %s", trackEvent.Text)})
	}

	// Sort by tick
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].tick < allEvents[j].tick
	})

	for _, event := range allEvents {
		fmt.Printf("      Tick %d: %s\n", event.tick, event.text)
	}
}
