package main

import (
	"gitlab.com/gomidi/midi/v2/smf"
)

// SongInterface defines a common interface for extracting timeline information from music files
type SongInterface interface {
	GetTimeline() (*Timeline, error)
	GetMetadata() map[string]string
	GetLyricsByMeasure() ([]MeasureLyrics, error)
}

// SMF wrapper so we can implement the interface
type MidiFile struct {
	*smf.SMF
}

func (m *MidiFile) GetMetadata() map[string]string {
	result := make(map[string]string)

	if len(m.Tracks) > 0 {
		trackName := getTrackName(m.Tracks[0])
		if trackName != "" {
			result["name"] = trackName
		}
	}

	return result
}

func (m *MidiFile) GetLyricsByMeasure() ([]MeasureLyrics, error) {
	// Get timeline for measure timing
	timeline, err := m.GetTimeline()
	if err != nil {
		return nil, err
	}

	// Extract lyric events with timing from MIDI file
	lyricEvents := extractLyricsWithTiming(m.SMF)
	if len(lyricEvents) == 0 {
		return []MeasureLyrics{}, nil
	}

	// Group lyrics by measure using existing logic
	measureLyrics := groupLyricsByMeasure(lyricEvents, timeline)
	return measureLyrics, nil
}
