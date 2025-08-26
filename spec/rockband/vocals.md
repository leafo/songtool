Adapted from: <http://docs.c3universe.com/rbndocs/index.php?title=Vocal_Authoring>

# Rock Band MIDI Vocal Encoding Guide

This document summarizes how vocals are encoded in Rock Band MIDI files, based on the official RBN/C3 documentation.

## Overview

Rock Band vocal charts represent simplified versions of the original vocal performance, focusing on playability rather than exact reproduction. Each MIDI note in the vocal part must have a corresponding lyric event placed exactly at the start of the note.

## MIDI Note Ranges

### Pitched Vocals
- **Range**: C1 (MIDI note 36) to C5 (MIDI note 84)
  - Note: Avoid note 84 as it doesn't display correctly in Blitz
- **Recommendation**: Author in the same octave as the original vocal for clarity and consistency

### Non-Pitched Vocals
- Use the same note range but mark with special lyric symbols (see Lyric Formatting section)

### Percussion
- **Playable percussion**: C6 (MIDI note 96)
- **Non-playable percussion samples**: C#6 (MIDI note 97)
- Types: tambourine, cowbell, hand clap (only one type per song)

### Phrase Markers
- **Location**: A6 (MIDI note 105)
- **Minimum length**: Quarter-note
- **Overdrive phrases**: Copy phrase marker to G#7 (MIDI note 116)

## Lyric Formatting and Syntax

### Basic Rules
- Each MIDI note requires exactly one lyric event
- One syllable per lyric event
- Capitalize first syllable of every phrase and proper nouns
- Only use question marks and exclamation marks for punctuation

### Multi-Syllable Words
Break words with dashes (`-`):
```
Hello → Hel- lo
Thunderstruck → Thun- der- struck
```

### Syllables with Multiple Notes (Slides/Bends)
Use plus signs (`+`) for additional notes on the same syllable:
```
Yeah (over 2 notes) → Yeah +
Thunderstruck (2 notes on "der") → Thun- der- + struck
```

### Special Markers

#### Hyphens in Lyrics
Use equals sign (`=`) to display actual hyphens:
```
Ex-Girlfriend → Ex= Girl- friend
```

#### Non-Pitched Syllables
- **Standard scoring**: `#` at end of syllable
  ```
  All right! → All# right!#
  ```
- **Generous scoring**: `^` at end of syllable (use for short phrases or soft consonants)

#### Multi-Syllable Non-Pitched Words
Place hyphen before the `#` or `^`:
```
indefatigably → in-# de-# fa-# ti-# ga-# bly#
cowardice → cow-# ard-^ ice#
```

#### Range Dividers
Use `%` at the end of a phrase's last lyric to separate vocal ranges (static HUD only):
```
Last word of phrase% → Creates range separation
```

## Track Structure

### Main Vocal Track: `PART VOCALS`
- Contains the primary vocal melody
- Include both pitched and non-pitched sections
- For songs with multiple singers, choose the part a listener would sing along with

### Harmony Tracks
- `HARM1`, `HARM2`, `HARM3` for additional vocal parts
- Each harmony track follows the same formatting rules as main vocals

## Authoring Guidelines

### Note Placement
- **Grid precision**: Use 1/64 grid (1/128 below 90 BPM)
- **Note timing**: Focus on accurate note-on timing; note-off is less critical
- **Consonant handling**: Generally exclude consonants from note tubes, include only vowels for pitch content
- **Note spacing**: All vocal notes need space between end of one note and start of next

### Phrase Construction
- **Length**: Approximately one breath for average player (~2 measures at mid-tempo)
- **Phrase markers**: Must begin on/before first note, end on/after last note
- **Gap for overdrive**: 600ms+ gap between phrases automatically creates overdrive deploy section
- **Screen limits**: Test on 4:3 displays to ensure lyrics fit on screen

### Special Situations
- **Overlapping vocals**: Choose most prominent part when vocals overlap
- **Vibrato/ornaments**: Generally omit minute details for playability
- **Phrase gaps**: Leave 16th note space (four 64th notes) between pitched phrase end and non-pitched phrase start

## Animation Markers (Text Events)

Animation markers control character behavior during performance:

| Text Event | Animation |
|------------|-----------|
| `[play]` | Standard singing state |
| `[intense]` | Hard, fast sections |
| `[mellow]` | Slow, quiet sections |
| `[idle]` | Mic down, dancing to beat |
| `[idle_intense]` | Intense idle animation |
| `[idle_realtime]` | Mic down, not synced to beat |

### Percussion Animation Markers
- `[tambourine_start]`/`[tambourine_end]`
- `[cowbell_start]`/`[cowbell_end]`
- `[clap_start]`/`[clap_end]`

## Technical Considerations

### File Alignment
- Ensure vocal stem and dry vocal file are aligned within 30-40ms
- Use dry vocal file for authoring reference when wet file has heavy effects

### Pitch Detection System
- Different difficulties change pitch detection leniency, not note content
- Vocal authoring is not quantized to song beat
- System automatically determines vocal range from highest/lowest notes

### Overdrive System
- Attached to specific phrases via G#7 marker
- Player earns energy by achieving "AWESOME" score on overdrive phrases
- Overdrive and phrase markers must align exactly

## Common Contractions
Preferred hyphenation for multi-syllable contractions:
- `It- 'd`, `It- 'll`
- `Must- 've`, `Should- 've`, `Would- 've`, `Could- 've`
- `Should- n't`, `Would- n't`, `Could- n't`, `Must- n't`

## Export Considerations

When converting Rock Band vocal data to other formats:
1. **Pitch data**: Extract from MIDI note numbers (C1-C5 range)
2. **Timing**: Use note-on times for syllable starts
3. **Lyrics**: Parse text events, removing formatting markers
4. **Slides**: Identify `+` markers for connected notes
5. **Non-pitched sections**: Identify `#` and `^` markers
6. **Phrase boundaries**: Extract from A6 note markers
7. **Animation cues**: Parse bracketed text events for performance context

This encoding system allows Rock Band to provide real-time pitch feedback, lyric display, and character animation synchronized to the vocal performance.
