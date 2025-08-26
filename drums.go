package main

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sort"
	"strings"

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

// ExportDrumsFromMidi extracts expert difficulty drums from a Rock Band MIDI file
// and converts them to GM standard drums, outputting to the provided writer
func ExportDrumsFromMidi(smfData *smf.SMF, writer io.Writer) error {
	// Find the PART DRUMS track
	var drumTrack smf.Track
	var drumTrackFound bool

	for _, track := range smfData.Tracks {
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

	// Extract Expert difficulty drum notes (MIDI range 96-100)
	expertDrumNotes := extractDrumNotes(drumTrack)

	if len(expertDrumNotes) == 0 {
		return fmt.Errorf("no expert drum notes found")
	}

	// Create new MIDI file with GM drums (Format 1 - multi-track)
	// Use the same time format as the original file
	newSMF := smf.NewSMF1()
	newSMF.TimeFormat = smfData.TimeFormat // Preserve original timing resolution

	// Track 0: Copy tempo/conductor information from original MIDI
	tempoTrack := extractTempoTrack(smfData)
	newSMF.Add(tempoTrack)

	// Track 1: Drums track
	drumsTrack := smf.Track{}

	// Add track name meta event
	trackNameMsg := smf.Message(smf.MetaTrackSequenceName("Drums"))
	drumsTrack = append(drumsTrack, smf.Event{Delta: 0, Message: trackNameMsg})

	// Collect all MIDI events with absolute timestamps
	type midiEvent struct {
		time    uint32
		message smf.Message
	}

	var allEvents []midiEvent

	// generate events for drum notes
	for _, note := range expertDrumNotes {
		// Convert to GM drums (handles both regular and Pro Drums)
		gmNote, err := note.toMidiKey()
		if err != nil {
			log.Printf("Error converting drum note to General MIDI key: %v\n", err)
			continue
		}

		// Note On event
		noteOnMsg := smf.Message(midi.NoteOn(gmDrumChannel, gmNote, note.Velocity))
		allEvents = append(allEvents, midiEvent{time: note.Time, message: noteOnMsg})

		// Note Off event at note time + duration
		noteOffMsg := smf.Message(midi.NoteOff(gmDrumChannel, gmNote))
		allEvents = append(allEvents, midiEvent{time: note.Time + hitDurationTicks, message: noteOffMsg})
	}

	// Sort events by time
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].time < allEvents[j].time
	})

	// Add events to track with proper delta times
	var lastTime uint32
	for _, event := range allEvents {
		delta := event.time - lastTime
		drumsTrack = append(drumsTrack, smf.Event{Delta: delta, Message: event.message})
		lastTime = event.time
	}

	// Add end of track meta event to drums track
	drumsTrack = append(drumsTrack, smf.Event{Delta: 0, Message: smf.EOT})
	newSMF.Add(drumsTrack)

	// Write the new MIDI file to the provided writer
	_, err := newSMF.WriteTo(writer)
	if err != nil {
		return fmt.Errorf("error writing MIDI file: %w", err)
	}

	return nil
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
		} else if msg.GetNoteOff(&ch, &key, &vel) {
			// Update end time for tom modifiers
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

// generateDrumOutputFilename creates an appropriate output filename for the GM drum export
func generateDrumOutputFilename(sourceFilename string) string {
	base := strings.TrimSuffix(sourceFilename, filepath.Ext(sourceFilename))
	if strings.Contains(base, "(from SNG)") {
		base = strings.Replace(base, "notes.mid (from SNG)", "drums_gm", 1)
	} else {
		base = base + "_drums_gm"
	}
	return base + ".mid"
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
			// Ignore all other events
		}
	}

	// Always end with End of Track
	tempoTrack = append(tempoTrack, smf.Event{Delta: 0, Message: smf.EOT})
	return tempoTrack
}
