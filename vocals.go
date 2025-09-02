package main

import (
	"fmt"
	"log"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/smf"
)

// the default voice used for vocal tracks
const (
	gmOboe uint8 = 68 // Oboe - melodic instrument for vocals
)

// VocalNote represents a single vocal note with timing, pitch, and lyric
type VocalNote struct {
	Time     uint32
	Key      uint8 // MIDI note number (C1=36 to C5=84)
	Velocity uint8
	Duration uint32 // Duration in ticks
	Lyric    string // Associated lyric text
}

// AddVocalTracks extracts vocal melody and harmonies from a Rock Band MIDI file
// and adds them as GM standard vocal tracks to the exporter
func (e *GeneralMidiExporter) AddVocalTracks(sourceData *smf.SMF) error {
	// Find all vocal tracks
	vocalTracks := make(map[string]smf.Track)
	vocalTrackNames := []string{"PART VOCALS", "HARM1", "HARM2", "HARM3"}

	for _, track := range sourceData.Tracks {
		trackName := getTrackName(track)
		for _, vocalTrackName := range vocalTrackNames {
			if trackName == vocalTrackName {
				vocalTracks[vocalTrackName] = track
				break
			}
		}
	}

	if len(vocalTracks) == 0 {
		return fmt.Errorf("no vocal tracks found")
	}

	log.Printf("Found %d vocal track(s)", len(vocalTracks))

	// Extract vocal notes from all tracks
	allVocalNotes := make(map[string][]VocalNote)
	totalNotes := 0

	for trackName, track := range vocalTracks {
		notes := extractVocalNotes(track)
		if len(notes) > 0 {
			allVocalNotes[trackName] = notes
			totalNotes += len(notes)
			log.Printf("Found %d vocal notes in %s", len(notes), trackName)
		}
	}

	if totalNotes == 0 {
		return fmt.Errorf("no vocal notes found in any vocal tracks")
	}

	log.Printf("Found %d total vocal notes across all tracks", totalNotes)

	// Create track info for each vocal part
	trackOrder := []string{"PART VOCALS", "HARM1", "HARM2", "HARM3"}
	channelMap := map[string]uint8{
		"PART VOCALS": 0, // Main vocals on channel 0
		"HARM1":       1, // Harmony 1 on channel 1
		"HARM2":       2, // Harmony 2 on channel 2
		"HARM3":       3, // Harmony 3 on channel 3
	}

	for _, trackName := range trackOrder {
		vocalNotes, exists := allVocalNotes[trackName]
		if !exists || len(vocalNotes) == 0 {
			continue
		}

		channel := channelMap[trackName]

		// Create display name
		displayName := trackName
		if trackName == "PART VOCALS" {
			displayName = "Lead Vocals"
		} else {
			displayName = "Harmony " + trackName[4:]
		}

		// Convert vocal notes to MIDI events
		var events []MidiEvent

		for i, note := range vocalNotes {
			// Skip notes outside valid range
			if note.Key < 36 || note.Key > 84 {
				log.Printf("Warning: skipping vocal note %d outside valid range (36-84)", note.Key)
				continue
			}

			// Add lyric event if present (only for main vocals)
			if note.Lyric != "" && trackName == "PART VOCALS" {
				lyricMsg := smf.Message(smf.MetaLyric(note.Lyric))
				events = append(events, MidiEvent{Time: note.Time, Message: lyricMsg})
			}

			// Add Note On event
			noteOnMsg := smf.Message(midi.NoteOn(channel, note.Key, note.Velocity))
			events = append(events, MidiEvent{Time: note.Time, Message: noteOnMsg})

			// Calculate end time with overlap detection
			endTime := note.Time + note.Duration
			for j := i + 1; j < len(vocalNotes); j++ {
				nextNote := vocalNotes[j]
				if nextNote.Time >= endTime {
					break
				}
				if nextNote.Key == note.Key {
					endTime = nextNote.Time
					break
				}
			}

			// Add Note Off event
			noteOffMsg := smf.Message(midi.NoteOff(channel, note.Key))
			events = append(events, MidiEvent{Time: endTime, Message: noteOffMsg})
		}

		// Add vocal track to exporter
		vocalTrackInfo := TrackInfo{
			Name:    displayName,
			Channel: channel,
			Program: gmOboe, // All vocals use oboe
			Events:  events,
		}

		err := e.addTrack(vocalTrackInfo)
		if err != nil {
			return err
		}
	}

	return nil
}

// extractVocalNotes finds all vocal notes (C1-C5: 36-84) in the vocal track
// and associates them with lyric events
func extractVocalNotes(vocalTrack smf.Track) []VocalNote {
	var vocalNotes []VocalNote
	var currentTime uint32

	// First pass: collect all lyric events with their timestamps
	lyricsByTime := make(map[uint32]string)
	currentTime = 0

	for _, event := range vocalTrack {
		currentTime += event.Delta
		msg := event.Message

		var lyric, text string
		if msg.GetMetaLyric(&lyric) {
			lyricsByTime[currentTime] = lyric
		} else if msg.GetMetaText(&text) {
			// Skip bracketed animation markers, look for actual lyrics
			if len(text) > 0 && text[0] != '[' {
				lyricsByTime[currentTime] = text
			}
		}
	}

	// Second pass: collect vocal notes and associate with lyrics
	currentTime = 0
	noteOnMap := make(map[uint8]uint32) // Track note-on times for duration calculation

	for _, event := range vocalTrack {
		currentTime += event.Delta
		msg := event.Message

		var ch, key, vel uint8
		if msg.GetNoteOn(&ch, &key, &vel) && vel > 0 {
			// Vocal notes are in the range C1-C5 (36-84)
			if key >= 36 && key <= 84 {
				// Store note-on time for duration calculation
				noteOnMap[key] = currentTime

				// Get associated lyric (if any)
				lyric := lyricsByTime[currentTime]

				vocalNotes = append(vocalNotes, VocalNote{
					Time:     currentTime,
					Key:      key,
					Velocity: vel,
					Duration: 0, // Duration set later when note-off is found
					Lyric:    lyric,
				})
			}
		} else if msg.GetNoteOff(&ch, &key, &vel) || (msg.GetNoteOn(&ch, &key, &vel) && vel == 0) {
			// Handle note-off events (including note-on with velocity 0)
			if key >= 36 && key <= 84 {
				if noteOnTime, exists := noteOnMap[key]; exists {
					// Find the corresponding note and update its duration
					for i := len(vocalNotes) - 1; i >= 0; i-- {
						if vocalNotes[i].Key == key && vocalNotes[i].Time == noteOnTime {
							duration := currentTime - noteOnTime
							if duration > 0 {
								vocalNotes[i].Duration = duration
							}
							break
						}
					}
					delete(noteOnMap, key)
				}
			}
		}
	}

	// Filter out any VocalNotes with a duration of 0
	filteredVocalNotes := vocalNotes[:0]
	for _, note := range vocalNotes {
		if note.Duration > 0 {
			filteredVocalNotes = append(filteredVocalNotes, note)
		} else {
			fmt.Printf("Invalid vocal note: Key=%d, Time=%d, missing NoteOff event\n", note.Key, note.Time)
		}
	}
	vocalNotes = filteredVocalNotes

	log.Printf("Extracted %d valid vocal notes", len(vocalNotes))
	return vocalNotes
}
