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
