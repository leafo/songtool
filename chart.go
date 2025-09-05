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

type NoteEvent struct {
	Tick    uint32 `json:"tick"`
	Fret    uint8  `json:"fret"`
	Sustain uint32 `json:"sustain"`
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

		// Handle BOM if present
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
		return nil // Skip malformed lines
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Remove quotes if present
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}

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
		return nil
	}

	tickStr := strings.TrimSpace(parts[0])
	tick, err := strconv.ParseUint(tickStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid tick value '%s': %w", tickStr, err)
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
					BPM:  uint32(bpm),
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
		return nil
	}

	tickStr := strings.TrimSpace(parts[0])
	tick, err := strconv.ParseUint(tickStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid tick value '%s': %w", tickStr, err)
	}

	eventParts := strings.Fields(strings.TrimSpace(parts[1]))
	if len(eventParts) < 2 || eventParts[0] != "E" {
		return nil
	}

	// Join remaining parts and remove quotes
	text := strings.Join(eventParts[1:], " ")
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		text = text[1 : len(text)-1]
	}

	chart.Events.GlobalEvents = append(chart.Events.GlobalEvents, GlobalEvent{
		Tick: uint32(tick),
		Text: text,
	})

	return nil
}

func parseTrackLine(chart *ChartFile, section, line string) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return nil
	}

	tickStr := strings.TrimSpace(parts[0])
	tick, err := strconv.ParseUint(tickStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid tick value '%s': %w", tickStr, err)
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
					track.Notes = append(track.Notes, NoteEvent{
						Tick:    uint32(tick),
						Fret:    uint8(fret),
						Sustain: uint32(sustain),
					})
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

func isTrackSection(section string) bool {
	difficulties := []string{"Easy", "Medium", "Hard", "Expert"}
	instruments := []string{"Single", "DoubleGuitar", "DoubleBass", "DoubleRhythm", "Drums", "Keyboard", "GHLGuitar", "GHLBass", "GHLCoop", "GHLRhythm"}

	for _, diff := range difficulties {
		for _, inst := range instruments {
			if section == diff+inst {
				return true
			}
		}
	}
	return false
}

func (c *ChartFile) GetBPMAtTick(tick uint32) float64 {
	var currentBPM uint32 = 120000 // Default 120 BPM

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

	return float64(currentBPM) / 1000.0
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
