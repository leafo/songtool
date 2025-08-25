package main

import (
	"strings"

	"gitlab.com/gomidi/midi/v2/smf"
)

// parseRockBandLyrics processes Rock Band MIDI vocal lyric events and converts them
// into readable text by handling the special formatting used in Rock Band vocal charts.
//
// This function implements the lyric parsing rules documented in rockband-format/vocals.md,
// which describes the Rock Band MIDI vocal encoding system used for pitch detection,
// syllable timing, and character animation.
//
// Key formatting rules handled:
// - Multi-syllable words: "Hel- lo" → "Hello"
// - Slide notes (multiple notes per syllable): "Yeah +" → "Yeah"
// - Non-pitched markers: "All#" or "All^" → "All"
// - Range dividers: "word%" → "word"
// - Actual hyphens in lyrics: "Ex= Girl- friend" → "Ex-Girlfriend"
//
// See rockband-format/vocals.md for complete specification.
func parseRockBandLyrics(rawLyrics []string) string {
	var result []string
	var currentWord strings.Builder

	for _, lyric := range rawLyrics {
		if lyric == "" {
			continue
		}

		// Skip if it's just a "+" (syllable continuation marker)
		if lyric == "+" {
			continue
		}

		// Clean up the lyric text
		cleaned := lyric

		// Remove non-pitched markers (#, ^) and range dividers (%)
		cleaned = strings.TrimSuffix(cleaned, "#")
		cleaned = strings.TrimSuffix(cleaned, "^")
		cleaned = strings.TrimSuffix(cleaned, "%")

		// Handle actual hyphens (= becomes -)
		cleaned = strings.ReplaceAll(cleaned, "=", "-")

		// Check if this syllable continues with "+"
		isSlideNote := strings.HasSuffix(cleaned, "+")
		if isSlideNote {
			cleaned = strings.TrimSuffix(cleaned, "+")
			cleaned = strings.TrimSpace(cleaned)
		}

		// Check if this is a syllable continuation (starts with hyphen after cleaning markers)
		isSyllableContinuation := strings.HasSuffix(cleaned, "-")
		if isSyllableContinuation {
			cleaned = strings.TrimSuffix(cleaned, "-")
			cleaned = strings.TrimSpace(cleaned)
		}

		// Add to current word
		currentWord.WriteString(cleaned)

		// If this syllable doesn't continue to next (no trailing hyphen), complete the word
		if !isSyllableContinuation && !isSlideNote {
			word := currentWord.String()
			if word != "" {
				result = append(result, word)
			}
			currentWord.Reset()
		}
	}

	// Handle any remaining word
	if currentWord.Len() > 0 {
		word := currentWord.String()
		if word != "" {
			result = append(result, word)
		}
	}

	return strings.Join(result, " ")
}

// extractLyrics extracts all lyric events from an SMF track and joins them into a single string.
// It looks for both MetaLyric events and MetaText events (excluding bracketed animation markers),
// then processes them through parseRockBandLyrics to handle Rock Band vocal formatting.
func extractLyrics(track smf.Track) string {
	var lyrics []string

	for _, event := range track {
		msg := event.Message

		var lyric, text string
		if msg.GetMetaLyric(&lyric) {
			lyrics = append(lyrics, lyric)
		} else if msg.GetMetaText(&text) {
			// Skip bracketed animation markers, look for actual lyrics
			if len(text) > 0 && text[0] != '[' {
				lyrics = append(lyrics, text)
			}
		}
	}

	return parseRockBandLyrics(lyrics)
}