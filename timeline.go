package main

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"gitlab.com/gomidi/midi/v2/smf"
)

// BeatNote represents a beat event from the BEAT track
type BeatNote struct {
	Time       uint32 // Absolute time in ticks
	IsDownbeat bool   // True if this is a downbeat (C-1), false for other beats (C#-1)
}

// Measure represents a musical measure with timing information
type Measure struct {
	StartTime        uint32  // Start time in ticks
	EndTime          uint32  // End time in ticks
	BeatsPerMeasure  int     // Number of beats in this measure
	BeatsPerMinute   float64 // Original BPM from MIDI tempo events
	PreciseStartTime float64 // High-precision expected start time in ticks
	PreciseEndTime   float64 // High-precision expected end time in ticks
}

// Timeline represents the complete beat timeline of a song
type Timeline struct {
	Measures     []Measure
	BeatNotes    []BeatNote
	TicksPerBeat float64 // Derived from time signature and tempo
}

// ExtractBeatTimeline analyzes the BEAT track and creates a timeline with measure information
func ExtractBeatTimeline(smfData *smf.SMF) (*Timeline, error) {
	// Find the BEAT track
	var beatTrack smf.Track
	var found bool

	for _, track := range smfData.Tracks {
		trackName := getTrackName(track)
		if trackName == "BEAT" {
			beatTrack = track
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("BEAT track not found")
	}

	// Extract beat notes from the BEAT track
	beatNotes, err := extractBeatNotes(beatTrack)
	if err != nil {
		return nil, fmt.Errorf("failed to extract beat notes: %w", err)
	}

	if len(beatNotes) == 0 {
		return nil, fmt.Errorf("no beat notes found in BEAT track")
	}

	// Extract tempo information from all tracks
	tempoMap, err := extractTempoMap(smfData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tempo map: %w", err)
	}

	// Get ticks per quarter note for BPM calculations
	ticksPerQuarter, ok := smfData.TimeFormat.(smf.MetricTicks)
	if !ok {
		return nil, fmt.Errorf("unsupported time format, expected MetricTicks")
	}

	// Create measures from beat pattern
	measures := createMeasuresFromBeats(beatNotes, tempoMap, float64(ticksPerQuarter))

	timeline := &Timeline{
		Measures:     measures,
		BeatNotes:    beatNotes,
		TicksPerBeat: float64(ticksPerQuarter),
	}

	return timeline, nil
}

// extractBeatNotes extracts beat events from the BEAT track
func extractBeatNotes(beatTrack smf.Track) ([]BeatNote, error) {
	var beatNotes []BeatNote
	var currentTime uint32

	for _, event := range beatTrack {
		currentTime += event.Delta

		msg := event.Message
		var ch, key, vel uint8

		// Check for note on events
		if msg.GetNoteOn(&ch, &key, &vel) {
			// noteoff events encoded as note on with velocity 0
			if vel == 0 {
				continue
			}

			var isDownbeat bool
			switch key {
			case 12: // C-1 - Downbeat
				isDownbeat = true
			case 13: // C#-1 - Other beats
				isDownbeat = false
			default:
				// Skip notes that aren't beat markers
				fmt.Printf("Invalid note detected at time %d with key %d\n", currentTime, key)
				continue
			}

			beatNotes = append(beatNotes, BeatNote{
				Time:       currentTime,
				IsDownbeat: isDownbeat,
			})
		}
	}

	// Sort by time to ensure proper order
	sort.Slice(beatNotes, func(i, j int) bool {
		return beatNotes[i].Time < beatNotes[j].Time
	})

	return beatNotes, nil
}

// TempoEvent represents a tempo change in the MIDI file
type TempoEvent struct {
	Time uint32  // Absolute time in ticks
	BPM  float64 // Beats per minute
}

// extractTempoMap extracts tempo changes from all tracks in the MIDI file
func extractTempoMap(smfData *smf.SMF) ([]TempoEvent, error) {
	var tempoEvents []TempoEvent

	// Verify time format is supported
	_, ok := smfData.TimeFormat.(smf.MetricTicks)
	if !ok {
		return nil, fmt.Errorf("unsupported time format")
	}

	// Search all tracks for tempo events
	for _, track := range smfData.Tracks {
		var currentTime uint32

		for _, event := range track {
			currentTime += event.Delta

			msg := event.Message
			var bpm float64

			// Check for tempo meta events
			if msg.GetMetaTempo(&bpm) {

				tempoEvents = append(tempoEvents, TempoEvent{
					Time: currentTime,
					BPM:  bpm,
				})
			}
		}
	}

	// Sort by time
	sort.Slice(tempoEvents, func(i, j int) bool {
		return tempoEvents[i].Time < tempoEvents[j].Time
	})

	// If no tempo events found, assume 120 BPM
	if len(tempoEvents) == 0 {
		tempoEvents = append(tempoEvents, TempoEvent{
			Time: 0,
			BPM:  120.0,
		})
	}

	return tempoEvents, nil
}

// createMeasuresFromBeats creates measure objects from beat pattern and tempo information
func createMeasuresFromBeats(beatNotes []BeatNote, tempoMap []TempoEvent, ticksPerQuarter float64) []Measure {
	var measures []Measure

	if len(beatNotes) == 0 {
		return measures
	}

	// Group beats into measures by finding downbeats
	var measureStarts []int
	for i, beat := range beatNotes {
		if beat.IsDownbeat {
			measureStarts = append(measureStarts, i)
		}
	}

	// Create measures from downbeat groupings
	for i, startIdx := range measureStarts {
		var endIdx int
		if i+1 < len(measureStarts) {
			endIdx = measureStarts[i+1]
		} else {
			endIdx = len(beatNotes)
		}

		// Count beats in this measure
		beatsInMeasure := endIdx - startIdx

		startTime := beatNotes[startIdx].Time
		var endTime uint32
		if endIdx < len(beatNotes) {
			endTime = beatNotes[endIdx].Time
		} else {
			// For the last measure, estimate end time
			if beatsInMeasure > 1 {
				beatDuration := beatNotes[endIdx-1].Time - startTime
				averageBeatLength := float64(beatDuration) / float64(beatsInMeasure-1)
				endTime = startTime + uint32(float64(beatsInMeasure)*averageBeatLength)
			} else {
				endTime = startTime + uint32(ticksPerQuarter) // Assume quarter note
			}
		}

		// Find the appropriate tempo for this measure
		bpm := findBPMAtTime(startTime, tempoMap)

		measure := Measure{
			StartTime:       startTime,
			EndTime:         endTime,
			BeatsPerMeasure: beatsInMeasure,
			BeatsPerMinute:  bpm,
			// Precise timing fields will be calculated later
			PreciseStartTime: 0,
			PreciseEndTime:   0,
		}

		measures = append(measures, measure)
	}

	// Calculate precise timing
	measures = calculatePreciseMeasureTiming(measures, tempoMap, ticksPerQuarter)

	return measures
}

// findBPMAtTime finds the BPM that applies at a given time
func findBPMAtTime(time uint32, tempoMap []TempoEvent) float64 {
	bpm := 120.0 // Default BPM

	for _, tempo := range tempoMap {
		if tempo.Time <= time {
			bpm = tempo.BPM
		} else {
			break
		}
	}

	return bpm
}

// GetMeasureAtTime finds the measure that contains the given time
func (t *Timeline) GetMeasureAtTime(time uint32) *Measure {
	for i := range t.Measures {
		if time >= t.Measures[i].StartTime && time < t.Measures[i].EndTime {
			return &t.Measures[i]
		}
	}
	return nil
}

// GetTotalDuration returns the total duration of the timeline in ticks
func (t *Timeline) GetTotalDuration() uint32 {
	if len(t.Measures) == 0 {
		return 0
	}
	return t.Measures[len(t.Measures)-1].EndTime
}

// String returns a string representation of the timeline
func (t *Timeline) String() string {
	result := fmt.Sprintf("Timeline: %d measures, %d beat notes\n", len(t.Measures), len(t.BeatNotes))

	var elapsedSeconds float64 = 0.0

	for i, measure := range t.Measures {
		result += fmt.Sprintf("Measure %d: %d/%d time, %.1f BPM, ticks %d-%d\n",
			i+1,
			measure.BeatsPerMeasure,
			4, // Assuming quarter note gets the beat for simplicity
			measure.BeatsPerMinute,
			measure.StartTime,
			measure.EndTime,
		)

		// Find beats within this measure
		measureBeatIndex := 1
		for _, beat := range t.BeatNotes {
			if beat.Time >= measure.StartTime && beat.Time < measure.EndTime {
				// Calculate time offset from measure start
				ticksFromMeasureStart := beat.Time - measure.StartTime
				// Convert ticks to seconds using measure's BPM
				ticksPerSecond := t.TicksPerBeat * measure.BeatsPerMinute / 60.0
				secondsFromMeasureStart := float64(ticksFromMeasureStart) / ticksPerSecond
				beatTimeSeconds := elapsedSeconds + secondsFromMeasureStart

				result += fmt.Sprintf("  * Beat %d: %.6f\n", measureBeatIndex, beatTimeSeconds)
				measureBeatIndex++
			}
		}

		// Add measure duration to elapsed time
		measureDurationTicks := measure.EndTime - measure.StartTime
		ticksPerSecond := t.TicksPerBeat * measure.BeatsPerMinute / 60.0
		measureDurationSeconds := float64(measureDurationTicks) / ticksPerSecond
		elapsedSeconds += measureDurationSeconds
	}

	return result
}

// calculatePreciseMeasureTiming calculates high-precision measure start/end times
// based on MIDI tempo events and beat pattern
func calculatePreciseMeasureTiming(measures []Measure, tempoMap []TempoEvent, ticksPerQuarter float64) []Measure {
	if len(measures) == 0 {
		return measures
	}

	// Create a copy of measures to modify
	preciseMeasures := make([]Measure, len(measures))
	copy(preciseMeasures, measures)

	// Calculate precise timing for each measure
	var currentPreciseTime float64 = float64(measures[0].StartTime)

	for i := range preciseMeasures {
		preciseMeasures[i].PreciseStartTime = currentPreciseTime

		// Calculate expected duration of this measure in ticks based on BPM and beat count
		bpm := findBPMAtTime(uint32(currentPreciseTime), tempoMap)
		measureDurationInMinutes := float64(preciseMeasures[i].BeatsPerMeasure) / bpm
		measureDurationInTicks := measureDurationInMinutes * 60.0 * ticksPerQuarter * bpm / 60.0 // Simplifies to: measureDurationInMinutes * ticksPerQuarter * bpm

		// More precise calculation: duration = (beats_per_measure / bpm) * 60 seconds * (ticks_per_quarter * (bpm/60)) ticks per second
		measureDurationInTicks = float64(preciseMeasures[i].BeatsPerMeasure) * ticksPerQuarter * (60.0 / bpm)

		currentPreciseTime += measureDurationInTicks
		preciseMeasures[i].PreciseEndTime = currentPreciseTime
	}

	return preciseMeasures
}

// ExtractAudioBeats uses aubiotrack to detect beats from an audio file
// Returns a slice of beat timestamps in seconds
func ExtractAudioBeats(audioFilePath string) ([]float64, error) {
	// Run aubiotrack on the audio file
	cmd := exec.Command("aubiotrack", audioFilePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aubiotrack failed: %w", err)
	}

	// Parse aubiotrack output (format: one beat time per line in seconds)
	var beats []float64
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse the beat time (in seconds)
		beatTime, err := strconv.ParseFloat(line, 64)
		if err != nil {
			log.Printf("Warning: failed to parse beat time '%s': %v", line, err)
			continue
		}

		beats = append(beats, beatTime)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading aubiotrack output: %w", err)
	}

	log.Printf("Extracted %d beats from aubiotrack", len(beats))
	return beats, nil
}
