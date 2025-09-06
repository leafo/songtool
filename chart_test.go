package main

import (
	"fmt"
	"strings"
	"testing"
)

// Test constants with sample chart data
const validChartData = `[Song]
{
  Name = "Test Song"
  Artist = "Test Artist" 
  Charter = "Test Charter"
  Album = "Test Album"
  Year = ", 2024"
  Offset = 0
  Resolution = 192
  Player2 = bass
  Difficulty = 5
  PreviewStart = 30000
  PreviewEnd = 60000
  Genre = "rock"
  MediaType = "cd"
  MusicStream = "song.ogg"
  GuitarStream = "guitar.ogg"
  BassStream = "bass.ogg"
  DrumStream = "drums.ogg"
}
[SyncTrack]
{
  0 = TS 4
  0 = B 120000
  768 = TS 3 3
  768 = B 140000
  1536 = TS 4 2
  1536 = B 120000
  2304 = A 2000000
}
[Events]
{
  0 = E "song_start"
  384 = E "section Verse 1"
  768 = E "section Chorus"
  1152 = E "lyric Hello world"
  1536 = E "section Bridge"
  1920 = E "end"
}
[ExpertSingle]
{
  192 = N 0 0
  384 = N 1 0
  576 = N 2 192
  768 = N 3 0
  960 = N 4 0
  1152 = N 7 0
  1344 = N 5 0
  1536 = N 6 0
  1728 = S 2 192
  1920 = E solostart
}
[HardDrums]
{
  192 = N 0 0
  384 = N 1 0
  576 = N 2 0
  768 = N 3 0
  960 = N 4 0
  1152 = N 32 0
  1344 = N 34 0
  1536 = N 66 0
  1728 = S 2 192
}
[MediumGHLGuitar]
{
  192 = N 0 0
  384 = N 1 0
  576 = N 2 0
  768 = N 3 0
  960 = N 4 0
  1152 = N 8 0
  1344 = N 5 0
  1536 = N 6 0
}`

const minimalChartData = `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  0 = B 120000
}
[ExpertSingle]
{
  192 = N 0 0
}`

const malformedChartData = `[Song]
{
  Name = "Test Song"
  Resolution = 192
  InvalidLine
  = AnotherBadLine
}
[SyncTrack]
{
  0 = B 120000
  invalid_tick = B 140000
  768 = B invalid_bpm
}
[Events]
{
  0 = E "unclosed quote
  192 = INVALID_EVENT_TYPE "test"
}
[ExpertSingle]
{
  192 = N 0 0
  bad_tick = N 1 0
  384 = N invalid_fret 0
}`

const chartWithBOM = "\ufeff[Song]\n{\n  Resolution = 192\n}\n[SyncTrack]\n{\n  0 = B 120000\n}"

const chartWithEscapes = `[Song]
{
  Name = "Song with \"quotes\" and\nnewlines"
  Artist = "Artist\twith\ttabs"
  Resolution = 192
}
[SyncTrack]
{
  0 = B 120000
}
[Events]
{
  0 = E "Text with \\backslash and \"quotes\""
  192 = E "Line with\nnewline"
}`

const emptyChart = ""

const noTracksChart = `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  0 = B 120000
}`

// Basic Parsing Tests
func TestParseValidChart(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse valid chart: %v", err)
	}

	// Test song metadata
	if chart.Song.Name != "Test Song" {
		t.Errorf("Expected Name 'Test Song', got '%s'", chart.Song.Name)
	}
	if chart.Song.Artist != "Test Artist" {
		t.Errorf("Expected Artist 'Test Artist', got '%s'", chart.Song.Artist)
	}
	if chart.Song.Charter != "Test Charter" {
		t.Errorf("Expected Charter 'Test Charter', got '%s'", chart.Song.Charter)
	}
	if chart.Song.Resolution != 192 {
		t.Errorf("Expected Resolution 192, got %d", chart.Song.Resolution)
	}
	if chart.Song.Difficulty != 5 {
		t.Errorf("Expected Difficulty 5, got %d", chart.Song.Difficulty)
	}
	if chart.Song.PreviewStart != 30000 {
		t.Errorf("Expected PreviewStart 30000, got %d", chart.Song.PreviewStart)
	}
	if chart.Song.MusicStream != "song.ogg" {
		t.Errorf("Expected MusicStream 'song.ogg', got '%s'", chart.Song.MusicStream)
	}
}

func TestParseMinimalChart(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(minimalChartData))
	if err != nil {
		t.Fatalf("Failed to parse minimal chart: %v", err)
	}

	if chart.Song.Resolution != 192 {
		t.Errorf("Expected Resolution 192, got %d", chart.Song.Resolution)
	}

	// Should have one BPM event
	if len(chart.SyncTrack.BPMEvents) != 1 {
		t.Errorf("Expected 1 BPM event, got %d", len(chart.SyncTrack.BPMEvents))
	}

	// Should have one track
	if len(chart.Tracks) != 1 {
		t.Errorf("Expected 1 track, got %d", len(chart.Tracks))
	}
}

func TestParseBOMChart(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(chartWithBOM))
	if err != nil {
		t.Fatalf("Failed to parse chart with BOM: %v", err)
	}

	if chart.Song.Resolution != 192 {
		t.Errorf("Expected Resolution 192, got %d", chart.Song.Resolution)
	}
}

func TestParseEmptyChart(t *testing.T) {
	_, err := ParseChartFile(strings.NewReader(emptyChart))
	if err == nil {
		t.Error("Expected error for empty chart, but parsing succeeded")
	}
}

func TestParseEscapeSequences(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(chartWithEscapes))
	if err != nil {
		t.Fatalf("Failed to parse chart with escapes: %v", err)
	}

	expectedName := "Song with \"quotes\" and\nnewlines"
	if chart.Song.Name != expectedName {
		t.Errorf("Expected Name '%s', got '%s'", expectedName, chart.Song.Name)
	}

	expectedArtist := "Artist\twith\ttabs"
	if chart.Song.Artist != expectedArtist {
		t.Errorf("Expected Artist '%s', got '%s'", expectedArtist, chart.Song.Artist)
	}
}

// SyncTrack Section Tests
func TestSyncTrackBPMEvents(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	if len(chart.SyncTrack.BPMEvents) != 3 {
		t.Fatalf("Expected 3 BPM events, got %d", len(chart.SyncTrack.BPMEvents))
	}

	// Test first BPM event
	bpm1 := chart.SyncTrack.BPMEvents[0]
	if bpm1.Tick != 0 {
		t.Errorf("Expected first BPM tick 0, got %d", bpm1.Tick)
	}
	if bpm1.BPM != 120000 {
		t.Errorf("Expected first BPM 120000, got %d", bpm1.BPM)
	}

	// Test second BPM event
	bpm2 := chart.SyncTrack.BPMEvents[1]
	if bpm2.Tick != 768 {
		t.Errorf("Expected second BPM tick 768, got %d", bpm2.Tick)
	}
	if bpm2.BPM != 140000 {
		t.Errorf("Expected second BPM 140000, got %d", bpm2.BPM)
	}
}

func TestSyncTrackTimeSignatures(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	if len(chart.SyncTrack.TimeSigEvents) != 3 {
		t.Fatalf("Expected 3 time signature events, got %d", len(chart.SyncTrack.TimeSigEvents))
	}

	// Test first time signature (4/4)
	ts1 := chart.SyncTrack.TimeSigEvents[0]
	if ts1.Tick != 0 {
		t.Errorf("Expected first TS tick 0, got %d", ts1.Tick)
	}
	if ts1.Numerator != 4 {
		t.Errorf("Expected first TS numerator 4, got %d", ts1.Numerator)
	}
	if ts1.Denominator != 2 { // Default is 4/4, stored as 2 (log2 of 4)
		t.Errorf("Expected first TS denominator 2, got %d", ts1.Denominator)
	}

	// Test second time signature (3/8, stored as 3 3)
	ts2 := chart.SyncTrack.TimeSigEvents[1]
	if ts2.Numerator != 3 {
		t.Errorf("Expected second TS numerator 3, got %d", ts2.Numerator)
	}
	if ts2.Denominator != 3 { // log2(8) = 3
		t.Errorf("Expected second TS denominator 3, got %d", ts2.Denominator)
	}
}

func TestSyncTrackAnchors(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	if len(chart.SyncTrack.AnchorEvents) != 1 {
		t.Fatalf("Expected 1 anchor event, got %d", len(chart.SyncTrack.AnchorEvents))
	}

	anchor := chart.SyncTrack.AnchorEvents[0]
	if anchor.Tick != 2304 {
		t.Errorf("Expected anchor tick 2304, got %d", anchor.Tick)
	}
	if anchor.Microseconds != 2000000 {
		t.Errorf("Expected anchor microseconds 2000000, got %d", anchor.Microseconds)
	}
}

// Events Section Tests
func TestEventsGlobalEvents(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	expectedEvents := []struct {
		tick uint32
		text string
	}{
		{0, "song_start"},
		{384, "section Verse 1"},
		{768, "section Chorus"},
		{1152, "lyric Hello world"},
		{1536, "section Bridge"},
		{1920, "end"},
	}

	if len(chart.Events.GlobalEvents) != len(expectedEvents) {
		t.Fatalf("Expected %d global events, got %d", len(expectedEvents), len(chart.Events.GlobalEvents))
	}

	for i, expected := range expectedEvents {
		event := chart.Events.GlobalEvents[i]
		if event.Tick != expected.tick {
			t.Errorf("Event %d: expected tick %d, got %d", i, expected.tick, event.Tick)
		}
		if event.Text != expected.text {
			t.Errorf("Event %d: expected text '%s', got '%s'", i, expected.text, event.Text)
		}
	}
}

func TestEventsWithEscapeSequences(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(chartWithEscapes))
	if err != nil {
		t.Fatalf("Failed to parse chart with escapes: %v", err)
	}

	if len(chart.Events.GlobalEvents) != 2 {
		t.Fatalf("Expected 2 global events, got %d", len(chart.Events.GlobalEvents))
	}

	// Test first event with quotes and backslashes
	event1 := chart.Events.GlobalEvents[0]
	expectedText1 := "Text with \\backslash and \"quotes\""
	if event1.Text != expectedText1 {
		t.Errorf("Expected event text '%s', got '%s'", expectedText1, event1.Text)
	}

	// Test second event with newline
	event2 := chart.Events.GlobalEvents[1]
	expectedText2 := "Line with\nnewline"
	if event2.Text != expectedText2 {
		t.Errorf("Expected event text '%s', got '%s'", expectedText2, event2.Text)
	}
}

// Track Section Tests
func TestTrackSectionParsing(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	// Should have 3 tracks: ExpertSingle, HardDrums, MediumGHLGuitar
	if len(chart.Tracks) != 3 {
		t.Fatalf("Expected 3 tracks, got %d", len(chart.Tracks))
	}

	// Test ExpertSingle track
	expertTrack, exists := chart.Tracks["ExpertSingle"]
	if !exists {
		t.Fatal("ExpertSingle track not found")
	}
	if expertTrack.Name != "ExpertSingle" {
		t.Errorf("Expected track name 'ExpertSingle', got '%s'", expertTrack.Name)
	}
}

func TestExpertSingleNotes(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	expertTrack := chart.Tracks["ExpertSingle"]

	expectedNotes := []struct {
		tick    uint32
		fret    uint8
		sustain uint32
		flags   NoteFlags
	}{
		{192, 0, 0, FlagNone},   // Green
		{384, 1, 0, FlagNone},   // Red
		{576, 2, 192, FlagNone}, // Yellow with sustain
		{768, 3, 0, FlagNone},   // Blue
		{960, 4, 0, FlagNone},   // Orange
		{1152, 7, 0, FlagOpen},  // Open note
		// Note: 1344 = N 5 0 (forced flag) and 1536 = N 6 0 (tap flag) are skipped by our implementation
	}

	if len(expertTrack.Notes) != len(expectedNotes) {
		t.Fatalf("Expected %d notes, got %d", len(expectedNotes), len(expertTrack.Notes))
	}

	for i, expected := range expectedNotes {
		note := expertTrack.Notes[i]
		if note.Tick != expected.tick {
			t.Errorf("Note %d: expected tick %d, got %d", i, expected.tick, note.Tick)
		}
		if note.Fret != expected.fret {
			t.Errorf("Note %d: expected fret %d, got %d", i, expected.fret, note.Fret)
		}
		if note.Sustain != expected.sustain {
			t.Errorf("Note %d: expected sustain %d, got %d", i, expected.sustain, note.Sustain)
		}
		if note.Flags != expected.flags {
			t.Errorf("Note %d: expected flags %v, got %v", i, expected.flags, note.Flags)
		}
	}
}

func TestDrumNotes(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	drumTrack := chart.Tracks["HardDrums"]

	expectedNotes := []struct {
		tick  uint32
		fret  uint8
		flags NoteFlags
	}{
		{192, 0, FlagNone},        // Kick
		{384, 1, FlagNone},        // Red pad
		{576, 2, FlagNone},        // Yellow pad
		{768, 3, FlagNone},        // Blue pad
		{960, 4, FlagNone},        // Orange pad
		{1152, 0, FlagDoubleKick}, // Double kick (note 32 -> fret 0 with flag)
		// Note: 1344 = N 34 0 (accent flag) and 1536 = N 66 0 (cymbal flag) are skipped by our implementation
	}

	if len(drumTrack.Notes) != len(expectedNotes) {
		t.Fatalf("Expected %d drum notes, got %d", len(expectedNotes), len(drumTrack.Notes))
	}

	for i, expected := range expectedNotes {
		note := drumTrack.Notes[i]
		if note.Fret != expected.fret {
			t.Errorf("Drum note %d: expected fret %d, got %d", i, expected.fret, note.Fret)
		}
		if note.Flags != expected.flags {
			t.Errorf("Drum note %d: expected flags %v, got %v", i, expected.flags, note.Flags)
		}
	}
}

func TestSpecialEvents(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	expertTrack := chart.Tracks["ExpertSingle"]
	drumTrack := chart.Tracks["HardDrums"]

	// ExpertSingle should have 1 starpower event
	if len(expertTrack.Specials) != 1 {
		t.Fatalf("Expected 1 special event in ExpertSingle, got %d", len(expertTrack.Specials))
	}

	special := expertTrack.Specials[0]
	if special.Tick != 1728 {
		t.Errorf("Expected special tick 1728, got %d", special.Tick)
	}
	if special.Type != 2 { // Starpower type
		t.Errorf("Expected special type 2, got %d", special.Type)
	}
	if special.Length != 192 {
		t.Errorf("Expected special length 192, got %d", special.Length)
	}

	// HardDrums should also have 1 starpower event
	if len(drumTrack.Specials) != 1 {
		t.Fatalf("Expected 1 special event in HardDrums, got %d", len(drumTrack.Specials))
	}
}

func TestTrackEvents(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	expertTrack := chart.Tracks["ExpertSingle"]

	// ExpertSingle should have 1 track event
	if len(expertTrack.TrackEvents) != 1 {
		t.Fatalf("Expected 1 track event in ExpertSingle, got %d", len(expertTrack.TrackEvents))
	}

	trackEvent := expertTrack.TrackEvents[0]
	if trackEvent.Tick != 1920 {
		t.Errorf("Expected track event tick 1920, got %d", trackEvent.Tick)
	}
	if trackEvent.Text != "solostart" {
		t.Errorf("Expected track event text 'solostart', got '%s'", trackEvent.Text)
	}
}

// Note Flag Processing Tests
func TestNoteFlagDetection(t *testing.T) {
	// Test Open note flag
	openNoteChart := `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  0 = B 120000
}
[ExpertSingle]
{
  192 = N 7 0
}`

	chart, err := ParseChartFile(strings.NewReader(openNoteChart))
	if err != nil {
		t.Fatalf("Failed to parse open note chart: %v", err)
	}

	expertTrack := chart.Tracks["ExpertSingle"]
	if len(expertTrack.Notes) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(expertTrack.Notes))
	}

	note := expertTrack.Notes[0]
	if note.Fret != 7 {
		t.Errorf("Expected open note fret 7, got %d", note.Fret)
	}
	if note.Flags&FlagOpen == 0 {
		t.Errorf("Expected open note to have FlagOpen, got flags %v", note.Flags)
	}
}

func TestDoubleKickFlag(t *testing.T) {
	doubleKickChart := `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  0 = B 120000
}
[ExpertDrums]
{
  192 = N 32 0
}`

	chart, err := ParseChartFile(strings.NewReader(doubleKickChart))
	if err != nil {
		t.Fatalf("Failed to parse double kick chart: %v", err)
	}

	drumTrack := chart.Tracks["ExpertDrums"]
	if len(drumTrack.Notes) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(drumTrack.Notes))
	}

	note := drumTrack.Notes[0]
	if note.Fret != 0 { // Should convert to kick fret
		t.Errorf("Expected double kick fret 0, got %d", note.Fret)
	}
	if note.Flags&FlagDoubleKick == 0 {
		t.Errorf("Expected double kick to have FlagDoubleKick, got flags %v", note.Flags)
	}
}

func TestGHLiveNoteMapping(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	ghlTrack := chart.Tracks["MediumGHLGuitar"]

	expectedGHLNotes := []struct {
		tick uint32
		fret uint8
	}{
		{192, 0},  // White 1 (N 0 0)
		{384, 1},  // White 2 (N 1 0)
		{576, 2},  // White 3 (N 2 0)
		{768, 3},  // Black 1 (N 3 0)
		{960, 4},  // Black 2 (N 4 0)
		{1152, 8}, // Black 3 (N 8 0) - raw fret value
		// Note: 1344 = N 5 0 (forced flag) and 1536 = N 6 0 (tap flag) are skipped by our implementation
	}

	if len(ghlTrack.Notes) != len(expectedGHLNotes) {
		t.Fatalf("Expected %d GHL notes, got %d", len(expectedGHLNotes), len(ghlTrack.Notes))
	}

	for i, expected := range expectedGHLNotes {
		note := ghlTrack.Notes[i]
		if note.Fret != expected.fret {
			t.Errorf("GHL note %d: expected fret %d, got %d", i, expected.fret, note.Fret)
		}
	}
}

// Error Handling and Edge Case Tests
func TestMalformedChartGracefulHandling(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(malformedChartData))
	if err != nil {
		t.Fatalf("Parser should handle malformed chart gracefully: %v", err)
	}

	// Should still parse the valid parts
	if chart.Song.Name != "Test Song" {
		t.Errorf("Expected Name 'Test Song', got '%s'", chart.Song.Name)
	}
	if chart.Song.Resolution != 192 {
		t.Errorf("Expected Resolution 192, got %d", chart.Song.Resolution)
	}

	// Should have one valid BPM event (the invalid ones are skipped)
	if len(chart.SyncTrack.BPMEvents) != 1 {
		t.Errorf("Expected 1 BPM event, got %d", len(chart.SyncTrack.BPMEvents))
	}

	// Should still create the ExpertSingle track with valid notes
	expertTrack, exists := chart.Tracks["ExpertSingle"]
	if !exists {
		t.Fatal("ExpertSingle track should exist")
	}
	if len(expertTrack.Notes) != 1 { // Only the first valid note
		t.Errorf("Expected 1 note, got %d", len(expertTrack.Notes))
	}
}

func TestMalformedSectionHeaders(t *testing.T) {
	malformedSections := `[Song]
{
  Resolution = 192
}
[]
{
  0 = B 120000
}
[SyncTrack]
{
  0 = B 120000
}`

	_, err := ParseChartFile(strings.NewReader(malformedSections))
	if err == nil {
		t.Error("Expected error for empty section name")
	}
}

func TestMissingSectionBraces(t *testing.T) {
	missingBraces := `[Song]
  Resolution = 192
[SyncTrack]
{
  0 = B 120000
}`

	_, err := ParseChartFile(strings.NewReader(missingBraces))
	if err == nil {
		t.Error("Expected error for missing opening brace")
	}
}

func TestInvalidTickValues(t *testing.T) {
	invalidTicks := `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  0 = B 120000
  -100 = B 140000
  abc = B 160000
}`

	chart, err := ParseChartFile(strings.NewReader(invalidTicks))
	if err != nil {
		t.Fatalf("Parser should handle invalid ticks gracefully: %v", err)
	}

	// Should only have the valid BPM event
	if len(chart.SyncTrack.BPMEvents) != 1 {
		t.Errorf("Expected 1 BPM event, got %d", len(chart.SyncTrack.BPMEvents))
	}
}

func TestEmptyEventsAndTracks(t *testing.T) {
	emptyEvents := `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  0 = B 120000
}
[Events]
{
}
[ExpertSingle]
{
}`

	chart, err := ParseChartFile(strings.NewReader(emptyEvents))
	if err != nil {
		t.Fatalf("Failed to parse chart with empty sections: %v", err)
	}

	if len(chart.Events.GlobalEvents) != 0 {
		t.Errorf("Expected 0 global events, got %d", len(chart.Events.GlobalEvents))
	}

	// Empty track sections don't create tracks in our implementation
	if len(chart.Tracks) != 0 {
		t.Errorf("Expected 0 tracks for empty sections, got %d", len(chart.Tracks))
	}
}

func TestUnclosedQuotes(t *testing.T) {
	unclosedQuote := `[Song]
{
  Name = "Unclosed quote
  Resolution = 192
}
[SyncTrack]
{
  0 = B 120000
}`

	chart, err := ParseChartFile(strings.NewReader(unclosedQuote))
	if err != nil {
		t.Fatalf("Parser should handle unclosed quotes gracefully: %v", err)
	}

	// The quote handling should not fail parsing
	if chart.Song.Resolution != 192 {
		t.Errorf("Expected Resolution 192, got %d", chart.Song.Resolution)
	}
}

// Validation Tests
func TestValidationMissingResolution(t *testing.T) {
	noResolution := `[Song]
{
  Name = "Test"
}
[SyncTrack]
{
  0 = B 120000
}`

	_, err := ParseChartFile(strings.NewReader(noResolution))
	if err == nil {
		t.Error("Expected validation error for missing resolution")
	}
}

func TestValidationExtremelyHighBPM(t *testing.T) {
	highBPM := `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  0 = B 2000000
}`

	_, err := ParseChartFile(strings.NewReader(highBPM))
	if err == nil {
		t.Error("Expected validation error for extremely high BPM")
	}
}

func TestValidationZeroBPM(t *testing.T) {
	zeroBPM := `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  0 = B 0
}`

	_, err := ParseChartFile(strings.NewReader(zeroBPM))
	if err == nil {
		t.Error("Expected validation error for zero BPM")
	}
}

func TestValidationInvalidFretRange(t *testing.T) {
	invalidFret := `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  0 = B 120000
}
[ExpertSingle]
{
  192 = N 99 0
}`

	_, err := ParseChartFile(strings.NewReader(invalidFret))
	if err == nil {
		t.Error("Expected validation error for invalid fret number")
	}
}

func TestValidationNoTracks(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(noTracksChart))
	if err != nil {
		t.Fatalf("Parser should handle chart with no tracks gracefully: %v", err)
	}

	// Should not error but should have zero tracks
	if len(chart.Tracks) != 0 {
		t.Errorf("Expected 0 tracks, got %d", len(chart.Tracks))
	}
}

func TestValidationDefaultBPMAdded(t *testing.T) {
	noBPM := `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
}
[ExpertSingle]
{
  192 = N 0 0
}`

	chart, err := ParseChartFile(strings.NewReader(noBPM))
	if err != nil {
		t.Fatalf("Parser should handle missing BPM by adding default: %v", err)
	}

	// Should auto-add default BPM
	if len(chart.SyncTrack.BPMEvents) != 1 {
		t.Fatalf("Expected 1 default BPM event, got %d", len(chart.SyncTrack.BPMEvents))
	}

	defaultBPM := chart.SyncTrack.BPMEvents[0]
	if defaultBPM.BPM != 120000 { // 120 BPM * 1000
		t.Errorf("Expected default BPM 120000, got %d", defaultBPM.BPM)
	}
	if defaultBPM.Tick != 0 {
		t.Errorf("Expected default BPM tick 0, got %d", defaultBPM.Tick)
	}
}

func TestValidationBPMTooLateInSong(t *testing.T) {
	lateBPM := `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  400 = B 120000
}
[ExpertSingle]
{
  192 = N 0 0
}`

	_, err := ParseChartFile(strings.NewReader(lateBPM))
	if err == nil {
		t.Error("Expected validation error for BPM event too late in song")
	}
}

// Utility Function Tests
func TestGetBPMAtTick(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	testCases := []struct {
		tick        uint32
		expectedBPM float64
	}{
		{0, 120.0},    // First BPM event
		{192, 120.0},  // Still in first BPM
		{768, 140.0},  // Second BPM event
		{1000, 140.0}, // Still in second BPM
		{1536, 120.0}, // Third BPM event
		{2000, 120.0}, // Still in third BPM
	}

	for _, tc := range testCases {
		actualBPM := chart.GetBPMAtTick(tc.tick)
		if actualBPM != tc.expectedBPM {
			t.Errorf("GetBPMAtTick(%d): expected %f, got %f", tc.tick, tc.expectedBPM, actualBPM)
		}
	}
}

func TestGetBPMAtTickNoBPMEvents(t *testing.T) {
	chart := &ChartFile{
		Song: SongSection{Resolution: 192},
		SyncTrack: SyncTrackSection{
			BPMEvents: []BPMEvent{}, // No BPM events
		},
	}

	bpm := chart.GetBPMAtTick(192)
	if bpm != 120.0 { // Default BPM
		t.Errorf("Expected default BPM 120.0, got %f", bpm)
	}
}

func TestIsTrackSection(t *testing.T) {
	testCases := []struct {
		section  string
		expected bool
	}{
		// Valid track sections
		{"ExpertSingle", true},
		{"HardDrums", true},
		{"MediumGHLGuitar", true},
		{"EasyKeyboard", true},
		{"ExpertDoubleBass", true},

		// Invalid track sections
		{"Song", false},
		{"SyncTrack", false},
		{"Events", false},
		{"InvalidTrack", false},
		{"ExpertInvalidInstrument", false},
		{"InvalidDifficultyGuitar", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := isTrackSection(tc.section)
		if result != tc.expected {
			t.Errorf("isTrackSection('%s'): expected %t, got %t", tc.section, tc.expected, result)
		}
	}
}

func TestUnquoteString(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		// Basic quote removal
		{`"hello"`, "hello"},
		{`'hello'`, "hello"},
		{"hello", "hello"}, // No quotes

		// Escape sequences
		{`"hello\nworld"`, "hello\nworld"},
		{`"tab\there"`, "tab\there"},
		{`"quote\"test"`, "quote\"test"},
		{`"backslash\\"`, "backslash\\"},
		{`"single\'"`, "single'"},

		// Mixed quotes (only outer quotes removed)
		{`"he said 'hello'"`, "he said 'hello'"},
		{`'she said "hello"'`, `she said "hello"`},

		// Edge cases
		{`""`, ""},                 // Empty quotes
		{`"`, `"`},                 // Single quote
		{`"unclosed`, `"unclosed`}, // Unclosed quote (quotes not removed)

		// Unknown escape sequences (preserved)
		{`"unknown\z"`, "unknown\\z"},
		{`"valid\tinvalid\q"`, "valid\tinvalid\\q"},
	}

	for _, tc := range testCases {
		result := unquoteString(tc.input)
		if result != tc.expected {
			t.Errorf("unquoteString('%s'): expected '%s', got '%s'", tc.input, tc.expected, result)
		}
	}
}

func TestGetMaxFretForTrack(t *testing.T) {
	testCases := []struct {
		trackName   string
		expectedMax int
	}{
		// Drum tracks
		{"ExpertDrums", 5},
		{"HardDrums", 5},

		// GHL tracks
		{"ExpertGHLGuitar", 8},
		{"MediumGHLBass", 8},

		// Standard guitar tracks
		{"ExpertSingle", 7},
		{"HardDoubleBass", 7},
		{"EasyKeyboard", 7},

		// Unknown track (default)
		{"UnknownTrack", 7},
	}

	for _, tc := range testCases {
		result := getMaxFretForTrack(tc.trackName)
		if result != tc.expectedMax {
			t.Errorf("getMaxFretForTrack('%s'): expected %d, got %d", tc.trackName, tc.expectedMax, result)
		}
	}
}

// Integration and Performance Tests
func TestLargeChartParsing(t *testing.T) {
	// Test with a chart that has many events
	largeChart := `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  0 = B 120000
}`

	// Add 1000 notes to test performance
	largeChart += "\n[ExpertSingle]\n{\n"
	for i := 0; i < 1000; i++ {
		tick := i * 192
		fret := i % 5
		largeChart += fmt.Sprintf("  %d = N %d 0\n", tick, fret)
	}
	largeChart += "}"

	chart, err := ParseChartFile(strings.NewReader(largeChart))
	if err != nil {
		t.Fatalf("Failed to parse large chart: %v", err)
	}

	expertTrack := chart.Tracks["ExpertSingle"]
	if len(expertTrack.Notes) != 1000 {
		t.Errorf("Expected 1000 notes, got %d", len(expertTrack.Notes))
	}

	// Verify first and last notes
	if expertTrack.Notes[0].Tick != 0 {
		t.Errorf("Expected first note tick 0, got %d", expertTrack.Notes[0].Tick)
	}
	if expertTrack.Notes[999].Tick != 999*192 {
		t.Errorf("Expected last note tick %d, got %d", 999*192, expertTrack.Notes[999].Tick)
	}
}

func TestChartFileString(t *testing.T) {
	chart, err := ParseChartFile(strings.NewReader(validChartData))
	if err != nil {
		t.Fatalf("Failed to parse chart: %v", err)
	}

	chart.Filename = "test.chart"
	str := chart.String()

	// Should contain basic info
	if !strings.Contains(str, "test.chart") {
		t.Error("Chart string should contain filename")
	}
	if !strings.Contains(str, "Test Song") {
		t.Error("Chart string should contain song name")
	}
	if !strings.Contains(str, "Test Artist") {
		t.Error("Chart string should contain artist")
	}
	if !strings.Contains(str, "Resolution: 192") {
		t.Error("Chart string should contain resolution")
	}
}

// Benchmark Tests
func BenchmarkParseValidChart(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseChartFile(strings.NewReader(validChartData))
		if err != nil {
			b.Fatalf("Failed to parse chart: %v", err)
		}
	}
}

func BenchmarkParseLargeChart(b *testing.B) {
	// Create a large chart with 1000 notes
	largeChart := `[Song]
{
  Resolution = 192
}
[SyncTrack]
{
  0 = B 120000
}
[ExpertSingle]
{`
	for i := 0; i < 1000; i++ {
		tick := i * 192
		fret := i % 5
		largeChart += fmt.Sprintf("  %d = N %d 0\n", tick, fret)
	}
	largeChart += "}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseChartFile(strings.NewReader(largeChart))
		if err != nil {
			b.Fatalf("Failed to parse large chart: %v", err)
		}
	}
}
