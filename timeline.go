package main

import (
	"fmt"
	"math"
	"sort"

	"gitlab.com/gomidi/midi/v2/smf"
)

// BeatNote represents a beat event from the BEAT track
type BeatNote struct {
	Time        uint32  `json:"time"`         // Absolute time in ticks
	TimeSeconds float64 `json:"time_seconds"` // Absolute time in seconds
	IsDownbeat  bool    `json:"is_downbeat"`  // True if this is a downbeat (C-1), false for other beats (C#-1)
}

// Measure represents a musical measure with timing information
type Measure struct {
	StartTime        uint32     `json:"start_time"`         // Start time in ticks
	EndTime          uint32     `json:"end_time"`           // End time in ticks
	StartTimeSeconds float64    `json:"start_time_seconds"` // Start time in seconds
	EndTimeSeconds   float64    `json:"end_time_seconds"`   // End time in seconds
	BeatsPerMeasure  int        `json:"beats_per_measure"`  // Number of beats in this measure
	BeatsPerMinute   float64    `json:"beats_per_minute"`   // Original BPM from MIDI tempo events
	BeatNotes        []BeatNote `json:"beat_notes"`         // Beat notes contained in this measure
}

// Timeline represents the complete beat timeline of a song
type Timeline struct {
	Measures     []Measure  `json:"measures"`
	BeatNotes    []BeatNote `json:"beat_notes"`
	TicksPerBeat float64    `json:"ticks_per_beat"` // Derived from time signature and tempo
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

	// Extract beat notes with accurate timing from all tracks
	beatNotes, err := extractBeatNotesWithTiming(smfData, beatTrack)
	if err != nil {
		return nil, fmt.Errorf("failed to extract beat notes: %w", err)
	}

	if len(beatNotes) == 0 {
		return nil, fmt.Errorf("no beat notes found in BEAT track")
	}

	// Get ticks per quarter note for BPM calculations
	ticksPerQuarter, ok := smfData.TimeFormat.(smf.MetricTicks)
	if !ok {
		return nil, fmt.Errorf("unsupported time format, expected MetricTicks")
	}

	// Create measures from beat pattern
	measures := createMeasuresFromBeats(beatNotes)

	timeline := &Timeline{
		Measures:     measures,
		BeatNotes:    beatNotes,
		TicksPerBeat: float64(ticksPerQuarter),
	}

	return timeline, nil
}

// extractBeatNotesWithTiming processes all MIDI events chronologically to extract beats with accurate timing
func extractBeatNotesWithTiming(smfData *smf.SMF, beatTrack smf.Track) ([]BeatNote, error) {
	// Get ticks per quarter note
	ticksPerQuarter, ok := smfData.TimeFormat.(smf.MetricTicks)
	if !ok {
		return nil, fmt.Errorf("unsupported time format, expected MetricTicks")
	}

	// Create a unified event stream with all events from all tracks
	type TimedEvent struct {
		Time    uint32
		Message smf.Message
		IsBeat  bool
		Key     uint8
	}

	var allEvents []TimedEvent

	// Process all tracks to collect tempo events and beat events
	for _, track := range smfData.Tracks {
		var currentTime uint32
		trackName := getTrackName(track)
		isBeatTrack := (trackName == "BEAT")

		for _, event := range track {
			currentTime += event.Delta

			// Add tempo events from any track
			var bpm float64
			if event.Message.GetMetaTempo(&bpm) {
				allEvents = append(allEvents, TimedEvent{
					Time:    currentTime,
					Message: event.Message,
					IsBeat:  false,
				})
			}

			// Add beat events only from BEAT track
			if isBeatTrack {
				var ch, key, vel uint8
				if event.Message.GetNoteOn(&ch, &key, &vel) && vel > 0 {
					if key == 12 || key == 13 { // C-1 or C#-1
						allEvents = append(allEvents, TimedEvent{
							Time:    currentTime,
							Message: event.Message,
							IsBeat:  true,
							Key:     key,
						})
					} else {
						// Warning for unexpected notes in beat track
						fmt.Printf("Warning: Unexpected note detected in BEAT track at time %d with key %d\n", currentTime, key)
					}
				}
			}
		}
	}

	// Sort all events by time
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Time < allEvents[j].Time
	})

	// Process events chronologically to build beat notes with accurate timing
	var beatNotes []BeatNote
	var currentSeconds float64 = 0.0
	var lastTick uint32 = 0
	var currentBPM float64 = 120.0 // Default BPM
	var hasTempoEvents bool = false
	var usedDefaultTempo bool = false

	for _, event := range allEvents {
		// Calculate time elapsed since last event
		ticksElapsed := event.Time - lastTick
		if ticksElapsed > 0 {
			// Check if we're using default tempo
			if !hasTempoEvents && currentBPM == 120.0 {
				usedDefaultTempo = true
			}
			// Convert ticks to seconds using current BPM
			ticksPerSecond := float64(ticksPerQuarter) * currentBPM / 60.0
			secondsElapsed := float64(ticksElapsed) / ticksPerSecond
			currentSeconds += secondsElapsed
		}

		// Update BPM if this is a tempo event
		var bpm float64
		if event.Message.GetMetaTempo(&bpm) {
			currentBPM = bpm
			hasTempoEvents = true
		}

		// Record beat event if this is a beat
		if event.IsBeat {
			var isDownbeat bool
			switch event.Key {
			case 12: // C-1 - Downbeat
				isDownbeat = true
			case 13: // C#-1 - Other beats
				isDownbeat = false
			}

			beatNotes = append(beatNotes, BeatNote{
				Time:        event.Time,
				TimeSeconds: currentSeconds,
				IsDownbeat:  isDownbeat,
			})
		}

		lastTick = event.Time
	}

	// Warn if we used default tempo
	if usedDefaultTempo {
		fmt.Printf("Warning: No tempo events found, using default 120 BPM for timing calculations\n")
	}

	return beatNotes, nil
}

// createMeasuresFromBeats creates measure objects from beat pattern
func createMeasuresFromBeats(beatNotes []BeatNote) []Measure {
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

		// Extract beats for this measure
		measureBeats := beatNotes[startIdx:endIdx]
		beatsInMeasure := len(measureBeats)

		// Get timing from first and last beat
		startTime := measureBeats[0].Time
		startTimeSeconds := measureBeats[0].TimeSeconds

		var endTime uint32
		var endTimeSeconds float64
		if endIdx < len(beatNotes) {
			// Use the start of the next measure as our end time
			endTime = beatNotes[endIdx].Time
			endTimeSeconds = beatNotes[endIdx].TimeSeconds
		} else {
			// For the last measure, estimate end time based on last beat
			lastBeat := measureBeats[len(measureBeats)-1]
			if beatsInMeasure > 1 {
				// Calculate average beat duration and project forward
				measureDuration := lastBeat.TimeSeconds - startTimeSeconds
				averageBeatDuration := measureDuration / float64(beatsInMeasure-1)
				endTimeSeconds = lastBeat.TimeSeconds + averageBeatDuration

				ticksDuration := lastBeat.Time - startTime
				averageTicksPerBeat := float64(ticksDuration) / float64(beatsInMeasure-1)
				endTime = lastBeat.Time + uint32(averageTicksPerBeat)
			} else {
				// Single beat measure - add a reasonable duration
				endTimeSeconds = lastBeat.TimeSeconds + 1.0 // 1 second default
				endTime = lastBeat.Time + 480               // Assume 480 ticks (quarter note at common resolution)
			}
		}

		// Calculate BPM from measure duration
		measureDurationSeconds := endTimeSeconds - startTimeSeconds
		bpm := float64(beatsInMeasure) * 60.0 / measureDurationSeconds

		measure := Measure{
			StartTime:        startTime,
			EndTime:          endTime,
			StartTimeSeconds: startTimeSeconds,
			EndTimeSeconds:   endTimeSeconds,
			BeatsPerMeasure:  beatsInMeasure,
			BeatsPerMinute:   bpm,
			BeatNotes:        measureBeats,
		}

		measures = append(measures, measure)
	}

	return measures
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

	for i, measure := range t.Measures {
		result += fmt.Sprintf("Measure %d: %d/%d time, %.1f BPM, ticks %d-%d, %.3fs-%.3fs\n",
			i+1,
			measure.BeatsPerMeasure,
			4, // Assuming quarter note gets the beat for simplicity
			measure.BeatsPerMinute,
			measure.StartTime,
			measure.EndTime,
			measure.StartTimeSeconds,
			measure.EndTimeSeconds,
		)

		// Print beats from this measure's BeatNotes
		for j, beat := range measure.BeatNotes {
			result += fmt.Sprintf("  * Beat %d: %.6f\n", j+1, beat.TimeSeconds)
		}
	}

	return result
}

// QuantizeBPMs takes a timeline with floating-point BPMs and returns a new timeline
// with integer BPMs selected to minimize cumulative timing drift
func QuantizeBPMs(timeline *Timeline) *Timeline {
	if len(timeline.Measures) == 0 {
		return timeline
	}

	quantizedTimeline := &Timeline{
		BeatNotes:    timeline.BeatNotes, // Keep original beat notes unchanged
		TicksPerBeat: timeline.TicksPerBeat,
	}

	quantizedMeasures := make([]Measure, len(timeline.Measures))
	quantizedCurrentTime := 0.0 // Track cumulative time with quantized BPMs

	for i, measure := range timeline.Measures {
		// Copy the original measure
		quantizedMeasures[i] = measure

		originalBPM := measure.BeatsPerMinute

		// Search range: try BPMs around the original value
		searchRange := 2                  // Try ±2 BPM from the rounded value
		baseBPM := int(originalBPM + 0.5) // Start with simple rounding

		bestBPM := -1
		bestDrift := math.Inf(1)

		// Search for better BPM values
		for testBPM := baseBPM - searchRange; testBPM <= baseBPM+searchRange; testBPM++ {
			if testBPM < 1 { // Ensure BPM is positive
				continue
			}

			drift := calculateDrift(testBPM, quantizedCurrentTime, measure)

			if drift < bestDrift {
				bestDrift = drift
				bestBPM = testBPM
			}
		}

		if bestBPM == -1 {
			panic("Failed to calculate best fit bpm")
		}

		// Update the measure with quantized BPM
		quantizedMeasures[i].BeatsPerMinute = float64(bestBPM)

		quantizedMeasureDuration := float64(measure.BeatsPerMeasure) * 60.0 / float64(bestBPM)

		// Update timing information for quantized timeline
		quantizedMeasures[i].StartTimeSeconds = quantizedCurrentTime
		quantizedMeasures[i].EndTimeSeconds = quantizedCurrentTime + quantizedMeasureDuration

		// Update quantized current time for next iteration
		quantizedCurrentTime += quantizedMeasureDuration
	}

	quantizedTimeline.Measures = quantizedMeasures
	return quantizedTimeline
}

// calculateDrift returns the absolute difference of end time when using a particular BPM for a measure
func calculateDrift(bpm int, currentTime float64, targetMeasure Measure) float64 {
	duration := float64(targetMeasure.BeatsPerMeasure) * 60.0 / float64(bpm)
	endTime := currentTime + duration

	return abs(targetMeasure.EndTimeSeconds - endTime)
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
