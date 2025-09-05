package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gomidi/midi/v2/smf"
)

// BeatMap holds detected beat information for backing track
type BeatMap struct {
	Beats    []ToneLibBackingBeat
	TotalNum string // Total number of beats detected
	NST      string // Normalized sample time or similar metric
}

// ToneLib Score XML structure - represents the complete the_song.dat file
type ToneLibScore struct {
	XMLName      xml.Name             `xml:"Score"`
	Info         ToneLibInfo          `xml:"info"`
	BarIndex     ToneLibBarIndex      `xml:"BarIndex"`
	Tracks       ToneLibTracks        `xml:"Tracks"`
	BackingTrack *ToneLibBackingTrack `xml:"Backing_track1,omitempty"`
}

// Song metadata
type ToneLibInfo struct {
	Name        string `xml:"name"`
	Artist      string `xml:"artist"`
	Album       string `xml:"album"`
	Author      string `xml:"author"`
	Date        string `xml:"date"`
	Copyright   string `xml:"copyright"`
	Writer      string `xml:"writer"`
	Transcriber string `xml:"transcriber"`
	Remarks     string `xml:"remarks"`
	ShowRemarks string `xml:"show_remarks"`
}

// Bar index for tempo and time signature
type ToneLibBarIndex struct {
	Bars []ToneLibBar `xml:"Bar"`
}

type ToneLibBar struct {
	ID       int                   `xml:"id,attr"`
	Tempo    int                   `xml:"tempo,attr,omitempty"`
	JamSet   int                   `xml:"jam_set,attr"`
	TimeSign *ToneLibTimeSignature `xml:"time_sign,omitempty"`
	Label    *ToneLibLabel         `xml:"label,omitempty"`
}

type ToneLibTimeSignature struct {
	Numerator int `xml:"numerator,attr"`
	Duration  int `xml:"duration,attr"`
}

type ToneLibLabel struct {
	Letter string `xml:"letter,attr"`
	Text   string `xml:"text,attr"`
}

// Tracks container
type ToneLibTracks struct {
	Tracks []ToneLibTrack `xml:"Track"`
}

// Individual track
type ToneLibTrack struct {
	Name     string           `xml:"name,attr"`
	Color    string           `xml:"color,attr"`
	Visible  int              `xml:"visible,attr"`
	Collapse int              `xml:"collapse,attr"`
	Lock     int              `xml:"lock,attr"`
	Solo     int              `xml:"solo,attr"`
	Mute     int              `xml:"mute,attr"`
	Opt      int              `xml:"opt,attr"`
	VolDB    string           `xml:"vol_db,attr"`
	Bank     int              `xml:"bank,attr"`
	Program  int              `xml:"program,attr"`
	Chorus   int              `xml:"chorus,attr"`
	Reverb   int              `xml:"reverb,attr"`
	Phaser   int              `xml:"phaser,attr"`
	Tremolo  int              `xml:"tremolo,attr"`
	ID       int              `xml:"id,attr"`
	Offset   int              `xml:"offset,attr"`
	Strings  ToneLibStrings   `xml:"Strings"`
	Bars     ToneLibTrackBars `xml:"Bars"`
}

// String definitions
type ToneLibStrings struct {
	Strings []ToneLibString `xml:"String"`
}

type ToneLibString struct {
	ID     int `xml:"id,attr"`
	Tuning int `xml:"tuning,attr"`
}

// Track bars container
type ToneLibTrackBars struct {
	Bars []ToneLibTrackBar `xml:"Bar"`
}

// Individual bar in a track
type ToneLibTrackBar struct {
	ID       int             `xml:"id,attr"`
	Clef     *ToneLibClef    `xml:"Clef,omitempty"`
	KeySign  *ToneLibKeySign `xml:"KeySign,omitempty"`
	Beats    []ToneLibBeat   `xml:"Beat"`
	BeatsEnd *string         `xml:"Beats"` // Required empty closing tag
}

type ToneLibClef struct {
	Value int `xml:"value,attr"`
}

type ToneLibKeySign struct {
	Value int `xml:"value,attr"`
}

// Beat element containing notes
type ToneLibBeat struct {
	Duration int           `xml:"duration,attr"`
	Dyn      string        `xml:"dyn,attr"`
	Dotted   int           `xml:"dotted,attr,omitempty"`
	Notes    []ToneLibNote `xml:"Note,omitempty"`
	Text     *ToneLibText  `xml:"Text,omitempty"`
}

// Note element
type ToneLibNote struct {
	Fret    int             `xml:"fret,attr"`
	String  int             `xml:"string,attr"`
	Tied    string          `xml:"tied,attr,omitempty"`
	Effects *ToneLibEffects `xml:"Effects,omitempty"`
}

// Text element for lyrics
type ToneLibText struct {
	Value string `xml:"value,attr"`
}

// Effects container
type ToneLibEffects struct {
	Ghost string        `xml:"ghost,attr,omitempty"`
	Grace *ToneLibGrace `xml:"Grace,omitempty"`
}

// Grace note
type ToneLibGrace struct {
	Fret       int `xml:"fret,attr"`
	Duration   int `xml:"duration,attr"`
	Dynamic    int `xml:"dynamic,attr"`
	Transition int `xml:"transition,attr"`
}

// Audio backing track
type ToneLibBackingTrack struct {
	Color    string       `xml:"color,attr"`
	Visible  int          `xml:"visible,attr"`
	Collapse int          `xml:"collapse,attr"`
	Lock     int          `xml:"lock,attr"`
	Solo     int          `xml:"solo,attr"`
	Mute     int          `xml:"mute,attr"`
	Opt      int          `xml:"opt,attr"`
	VolDB    string       `xml:"vol_db,attr"`
	Audio    ToneLibAudio `xml:"audio"`
}

type ToneLibAudio struct {
	Name        string             `xml:"name"`
	DataFile    string             `xml:"data_file"`
	DataLen     int                `xml:"data_len"`
	TimeOffset  string             `xml:"time_offset"`
	Gain        string             `xml:"gain"`
	ChannelMode int                `xml:"channel_mode"`
	Bars        ToneLibBackingBars `xml:"bars"`
}

// Bars element for backing track
type ToneLibBackingBars struct {
	Num   string               `xml:"num,attr"`
	NST   string               `xml:"nst,attr"`
	Beats []ToneLibBackingBeat `xml:"-"` // Don't marshal automatically
}

// Beat element for backing track bars
type ToneLibBackingBeat struct {
	N int    `xml:"n,attr"`
	T string `xml:"t,attr"`
}

// Custom marshaling to create beat0, beat1, beat2, etc.
func (b ToneLibBackingBars) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Marshal attributes
	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "num"}, Value: b.Num})
	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "nst"}, Value: b.NST})

	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Marshal each beat with dynamic tag names
	for i, beat := range b.Beats {
		beatStart := xml.StartElement{Name: xml.Name{Local: fmt.Sprintf("beat%d", i)}}
		if err := e.EncodeElement(beat, beatStart); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// WriteToneLibXMLTo writes a MIDI file as ToneLib the_song.dat XML format to the writer
func WriteToneLibXMLTo(writer io.Writer, midiFile *smf.SMF, sngFile *SngFile, beatMap *BeatMap) error {
	score := createToneLibScore(midiFile, sngFile, beatMap)
	return writeScoreXML(score, writer)
}

// createDefaultBarIndex creates a simple bar structure with default 120 BPM tempo
func createDefaultBarIndex(midiFile *smf.SMF) ToneLibBarIndex {
	// Estimate number of bars based on MIDI length
	// This is a rough calculation - in practice you'd want tempo events
	numBars := estimateBarCount(midiFile)

	bars := make([]ToneLibBar, numBars)
	for i := 0; i < numBars; i++ {
		bar := ToneLibBar{
			ID:     i + 1,
			JamSet: 0,
		}

		// Set tempo on first bar, and time signature
		if i == 0 {
			bar.Tempo = 120 // Default BPM
			bar.TimeSign = &ToneLibTimeSignature{
				Numerator: 4,
				Duration:  4,
			}
		}

		bars[i] = bar
	}

	return ToneLibBarIndex{Bars: bars}
}

// createBarIndexFromTimeline creates bar index from extracted BEAT track timeline
func createBarIndexFromTimeline(timeline *Timeline) ToneLibBarIndex {
	if len(timeline.Measures) == 0 {
		// Fallback to simple structure
		return ToneLibBarIndex{
			Bars: []ToneLibBar{{
				ID: 1, Tempo: 120, JamSet: 0,
				TimeSign: &ToneLibTimeSignature{Numerator: 4, Duration: 4},
			}},
		}
	}

	bars := make([]ToneLibBar, len(timeline.Measures))
	var lastTempo int

	for i, measure := range timeline.Measures {
		bar := ToneLibBar{
			ID:     i + 1,
			JamSet: 0,
		}

		// Set tempo using BPM from MIDI tempo events, rounded to nearest integer for ToneLib format
		currentTempo := int(measure.BeatsPerMinute + 0.5) // Round to nearest integer
		if i == 0 || currentTempo != lastTempo {
			bar.Tempo = currentTempo
			lastTempo = currentTempo
		}

		// Set time signature if it's different from 4/4 or first bar
		if i == 0 || measure.BeatsPerMeasure != 4 {
			bar.TimeSign = &ToneLibTimeSignature{
				Numerator: measure.BeatsPerMeasure,
				Duration:  4, // Assuming quarter note base
			}
		}

		bars[i] = bar
	}

	return ToneLibBarIndex{Bars: bars}
}

// createTracksFromMidi analyzes MIDI tracks and creates ToneLib tracks
// For now, only creates drum tracks from Rock Band "PART DRUMS" track
func createTracksFromMidi(midiFile *smf.SMF, numBars int) ToneLibTracks {
	var tracks []ToneLibTrack

	// Find the "PART DRUMS" track specifically
	var drumTrack smf.Track
	var drumTrackFound bool

	for _, track := range midiFile.Tracks {
		trackName := getTrackName(track)
		if trackName == "PART DRUMS" {
			drumTrack = track
			drumTrackFound = true
			break
		}
	}

	// Only create drum track if found
	if drumTrackFound {
		// Extract Rock Band expert drum notes
		expertDrumNotes := extractDrumNotes(drumTrack)

		if len(expertDrumNotes) > 0 {
			toneLibTrack := ToneLibTrack{
				Name:     "Drum",
				Color:    "fffad11c", // Orange color for drums
				Visible:  1,
				Collapse: 0,
				Lock:     0,
				Solo:     0,
				Mute:     0,
				Opt:      0,
				VolDB:    "0",
				Bank:     128, // Percussion bank
				Program:  0,   // Standard drum kit
				Chorus:   0,
				Reverb:   0,
				Phaser:   0,
				Tremolo:  0,
				ID:       1,
				Offset:   24, // Required for correct pitch playback
				Strings:  createDrumStrings(),
				Bars:     createDrumBarsFromNotes(expertDrumNotes, midiFile, numBars),
			}

			tracks = append(tracks, toneLibTrack)
		}
	}

	return ToneLibTracks{Tracks: tracks}
}

// Helper functions for track creation
func getTrackColor(trackName string, isDrumTrack bool) string {
	if isDrumTrack {
		return "fffad11c" // Orange for drums
	}
	switch strings.ToUpper(trackName) {
	case "VOCALS", "VOICE":
		return "fff5a41c" // Red for vocals
	case "GUITAR":
		return "ff00ff00" // Green for guitar
	case "BASS":
		return "ff0000ff" // Blue for bass
	default:
		return "ffffffff" // White for others
	}
}

func getBankForTrack(isDrumTrack bool) int {
	if isDrumTrack {
		return 128 // Percussion bank
	}
	return 0 // Standard bank
}

func getProgramForTrack(trackName string, isDrumTrack bool) int {
	if isDrumTrack {
		return 0 // Standard drum kit
	}
	switch strings.ToUpper(trackName) {
	case "VOCALS", "VOICE":
		return 27 // Distorted guitar (often used for vocals)
	case "GUITAR":
		return 26 // Electric guitar (jazz)
	case "BASS":
		return 34 // Electric bass (fingered)
	default:
		return 1 // Acoustic piano
	}
}

// createDrumStrings creates string definitions for drum tracks
func createDrumStrings() ToneLibStrings {
	strings := make([]ToneLibString, 6)

	// All drums use tuning 0 so fret = MIDI note directly
	for i := 0; i < 6; i++ {
		strings[i] = ToneLibString{ID: i + 1, Tuning: 0}
	}

	return ToneLibStrings{Strings: strings}
}

// createDrumBarsFromNotes converts Rock Band drum notes to ToneLib bars
func createDrumBarsFromNotes(drumNotes []DrumNote, midiFile *smf.SMF, numBars int) ToneLibTrackBars {
	if len(drumNotes) == 0 {
		// Create empty bars matching BarIndex count
		emptyBeats := ""
		bars := make([]ToneLibTrackBar, numBars)

		for i := 0; i < numBars; i++ {
			bar := ToneLibTrackBar{
				ID:       i + 1,
				Beats:    []ToneLibBeat{{Duration: 1, Dyn: "mf"}}, // Whole rest
				BeatsEnd: &emptyBeats,
			}

			// Add clef and key signature to first bar
			if i == 0 {
				bar.Clef = &ToneLibClef{Value: 5} // Percussion clef
				bar.KeySign = &ToneLibKeySign{Value: 0}
			}

			bars[i] = bar
		}

		return ToneLibTrackBars{Bars: bars}
	}

	// Get ticks per quarter note for timing calculations
	ticksPerQuarter := int(480) // Default
	if tf, ok := midiFile.TimeFormat.(smf.MetricTicks); ok {
		ticksPerQuarter = int(tf)
	}

	// Simple quantization: group notes into bars of 4/4 time
	// Each bar = 4 quarter notes = 4 * ticksPerQuarter ticks
	ticksPerBar := ticksPerQuarter * 4

	// Group notes by bar
	barNotes := make(map[int][]DrumNote)

	// DrumNote.Time is already absolute time in ticks
	for _, note := range drumNotes {
		barNum := int(note.Time/uint32(ticksPerBar)) + 1
		// Only include notes within the expected bar count
		if barNum <= numBars {
			barNotes[barNum] = append(barNotes[barNum], note)
		}
	}

	// Create ToneLib bars - exactly numBars to match BarIndex
	var bars []ToneLibTrackBar
	emptyBeats := ""

	for barID := 1; barID <= numBars; barID++ {
		bar := ToneLibTrackBar{
			ID:       barID,
			Beats:    []ToneLibBeat{},
			BeatsEnd: &emptyBeats, // Required empty closing tag for each bar
		}

		// Add clef and key signature to first bar
		if barID == 1 {
			bar.Clef = &ToneLibClef{Value: 5} // Percussion clef
			bar.KeySign = &ToneLibKeySign{Value: 0}
		}

		// Convert notes in this bar to beats
		notesInBar := barNotes[barID]
		if len(notesInBar) > 0 {
			bar.Beats = convertDrumNotesToBeats(notesInBar, barID, ticksPerQuarter)
		} else {
			// Empty bar - whole rest
			bar.Beats = []ToneLibBeat{{Duration: 1, Dyn: "mf"}}
		}

		bars = append(bars, bar)
	}

	return ToneLibTrackBars{
		Bars: bars,
	}
}

// convertDrumNotesToBeats converts drum notes in a bar to ToneLib beats
// This is a simplified quantization - groups notes into eighth note beats
func convertDrumNotesToBeats(notesInBar []DrumNote, barID int, ticksPerQuarter int) []ToneLibBeat {
	if len(notesInBar) == 0 {
		return []ToneLibBeat{{Duration: 1, Dyn: "mf"}} // Whole rest
	}

	// Calculate bar start time
	barStartTime := uint32((barID - 1) * ticksPerQuarter * 4)

	// Quantize to eighth notes (duration = 8 in ToneLib)
	ticksPerEighth := ticksPerQuarter / 2
	numEighths := 8 // 8 eighth notes per 4/4 bar

	// Group notes by eighth note position
	eighthNotes := make(map[int][]DrumNote)

	for _, note := range notesInBar {
		relativeTime := int(note.Time - barStartTime)
		eighthPos := relativeTime / ticksPerEighth
		if eighthPos >= numEighths {
			eighthPos = numEighths - 1 // Clamp to last eighth
		}
		eighthNotes[eighthPos] = append(eighthNotes[eighthPos], note)
	}

	// Create beats
	var beats []ToneLibBeat

	for eighthPos := 0; eighthPos < numEighths; eighthPos++ {
		notes := eighthNotes[eighthPos]

		if len(notes) > 0 {
			// Create beat with notes
			beat := ToneLibBeat{
				Duration: 8, // Eighth note
				Dyn:      "mf",
				Notes:    []ToneLibNote{},
			}

			// Convert each Rock Band drum note to ToneLib note
			stringID := 1
			for _, drumNote := range notes {
				// Convert Rock Band key (96-100) to GM drum note
				gmKey, err := drumNote.toMidiKey()
				if err != nil {
					continue // Skip invalid notes
				}

				// Create ToneLib note - fret = GM MIDI note, since string tuning = 0
				toneLibNote := ToneLibNote{
					Fret:   int(gmKey),
					String: stringID,
				}

				beat.Notes = append(beat.Notes, toneLibNote)
				stringID++ // Use different strings for visual separation
				if stringID > 6 {
					stringID = 1 // Wrap around
				}
			}

			beats = append(beats, beat)
		} else {
			// Create rest beat
			beats = append(beats, ToneLibBeat{
				Duration: 8,
				Dyn:      "mf",
			})
		}
	}

	return beats
}

func getClefValue(isDrumTrack bool) int {
	if isDrumTrack {
		return 5 // Percussion clef
	}
	return 1 // Treble clef
}

// Utility functions
func isDrumTrack(track smf.Track) bool {
	// Check if track uses MIDI channel 10 (drums)
	for _, event := range track {
		msg := event.Message
		var ch, key, vel uint8
		if msg.GetNoteOn(&ch, &key, &vel) && ch == 9 { // Channel 10 is index 9
			return true
		}
	}
	return false
}

func estimateBarCount(midiFile *smf.SMF) int {
	// Simple estimation based on file length
	// In practice, you'd analyze the actual MIDI events
	maxTicks := uint32(0)

	for _, track := range midiFile.Tracks {
		currentTick := uint32(0)
		for _, event := range track {
			currentTick += event.Delta
		}
		if currentTick > maxTicks {
			maxTicks = currentTick
		}
	}

	// Assume 480 ticks per quarter note, 4 beats per bar
	if tf, ok := midiFile.TimeFormat.(smf.MetricTicks); ok {
		ticksPerBar := uint32(tf) * 4
		bars := int((maxTicks + ticksPerBar - 1) / ticksPerBar) // Ceiling division
		if bars < 1 {
			bars = 1
		}
		return bars
	}

	return 4 // Default fallback
}

// printXML outputs the ToneLib score as XML to stdout
func writeScoreXML(score *ToneLibScore, writer io.Writer) error {
	// Buffer the XML output for post-processing
	var buf bytes.Buffer

	buf.Write([]byte(xml.Header))
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")

	if err := encoder.Encode(score); err != nil {
		return fmt.Errorf("failed to encode XML: %w", err)
	}

	buf.Write([]byte("\n")) // Add final newline

	// Apply post-processing transformations
	xmlString := buf.String()

	// 1. Convert empty tags to self-closing format
	// Pattern matches: <tagname attributes></tagname> where tagname is repeated
	emptyTagRegex := regexp.MustCompile(`<(\w+)([^>]*?)></\w+>`)
	xmlString = emptyTagRegex.ReplaceAllStringFunc(xmlString, func(match string) string {
		// Extract tag name and attributes more carefully
		openTagRegex := regexp.MustCompile(`<(\w+)([^>]*?)></\w+>`)
		matches := openTagRegex.FindStringSubmatch(match)
		if len(matches) >= 3 {
			tagName := matches[1]
			attributes := matches[2]
			// Verify the closing tag matches the opening tag
			if strings.Contains(match, "</"+tagName+">") {
				return "<" + tagName + attributes + "/>"
			}
		}
		return match // Return unchanged if pattern doesn't match properly
	})

	// 2. Convert Unix line endings (LF) to DOS line endings (CRLF)
	xmlString = strings.ReplaceAll(xmlString, "\n", "\r\n")

	// Write the transformed XML to the final writer
	_, err := writer.Write([]byte(xmlString))
	if err != nil {
		return fmt.Errorf("failed to write transformed XML: %w", err)
	}

	return nil
}

// createZipEntryWithCurrentTime creates a new ZIP entry with the current timestamp
func createZipEntryWithCurrentTime(w *zip.Writer, name string) (io.Writer, error) {
	header := &zip.FileHeader{
		Name:     name,
		Modified: time.Now(),
		Method:   zip.Deflate,
	}
	return w.CreateHeader(header)
}

// WriteToneLibSongTo writes a complete ToneLib .song ZIP archive to the writer
func WriteToneLibSongTo(writer io.Writer, midiFile *smf.SMF, sngFile *SngFile, useAubio bool) error {
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	// 1. Create version.info (4 bytes: "3.1\0")
	versionWriter, err := createZipEntryWithCurrentTime(zipWriter, "version.info")
	if err != nil {
		return fmt.Errorf("failed to create version.info: %w", err)
	}
	versionBytes := []byte{0x33, 0x2e, 0x31, 0x00} // "3.1" + null terminator
	if _, err := versionWriter.Write(versionBytes); err != nil {
		return fmt.Errorf("failed to write version.info: %w", err)
	}

	// 2. Convert audio first to get the converted data length and path
	var convertedAudioLen int
	var audioFilePath string
	var mergedAudio *MergedAudio
	if sngFile != nil {
		// Merge all opus files into a single audio file
		mergedAudio, err = sngFile.GetMergedAudio()
		if err != nil {
			return fmt.Errorf("failed to merge audio files: %w", err)
		}
		defer mergedAudio.Close()

		// Read the converted audio data
		convertedData, err := os.ReadFile(mergedAudio.FilePath)
		if err != nil {
			return fmt.Errorf("failed to read merged audio: %w", err)
		}
		convertedAudioLen = len(convertedData)

		// Create hash for filename
		audioFilename := "merged_audio.opus"
		hash := sha256.Sum256([]byte(audioFilename))
		audioHash := hex.EncodeToString(hash[:])[:16]
		audioFilePath = fmt.Sprintf("audio/%s.snd", audioHash)

		// Write converted audio to ZIP
		audioWriter, err := createZipEntryWithCurrentTime(zipWriter, audioFilePath)
		if err != nil {
			return fmt.Errorf("failed to create audio file in ZIP: %w", err)
		}

		if _, err := audioWriter.Write(convertedData); err != nil {
			return fmt.Errorf("failed to write audio data: %w", err)
		}
	}

	// 3. Extract beats using aubiotrack if requested and audio is available
	var beatMap *BeatMap
	if useAubio && mergedAudio != nil {
		log.Printf("Running aubiotrack on merged audio file...")
		beatTimes, err := ExtractAudioBeats(mergedAudio.FilePath)
		if err != nil {
			log.Printf("Warning: aubiotrack failed: %v", err)
			beatMap = nil
		} else {
			beatMap = convertBeatsTimesToBeatMap(beatTimes)
		}
	}

	// 4. Create the_song.dat XML file with correct audio data length and path
	score := createToneLibScore(midiFile, sngFile, beatMap)
	if score.BackingTrack != nil {
		score.BackingTrack.Audio.DataLen = convertedAudioLen
		score.BackingTrack.Audio.DataFile = audioFilePath
	}

	songWriter, err := createZipEntryWithCurrentTime(zipWriter, "the_song.dat")
	if err != nil {
		return fmt.Errorf("failed to create the_song.dat: %w", err)
	}

	if err := writeScoreXML(score, songWriter); err != nil {
		return fmt.Errorf("failed to write the_song.dat: %w", err)
	}

	return nil
}

// convertBeatsTimesToBeatMap converts raw beat timestamps to ToneLib BeatMap format
func convertBeatsTimesToBeatMap(beatTimes []float64) *BeatMap {
	beats := make([]ToneLibBackingBeat, len(beatTimes))

	for i, beatTime := range beatTimes {
		beats[i] = ToneLibBackingBeat{
			N: i,
			T: fmt.Sprintf("%.15f", beatTime), // High precision for timing
		}
	}

	return &BeatMap{
		Beats:    beats,
		TotalNum: strconv.Itoa(len(beats)),
		NST:      "", // Unknown meaning, leave blank
	}
}

func createToneLibScore(midiFile *smf.SMF, sngFile *SngFile, beatMap *BeatMap) *ToneLibScore {
	score := &ToneLibScore{
		Info: ToneLibInfo{
			ShowRemarks: "no",
		},
	}

	// Fill in metadata from SNG file if available, otherwise from MIDI
	if sngFile != nil {
		metadata := sngFile.GetMetadata()
		score.Info.Name = metadata["name"]
		score.Info.Artist = metadata["artist"]
		score.Info.Album = metadata["album"]
		score.Info.Author = metadata["author"]
		score.Info.Writer = metadata["writer"]
	} else {
		// Use track 0 name as song title if no SNG metadata
		if len(midiFile.Tracks) > 0 {
			trackName := getTrackName(midiFile.Tracks[0])
			if trackName != "" {
				score.Info.Name = trackName
			}
		}
	}

	// Extract timeline for tempo mapping
	timeline, err := ExtractBeatTimeline(midiFile)
	if err != nil {
		// If no BEAT track, create a simple bar structure with default tempo
		score.BarIndex = createDefaultBarIndex(midiFile)
	} else {
		score.BarIndex = createBarIndexFromTimeline(timeline)
	}

	// Create tracks from MIDI - ensure bar count matches BarIndex
	numBars := len(score.BarIndex.Bars)
	score.Tracks = createTracksFromMidi(midiFile, numBars)

	// Add backing track reference if SNG file has audio
	if sngFile != nil {
		// Check for any opus files in SNG
		files := sngFile.ListFiles()
		hasOpusFiles := false
		for _, filename := range files {
			if strings.HasSuffix(filename, ".opus") {
				hasOpusFiles = true
				break
			}
		}

		if hasOpusFiles {
			// Create bars structure with beat map data if available
			bars := ToneLibBackingBars{
				Num:   "0",
				NST:   "",
				Beats: []ToneLibBackingBeat{},
			}

			if beatMap != nil {
				bars.Num = beatMap.TotalNum
				bars.NST = beatMap.NST
				bars.Beats = beatMap.Beats
			}

			score.BackingTrack = &ToneLibBackingTrack{
				Color:    "ff40a0a0",
				Visible:  1,
				Collapse: 0,
				Lock:     0,
				Solo:     0,
				Mute:     0,
				Opt:      0,
				VolDB:    "0",
				Audio: ToneLibAudio{
					Name:        "merged.ogg", // Display name for merged audio
					DataFile:    "",           // Will be updated with actual path from conversion
					DataLen:     0,            // Will be updated with actual converted size
					TimeOffset:  "0",
					Gain:        "1",
					ChannelMode: 0,
					Bars:        bars,
				},
			}
		}
	}

	return score
}
