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
	"sort"
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

// LyricEvent represents a lyric event with timing information
type LyricEvent struct {
	Time  uint32 // Absolute time in ticks
	Lyric string // Raw lyric text from MIDI (preserves Rock Band formatting)
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

type ToneLibTracks struct {
	Tracks []ToneLibTrack `xml:"Track"`
}

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

type ToneLibStrings struct {
	Strings []ToneLibString `xml:"String"`
}

type ToneLibString struct {
	ID     int `xml:"id,attr"`
	Tuning int `xml:"tuning,attr"`
}

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

// Clef types
const (
	ToneLibTrebleClef     = 1
	ToneLibBassClef       = 2
	ToneLibPercussionClef = 5
)

// Default track colors
const (
	ToneLibDrumColor    = "fffad11c" // Orange
	ToneLibBassColor    = "ff0000ff" // Blue
	ToneLibLyricsColor  = "ff00ff00" // Green
	ToneLibBackingColor = "ff40a0a0" // Teal
)

// Note duration constants
const (
	ToneLibWholeNoteDuration        = 1
	ToneLibHalfNoteDuration         = 2
	ToneLibQuarterNoteDuration      = 4
	ToneLibEighthNoteDuration       = 8
	ToneLibSixteenthNoteDuration    = 16
	ToneLibThirtySecondNoteDuration = 32
)

// ToneLib default values
const (
	ToneLibDefaultVolDB           = "-0.1574783325195312"
	ToneLibDefaultOffset          = 24 // this is reuqired for the notes to be pitched correctly
	ToneLibDefaultTempo           = 120
	ToneLibDefaultBeatsPerMeasure = 4
	ToneLibDefaultDynamic         = "mf"
)

// MusicalNote represents a musical note that can be converted to ToneLib format.
type MusicalNote interface {
	GetTime() uint32 // returns the absolute timing of the note in MIDI ticks
	ConvertToToneLibNote() (ToneLibNote, error)
}

type BarCreationConfig struct {
	ClefValue        int // ToneLib clef type (percussion, treble, or bass)
	TicksPerQuarter  int // MIDI timing resolution
	NumBars          int // Total number of bars to create
	NumEighthsPerBar int // Number of eighth-note subdivisions per bar (typically 8 for 4/4 time)
}

type TrackCreationContext struct {
	MidiFile *smf.SMF  // Source MIDI file containing Rock Band data
	NumBars  int       // Total number of bars in the song
	Timeline *Timeline // Extracted beat timeline for accurate timing
	TrackID  *int      // Pointer to current track ID counter (auto-incremented)
}

type AudioProcessingResult struct {
	MergedAudio       *MergedAudio // Temporary merged audio file (needs cleanup)
	ConvertedAudioLen int          // Size of converted audio data in bytes
	AudioFilePath     string       // Path within ZIP archive for audio file
}

func (d DrumNote) GetTime() uint32 {
	return d.Time
}

func (d DrumNote) ConvertToToneLibNote() (ToneLibNote, error) {
	gmKey, err := d.toMidiKey()
	if err != nil {
		return ToneLibNote{}, err
	}

	return ToneLibNote{
		Fret:   int(gmKey),
		String: 1, // Will be assigned by the caller for visual separation
	}, nil
}

func (b BassNote) GetTime() uint32 {
	return b.Time
}

func (b BassNote) ConvertToToneLibNote() (ToneLibNote, error) {
	midiNote, err := b.toMidiNote()
	if err != nil {
		return ToneLibNote{}, err
	}

	// Map Rock Band bass strings to ToneLib strings (reverse order)
	toneLibStringID := 4 - int(b.String)

	// Use standard bass tuning from constants
	stringTuning := BassTuning[toneLibStringID-1] // Convert to 0-indexed
	fret := int(midiNote) - stringTuning

	if fret < 0 {
		fret = 0
	}

	return ToneLibNote{
		Fret:   fret,
		String: toneLibStringID,
	}, nil
}

// Group a list of notes into the bars (aka measures) for tonelib export
// 1. Groups notes by measure using timing calculations
// 2. Creates empty bars with appropriate clef and key signature
// 3. Converts notes to beats using convertNotesToBeats
// 4. Handles empty bars with whole rests
func createBarsFromNotes[T MusicalNote](notes []T, config BarCreationConfig) ToneLibTrackBars {
	// Calculate timing values
	ticksPerBar := config.TicksPerQuarter * ToneLibDefaultBeatsPerMeasure

	// Group notes by bar
	barNotes := make(map[int][]T)
	for _, note := range notes {
		barNum := int(note.GetTime()/uint32(ticksPerBar)) + 1
		if barNum <= config.NumBars {
			barNotes[barNum] = append(barNotes[barNum], note)
		}
	}

	// Create ToneLib bars
	var bars []ToneLibTrackBar
	emptyBeats := ""

	for barID := 1; barID <= config.NumBars; barID++ {
		bar := ToneLibTrackBar{
			ID:       barID,
			Beats:    []ToneLibBeat{},
			BeatsEnd: &emptyBeats,
		}

		// Add clef and key signature to first bar only
		if barID == 1 {
			bar.Clef = &ToneLibClef{Value: config.ClefValue}
			bar.KeySign = &ToneLibKeySign{Value: 0}
		}

		// Convert notes in this bar to beats
		notesInBar := barNotes[barID]
		if len(notesInBar) > 0 {
			bar.Beats = convertNotesToBeats(notesInBar, barID, config)
		} else {
			// Empty bar - whole rest
			bar.Beats = []ToneLibBeat{{Duration: ToneLibWholeNoteDuration, Dyn: ToneLibDefaultDynamic}}
		}

		bars = append(bars, bar)
	}

	return ToneLibTrackBars{Bars: bars}
}

// convertNotesToBeats converts notes in a bar to ToneLib beats with eighth note quantization
func convertNotesToBeats[T MusicalNote](notesInBar []T, barID int, config BarCreationConfig) []ToneLibBeat {
	if len(notesInBar) == 0 {
		return []ToneLibBeat{{Duration: ToneLibWholeNoteDuration, Dyn: ToneLibDefaultDynamic}}
	}

	// Calculate bar start time and eighth note positions
	barStartTime := uint32((barID - 1) * config.TicksPerQuarter * ToneLibDefaultBeatsPerMeasure)
	ticksPerEighth := config.TicksPerQuarter / 2

	// Group notes by eighth note position
	eighthNotes := make(map[int][]T)
	for _, note := range notesInBar {
		relativeTime := int(note.GetTime() - barStartTime)
		eighthPos := relativeTime / ticksPerEighth
		if eighthPos >= config.NumEighthsPerBar {
			eighthPos = config.NumEighthsPerBar - 1
		}
		eighthNotes[eighthPos] = append(eighthNotes[eighthPos], note)
	}

	// Create beats
	var beats []ToneLibBeat
	for eighthPos := 0; eighthPos < config.NumEighthsPerBar; eighthPos++ {
		notes := eighthNotes[eighthPos]

		if len(notes) > 0 {
			beat := ToneLibBeat{
				Duration: ToneLibEighthNoteDuration,
				Dyn:      ToneLibDefaultDynamic,
				Notes:    []ToneLibNote{},
			}

			// Convert each note to ToneLib format
			stringID := 1
			for _, note := range notes {
				toneLibNote, err := note.ConvertToToneLibNote()
				if err != nil {
					continue // Skip invalid notes
				}

				// For drums, assign different strings for visual separation
				if config.ClefValue == ToneLibPercussionClef {
					toneLibNote.String = stringID
					stringID++
					if stringID > 6 {
						stringID = 1 // Wrap around
					}
				}

				beat.Notes = append(beat.Notes, toneLibNote)
			}

			beats = append(beats, beat)
		} else {
			// Create rest beat
			beats = append(beats, ToneLibBeat{
				Duration: ToneLibEighthNoteDuration,
				Dyn:      ToneLibDefaultDynamic,
			})
		}
	}

	return beats
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
			bar.Tempo = ToneLibDefaultTempo
			bar.TimeSign = &ToneLibTimeSignature{
				Numerator: ToneLibDefaultBeatsPerMeasure,
				Duration:  ToneLibQuarterNoteDuration,
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
				ID: 1, Tempo: ToneLibDefaultTempo, JamSet: 0,
				TimeSign: &ToneLibTimeSignature{
					Numerator: ToneLibDefaultBeatsPerMeasure,
					Duration:  ToneLibQuarterNoteDuration,
				},
			}},
		}
	}

	// Quantize BPMs to minimize cumulative drift
	quantizedTimeline := QuantizeBPMs(timeline)

	bars := make([]ToneLibBar, len(quantizedTimeline.Measures))
	var lastTempo int

	for i, measure := range quantizedTimeline.Measures {
		bar := ToneLibBar{
			ID:     i + 1,
			JamSet: 0,
		}

		// BPM is now already an integer from quantization process
		currentTempo := int(measure.BeatsPerMinute)
		if i == 0 || currentTempo != lastTempo {
			bar.Tempo = currentTempo
			lastTempo = currentTempo
		}

		// Set time signature if it's different from 4/4 or first bar
		if i == 0 || measure.BeatsPerMeasure != ToneLibDefaultBeatsPerMeasure {
			bar.TimeSign = &ToneLibTimeSignature{
				Numerator: measure.BeatsPerMeasure,
				Duration:  ToneLibQuarterNoteDuration,
			}
		}

		bars[i] = bar
	}

	return ToneLibBarIndex{Bars: bars}
}

// Create all the ToneLib tracks from the source MIDI file
func createTracksFromMidi(midiFile *smf.SMF, numBars int, timeline *Timeline) ToneLibTracks {
	var tracks []ToneLibTrack
	trackID := 1

	ctx := &TrackCreationContext{
		MidiFile: midiFile,
		NumBars:  numBars,
		Timeline: timeline,
		TrackID:  &trackID,
	}

	// Create tracks in order: lyrics, drums, bass
	if lyricsTrack := createLyricsTrackFromMidi(ctx); lyricsTrack != nil {
		tracks = append(tracks, *lyricsTrack)
	}

	if bassTrack := createBassTrackFromMidi(ctx); bassTrack != nil {
		tracks = append(tracks, *bassTrack)
	}

	if drumTrack := createDrumTrackFromMidi(ctx); drumTrack != nil {
		tracks = append(tracks, *drumTrack)
	}

	return ToneLibTracks{Tracks: tracks}
}

// createLyricsTrackFromMidi extracts and creates a lyrics track if available
func createLyricsTrackFromMidi(ctx *TrackCreationContext) *ToneLibTrack {
	lyricEvents := extractLyricsWithTiming(ctx.MidiFile)
	if len(lyricEvents) == 0 || ctx.Timeline == nil {
		return nil
	}

	measureLyrics := groupLyricsByMeasure(lyricEvents, ctx.Timeline)
	if len(measureLyrics) == 0 {
		return nil
	}

	lyricsTrack := createLyricsTrack(measureLyrics, ctx.MidiFile, ctx.NumBars, *ctx.TrackID, ctx.Timeline)
	*ctx.TrackID++
	log.Printf("Created lyrics track with %d measures containing lyrics", len(measureLyrics))
	return &lyricsTrack
}

// createDrumTrackFromMidi extracts and creates a drum track if available
func createDrumTrackFromMidi(ctx *TrackCreationContext) *ToneLibTrack {
	// Find the "PART DRUMS" track specifically
	var drumTrack smf.Track
	var drumTrackFound bool

	for _, track := range ctx.MidiFile.Tracks {
		trackName := getTrackName(track)
		if trackName == "PART DRUMS" {
			drumTrack = track
			drumTrackFound = true
			break
		}
	}

	if !drumTrackFound {
		return nil
	}

	// Extract Rock Band expert drum notes
	expertDrumNotes := extractDrumNotes(drumTrack)
	if len(expertDrumNotes) == 0 {
		return nil
	}

	toneLibTrack := ToneLibTrack{
		Name:     "Drum",
		Color:    ToneLibDrumColor,
		Visible:  1,
		Collapse: 0,
		Lock:     0,
		Solo:     0,
		Mute:     0,
		Opt:      0,
		VolDB:    ToneLibDefaultVolDB,
		Bank:     128, // Percussion bank
		Program:  0,   // Standard drum kit
		Chorus:   0,
		Reverb:   0,
		Phaser:   0,
		Tremolo:  0,
		ID:       *ctx.TrackID,
		Offset:   ToneLibDefaultOffset,
		Strings:  createDrumStrings(),
		Bars:     createDrumBarsFromNotes(expertDrumNotes, ctx.MidiFile, ctx.NumBars),
	}

	*ctx.TrackID++
	return &toneLibTrack
}

// createBassTrackFromMidi extracts and creates a bass track if available
func createBassTrackFromMidi(ctx *TrackCreationContext) *ToneLibTrack {
	// Find pro bass tracks
	var bassTrackConfig BassTrackInfo
	var bassTrack smf.Track
	var bassTrackFound bool

	// Try expert pro bass track first, then fall back to combined track
	bassTrackConfig, bassTrack, bassTrackFound = findBassTrack(ctx.MidiFile, "PART REAL_BASS_X")
	if !bassTrackFound {
		// Try combined track format
		bassTrackConfig, bassTrack, bassTrackFound = findBassTrack(ctx.MidiFile, "PART REAL_BASS")
	}

	if !bassTrackFound {
		return nil
	}

	// Extract pro bass notes
	expertBassNotes := extractBassNotes(bassTrack, bassTrackConfig)
	if len(expertBassNotes) == 0 {
		return nil
	}

	toneLibTrack := ToneLibTrack{
		Name:     "Bass",
		Color:    ToneLibBassColor,
		Visible:  1,
		Collapse: 0,
		Lock:     0,
		Solo:     0,
		Mute:     0,
		Opt:      0,
		VolDB:    ToneLibDefaultVolDB,
		Bank:     0,  // Standard bank
		Program:  33, // Electric Bass (finger)
		Chorus:   0,
		Reverb:   0,
		Phaser:   0,
		Tremolo:  0,
		ID:       *ctx.TrackID,
		Offset:   ToneLibDefaultOffset,
		Strings:  createBassStrings(),
		Bars:     createBassBarsFromNotes(expertBassNotes, ctx.MidiFile, ctx.NumBars),
	}

	*ctx.TrackID++
	return &toneLibTrack
}

// Standard tuning configurations
var (
	DrumTuning   = []int{0, 0, 0, 0, 0, 0}       // All drums use tuning 0
	BassTuning   = []int{43, 38, 33, 28}         // G, D, A, E (high to low)
	GuitarTuning = []int{64, 59, 55, 50, 45, 40} // E, B, G, D, A, E (high to low)
)

func createStringsWithTuning(tunings []int) ToneLibStrings {
	strings := make([]ToneLibString, len(tunings))

	for i, tuning := range tunings {
		strings[i] = ToneLibString{
			ID:     i + 1,
			Tuning: tuning,
		}
	}

	return ToneLibStrings{Strings: strings}
}

func createDrumStrings() ToneLibStrings {
	return createStringsWithTuning(DrumTuning)
}

func createBassStrings() ToneLibStrings {
	return createStringsWithTuning(BassTuning)
}

func createGuitarStrings() ToneLibStrings {
	return createStringsWithTuning(GuitarTuning)
}

// createDrumBarsFromNotes converts Rock Band drum notes to ToneLib bars using generic bar creation
func createDrumBarsFromNotes(drumNotes []DrumNote, midiFile *smf.SMF, numBars int) ToneLibTrackBars {
	// Get ticks per quarter note for timing calculations
	ticksPerQuarter := int(480) // Default
	if tf, ok := midiFile.TimeFormat.(smf.MetricTicks); ok {
		ticksPerQuarter = int(tf)
	}

	config := BarCreationConfig{
		ClefValue:        ToneLibPercussionClef,
		TicksPerQuarter:  ticksPerQuarter,
		NumBars:          numBars,
		NumEighthsPerBar: 8, // 8 eighth notes per 4/4 bar
	}

	return createBarsFromNotes(drumNotes, config)
}

// createBassBarsFromNotes converts Rock Band pro bass notes to ToneLib bars using generic bar creation
func createBassBarsFromNotes(bassNotes []BassNote, midiFile *smf.SMF, numBars int) ToneLibTrackBars {
	// Get ticks per quarter note for timing calculations
	ticksPerQuarter := int(480) // Default
	if tf, ok := midiFile.TimeFormat.(smf.MetricTicks); ok {
		ticksPerQuarter = int(tf)
	}

	config := BarCreationConfig{
		ClefValue:        ToneLibBassClef,
		TicksPerQuarter:  ticksPerQuarter,
		NumBars:          numBars,
		NumEighthsPerBar: 8, // 8 eighth notes per 4/4 bar
	}

	return createBarsFromNotes(bassNotes, config)
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
		matches := emptyTagRegex.FindStringSubmatch(match)
		if len(matches) >= 3 {
			tagName := matches[1]
			attributes := matches[2]
			// Verify the closing tag matches the opening tag
			if strings.Contains(match, "</"+tagName+">") {
				return "<" + tagName + attributes + "/>"
			}
		}
		return match
	})

	// 2. Convert Unix line endings (LF) to DOS line endings (CRLF)
	// xmlString = strings.ReplaceAll(xmlString, "\n", "\r\n")

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

// Generate and write a complete ToneLib .song ZIP archive to the writer
func WriteToneLibSongTo(writer io.Writer, midiFile *smf.SMF, sngFile *SngFile) error {
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	// 1. Create version.info
	if err := createVersionInfo(zipWriter); err != nil {
		return err
	}

	// 2. Process and add audio to ZIP
	audioResult, err := processAudioForZip(zipWriter, sngFile)
	if err != nil {
		return err
	}
	if audioResult != nil {
		defer audioResult.MergedAudio.Close()
	}

	// 3. Generate beats from timeline
	timeline, _ := ExtractBeatTimeline(midiFile)
	beatMap := generateBeatsFromTimeline(timeline)

	// 4. Create and write the_song.dat XML
	if err := writeToneLibXMLToZip(zipWriter, midiFile, sngFile, beatMap, audioResult); err != nil {
		return err
	}

	return nil
}

// createVersionInfo creates and writes the ToneLib version.info file to the ZIP
func createVersionInfo(zipWriter *zip.Writer) error {
	versionWriter, err := createZipEntryWithCurrentTime(zipWriter, "version.info")
	if err != nil {
		return fmt.Errorf("failed to create version.info: %w", err)
	}

	versionBytes := []byte{0x33, 0x2e, 0x31, 0x00} // "3.1" + null terminator
	if _, err := versionWriter.Write(versionBytes); err != nil {
		return fmt.Errorf("failed to write version.info: %w", err)
	}

	return nil
}

// processAudioForZip processes audio from SNG file and adds it to the ZIP
func processAudioForZip(zipWriter *zip.Writer, sngFile *SngFile) (*AudioProcessingResult, error) {
	if sngFile == nil {
		return nil, nil
	}

	// Merge all opus files into a single audio file
	mergedAudio, err := sngFile.GetMergedAudio()
	if err != nil {
		return nil, fmt.Errorf("failed to merge audio files: %w", err)
	}

	// Read the converted audio data
	convertedData, err := os.ReadFile(mergedAudio.FilePath)
	if err != nil {
		mergedAudio.Close()
		return nil, fmt.Errorf("failed to read merged audio: %w", err)
	}

	// Create hash for filename
	audioFilename := "merged_audio.opus"
	hash := sha256.Sum256([]byte(audioFilename))
	audioHash := hex.EncodeToString(hash[:])[:16]
	audioFilePath := fmt.Sprintf("audio/%s.snd", audioHash)

	// Write converted audio to ZIP
	audioWriter, err := createZipEntryWithCurrentTime(zipWriter, audioFilePath)
	if err != nil {
		mergedAudio.Close()
		return nil, fmt.Errorf("failed to create audio file in ZIP: %w", err)
	}

	if _, err := audioWriter.Write(convertedData); err != nil {
		mergedAudio.Close()
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}

	return &AudioProcessingResult{
		MergedAudio:       mergedAudio,
		ConvertedAudioLen: len(convertedData),
		AudioFilePath:     audioFilePath,
	}, nil
}

// generateBeatsFromTimeline generates beats from timeline data instead of audio analysis
func generateBeatsFromTimeline(timeline *Timeline) *BeatMap {
	if timeline == nil || len(timeline.BeatNotes) == 0 {
		return nil
	}

	beats := make([]ToneLibBackingBeat, len(timeline.BeatNotes))

	for i, beatNote := range timeline.BeatNotes {
		beats[i] = ToneLibBackingBeat{
			N: i,
			T: fmt.Sprintf("%.15f", beatNote.TimeSeconds), // High precision for timing
		}
	}

	log.Printf("Generated %d beats from timeline data", len(beats))
	return &BeatMap{
		Beats:    beats,
		TotalNum: fmt.Sprintf("%d", len(beats)),
		NST:      "0", // Unknown meaning, leave blank
	}
}

// writeToneLibXMLToZip creates and writes the_song.dat XML file to the ZIP
func writeToneLibXMLToZip(zipWriter *zip.Writer, midiFile *smf.SMF, sngFile *SngFile,
	beatMap *BeatMap, audioResult *AudioProcessingResult) error {

	// Create the score with audio metadata if available
	score := createToneLibScore(midiFile, sngFile, beatMap)
	if score.BackingTrack != nil && audioResult != nil {
		score.BackingTrack.Audio.DataLen = audioResult.ConvertedAudioLen
		score.BackingTrack.Audio.DataFile = audioResult.AudioFilePath
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

// createToneLibInfo extracts metadata and creates the ToneLib info section
func createToneLibInfo(midiFile *smf.SMF, sngFile *SngFile) ToneLibInfo {
	info := ToneLibInfo{
		ShowRemarks: "no",
	}

	if sngFile != nil {
		metadata := sngFile.GetMetadata()
		info.Name = metadata["name"]
		info.Artist = metadata["artist"]
		info.Album = metadata["album"]
		info.Author = metadata["author"]
		info.Writer = metadata["writer"]
	} else {
		// Use track 0 name as song title if no SNG metadata
		if len(midiFile.Tracks) > 0 {
			trackName := getTrackName(midiFile.Tracks[0])
			if trackName != "" {
				info.Name = trackName
			}
		}
	}

	return info
}

// createToneLibBarIndex extracts timeline and creates the bar index
func createToneLibBarIndex(midiFile *smf.SMF) (ToneLibBarIndex, *Timeline, error) {
	timeline, err := ExtractBeatTimeline(midiFile)
	if err != nil {
		// If no BEAT track, create a simple bar structure with default tempo
		return createDefaultBarIndex(midiFile), nil, nil
	}

	barIndex := createBarIndexFromTimeline(timeline)
	return barIndex, timeline, nil
}

// createBackingTrackIfNeeded creates backing track if SNG has audio files
func createBackingTrackIfNeeded(sngFile *SngFile, beatMap *BeatMap) *ToneLibBackingTrack {
	if sngFile == nil {
		return nil
	}

	// Check for any opus files in SNG
	files := sngFile.ListFiles()
	hasOpusFiles := false
	for _, filename := range files {
		if strings.HasSuffix(filename, ".opus") {
			hasOpusFiles = true
			break
		}
	}

	if !hasOpusFiles {
		return nil
	}

	// Create bars structure with beat map data if available
	bars := ToneLibBackingBars{
		Num:   "0",
		NST:   "0",
		Beats: []ToneLibBackingBeat{},
	}

	if beatMap != nil {
		bars.Num = beatMap.TotalNum
		bars.NST = beatMap.NST
		bars.Beats = beatMap.Beats
	}

	return &ToneLibBackingTrack{
		Color:    ToneLibBackingColor,
		Visible:  1,
		Collapse: 0,
		Lock:     0,
		Solo:     0,
		Mute:     0,
		Opt:      0,
		VolDB:    "0",
		Audio: ToneLibAudio{
			Name:        "merged.ogg",
			DataFile:    "", // Will be updated with actual path from conversion
			DataLen:     0,  // Will be updated with actual converted size
			TimeOffset:  "0",
			Gain:        "1",
			ChannelMode: 0,
			Bars:        bars,
		},
	}
}

// createToneLibScore creates a complete ToneLib score from MIDI and SNG data
func createToneLibScore(midiFile *smf.SMF, sngFile *SngFile, beatMap *BeatMap) *ToneLibScore {
	// Create the base score structure
	score := &ToneLibScore{}

	// 1. Extract and set metadata
	score.Info = createToneLibInfo(midiFile, sngFile)

	// 2. Create bar index and extract timeline
	barIndex, timeline, _ := createToneLibBarIndex(midiFile)
	score.BarIndex = barIndex

	// 3. Create tracks from MIDI - ensure bar count matches BarIndex
	numBars := len(score.BarIndex.Bars)
	score.Tracks = createTracksFromMidi(midiFile, numBars, timeline)

	// 4. Add backing track if needed
	score.BackingTrack = createBackingTrackIfNeeded(sngFile, beatMap)

	return score
}

// extractLyricsWithTiming extracts lyric events with timing from PART VOCALS track
func extractLyricsWithTiming(midiFile *smf.SMF) []LyricEvent {
	var lyricEvents []LyricEvent

	// Find the PART VOCALS track
	var vocalTrack smf.Track
	var found bool

	for _, track := range midiFile.Tracks {
		trackName := getTrackName(track)
		if trackName == "PART VOCALS" {
			vocalTrack = track
			found = true
			break
		}
	}

	if !found {
		return lyricEvents
	}

	// Extract lyric events with timing
	var currentTime uint32
	for _, event := range vocalTrack {
		currentTime += event.Delta
		msg := event.Message

		var lyric, text string
		if msg.GetMetaLyric(&lyric) {
			lyricEvents = append(lyricEvents, LyricEvent{
				Time:  currentTime,
				Lyric: lyric,
			})
		} else if msg.GetMetaText(&text) {
			// Skip bracketed animation markers, look for actual lyrics
			if len(text) > 0 && text[0] != '[' {
				lyricEvents = append(lyricEvents, LyricEvent{
					Time:  currentTime,
					Lyric: text,
				})
			}
		}
	}

	log.Printf("Extracted %d lyric events from PART VOCALS", len(lyricEvents))
	return lyricEvents
}

// MeasureLyrics represents lyrics grouped by measure
type MeasureLyrics struct {
	MeasureNum int    // 1-based measure number
	StartTime  uint32 // Time of first lyric in measure
	Text       string // Merged text for the measure
}

// groupLyricsByMeasure groups lyric events by measure and merges adjacent lyrics within each measure
func groupLyricsByMeasure(lyricEvents []LyricEvent, timeline *Timeline) []MeasureLyrics {
	var measureLyrics []MeasureLyrics

	if len(lyricEvents) == 0 || timeline == nil || len(timeline.Measures) == 0 {
		return measureLyrics
	}

	// Group lyrics by measure
	measureGroups := make(map[int][]LyricEvent)

	for _, lyricEvent := range lyricEvents {
		// Find which measure this lyric belongs to
		measureNum := -1
		for i, measure := range timeline.Measures {
			if lyricEvent.Time >= measure.StartTime && lyricEvent.Time < measure.EndTime {
				measureNum = i + 1 // 1-based measure numbering
				break
			}
		}

		if measureNum > 0 {
			measureGroups[measureNum] = append(measureGroups[measureNum], lyricEvent)
		}
	}

	// Process each measure's lyrics
	for measureNum := 1; measureNum <= len(timeline.Measures); measureNum++ {
		events, exists := measureGroups[measureNum]
		if !exists || len(events) == 0 {
			continue
		}

		// Sort events by time within the measure
		// (they should already be sorted, but ensure consistency)
		sort.Slice(events, func(i, j int) bool {
			return events[i].Time < events[j].Time
		})

		// Collect raw lyrics for this measure
		var rawLyrics []string
		for _, event := range events {
			if event.Lyric != "" {
				rawLyrics = append(rawLyrics, event.Lyric)
			}
		}

		if len(rawLyrics) > 0 {
			// Use existing Rock Band lyric parsing to merge and clean up lyrics
			mergedText := parseRockBandLyrics(rawLyrics)

			measureLyrics = append(measureLyrics, MeasureLyrics{
				MeasureNum: measureNum,
				StartTime:  events[0].Time,
				Text:       mergedText,
			})
		}
	}

	log.Printf("Grouped lyrics into %d measures", len(measureLyrics))
	return measureLyrics
}

// createLyricsTrack creates a ToneLib lyrics track from measure-grouped lyrics
func createLyricsTrack(measureLyrics []MeasureLyrics, midiFile *smf.SMF, numBars int, trackID int, timeline *Timeline) ToneLibTrack {
	toneLibTrack := ToneLibTrack{
		Name:     "Lyrics",
		Color:    ToneLibLyricsColor,
		Visible:  1,
		Collapse: 0,
		Lock:     0,
		Solo:     0,
		Mute:     0,
		Opt:      0,
		VolDB:    ToneLibDefaultVolDB,
		Bank:     0, // Standard bank
		Program:  1, // Acoustic piano
		Chorus:   0,
		Reverb:   0,
		Phaser:   0,
		Tremolo:  0,
		ID:       trackID,
		Offset:   ToneLibDefaultOffset,
		Strings:  createGuitarStrings(), // no notes are used here, use standard tuning
		Bars:     createLyricsBarsFromMeasures(measureLyrics, midiFile, numBars, timeline),
	}

	return toneLibTrack
}

// createLyricsBarsFromMeasures converts measure-grouped lyrics to ToneLib bars
func createLyricsBarsFromMeasures(measureLyrics []MeasureLyrics, midiFile *smf.SMF, numBars int, timeline *Timeline) ToneLibTrackBars {
	// Get ticks per quarter note for beat calculations
	ticksPerQuarter := int(480) // Default
	if tf, ok := midiFile.TimeFormat.(smf.MetricTicks); ok {
		ticksPerQuarter = int(tf)
	}
	ticksPerEighth := ticksPerQuarter / 2

	// Create a map for quick lookup of lyrics by measure number
	lyricsByMeasure := make(map[int]MeasureLyrics)
	for _, measureLyric := range measureLyrics {
		lyricsByMeasure[measureLyric.MeasureNum] = measureLyric
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

		// Add clef and key signature to first bar only
		if barID == 1 {
			bar.Clef = &ToneLibClef{Value: ToneLibTrebleClef}
			bar.KeySign = &ToneLibKeySign{Value: 0}
		}

		// Check if this measure has lyrics
		if measureLyric, hasLyrics := lyricsByMeasure[barID]; hasLyrics && measureLyric.Text != "" {
			// Calculate the correct beat position within the measure
			var beats []ToneLibBeat

			if timeline != nil && barID <= len(timeline.Measures) {
				measure := timeline.Measures[barID-1] // Convert to 0-based index

				// Calculate relative position within measure
				relativeTicks := int(measureLyric.StartTime - measure.StartTime)

				// Quantize to nearest eighth note position (0-7 for 4/4 time)
				eighthNotePosition := (relativeTicks + ticksPerEighth/2) / ticksPerEighth
				if eighthNotePosition < 0 {
					eighthNotePosition = 0
				}
				if eighthNotePosition > 7 {
					eighthNotePosition = 7
				}

				// Create beats with text at calculated position
				for i := 0; i < 8; i++ {
					if i == eighthNotePosition {
						// Text beat at the calculated position
						beats = append(beats, ToneLibBeat{
							Duration: ToneLibEighthNoteDuration,
							Dyn:      ToneLibDefaultDynamic,
							Text:     &ToneLibText{Value: measureLyric.Text},
						})
					} else {
						// Rest beat
						beats = append(beats, ToneLibBeat{
							Duration: ToneLibEighthNoteDuration,
							Dyn:      ToneLibDefaultDynamic,
						})
					}
				}
			} else {
				// Fallback: place text at beginning if no timeline info
				beats = []ToneLibBeat{
					{Duration: ToneLibQuarterNoteDuration, Dyn: ToneLibDefaultDynamic, Text: &ToneLibText{Value: measureLyric.Text}},
					{Duration: ToneLibQuarterNoteDuration, Dyn: ToneLibDefaultDynamic},
					{Duration: ToneLibQuarterNoteDuration, Dyn: ToneLibDefaultDynamic},
					{Duration: ToneLibQuarterNoteDuration, Dyn: ToneLibDefaultDynamic},
				}
			}

			bar.Beats = beats
		} else {
			// Empty measure - whole rest
			bar.Beats = []ToneLibBeat{{Duration: ToneLibWholeNoteDuration, Dyn: ToneLibDefaultDynamic}}
		}

		bars = append(bars, bar)
	}

	return ToneLibTrackBars{Bars: bars}
}
