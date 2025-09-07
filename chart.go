package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type ChartFile struct {
	Song      SongSection             `json:"song"`
	SyncTrack SyncTrackSection        `json:"syncTrack"`
	Events    EventsSection           `json:"events"`
	Tracks    map[string]TrackSection `json:"tracks"`
	Filename  string                  `json:"filename"`
}

type SongSection struct {
	Name         string `json:"name,omitempty"`
	Artist       string `json:"artist,omitempty"`
	Charter      string `json:"charter,omitempty"`
	Album        string `json:"album,omitempty"`
	Year         string `json:"year,omitempty"`
	Offset       int    `json:"offset"`
	Resolution   int    `json:"resolution"`
	Player2      string `json:"player2,omitempty"`
	Difficulty   int    `json:"difficulty"`
	PreviewStart int    `json:"previewStart"`
	PreviewEnd   int    `json:"previewEnd"`
	Genre        string `json:"genre,omitempty"`
	MediaType    string `json:"mediaType,omitempty"`
	MusicStream  string `json:"musicStream,omitempty"`
	GuitarStream string `json:"guitarStream,omitempty"`
	RhythmStream string `json:"rhythmStream,omitempty"`
	BassStream   string `json:"bassStream,omitempty"`
	DrumStream   string `json:"drumStream,omitempty"`
	Drum2Stream  string `json:"drum2Stream,omitempty"`
	Drum3Stream  string `json:"drum3Stream,omitempty"`
	Drum4Stream  string `json:"drum4Stream,omitempty"`
	VocalStream  string `json:"vocalStream,omitempty"`
	KeysStream   string `json:"keysStream,omitempty"`
	CrowdStream  string `json:"crowdStream,omitempty"`
}

type SyncTrackSection struct {
	BPMEvents     []BPMEvent     `json:"bpmEvents"`
	TimeSigEvents []TimeSigEvent `json:"timeSigEvents"`
	AnchorEvents  []AnchorEvent  `json:"anchorEvents"`
}

type BPMEvent struct {
	Tick uint32 `json:"tick"`
	BPM  uint32 `json:"bpm"` // BPM * 1000
}

type TimeSigEvent struct {
	Tick        uint32 `json:"tick"`
	Numerator   uint8  `json:"numerator"`
	Denominator uint8  `json:"denominator"` // stored as log2 of actual denominator
}

type AnchorEvent struct {
	Tick         uint32 `json:"tick"`
	Microseconds uint64 `json:"microseconds"`
}

type EventsSection struct {
	GlobalEvents []GlobalEvent `json:"globalEvents"`
}

type GlobalEvent struct {
	Tick uint32 `json:"tick"`
	Text string `json:"text"`
}

type TrackSection struct {
	Name        string         `json:"name"`
	Notes       []NoteEvent    `json:"notes"`
	Specials    []SpecialEvent `json:"specials"`
	TrackEvents []TrackEvent   `json:"trackEvents"`
}

// NoteFlags represents various flags that can be applied to notes
type NoteFlags int

const (
	FlagNone       NoteFlags = 0
	FlagForced     NoteFlags = 1 << iota // Note 5 in chart
	FlagTap                              // Note 6 in chart
	FlagOpen                             // Note 7 in chart
	FlagDoubleKick                       // Drums: Note 32
	FlagCymbal                           // Pro drums: Notes 66,67,68
	FlagAccent                           // Drums: Notes 34-39
	FlagGhost                            // Drums: Notes 40-45
)

type NoteEvent struct {
	Tick    uint32    `json:"tick"`
	Fret    uint8     `json:"fret"`
	Sustain uint32    `json:"sustain"`
	Flags   NoteFlags `json:"flags"`
}

type SpecialEvent struct {
	Tick   uint32 `json:"tick"`
	Type   uint8  `json:"type"`
	Length uint32 `json:"length"`
}

type TrackEvent struct {
	Tick uint32 `json:"tick"`
	Text string `json:"text"`
}

// PendingFlag represents a flag that needs to be applied to notes after all notes are parsed
type PendingFlag struct {
	Tick     uint32
	NoteNum  int
	Flag     NoteFlags
	ApplyAll bool // If true, apply to all notes at this tick
}

func OpenChartFile(filename string) (*ChartFile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening chart file: %w", err)
	}
	defer file.Close()

	chart, err := ParseChartFile(file)
	if err != nil {
		return nil, fmt.Errorf("error parsing chart file: %w", err)
	}

	chart.Filename = filename
	return chart, nil
}

func ParseChartFile(reader io.Reader) (*ChartFile, error) {
	chart := &ChartFile{
		Tracks: make(map[string]TrackSection),
	}

	scanner := bufio.NewScanner(reader)
	var currentSection string
	var inSection bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Handle BOM (Byte Order Mark) if present at start of file
		if strings.HasPrefix(line, "\ufeff") {
			line = strings.TrimPrefix(line, "\ufeff")
		}

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			// Validate section name is not empty
			if strings.TrimSpace(currentSection) == "" {
				return nil, fmt.Errorf("empty section name at line: %s", line)
			}
			inSection = false
			continue
		}

		// Check for section start
		if line == "{" {
			inSection = true
			continue
		}

		// Check for section end
		if line == "}" {
			inSection = false
			currentSection = ""
			continue
		}

		if !inSection || currentSection == "" {
			continue
		}

		// Parse section content
		err := parseSectionLine(chart, currentSection, line)
		if err != nil {
			return nil, fmt.Errorf("error parsing line '%s' in section '%s': %w", line, currentSection, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading chart file: %w", err)
	}

	// Validate the parsed chart
	if err := validateChart(chart); err != nil {
		return nil, fmt.Errorf("chart validation failed: %w", err)
	}

	return chart, nil
}

func parseSectionLine(chart *ChartFile, section, line string) error {
	switch section {
	case "Song":
		return parseSongLine(chart, line)
	case "SyncTrack":
		return parseSyncTrackLine(chart, line)
	case "Events":
		return parseEventsLine(chart, line)
	default:
		// Check if it's a track section
		if isTrackSection(section) {
			return parseTrackLine(chart, section, line)
		}
	}
	return nil
}

func parseSongLine(chart *ChartFile, line string) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		// Skip malformed lines but don't return error to allow parsing to continue
		return nil
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Remove quotes and handle escape sequences
	value = unquoteString(value)

	switch key {
	case "Name":
		chart.Song.Name = value
	case "Artist":
		chart.Song.Artist = value
	case "Charter":
		chart.Song.Charter = value
	case "Album":
		chart.Song.Album = value
	case "Year":
		chart.Song.Year = value
	case "Offset":
		if val, err := strconv.Atoi(value); err == nil {
			chart.Song.Offset = val
		}
	case "Resolution":
		if val, err := strconv.Atoi(value); err == nil {
			chart.Song.Resolution = val
		}
	case "Player2":
		chart.Song.Player2 = value
	case "Difficulty":
		if val, err := strconv.Atoi(value); err == nil {
			chart.Song.Difficulty = val
		}
	case "PreviewStart":
		if val, err := strconv.Atoi(value); err == nil {
			chart.Song.PreviewStart = val
		}
	case "PreviewEnd":
		if val, err := strconv.Atoi(value); err == nil {
			chart.Song.PreviewEnd = val
		}
	case "Genre":
		chart.Song.Genre = value
	case "MediaType":
		chart.Song.MediaType = value
	case "MusicStream":
		chart.Song.MusicStream = value
	case "GuitarStream":
		chart.Song.GuitarStream = value
	case "RhythmStream":
		chart.Song.RhythmStream = value
	case "BassStream":
		chart.Song.BassStream = value
	case "DrumStream":
		chart.Song.DrumStream = value
	case "Drum2Stream":
		chart.Song.Drum2Stream = value
	case "Drum3Stream":
		chart.Song.Drum3Stream = value
	case "Drum4Stream":
		chart.Song.Drum4Stream = value
	case "VocalStream":
		chart.Song.VocalStream = value
	case "KeysStream":
		chart.Song.KeysStream = value
	case "CrowdStream":
		chart.Song.CrowdStream = value
	}

	return nil
}

func parseSyncTrackLine(chart *ChartFile, line string) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		// Skip malformed lines and continue parsing
		return nil
	}

	tickStr := strings.TrimSpace(parts[0])
	tick, err := strconv.ParseUint(tickStr, 10, 32)
	if err != nil {
		// Skip lines with invalid tick values and continue parsing
		return nil
	}

	eventParts := strings.Fields(strings.TrimSpace(parts[1]))
	if len(eventParts) < 2 {
		return nil
	}

	eventType := eventParts[0]

	switch eventType {
	case "B": // BPM event
		if len(eventParts) >= 2 {
			if bpm, err := strconv.ParseUint(eventParts[1], 10, 32); err == nil {
				chart.SyncTrack.BPMEvents = append(chart.SyncTrack.BPMEvents, BPMEvent{
					Tick: uint32(tick),
					BPM:  uint32(bpm), // BPM stored as BPM * 1000 per spec
				})
			}
		}
	case "TS": // Time Signature event
		if len(eventParts) >= 2 {
			if num, err := strconv.ParseUint(eventParts[1], 10, 8); err == nil {
				timeSig := TimeSigEvent{
					Tick:        uint32(tick),
					Numerator:   uint8(num),
					Denominator: 2, // Default is 4/4, stored as log2(4) = 2
				}
				if len(eventParts) >= 3 {
					if denom, err := strconv.ParseUint(eventParts[2], 10, 8); err == nil {
						timeSig.Denominator = uint8(denom)
					}
				}
				chart.SyncTrack.TimeSigEvents = append(chart.SyncTrack.TimeSigEvents, timeSig)
			}
		}
	case "A": // Anchor event
		if len(eventParts) >= 2 {
			if us, err := strconv.ParseUint(eventParts[1], 10, 64); err == nil {
				chart.SyncTrack.AnchorEvents = append(chart.SyncTrack.AnchorEvents, AnchorEvent{
					Tick:         uint32(tick),
					Microseconds: us,
				})
			}
		}
	}

	return nil
}

func parseEventsLine(chart *ChartFile, line string) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		// Skip malformed lines and continue parsing
		return nil
	}

	tickStr := strings.TrimSpace(parts[0])
	tick, err := strconv.ParseUint(tickStr, 10, 32)
	if err != nil {
		// Skip lines with invalid tick values and continue parsing
		return nil
	}

	eventParts := strings.Fields(strings.TrimSpace(parts[1]))
	if len(eventParts) < 2 || eventParts[0] != "E" {
		return nil
	}

	// Join remaining parts and remove quotes with escape sequence handling
	text := strings.Join(eventParts[1:], " ")
	text = unquoteString(text)

	chart.Events.GlobalEvents = append(chart.Events.GlobalEvents, GlobalEvent{
		Tick: uint32(tick),
		Text: text,
	})

	return nil
}

func parseTrackLine(chart *ChartFile, section, line string) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		// Skip malformed lines and continue parsing
		return nil
	}

	tickStr := strings.TrimSpace(parts[0])
	tick, err := strconv.ParseUint(tickStr, 10, 32)
	if err != nil {
		// Skip lines with invalid tick values and continue parsing
		return nil
	}

	eventParts := strings.Fields(strings.TrimSpace(parts[1]))
	if len(eventParts) < 2 {
		return nil
	}

	// Initialize track if it doesn't exist
	if _, exists := chart.Tracks[section]; !exists {
		chart.Tracks[section] = TrackSection{
			Name: section,
		}
	}

	track := chart.Tracks[section]
	eventType := eventParts[0]

	switch eventType {
	case "N": // Note event
		if len(eventParts) >= 3 {
			if fret, err := strconv.ParseUint(eventParts[1], 10, 8); err == nil {
				if sustain, err := strconv.ParseUint(eventParts[2], 10, 32); err == nil {
					note := NoteEvent{
						Tick:    uint32(tick),
						Fret:    uint8(fret),
						Sustain: uint32(sustain),
						Flags:   FlagNone,
					}

					// Handle special note types based on fret number
					switch fret {
					case 5: // Forced flag - don't add as note, will need post-processing
						// Skip for now - would need proper flag processing system
						return nil
					case 6: // Tap flag - don't add as note, will need post-processing
						// Skip for now - would need proper flag processing system
						return nil
					case 7: // Open note
						note.Flags |= FlagOpen
					case 32: // Double kick (drums)
						note.Fret = 0 // Convert to kick
						note.Flags |= FlagDoubleKick
					default:
						// Check for drum accent/ghost flags
						if fret >= 34 && fret <= 39 { // Accent flags
							// Skip for now - would need proper flag processing system
							return nil
						}
						if fret >= 40 && fret <= 45 { // Ghost flags
							// Skip for now - would need proper flag processing system
							return nil
						}
						if fret >= 66 && fret <= 68 { // Cymbal flags
							// Skip for now - would need proper flag processing system
							return nil
						}
					}

					track.Notes = append(track.Notes, note)
				}
			}
		}
	case "S": // Special event
		if len(eventParts) >= 3 {
			if sType, err := strconv.ParseUint(eventParts[1], 10, 8); err == nil {
				if length, err := strconv.ParseUint(eventParts[2], 10, 32); err == nil {
					track.Specials = append(track.Specials, SpecialEvent{
						Tick:   uint32(tick),
						Type:   uint8(sType),
						Length: uint32(length),
					})
				}
			}
		}
	case "E": // Track event
		if len(eventParts) >= 2 {
			text := strings.Join(eventParts[1:], " ")
			track.TrackEvents = append(track.TrackEvents, TrackEvent{
				Tick: uint32(tick),
				Text: text,
			})
		}
	}

	chart.Tracks[section] = track
	return nil
}

// sectionNameToTrackInfo maps section names to track information
var sectionNameToTrackInfo = map[string]bool{
	// Guitar tracks
	"EasySingle":   true,
	"MediumSingle": true,
	"HardSingle":   true,
	"ExpertSingle": true,

	// Guitar Coop tracks
	"EasyDoubleGuitar":   true,
	"MediumDoubleGuitar": true,
	"HardDoubleGuitar":   true,
	"ExpertDoubleGuitar": true,

	// Bass tracks
	"EasyDoubleBass":   true,
	"MediumDoubleBass": true,
	"HardDoubleBass":   true,
	"ExpertDoubleBass": true,

	// Rhythm tracks
	"EasyDoubleRhythm":   true,
	"MediumDoubleRhythm": true,
	"HardDoubleRhythm":   true,
	"ExpertDoubleRhythm": true,

	// Drums tracks
	"EasyDrums":   true,
	"MediumDrums": true,
	"HardDrums":   true,
	"ExpertDrums": true,

	// Keys tracks
	"EasyKeyboard":   true,
	"MediumKeyboard": true,
	"HardKeyboard":   true,
	"ExpertKeyboard": true,

	// GH Live Guitar tracks
	"EasyGHLGuitar":   true,
	"MediumGHLGuitar": true,
	"HardGHLGuitar":   true,
	"ExpertGHLGuitar": true,

	// GH Live Bass tracks
	"EasyGHLBass":   true,
	"MediumGHLBass": true,
	"HardGHLBass":   true,
	"ExpertGHLBass": true,

	// GH Live Rhythm tracks
	"EasyGHLRhythm":   true,
	"MediumGHLRhythm": true,
	"HardGHLRhythm":   true,
	"ExpertGHLRhythm": true,

	// GH Live Coop tracks
	"EasyGHLCoop":   true,
	"MediumGHLCoop": true,
	"HardGHLCoop":   true,
	"ExpertGHLCoop": true,
}

func isTrackSection(section string) bool {
	return sectionNameToTrackInfo[section]
}

// validateChart performs basic validation on the parsed chart
func validateChart(chart *ChartFile) error {
	// Check resolution is valid
	if chart.Song.Resolution == 0 {
		return fmt.Errorf("invalid resolution: %d", chart.Song.Resolution)
	}

	// Check that we have at least one tempo event
	if len(chart.SyncTrack.BPMEvents) == 0 {
		// This is a common issue - add a default BPM event rather than failing
		chart.SyncTrack.BPMEvents = append(chart.SyncTrack.BPMEvents, BPMEvent{
			Tick: 0,
			BPM:  120000, // Default 120 BPM * 1000
		})
	}

	// Check that first tempo event is at tick 0 or very close to it
	if len(chart.SyncTrack.BPMEvents) > 0 && chart.SyncTrack.BPMEvents[0].Tick > uint32(chart.Song.Resolution) {
		return fmt.Errorf("first BPM event should be near the beginning of the chart")
	}

	// Validate BPM values are reasonable (stored as BPM * 1000)
	for _, bpmEvent := range chart.SyncTrack.BPMEvents {
		actualBPM := float64(bpmEvent.BPM) / 1000.0
		if actualBPM < 1.0 || actualBPM > 1000.0 {
			return fmt.Errorf("invalid BPM value: %f at tick %d", actualBPM, bpmEvent.Tick)
		}
	}

	// Check that we have at least one track (warn but don't fail)
	if len(chart.Tracks) == 0 {
		// This is unusual but not necessarily an error - continue parsing
	}

	// Validate each track
	for trackName, track := range chart.Tracks {
		if err := validateTrack(&track, trackName); err != nil {
			return fmt.Errorf("track validation failed for %s: %w", trackName, err)
		}
	}

	return nil
}

// validateTrack performs validation on an individual track
func validateTrack(track *TrackSection, trackName string) error {
	if track == nil {
		return fmt.Errorf("track is nil")
	}

	// Get expected max fret for this track type
	maxFret := getMaxFretForTrack(trackName)

	// Validate note fret ranges
	for i, note := range track.Notes {
		if note.Fret < 0 || int(note.Fret) > maxFret {
			return fmt.Errorf("note %d has invalid fret %d for track %s (max: %d)",
				i, note.Fret, trackName, maxFret)
		}
	}

	return nil
}

// getMaxFretForTrack returns the maximum fret number for a track type
func getMaxFretForTrack(trackName string) int {
	// Determine instrument type from track name
	if strings.Contains(trackName, "Drums") {
		return 5 // 0-5 pads
	} else if strings.Contains(trackName, "GHL") {
		return 8 // GH Live supports up to note 8
	} else {
		return 7 // Guitar/Bass/Keys: 0-4 frets + open (7)
	}
}

// unquoteString removes quotes and handles escape sequences
func unquoteString(s string) string {
	// Remove surrounding quotes if present
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			s = s[1 : len(s)-1]
		}
	}

	// Handle escape sequences
	result := strings.Builder{}
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			case '\'':
				result.WriteByte('\'')
			default:
				// Unknown escape, keep both characters
				result.WriteByte(s[i])
				result.WriteByte(s[i+1])
			}
			i++ // Skip the next character
		} else {
			result.WriteByte(s[i])
		}
	}

	return result.String()
}

func (c *ChartFile) GetBPMAtTick(tick uint32) float64 {
	var currentBPM uint32 = 120000 // Default 120 BPM * 1000

	// Handle case where there are no BPM events
	if len(c.SyncTrack.BPMEvents) == 0 {
		return 120.0
	}

	for _, event := range c.SyncTrack.BPMEvents {
		if event.Tick <= tick {
			currentBPM = event.BPM
		} else {
			break
		}
	}

	return float64(currentBPM) / 1000.0 // Convert from BPM*1000 to actual BPM
}

func (c *ChartFile) GetMetadata() map[string]string {
	result := make(map[string]string)

	if c.Song.Name != "" {
		result["name"] = c.Song.Name
	}
	if c.Song.Artist != "" {
		result["artist"] = c.Song.Artist
	}
	if c.Song.Album != "" {
		result["album"] = c.Song.Album
	}
	if c.Song.Charter != "" {
		result["charter"] = c.Song.Charter
	}
	if c.Song.Year != "" {
		result["year"] = c.Song.Year
	}
	if c.Song.Genre != "" {
		result["genre"] = c.Song.Genre
	}

	return result
}

func (c *ChartFile) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Chart File: %s\n", c.Filename))
	if c.Song.Name != "" {
		sb.WriteString(fmt.Sprintf("Title: %s\n", c.Song.Name))
	}
	if c.Song.Artist != "" {
		sb.WriteString(fmt.Sprintf("Artist: %s\n", c.Song.Artist))
	}
	if c.Song.Album != "" {
		sb.WriteString(fmt.Sprintf("Album: %s\n", c.Song.Album))
	}
	if c.Song.Charter != "" {
		sb.WriteString(fmt.Sprintf("Charter: %s\n", c.Song.Charter))
	}
	sb.WriteString(fmt.Sprintf("Resolution: %d\n", c.Song.Resolution))
	sb.WriteString(fmt.Sprintf("BPM Events: %d\n", len(c.SyncTrack.BPMEvents)))
	sb.WriteString(fmt.Sprintf("Global Events: %d\n", len(c.Events.GlobalEvents)))
	sb.WriteString(fmt.Sprintf("Tracks: %d\n", len(c.Tracks)))

	return sb.String()
}
