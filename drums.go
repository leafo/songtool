package main

import (
	"fmt"
	"log"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/smf"
)

const gmDrumChannel uint8 = 9       // default percussion channel in GM
const hitDurationTicks uint32 = 120 // a 16th note at 480 ticks per quarter note

// all of these mapping are from the Expert drum range
// **MIDI Range:** 96 (C6) - 100 (E6)

// GM Drum mapping for standard MIDI drums
var gmDrumMap = map[uint8]uint8{
	96: BassDrum1,     // Kick Drum -> Bass Drum 1 (GM)
	97: AcousticSnare, // Snare Drum -> Acoustic Snare (GM)

	98:  ClosedHiHat,  // Hi-Hat -> Closed Hi Hat
	99:  RideCymbal1,  // Ride -> Ride Cymbal 1
	100: CrashCymbal1, // Crash -> Crash Cymbal 1
}

// GM mapping when tom modifier is applied (pro drums)
var gmTomMap = map[uint8]uint8{
	98:  LowMidTom,   // Hi-Hat -> Mid Tom 1 (when tom modifier active)
	99:  LowTom,      // Ride -> Low Tom (when tom modifier active)
	100: LowFloorTom, // Crash -> Low Floor Tom (when tom modifier active)
}

// DrumNote represents a single drum hit with timing and velocity
type DrumNote struct {
	Time          uint32
	Key           uint8 // the raw key event from rockband
	Velocity      uint8
	IsTomModified bool // For Pro Drums: true if this note should be a tom instead of cymbal
}

// Represents a range of time where cymbols are converted into toms
// Only applies to the notes that match Pad color
type TomModifier struct {
	StartTime uint32
	EndTime   uint32
	Pad       uint8 // 98 (yellow), 99 (blue), 100 (green)
}

// converts a DrumNote to general MIDI drum key
func (dn *DrumNote) toMidiKey() (uint8, error) {
	var gmKey uint8

	if dn.IsTomModified {
		if gmKeyVal, ok := gmTomMap[dn.Key]; ok {
			gmKey = gmKeyVal
		} else {
			return 0, fmt.Errorf("error: failed to get tom modified GM MIDI note for %d", dn.Key)
		}
	} else {
		if gmKeyVal, ok := gmDrumMap[dn.Key]; ok {
			gmKey = gmKeyVal
		} else {
			return 0, fmt.Errorf("error: failed to get GM MIDI note for %d", dn.Key)
		}
	}

	return gmKey, nil
}

// AddDrumTracks extracts expert difficulty drums from a Rock Band MIDI file
// and adds them as GM standard drums to the exporter
func (e *GeneralMidiExporter) AddDrumTracks(sourceData *smf.SMF) error {
	// Find the PART DRUMS track
	var drumTrack smf.Track
	var drumTrackFound bool

	for _, track := range sourceData.Tracks {
		trackName := getTrackName(track)
		if trackName == "PART DRUMS" {
			drumTrack = track
			drumTrackFound = true
			break
		}
	}

	if !drumTrackFound {
		return fmt.Errorf("no 'PART DRUMS' track found")
	}

	// Extract drum notes
	drumNotes := extractDrumNotes(drumTrack)
	if len(drumNotes) == 0 {
		return fmt.Errorf("no expert drum notes found")
	}

	// Convert drum notes to MIDI events
	var events []MidiEvent

	for i, note := range drumNotes {
		// Convert to GM drums
		gmNote, err := note.toMidiKey()
		if err != nil {
			log.Printf("Error converting drum note to General MIDI key: %v", err)
			continue
		}

		// Add Note On event
		noteOnMsg := smf.Message(midi.NoteOn(gmDrumChannel, gmNote, note.Velocity))
		events = append(events, MidiEvent{Time: note.Time, Message: noteOnMsg})

		// Calculate end time with overlap detection
		endTime := note.Time + hitDurationTicks
		for j := i + 1; j < len(drumNotes); j++ {
			nextNote := drumNotes[j]
			if nextNote.Time >= endTime {
				break
			}
			nextGmNote, err := nextNote.toMidiKey()
			if err != nil {
				continue
			}
			if nextGmNote == gmNote {
				endTime = nextNote.Time
				break
			}
		}

		// Add Note Off event
		noteOffMsg := smf.Message(midi.NoteOff(gmDrumChannel, gmNote))
		events = append(events, MidiEvent{Time: endTime, Message: noteOffMsg})
	}

	// Add drum track to exporter
	drumTrackInfo := TrackInfo{
		Name:    "Drums",
		Channel: gmDrumChannel,
		Program: 0, // Drums don't use program change (channel 9)
		Events:  events,
	}

	return e.addTrack(drumTrackInfo)
}

// extractDrumNotes finds all expert difficulty drum notes (96-100) in the drum track
// Handles both regular drums and Pro Drums with tom modifiers
func extractDrumNotes(drumTrack smf.Track) []DrumNote {
	var drumNotes []DrumNote
	var tomModifiers []TomModifier
	var currentTime uint32

	isTomModified := func(time uint32, key uint8) bool {
		for _, modifier := range tomModifiers {
			if modifier.Pad == key &&
				time >= modifier.StartTime &&
				time <= modifier.EndTime &&
				(key == 98 || key == 99 || key == 100) {
				return true
			}
		}
		return false
	}

	// first pass: collect all tom modifier events and ranges
	for _, event := range drumTrack {
		currentTime += event.Delta
		msg := event.Message

		var ch, key, vel uint8
		if msg.GetNoteOn(&ch, &key, &vel) && vel > 0 {
			// Tom modifiers (110-112)
			if key >= 110 && key <= 112 {
				padNote := uint8(98 + (key - 110)) // Map 110->98, 111->99, 112->100
				tomModifiers = append(tomModifiers, TomModifier{
					StartTime: currentTime,
					EndTime:   currentTime, // Will be updated when we find the note off
					Pad:       padNote,
				})
			}
		} else if msg.GetNoteOff(&ch, &key, &vel) || (msg.GetNoteOn(&ch, &key, &vel) && vel == 0) {
			// Update end time for tom modifiers (handles both explicit NoteOff and NoteOn with velocity 0)
			if key >= 110 && key <= 112 {
				padNote := uint8(98 + (key - 110))
				// Find the most recent tom modifier for this pad and update its end time
				for i := len(tomModifiers) - 1; i >= 0; i-- {
					if tomModifiers[i].Pad == padNote && tomModifiers[i].EndTime == tomModifiers[i].StartTime {
						tomModifiers[i].EndTime = currentTime
						break
					}
				}
			}
		}
	}

	// Second pass: collect drum notes
	currentTime = 0
	for _, event := range drumTrack {
		currentTime += event.Delta
		msg := event.Message

		var ch, key, vel uint8
		if msg.GetNoteOn(&ch, &key, &vel) && vel > 0 {
			// Expert drums are in the range 96-100 (C6-E6)
			if key >= 96 && key <= 100 {
				drumNotes = append(drumNotes, DrumNote{
					Time:          currentTime,
					Key:           key,
					Velocity:      vel,
					IsTomModified: isTomModified(currentTime, key),
				})
			}
		}
	}

	return drumNotes
}
