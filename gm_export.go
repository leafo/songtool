package main

import (
	"fmt"
	"io"
	"log"
	"sort"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/smf"
)

// MidiEvent represents a MIDI event with absolute timing
type MidiEvent struct {
	Time    uint32
	Message smf.Message
}

// TrackInfo contains information needed to create a MIDI track
type TrackInfo struct {
	Name    string      // Track name for meta event
	Channel uint8       // MIDI channel
	Program uint8       // GM program number (ignored for drums on channel 9)
	Events  []MidiEvent // All MIDI events for this track
}

// GeneralMidiExporter manages the construction of a General MIDI file
type GeneralMidiExporter struct {
	smf    *smf.SMF    // Target MIDI file being built
	tracks []TrackInfo // Accumulated track information
}

// NewGeneralMidiExporter creates a new MIDI exporter
func NewGeneralMidiExporter() *GeneralMidiExporter {
	return &GeneralMidiExporter{
		smf:    smf.NewSMF1(),
		tracks: make([]TrackInfo, 0),
	}
}

// SetupTimingTrack copies tempo/conductor information from the source MIDI file
func (e *GeneralMidiExporter) SetupTimingTrack(sourceData *smf.SMF) error {
	if sourceData == nil {
		return fmt.Errorf("source MIDI data is nil")
	}

	// Preserve original timing resolution
	e.smf.TimeFormat = sourceData.TimeFormat

	// Extract and add tempo track (Track 0)
	tempoTrack := extractTempoTrack(sourceData)
	e.smf.Add(tempoTrack)

	return nil
}

// AddTrack adds a track to the exporter's track list
func (e *GeneralMidiExporter) addTrack(trackInfo TrackInfo) error {
	e.tracks = append(e.tracks, trackInfo)
	return nil
}

// WriteTo finalizes the MIDI file and writes it to the provided writer
func (e *GeneralMidiExporter) WriteTo(writer io.Writer) error {
	if len(e.tracks) == 0 {
		return fmt.Errorf("no tracks to export")
	}

	// Create MIDI tracks from the accumulated track info
	for _, trackInfo := range e.tracks {
		midiTrack := createMidiTrack(trackInfo)
		e.smf.Add(midiTrack)
	}

	// Write the complete MIDI file
	_, err := e.smf.WriteTo(writer)
	if err != nil {
		return fmt.Errorf("error writing MIDI file: %w", err)
	}

	return nil
}

// createMidiTrack builds a complete MIDI track from TrackInfo
func createMidiTrack(trackInfo TrackInfo) smf.Track {
	track := smf.Track{}

	// Add track name
	trackNameMsg := smf.Message(smf.MetaTrackSequenceName(trackInfo.Name))
	track = append(track, smf.Event{Delta: 0, Message: trackNameMsg})

	// Add program change (except for drums on channel 9)
	if trackInfo.Channel != 9 { // Channel 9 is GM drums, no program change needed
		programChangeMsg := smf.Message(midi.ProgramChange(trackInfo.Channel, trackInfo.Program))
		track = append(track, smf.Event{Delta: 0, Message: programChangeMsg})
	}

	// Sort events by time
	events := make([]MidiEvent, len(trackInfo.Events))
	copy(events, trackInfo.Events)
	sort.Slice(events, func(i, j int) bool {
		if events[i].Time == events[j].Time {
			// Prioritize lyrics first, then note-offs, then note-ons
			msg1 := events[i].Message
			msg2 := events[j].Message

			isLyric1 := msg1.Type() == smf.MetaLyricMsg
			isLyric2 := msg2.Type() == smf.MetaLyricMsg

			if isLyric1 && !isLyric2 {
				return true
			}
			if !isLyric1 && isLyric2 {
				return false
			}

			var ch1, note1, vel1 uint8
			var ch2, note2, vel2 uint8
			isNoteOff1 := msg1.GetNoteOff(&ch1, &note1, &vel1)
			isNoteOn2 := msg2.GetNoteOn(&ch2, &note2, &vel2)

			if (isNoteOff1 || (isNoteOn2 && vel2 == 0)) && ch1 == ch2 && note1 == note2 {
				return true
			}
		}
		return events[i].Time < events[j].Time
	})

	// Add events with proper delta times
	var lastTime uint32
	for _, event := range events {
		delta := event.Time - lastTime
		track = append(track, smf.Event{Delta: delta, Message: event.Message})
		lastTime = event.Time
	}

	// Add end of track
	track = append(track, smf.Event{Delta: 0, Message: smf.EOT})
	return track
}

// AddChartDrumTracks extracts drums from a Chart file and adds them as GM drums to the exporter
func (e *GeneralMidiExporter) AddChartDrumTracks(chartFile *ChartFile) error {
	if chartFile == nil {
		return fmt.Errorf("chart file is nil")
	}

	// Find highest difficulty drum track available
	var drumTrack *TrackSection
	var trackName string

	difficulties := []string{"ExpertDrums", "HardDrums", "MediumDrums", "EasyDrums"}
	for _, diff := range difficulties {
		if track, exists := chartFile.Tracks[diff]; exists && len(track.Notes) > 0 {
			drumTrack = &track
			trackName = diff
			break
		}
	}

	if drumTrack == nil {
		return fmt.Errorf("no drum tracks found in chart file")
	}

	log.Printf("Found %s track with %d notes", trackName, len(drumTrack.Notes))

	// Convert chart drum notes to MIDI events
	var events []MidiEvent

	for _, note := range drumTrack.Notes {
		// Convert chart fret to MIDI key
		midiKey, err := chartFretToMidiKey(note.Fret)
		if err != nil {
			log.Printf("Warning: Could not convert chart fret %d: %v", note.Fret, err)
			continue
		}

		// Convert to GM drum key
		gmKey, err := midiKeyToGMKey(midiKey)
		if err != nil {
			log.Printf("Warning: Could not convert MIDI key %d to GM: %v", midiKey, err)
			continue
		}

		// Calculate absolute time in ticks
		absoluteTime := tickFromChart(chartFile, note.Tick)

		// Use reasonable velocity (chart files don't have velocity info)
		velocity := uint8(100)

		// Add Note On event
		noteOnMsg := smf.Message(midi.NoteOn(gmDrumChannel, gmKey, velocity))
		events = append(events, MidiEvent{Time: absoluteTime, Message: noteOnMsg})

		// Calculate note duration
		endTime := absoluteTime + hitDurationTicks

		// If this is a sustained note, use the sustain length
		if note.Sustain > 0 {
			sustainTicks := tickFromChart(chartFile, note.Sustain)
			endTime = absoluteTime + sustainTicks
		}

		// Add Note Off event
		noteOffMsg := smf.Message(midi.NoteOff(gmDrumChannel, gmKey))
		events = append(events, MidiEvent{Time: endTime, Message: noteOffMsg})
	}

	if len(events) == 0 {
		return fmt.Errorf("no valid drum events found")
	}

	// Add drum track to exporter
	drumTrackInfo := TrackInfo{
		Name:    "Drums",
		Channel: gmDrumChannel,
		Program: 0, // Drums don't use program change (channel 9)
		Events:  events,
	}

	log.Printf("Generated %d MIDI events from chart drums", len(events))
	return e.addTrack(drumTrackInfo)
}

// chartFretToMidiKey converts chart fret numbers to equivalent MIDI keys
func chartFretToMidiKey(fret uint8) (uint8, error) {
	// Chart drum frets map to MIDI keys like this:
	// Fret 0 = Kick = MIDI key 96 (C6)
	// Fret 1 = Red/Snare = MIDI key 97 (C#6)
	// Fret 2 = Yellow/Hi-Hat = MIDI key 98 (D6)
	// Fret 3 = Blue/Ride = MIDI key 99 (D#6)
	// Fret 4 = Orange/Crash = MIDI key 100 (E6)

	switch fret {
	case 0:
		return 96, nil // Kick
	case 1:
		return 97, nil // Snare
	case 2:
		return 98, nil // Hi-Hat
	case 3:
		return 99, nil // Ride
	case 4:
		return 100, nil // Crash
	case 7: // Open note (kick variant)
		return 96, nil
	default:
		return 0, fmt.Errorf("unsupported drum fret: %d", fret)
	}
}

// midiKeyToGMKey converts Rock Band MIDI keys to GM drum keys
func midiKeyToGMKey(midiKey uint8) (uint8, error) {
	gmKey, exists := gmDrumMap[midiKey]
	if !exists {
		return 0, fmt.Errorf("no GM mapping for MIDI key %d", midiKey)
	}
	return gmKey, nil
}

// tickFromChart converts chart ticks to absolute ticks (accounting for resolution differences)
func tickFromChart(chart *ChartFile, chartTick uint32) uint32 {
	// Chart files use their own resolution (typically 192 ticks per quarter note)
	// We need to convert to our target resolution (typically 480 for MIDI export)
	// For now, return the raw tick value - this assumes both use same resolution
	// TODO: Add proper resolution conversion if needed
	return chartTick
}

// SetupTimingTrackFromChart creates timing track from Chart file tempo/time signature data
func (e *GeneralMidiExporter) SetupTimingTrackFromChart(chartFile *ChartFile) error {
	if chartFile == nil {
		return fmt.Errorf("chart file is nil")
	}

	// Set MIDI resolution to match chart
	ticksPerQuarter := smf.MetricTicks(chartFile.Song.Resolution)
	e.smf.TimeFormat = ticksPerQuarter

	tempoTrack := smf.Track{}

	// Add tempo events from chart
	for _, bpmEvent := range chartFile.SyncTrack.BPMEvents {
		bpm := float64(bpmEvent.BPM) / 1000.0 // Chart stores BPM * 1000
		tempoMsg := smf.Message(smf.MetaTempo(bpm))
		tempoTrack = append(tempoTrack, smf.Event{Delta: bpmEvent.Tick, Message: tempoMsg})
	}

	// Add time signature events from chart
	for _, tsEvent := range chartFile.SyncTrack.TimeSigEvents {
		denominator := uint8(1 << tsEvent.Denominator) // Convert from log2 to actual value
		timeSigMsg := smf.Message(smf.MetaTimeSig(tsEvent.Numerator, denominator, 24, 8))
		tempoTrack = append(tempoTrack, smf.Event{Delta: tsEvent.Tick, Message: timeSigMsg})
	}

	// If no tempo events, add default
	if len(chartFile.SyncTrack.BPMEvents) == 0 {
		log.Println("Warning: No tempo events found, using default 120 BPM")
		tempoMsg := smf.Message(smf.MetaTempo(120.0))
		tempoTrack = append(tempoTrack, smf.Event{Delta: 0, Message: tempoMsg})
	}

	// Add track name
	trackNameMsg := smf.Message(smf.MetaTrackSequenceName("Tempo"))
	tempoTrack = append(tempoTrack, smf.Event{Delta: 0, Message: trackNameMsg})

	// Convert absolute deltas to relative deltas
	tempoTrack = convertToRelativeDeltas(tempoTrack)

	// Always end with End of Track
	tempoTrack = append(tempoTrack, smf.Event{Delta: 0, Message: smf.EOT})

	e.smf.Add(tempoTrack)
	return nil
}

// convertToRelativeDeltas converts absolute delta times to relative delta times
func convertToRelativeDeltas(track smf.Track) smf.Track {
	var result smf.Track
	var lastTime uint32

	for _, event := range track {
		delta := event.Delta - lastTime
		result = append(result, smf.Event{Delta: delta, Message: event.Message})
		lastTime = event.Delta
	}

	return result
}

// extractTempoTrack copies only essential timing events from the original MIDI file's first track
func extractTempoTrack(smfData *smf.SMF) smf.Track {
	tempoTrack := smf.Track{}

	if len(smfData.Tracks) == 0 {
		log.Println("Warning: Missing tempo track, creating default tempo 120bpm")
		// No tracks, create a basic tempo track
		tempoMsg := smf.Message(smf.MetaTempo(120.0))
		tempoTrack = append(tempoTrack, smf.Event{Delta: 0, Message: tempoMsg})
		timeSigMsg := smf.Message(smf.MetaTimeSig(4, 4, 24, 8))
		tempoTrack = append(tempoTrack, smf.Event{Delta: 0, Message: timeSigMsg})
	} else {
		// Copy only tempo, time signature, key signature, and track name events
		firstTrack := smfData.Tracks[0]

		for _, event := range firstTrack {
			msg := event.Message
			msgType := msg.Type()

			// Copy only the specific meta events we want
			if msgType == smf.MetaTempoMsg ||
				msgType == smf.MetaTimeSigMsg ||
				msgType == smf.MetaKeySigMsg ||
				msgType == smf.MetaTrackNameMsg {
				tempoTrack = append(tempoTrack, event)
			}
		}
	}

	// Always end with End of Track
	tempoTrack = append(tempoTrack, smf.Event{Delta: 0, Message: smf.EOT})
	return tempoTrack
}
