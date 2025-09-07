package main

import (
	"gitlab.com/gomidi/midi/v2/smf"
)

// SongInterface defines a common interface for extracting timeline information from music files
type SongInterface interface {
	GetTimeline() (*Timeline, error)
	GetMetadata() map[string]string
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
