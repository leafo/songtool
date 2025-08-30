package main

import (
	"fmt"
	"log"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/smf"
)

const gmBassChannel uint8 = 1            // Standard GM bass channel
const gmBassProgram uint8 = 33           // Electric Bass (finger) - GM program 34 (0-indexed as 33)
const bassNoteDurationTicks uint32 = 240 // Half note at 480 ticks per quarter note

// Bass difficulty levels - MIDI note base values for different difficulties
const (
	BassExpertBase = 96 // C6 - Expert difficulty base note
	BassHardBase   = 72 // C4 - Hard difficulty base note
	BassMediumBase = 48 // C2 - Medium difficulty base note
	BassEasyBase   = 24 // C0 - Easy difficulty base note
)

// Bass string mapping for 4-string bass (E-A-D-G standard tuning)
// Based on Rock Band Pro Bass specification
const (
	BassString4 = 0 // E (Low) - Base + 0 (C)
	BassString3 = 1 // A - Base + 1 (C#)
	BassString2 = 2 // D - Base + 2 (D)
	BassString1 = 3 // G - Base + 3 (D#)
)

// BassDifficulty represents the difficulty level for pro bass tracks
type BassDifficulty int

const (
	BassExpert BassDifficulty = iota
	BassHard
	BassMedium
	BassEasy
)

// BassNote represents a single bass note with all its attributes
type BassNote struct {
	Time     uint32 // Absolute timing in MIDI ticks
	String   uint8  // Bass string number (0-3 for 4-string bass)
	Fret     uint8  // Fret position (0 = open, 1-22 = fret numbers)
	Velocity uint8  // Original MIDI velocity
	Channel  uint8  // MIDI channel (technique indicator)
	RawKey   uint8  // Original MIDI key for debugging
}

// BassTrackInfo contains information about a bass difficulty track
type BassTrackInfo struct {
	TrackName  string
	Difficulty BassDifficulty
	BaseNote   uint8    // MIDI base note for this difficulty
	NoteRange  [2]uint8 // [min, max] MIDI note range for this difficulty
}

// Bass track configuration for different difficulties
var bassTrackConfigs = map[string]BassTrackInfo{
	"PART REAL_BASS_X": {
		TrackName:  "PART REAL_BASS_X",
		Difficulty: BassExpert,
		BaseNote:   BassExpertBase,
		NoteRange:  [2]uint8{96, 99}, // C6 to D#6
	},
	"PART REAL_BASS_H": {
		TrackName:  "PART REAL_BASS_H",
		Difficulty: BassHard,
		BaseNote:   BassHardBase,
		NoteRange:  [2]uint8{72, 75}, // C4 to D#4
	},
	"PART REAL_BASS_M": {
		TrackName:  "PART REAL_BASS_M",
		Difficulty: BassMedium,
		BaseNote:   BassMediumBase,
		NoteRange:  [2]uint8{48, 51}, // C2 to D#2
	},
	"PART REAL_BASS_E": {
		TrackName:  "PART REAL_BASS_E",
		Difficulty: BassEasy,
		BaseNote:   BassEasyBase,
		NoteRange:  [2]uint8{24, 27}, // C0 to D#0
	},
	// Combined track containing all difficulties
	"PART REAL_BASS": {
		TrackName:  "PART REAL_BASS",
		Difficulty: BassExpert, // Default to expert for combined tracks
		BaseNote:   BassExpertBase,
		NoteRange:  [2]uint8{96, 99}, // C6 to D#6 (expert range)
	},
}

// toMidiNote converts a BassNote to a MIDI note number based on string and fret
// Uses standard 4-string bass tuning: E(28), A(33), D(38), G(43)
func (bn *BassNote) toMidiNote() (uint8, error) {
	// Standard bass tuning in MIDI note numbers (E1, A1, D2, G2)
	baseTuning := [4]uint8{28, 33, 38, 43} // E, A, D, G

	if bn.String > 3 {
		return 0, fmt.Errorf("invalid bass string number: %d (must be 0-3)", bn.String)
	}

	if bn.Fret > 22 {
		return 0, fmt.Errorf("invalid fret number: %d (must be 0-22)", bn.Fret)
	}

	midiNote := baseTuning[bn.String] + bn.Fret
	if midiNote > 127 {
		return 0, fmt.Errorf("resulting MIDI note %d exceeds maximum (127)", midiNote)
	}

	return midiNote, nil
}

// getTechniqueInfo returns human-readable technique information based on MIDI channel
func (bn *BassNote) getTechniqueInfo() string {
	switch bn.Channel {
	case 1:
		return "Normal"
	case 2:
		return "Arpeggio"
	case 3:
		return "Bend"
	case 4:
		return "Muted"
	case 5:
		return "HOPO" // Hammer-on/Pull-off
	case 6:
		return "Harmonic"
	case 12:
		return "Reverse Slide"
	case 13:
		return "Force HOPO Off"
	default:
		return fmt.Sprintf("Unknown (ch %d)", bn.Channel)
	}
}

// AddBassTracks extracts expert difficulty bass from a Rock Band MIDI file
// and adds it as GM bass to the exporter
func (e *GeneralMidiExporter) AddBassTracks(sourceData *smf.SMF) error {
	// Try to find expert pro bass track first, then fall back to combined track
	trackConfig, track, found := findBassTrack(sourceData, "PART REAL_BASS_X")
	if !found {
		// Try combined track format
		trackConfig, track, found = findBassTrack(sourceData, "PART REAL_BASS")
		if !found {
			return fmt.Errorf("no pro bass track found (tried 'PART REAL_BASS_X' and 'PART REAL_BASS')")
		}
		log.Printf("Found combined pro bass track, extracting expert difficulty")
	} else {
		log.Printf("Found dedicated expert pro bass track")
	}

	// Extract bass notes from the track
	bassNotes := extractBassNotes(track, trackConfig)
	if len(bassNotes) == 0 {
		return fmt.Errorf("no expert pro bass notes found")
	}

	log.Printf("Found %d pro bass notes", len(bassNotes))

	// Convert bass notes to MIDI events
	var events []MidiEvent

	for i, note := range bassNotes {
		// Convert to GM bass note
		gmNote, err := note.toMidiNote()
		if err != nil {
			log.Printf("Error converting bass note to MIDI: %v", err)
			continue
		}

		// Add Note On event
		noteOnMsg := smf.Message(midi.NoteOn(gmBassChannel, gmNote, note.Velocity))
		events = append(events, MidiEvent{Time: note.Time, Message: noteOnMsg})

		// Calculate end time with overlap detection
		endTime := note.Time + bassNoteDurationTicks
		for j := i + 1; j < len(bassNotes); j++ {
			nextNote := bassNotes[j]
			if nextNote.Time >= endTime {
				break
			}
			nextGmNote, err := nextNote.toMidiNote()
			if err != nil {
				continue
			}
			// End current note if same MIDI note starts
			if nextGmNote == gmNote {
				endTime = nextNote.Time
				break
			}
		}

		// Add Note Off event
		noteOffMsg := smf.Message(midi.NoteOff(gmBassChannel, gmNote))
		events = append(events, MidiEvent{Time: endTime, Message: noteOffMsg})
	}

	// Add bass track to exporter
	bassTrackInfo := TrackInfo{
		Name:    "Pro Bass",
		Channel: gmBassChannel,
		Program: gmBassProgram,
		Events:  events,
	}

	return e.addTrack(bassTrackInfo)
}

// findBassTrack locates a specific bass track in the MIDI file
func findBassTrack(sourceData *smf.SMF, trackName string) (BassTrackInfo, smf.Track, bool) {
	config, exists := bassTrackConfigs[trackName]
	if !exists {
		return BassTrackInfo{}, nil, false
	}

	for _, track := range sourceData.Tracks {
		if getTrackName(track) == trackName {
			return config, track, true
		}
	}

	return BassTrackInfo{}, nil, false
}

// extractBassNotes finds all pro bass notes in the specified track and difficulty
func extractBassNotes(track smf.Track, config BassTrackInfo) []BassNote {
	var bassNotes []BassNote
	var currentTime uint32

	for _, event := range track {
		currentTime += event.Delta
		msg := event.Message

		var ch, key, vel uint8
		if msg.GetNoteOn(&ch, &key, &vel) && vel > 0 {
			// Check if this note is in the bass range for this difficulty
			if key >= config.NoteRange[0] && key <= config.NoteRange[1] {
				// Convert MIDI key to string and fret
				stringNum := key - config.BaseNote
				fret := getFretFromVelocity(vel)

				if stringNum <= 3 && fret <= 22 { // Valid bass string and fret range
					bassNotes = append(bassNotes, BassNote{
						Time:     currentTime,
						String:   stringNum,
						Fret:     fret,
						Velocity: vel,
						Channel:  ch,
						RawKey:   key,
					})
				}
			}
		}
	}

	log.Printf("Extracted %d bass notes from %s", len(bassNotes), config.TrackName)
	return bassNotes
}

// getFretFromVelocity converts MIDI velocity to fret position
// Rock Band Pro format: velocity 100 = open string, 101+ = fret numbers
func getFretFromVelocity(velocity uint8) uint8 {
	if velocity < 100 {
		return 0 // Treat as open string if velocity is below expected range
	}

	fret := velocity - 100
	if fret > 22 {
		return 22 // Cap at 22nd fret maximum
	}

	return fret
}
