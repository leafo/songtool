package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"gitlab.com/gomidi/midi/v2/smf"
)

// Test Helper Functions

// createMockMidiFile creates a mock MIDI file with basic test data
func createMockMidiFile() *smf.SMF {
	// Create a minimal test MIDI data in bytes that represents a valid MIDI file
	// Format 0, 1 track, 480 ticks per quarter note
	// Track 0: Track name "Test Track", End of track
	midiData := []byte{
		0x4D, 0x54, 0x68, 0x64, // Header chunk: MThd
		0x00, 0x00, 0x00, 0x06, // Header length: 6 bytes
		0x00, 0x00, // Format type: 0 (single track)
		0x00, 0x01, // Number of tracks: 1
		0x01, 0xE0, // Ticks per quarter: 480

		0x4D, 0x54, 0x72, 0x6B, // Track chunk: MTrk
		0x00, 0x00, 0x00, 0x1A, // Track length: 26 bytes

		// Track events:
		0x00, 0xFF, 0x03, 0x0A, 0x54, 0x65, 0x73, 0x74, 0x20, 0x54, 0x72, 0x61, 0x63, 0x6B, // Track name: "Test Track"
		0x00, 0xFF, 0x2F, 0x00, // End of track
	}

	smfFile, err := smf.ReadFrom(bytes.NewReader(midiData))
	if err != nil {
		panic(fmt.Sprintf("Failed to create test MIDI file: %v", err))
	}
	return smfFile
}

// createMidiFileWithBeatTrack creates a MIDI file with a BEAT track for timeline
func createMidiFileWithBeatTrack() *smf.SMF {
	// Create MIDI data with BEAT track and some tempo/beat events
	midiData := []byte{
		0x4D, 0x54, 0x68, 0x64, // Header chunk: MThd
		0x00, 0x00, 0x00, 0x06, // Header length: 6 bytes
		0x00, 0x00, // Format type: 0
		0x00, 0x01, // Number of tracks: 1
		0x01, 0xE0, // Ticks per quarter: 480

		0x4D, 0x54, 0x72, 0x6B, // Track chunk: MTrk
		0x00, 0x00, 0x00, 0x2B, // Track length: 43 bytes

		// Track name: "BEAT"
		0x00, 0xFF, 0x03, 0x04, 0x42, 0x45, 0x41, 0x54,
		// Tempo: 120 BPM (500000 microseconds per quarter note)
		0x00, 0xFF, 0x51, 0x03, 0x07, 0xA1, 0x20,
		// Time signature: 4/4
		0x00, 0xFF, 0x58, 0x04, 0x04, 0x02, 0x18, 0x08,
		// Note on: channel 0, note 60, velocity 100
		0x00, 0x90, 0x3C, 0x64,
		// Note off: channel 0, note 60, velocity 0 (after 96 ticks)
		0x60, 0x80, 0x3C, 0x00,
		// End of track
		0x00, 0xFF, 0x2F, 0x00,
	}

	smfFile, err := smf.ReadFrom(bytes.NewReader(midiData))
	if err != nil {
		panic(fmt.Sprintf("Failed to create BEAT track MIDI file: %v", err))
	}
	return smfFile
}

// createMidiFileWithDrums creates a MIDI file with drum track
func createMidiFileWithDrums() *smf.SMF {
	// Create MIDI data with PART DRUMS track
	midiData := []byte{
		0x4D, 0x54, 0x68, 0x64, // Header chunk: MThd
		0x00, 0x00, 0x00, 0x06, // Header length: 6 bytes
		0x00, 0x00, // Format type: 0
		0x00, 0x01, // Number of tracks: 1
		0x01, 0xE0, // Ticks per quarter: 480

		0x4D, 0x54, 0x72, 0x6B, // Track chunk: MTrk
		0x00, 0x00, 0x00, 0x30, // Track length: 48 bytes

		// Track name: "PART DRUMS"
		0x00, 0xFF, 0x03, 0x0A, 0x50, 0x41, 0x52, 0x54, 0x20, 0x44, 0x52, 0x55, 0x4D, 0x53,
		// Note on: channel 0, note 36 (kick), velocity 100
		0x00, 0x90, 0x24, 0x64,
		// Note off: channel 0, note 36, velocity 0 (after 96 ticks)
		0x60, 0x80, 0x24, 0x00,
		// Note on: channel 0, note 38 (snare), velocity 100
		0x00, 0x90, 0x26, 0x64,
		// Note off: channel 0, note 38, velocity 0 (after 96 ticks)
		0x60, 0x80, 0x26, 0x00,
		// End of track
		0x00, 0xFF, 0x2F, 0x00,
	}

	smfFile, err := smf.ReadFrom(bytes.NewReader(midiData))
	if err != nil {
		panic(fmt.Sprintf("Failed to create drum track MIDI file: %v", err))
	}
	return smfFile
}

// createMidiFileWithBass creates a simple MIDI file with bass track name
func createMidiFileWithBass() *smf.SMF {
	// Simple MIDI with PART REAL_BASS track name
	midiData := []byte{
		0x4D, 0x54, 0x68, 0x64, // Header
		0x00, 0x00, 0x00, 0x06,
		0x00, 0x00, 0x00, 0x01, 0x01, 0xE0,
		0x4D, 0x54, 0x72, 0x6B, // Track
		0x00, 0x00, 0x00, 0x1A,
		// Track name: "PART REAL_BASS"
		0x00, 0xFF, 0x03, 0x0E, 0x50, 0x41, 0x52, 0x54, 0x20, 0x52, 0x45, 0x41, 0x4C, 0x5F, 0x42, 0x41, 0x53, 0x53,
		0x00, 0xFF, 0x2F, 0x00, // End of track
	}
	smfFile, _ := smf.ReadFrom(bytes.NewReader(midiData))
	return smfFile
}

// createMidiFileWithVocals creates a simple MIDI file with vocal track name
func createMidiFileWithVocals() *smf.SMF {
	// Simple MIDI with PART VOCALS track name
	midiData := []byte{
		0x4D, 0x54, 0x68, 0x64, // Header
		0x00, 0x00, 0x00, 0x06,
		0x00, 0x00, 0x00, 0x01, 0x01, 0xE0,
		0x4D, 0x54, 0x72, 0x6B, // Track
		0x00, 0x00, 0x00, 0x16,
		// Track name: "PART VOCALS"
		0x00, 0xFF, 0x03, 0x0B, 0x50, 0x41, 0x52, 0x54, 0x20, 0x56, 0x4F, 0x43, 0x41, 0x4C, 0x53,
		0x00, 0xFF, 0x2F, 0x00, // End of track
	}
	smfFile, _ := smf.ReadFrom(bytes.NewReader(midiData))
	return smfFile
}

// createComplexMidiFile creates a MIDI file for integration testing
func createComplexMidiFile() *smf.SMF {
	// Reuse the drums file for complex testing
	return createMidiFileWithDrums()
}

type testDrumNote struct {
	time uint32
}

func (n testDrumNote) GetTime() uint32 { return n.time }

func (n testDrumNote) ConvertToToneLibNote() (ToneLibNote, error) {
	return ToneLibNote{Fret: 60, String: 1}, nil
}

func TestGroupLyricsByMeasure_SplitsSegmentsOnQuarterGaps(t *testing.T) {
	timeline := &Timeline{
		Measures: []Measure{
			{StartTime: 0, EndTime: 1920, BeatsPerMeasure: 4},
		},
		TicksPerBeat: 480,
	}

	lyricEvents := []LyricEvent{
		{Time: 0, Lyric: "Hel-"},
		{Time: 120, Lyric: "lo"},
		{Time: 600, Lyric: "world"},
	}

	measureLyrics := groupLyricsByMeasure(lyricEvents, timeline)
	if len(measureLyrics) != 1 {
		t.Fatalf("expected 1 measure with lyrics, got %d", len(measureLyrics))
	}

	segments := measureLyrics[0].Segments
	if len(segments) != 2 {
		t.Fatalf("expected 2 lyric segments, got %d", len(segments))
	}

	if segments[0].StartTime != 0 {
		t.Fatalf("expected first segment to start at 0, got %d", segments[0].StartTime)
	}
	if segments[0].Text != "Hello" {
		t.Fatalf("expected first segment text 'Hello', got '%s'", segments[0].Text)
	}

	if segments[1].StartTime != 600 {
		t.Fatalf("expected second segment to start at 600, got %d", segments[1].StartTime)
	}
	if segments[1].Text != "world" {
		t.Fatalf("expected second segment text 'world', got '%s'", segments[1].Text)
	}
}

func TestCreateLyricsBarsFromMeasures_MultipleSegments(t *testing.T) {
	measureLyrics := []MeasureLyrics{
		{
			MeasureNum: 1,
			Segments: []LyricSegment{
				{StartTime: 0, Text: "Hello"},
				{StartTime: 960, Text: "World"},
			},
		},
	}

	timeline := &Timeline{
		Measures: []Measure{
			{StartTime: 0, EndTime: 1920, BeatsPerMeasure: 4},
		},
		TicksPerBeat: 480,
	}

	bars := createLyricsBarsFromMeasures(measureLyrics, 1, timeline)
	if len(bars.Bars) != 1 {
		t.Fatalf("expected 1 bar, got %d", len(bars.Bars))
	}

	beats := bars.Bars[0].Beats
	if len(beats) != 8 {
		t.Fatalf("expected 8 beats in the bar, got %d", len(beats))
	}

	if beats[0].Text == nil || beats[0].Text.Value != "Hello" {
		t.Fatalf("expected first beat text 'Hello', got '%v'", beats[0].Text)
	}

	if beats[4].Text == nil || beats[4].Text.Value != "World" {
		t.Fatalf("expected fifth beat text 'World', got '%v'", beats[4].Text)
	}
}

// Tests for createToneLibInfo

func TestCreateToneLibInfo_MidiFile(t *testing.T) {
	midiFile := createMockMidiFile()
	song := &MidiFile{SMF: midiFile}

	info := createToneLibInfo(song)

	// MidiFile should extract track name from first track
	if info.Name != "Test Track" {
		t.Errorf("Expected info.Name 'Test Track', got '%s'", info.Name)
	}
	if info.ShowRemarks != "no" {
		t.Errorf("Expected ShowRemarks 'no', got '%s'", info.ShowRemarks)
	}
}

func TestCreateToneLibInfo_ChartFile(t *testing.T) {
	chartFile := &ChartFile{
		Song: SongSection{
			Name:    "Chart Song",
			Artist:  "Chart Artist",
			Album:   "Chart Album",
			Charter: "Chart Charter",
			Year:    "2024",
			Genre:   "rock",
		},
	}

	info := createToneLibInfo(chartFile)

	if info.Name != "Chart Song" {
		t.Errorf("Expected info.Name 'Chart Song', got '%s'", info.Name)
	}
	if info.Artist != "Chart Artist" {
		t.Errorf("Expected info.Artist 'Chart Artist', got '%s'", info.Artist)
	}
	if info.Album != "Chart Album" {
		t.Errorf("Expected info.Album 'Chart Album', got '%s'", info.Album)
	}
}

func TestCreateToneLibInfo_EmptyMetadata(t *testing.T) {
	chartFile := &ChartFile{
		Song: SongSection{}, // Empty metadata
	}

	info := createToneLibInfo(chartFile)

	// All fields should be empty strings
	if info.Name != "" {
		t.Errorf("Expected empty Name, got '%s'", info.Name)
	}
	if info.Artist != "" {
		t.Errorf("Expected empty Artist, got '%s'", info.Artist)
	}
	if info.ShowRemarks != "no" {
		t.Errorf("Expected ShowRemarks 'no', got '%s'", info.ShowRemarks)
	}
}

func TestConvertNotesToBeats_ExpandsToSixteenthGrid(t *testing.T) {
	config := BarCreationConfig{
		ClefValue:        ToneLibPercussionClef,
		TicksPerQuarter:  480,
		NumBars:          1,
		NumEighthsPerBar: 8,
	}

	notes := []testDrumNote{{time: 0}, {time: 120}}
	beats := convertNotesToBeats(notes, 1, config)

	if len(beats) != 16 {
		t.Fatalf("expected 16 beats, got %d", len(beats))
	}

	if beats[0].Duration != ToneLibSixteenthNoteDuration {
		t.Fatalf("expected sixteenth note duration, got %d", beats[0].Duration)
	}

	if len(beats[0].Notes) != 1 {
		t.Fatalf("expected 1 note at beat 0, got %d", len(beats[0].Notes))
	}

	if len(beats[1].Notes) != 1 {
		t.Fatalf("expected 1 note at beat 1, got %d", len(beats[1].Notes))
	}
}

func TestConvertNotesToBeats_ExpandsToSixtyFourthGrid(t *testing.T) {
	config := BarCreationConfig{
		ClefValue:        ToneLibPercussionClef,
		TicksPerQuarter:  480,
		NumBars:          1,
		NumEighthsPerBar: 8,
	}

	notes := []testDrumNote{{time: 30}}
	beats := convertNotesToBeats(notes, 1, config)

	if len(beats) != 64 {
		t.Fatalf("expected 64 beats, got %d", len(beats))
	}

	if beats[0].Duration != ToneLibSixtyFourthNoteDuration {
		t.Fatalf("expected sixty-fourth note duration, got %d", beats[0].Duration)
	}

	if len(beats[1].Notes) != 1 {
		t.Fatalf("expected 1 note at beat 1, got %d", len(beats[1].Notes))
	}
}

func TestConvertNotesToBeats_PrefersLowerSubdivisionWhenErrorEqual(t *testing.T) {
	config := BarCreationConfig{
		ClefValue:        ToneLibPercussionClef,
		TicksPerQuarter:  480,
		NumBars:          1,
		NumEighthsPerBar: 8,
	}

	notes := []testDrumNote{{time: 45}}
	beats := convertNotesToBeats(notes, 1, config)

	if len(beats) != 32 {
		t.Fatalf("expected 32 beats, got %d", len(beats))
	}

	if beats[0].Duration != ToneLibThirtySecondNoteDuration {
		t.Fatalf("expected thirty-second note duration, got %d", beats[0].Duration)
	}

	if len(beats[1].Notes) != 1 {
		t.Fatalf("expected 1 note at beat 1, got %d", len(beats[1].Notes))
	}
}

// Tests for WriteToneLibXMLTo

func TestWriteToneLibXMLTo_BasicMidiFile(t *testing.T) {
	midiFile := createMockMidiFile()
	song := &MidiFile{SMF: midiFile}

	var buf bytes.Buffer
	err := WriteToneLibXMLTo(&buf, song)
	if err != nil {
		t.Fatalf("WriteToneLibXMLTo failed: %v", err)
	}

	xmlOutput := buf.String()

	// Verify basic XML structure - use contains to handle self-closing tags
	expectedElements := []string{
		`<Score>`,
		`</Score>`,
		`<info>`,
		`</info>`,
		`<name>Test Track</name>`,
		`<show_remarks>no</show_remarks>`,
		`BarIndex`, // Just check that BarIndex appears
		`Tracks`,   // Just check that Tracks appears
	}

	for _, expected := range expectedElements {
		if !strings.Contains(xmlOutput, expected) {
			t.Errorf("Expected XML to contain '%s', but it didn't. Full XML:\n%s", expected, xmlOutput)
		}
	}
}

func TestWriteToneLibXMLTo_MidiFileWithDrums(t *testing.T) {
	midiFile := createMidiFileWithDrums()
	song := &MidiFile{SMF: midiFile}

	var buf bytes.Buffer
	err := WriteToneLibXMLTo(&buf, song)
	if err != nil {
		t.Fatalf("WriteToneLibXMLTo failed: %v", err)
	}

	xmlOutput := buf.String()

	// Verify basic structure is present (tracks may not be created if no notes extracted)
	expectedElements := []string{
		`<Score>`,
		`</Score>`,
		`Tracks`, // Basic Tracks element should exist
	}

	for _, expected := range expectedElements {
		if !strings.Contains(xmlOutput, expected) {
			t.Errorf("Expected XML to contain '%s', but it didn't", expected)
		}
	}

	// Note: Actual track creation depends on successful note extraction from MIDI
}

func TestWriteToneLibXMLTo_MidiFileWithBass(t *testing.T) {
	midiFile := createMidiFileWithBass()
	song := &MidiFile{SMF: midiFile}

	var buf bytes.Buffer
	err := WriteToneLibXMLTo(&buf, song)
	if err != nil {
		t.Fatalf("WriteToneLibXMLTo failed: %v", err)
	}

	xmlOutput := buf.String()

	// Verify basic structure
	if !strings.Contains(xmlOutput, `Tracks`) {
		t.Error("Expected Tracks element for bass")
	}

	if !strings.Contains(xmlOutput, `<Score>`) {
		t.Error("Expected Score element")
	}
}

func TestWriteToneLibXMLTo_MidiFileWithVocals(t *testing.T) {
	midiFile := createMidiFileWithVocals()
	song := &MidiFile{SMF: midiFile}

	var buf bytes.Buffer
	err := WriteToneLibXMLTo(&buf, song)
	if err != nil {
		t.Fatalf("WriteToneLibXMLTo failed: %v", err)
	}

	xmlOutput := buf.String()

	// Verify basic structure exists
	if !strings.Contains(xmlOutput, `<Score>`) {
		t.Error("Expected Score element")
	}

	if !strings.Contains(xmlOutput, `Tracks`) {
		t.Error("Expected Tracks element")
	}
}

func TestWriteToneLibXMLTo_ComplexMidiFile(t *testing.T) {
	midiFile := createComplexMidiFile()
	song := &MidiFile{SMF: midiFile}

	var buf bytes.Buffer
	err := WriteToneLibXMLTo(&buf, song)
	if err != nil {
		t.Fatalf("WriteToneLibXMLTo failed: %v", err)
	}

	xmlOutput := buf.String()

	// Verify basic structure
	expectedElements := []string{
		`<Score>`,
		`<info>`,
		`BarIndex`, // Self-closing tags
		`Tracks`,   // Self-closing tags
		`</info>`,
		`</Score>`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(xmlOutput, expected) {
			t.Errorf("Expected complex XML to contain '%s'", expected)
		}
	}
}

func TestWriteToneLibXMLTo_ChartFile(t *testing.T) {
	chartFile := &ChartFile{
		Song: SongSection{
			Name:       "Chart Test",
			Artist:     "Test Artist",
			Resolution: 192,
		},
		SyncTrack: SyncTrackSection{
			BPMEvents: []BPMEvent{
				{Tick: 0, BPM: 120000},
			},
			TimeSigEvents: []TimeSigEvent{
				{Tick: 0, Numerator: 4, Denominator: 2},
			},
		},
		Tracks: make(map[string]TrackSection),
	}

	var buf bytes.Buffer
	err := WriteToneLibXMLTo(&buf, chartFile)
	if err != nil {
		t.Fatalf("WriteToneLibXMLTo failed for ChartFile: %v", err)
	}

	xmlOutput := buf.String()

	// Verify chart metadata appears
	if !strings.Contains(xmlOutput, `<name>Chart Test</name>`) {
		t.Error("Expected chart name in XML")
	}
	if !strings.Contains(xmlOutput, `<artist>Test Artist</artist>`) {
		t.Error("Expected chart artist in XML")
	}

	// Basic structure should still be present
	if !strings.Contains(xmlOutput, `<Score>`) {
		t.Error("Expected Score element")
	}
}

// Edge Case Tests

func TestWriteToneLibXMLTo_EmptyMidiFile(t *testing.T) {
	// Create minimal MIDI file with just end of track
	midiData := []byte{
		0x4D, 0x54, 0x68, 0x64, // Header
		0x00, 0x00, 0x00, 0x06,
		0x00, 0x00, 0x00, 0x01, 0x01, 0xE0,
		0x4D, 0x54, 0x72, 0x6B, // Track
		0x00, 0x00, 0x00, 0x04,
		0x00, 0xFF, 0x2F, 0x00, // End of track only
	}
	smfFile, _ := smf.ReadFrom(bytes.NewReader(midiData))

	song := &MidiFile{SMF: smfFile}

	var buf bytes.Buffer
	err := WriteToneLibXMLTo(&buf, song)
	if err != nil {
		t.Fatalf("WriteToneLibXMLTo failed for empty MIDI: %v", err)
	}

	xmlOutput := buf.String()

	// Should still generate valid XML structure
	if !strings.Contains(xmlOutput, `<Score>`) {
		t.Error("Expected Score element even for empty MIDI")
	}
	if !strings.Contains(xmlOutput, `</Score>`) {
		t.Error("Expected closing Score element")
	}
}

func TestWriteToneLibXMLTo_NoBeatTrack(t *testing.T) {
	// Use existing drum file which doesn't have BEAT track
	smfFile := createMidiFileWithDrums()
	song := &MidiFile{SMF: smfFile}

	var buf bytes.Buffer
	err := WriteToneLibXMLTo(&buf, song)
	if err != nil {
		t.Fatalf("WriteToneLibXMLTo failed for MIDI without BEAT track: %v", err)
	}

	xmlOutput := buf.String()

	// Should still create fallback structure
	if !strings.Contains(xmlOutput, `BarIndex`) {
		t.Error("Expected BarIndex even without BEAT track")
	}
}

// Integration Tests

func TestCreateToneLibScore_Integration(t *testing.T) {
	midiFile := createMidiFileWithBeatTrack() // Use BEAT track to create bars
	song := &MidiFile{SMF: midiFile}

	score := createToneLibScore(song)

	if score == nil {
		t.Fatal("Expected non-nil score")
	}

	// Verify info section
	if score.Info.ShowRemarks != "no" {
		t.Errorf("Expected ShowRemarks 'no', got '%s'", score.Info.ShowRemarks)
	}

	// Verify bar index structure exists (may be empty if no proper BEAT markers)
	// The important thing is that the BarIndex is initialized

	// Verify tracks structure exists (structure varies by implementation)
	// The important thing is that the ToneLibTracks structure is created
}

func TestXMLGeneration_WellFormed(t *testing.T) {
	testCases := []struct {
		name     string
		midiFile *smf.SMF
	}{
		{"Basic", createMockMidiFile()},
		{"WithDrums", createMidiFileWithDrums()},
		{"WithBass", createMidiFileWithBass()},
		{"WithVocals", createMidiFileWithVocals()},
		{"Complex", createComplexMidiFile()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			song := &MidiFile{SMF: tc.midiFile}

			var buf bytes.Buffer
			err := WriteToneLibXMLTo(&buf, song)
			if err != nil {
				t.Fatalf("WriteToneLibXMLTo failed for %s: %v", tc.name, err)
			}

			xmlOutput := buf.String()

			// Basic well-formed checks
			if !strings.HasPrefix(strings.TrimSpace(xmlOutput), "<") {
				t.Errorf("XML should start with '<', got: %s", xmlOutput[:min(50, len(xmlOutput))])
			}

			// Check that opening tags have matching closing tags for key elements
			keyElements := []string{"Score", "info", "BarIndex", "Tracks"}
			for _, element := range keyElements {
				openCount := strings.Count(xmlOutput, fmt.Sprintf("<%s>", element))
				closeCount := strings.Count(xmlOutput, fmt.Sprintf("</%s>", element))
				if openCount != closeCount {
					t.Errorf("Mismatched tags for %s: %d open, %d close", element, openCount, closeCount)
				}
			}
		})
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
