Adapted from: <https://therogerland.tumblr.com/proguide/authoringbasics/notes> and <https://therogerland.tumblr.com/proguide/authoringbasics/chords>

# Rock Band Pro Guitar/Bass MIDI Authoring Guide

This document provides a comprehensive overview of how Pro Guitar and Pro Bass parts are authored in Rock Band MIDI files, including string mapping, fret positioning, MIDI channel encodings, technique representations, and chord authoring systems.

## Overview

Pro Guitar and Pro Bass in Rock Band represent actual guitar/bass gameplay using real fret positions and strings rather than the simplified 5-button system. The authoring system uses MIDI note numbers to represent strings and MIDI velocity values to represent fret positions, with different MIDI channels encoding various playing techniques.

## Track Structure

### Required Pro Guitar Tracks

| Track Name | Difficulty | String Range | Description |
|------------|------------|--------------|-------------|
| `PART REAL_GUITAR_X` | Expert | C6 (96) - F6 (101) | Most complex, exact transcription |
| `PART REAL_GUITAR_H` | Hard | C4 (72) - F4 (77) | Reduced complexity from Expert |
| `PART REAL_GUITAR_M` | Medium | C2 (48) - F2 (53) | Basic chord progressions |
| `PART REAL_GUITAR_E` | Easy | C0 (24) - F0 (29) | Single notes and simple chords |

### Required Pro Bass Tracks

| Track Name | Difficulty | String Range | Description |
|------------|------------|--------------|-------------|
| `PART REAL_BASS_X` | Expert | C6 (96) - D6 (98) | Most complex, exact transcription |
| `PART REAL_BASS_H` | Hard | C4 (72) - D4 (74) | Reduced complexity from Expert |
| `PART REAL_BASS_M` | Medium | C2 (48) - D2 (50) | Basic bass patterns |
| `PART REAL_BASS_E` | Easy | C0 (24) - D0 (26) | Simplified root notes |

*Note: Bass uses only 4 strings (E-A-D-G) while guitar uses 6 strings (E-A-D-G-B-E)*

## String Mapping System

### Guitar String Assignments

| MIDI Note | String | Tuning (Standard) |
|-----------|--------|-------------------|
| Base + 0 (C) | 6th String | E (Low) |
| Base + 1 (C#) | 5th String | A |
| Base + 2 (D) | 4th String | D |
| Base + 3 (D#) | 3rd String | G |
| Base + 4 (E) | 2nd String | B |
| Base + 5 (F) | 1st String | E (High) |

### Bass String Assignments

| MIDI Note | String | Tuning (Standard) |
|-----------|--------|-------------------|
| Base + 0 (C) | 4th String | E (Low) |
| Base + 1 (C#) | 3rd String | A |
| Base + 2 (D) | 2nd String | D |
| Base + 3 (D#) | 1st String | G |

### Difficulty Base Note Mapping

| Difficulty | Base MIDI Note | Guitar Range | Bass Range |
|------------|----------------|--------------|------------|
| Expert | 96 (C6) | 96-101 (C6-F6) | 96-99 (C6-D#6) |
| Hard | 72 (C4) | 72-77 (C4-F4) | 72-75 (C4-D#4) |
| Medium | 48 (C2) | 48-53 (C2-F2) | 48-51 (C2-D#2) |
| Easy | 24 (C0) | 24-29 (C0-F0) | 24-27 (C0-D#0) |

## Fret Position Encoding

### Velocity-Based Fret Mapping

| Velocity Value | Fret Position | Description |
|----------------|---------------|-------------|
| 100 | 0 | Open string |
| 101 | 1 | 1st fret |
| 102 | 2 | 2nd fret |
| 103 | 3 | 3rd fret |
| ... | ... | ... |
| 117 | 17 | 17th fret (standard track) |
| 118-122 | 18-22 | Extended frets (22-fret track) |

### Fret Range Limitations

- **17-Fret Tracks:** Velocity range 100-117 (open through 17th fret)
- **22-Fret Tracks:** Velocity range 100-122 (open through 22nd fret)
- **Open String:** Always velocity 100 regardless of track type

## MIDI Channel Encodings

### Playing Technique Channels

| MIDI Channel | Technique | Description |
|--------------|-----------|-------------|
| C1 | Normal Notes | Standard fretted notes |
| C2 | Arpeggio (Non-played) | Notes not played but part of chord shape |
| C3 | Bend Notes | Notes that require string bending |
| C4 | Muted Notes | Palm muted or string muted notes |
| C5 | Tapped Notes | Hammer-ons, pull-offs, and tapping |
| C6 | Harmonic/Pinch | Natural and artificial harmonics |

### Special Technique Channels

| MIDI Channel | Technique | Description |
|--------------|-----------|-------------|
| C12 | Reverse Slide | Slide from higher to lower fret |
| C13 | Force HOPO Off | Disable automatic hammer-on/pull-off |
| C14 | String Emphasis 1 | Accent specific string in chord |
| C15 | String Emphasis 2 | Secondary string emphasis |
| C16 | String Emphasis 3 | Tertiary string emphasis |

## Advanced Techniques

### Hammer-ons and Pull-offs (HOPOs)

- **Automatic Detection:** Game automatically detects based on note timing and pitch
- **Manual Override:** Use Channel 13 to disable automatic HOPO detection
- **Channel 5 (Tapped):** Force HOPO regardless of timing

### String Bending

- **Channel 3:** Indicates notes requiring string bending technique
- **Implementation:** Bend to reach target pitch rather than fretting directly
- **Common Usage:** Blues and rock lead guitar passages

### Muted Notes

- **Channel 4:** Palm muted or string dampened notes
- **Fret Position:** Still uses velocity to indicate fret, but muted execution
- **Common Usage:** Percussive rhythm parts, metal chugging

### Harmonics

- **Channel 6:** Natural harmonics (12th, 7th, 5th fret positions)
- **Pinch Harmonics:** Artificial harmonics using pick and thumb
- **Fret Encoding:** Velocity indicates where harmonic is produced

### Slides

- **Forward Slides:** Normal note followed by higher pitch note
- **Reverse Slides:** Channel 12 indicates slide from high to low
- **Implementation:** Start at one fret, slide to target fret

### Chord Arpeggiation

- **Channel 2:** Non-played notes that complete chord shape
- **Purpose:** Show complete chord fingering without requiring all notes
- **Technique:** Player forms full chord but only plays specific strings

## String Emphasis System

### Multi-String Chord Handling

When multiple strings are played simultaneously, emphasis channels indicate which strings to prioritize:

- **Channel 14:** Primary emphasis (most important string in chord)
- **Channel 15:** Secondary emphasis (second most important)
- **Channel 16:** Tertiary emphasis (supporting chord tone)

### Usage Guidelines

- **Complex Chords:** Use emphasis to guide player attention
- **Strumming Patterns:** Indicate which strings drive the rhythm
- **Melodic Lines:** Emphasize melody notes within chord context

## Chord Authoring System

### Chord Visual Representation

- **Chord Display:** Chords appear as blue shapes in-game interface
- **Open Notes:** Open strings are displayed as white notes
- **Fret Pattern:** Visual representation shows complete fingering pattern
- **Chord Name:** Game displays chord name based on note intervals and root

### Root Note Indicators

| MIDI Note | Note Name | Purpose |
|-----------|-----------|---------|
| 4 | E-2 | Root note indicator |
| 5 | F-2 | Root note indicator |
| 6 | F#-2 | Root note indicator |
| 7 | G-2 | Root note indicator |
| 8 | G#-2 | Root note indicator |
| 9 | A-2 | Root note indicator |
| 10 | A#-2 | Root note indicator |
| 11 | B-2 | Root note indicator |
| 12 | C-1 | Root note indicator |
| 13 | C#-1 | Root note indicator |
| 14 | D-1 | Root note indicator |
| 15 | D#-1 | Root note indicator |

### Special Chord Markers

| MIDI Note | Function | Description |
|-----------|----------|-------------|
| 16 (E-1) | Slash Chord Marker | Indicates alternate bass note (e.g., G/B) |
| 17 (F-1) | Hidden Chord Name | Suppresses chord name display |

### Chord Encoding Rules

#### Root Note System
- **Purpose:** Defines the scale and chord naming context
- **Placement:** One root note can define scale for multiple consecutive chords
- **Velocity:** Must always use velocity 100 (fixed requirement)
- **Function:** Game calculates chord names based on note intervals from root

#### Automatic Chord Naming
- **Algorithm:** Game automatically calculates note intervals
- **Root Dependency:** Chord name determined by active root note indicator
- **Interval Analysis:** Major, minor, diminished, augmented intervals recognized
- **Extensions:** 7th, 9th, 11th, 13th extensions automatically detected

#### Slash Chord Implementation
- **Marker Note:** MIDI note 16 (E-1) indicates slash chord
- **Function:** Shows alternate bass note in chord name
- **Example:** G major chord with B in bass = "G/B"
- **Velocity:** Must use velocity 100
- **Timing:** Place marker at same time as chord

#### Hidden Chord Names
- **Marker Note:** MIDI note 17 (F-1) suppresses chord name display
- **Usage:** For transitional chords or when name would be confusing
- **Velocity:** Must use velocity 100
- **Inheritance:** Uses last placed root note for internal naming

### Chord Authoring Guidelines

#### String Selection
- **Omit Muted Strings:** Do not include muted strings in chord encoding
- **Complete Muting:** Single muted string marks entire chord as muted
- **Channel 4 Usage:** Use muted note channel for palm-muted chord sections

#### Chord Shape Considerations
- **Physical Playability:** Ensure chords are physically possible to fret
- **Finger Stretches:** Consider human hand limitations for chord spans
- **Open String Integration:** Use open strings to reduce fret hand complexity
- **Barre Chord Support:** Support for barre chords across multiple strings

#### Difficulty-Specific Chord Authoring

**Expert Chords:**
- **Complexity:** Full chord voicings including extensions
- **Root Position:** Any inversion or voicing
- **Extended Chords:** 7th, 9th, 11th, 13th chords allowed
- **Jazz Voicings:** Complex jazz chord structures supported

**Hard Chords:**
- **Simplification:** Reduce complex extensions to basic triads
- **Root Position Focus:** Prefer root position chords
- **Open Chord Priority:** Use open chord shapes when possible
- **Limited Extensions:** Major 7th and dominant 7th primarily

**Medium Chords:**
- **Basic Triads:** Major, minor, and dominant 7th chords only
- **Open Chord Shapes:** Standard open chord forms
- **First Position:** Primarily first 3-5 frets
- **Common Progressions:** Focus on typical chord progressions

**Easy Chords:**
- **Power Chords:** Root and fifth intervals primarily
- **Open Chords:** Basic open major and minor chords
- **Single Notes:** Often reduce chords to single root notes
- **Simplified Rhythm:** Basic strumming patterns only

## Difficulty-Specific Authoring

### Expert Pro Guitar/Bass
**Philosophy:** Exact transcription of guitar/bass part

**Technical Specs:**
- **Fret Range:** Full 17-22 fret capability
- **Techniques:** All channels and techniques available
- **Chord Complexity:** Up to 6-string chords (guitar) or 4-string (bass)
- **Timing:** Precise rhythmic accuracy required

### Hard Pro Guitar/Bass
**Philosophy:** Simplified but recognizable version

**Technical Specs:**
- **Fret Limitation:** Generally limited to first 12 frets
- **Technique Reduction:** Fewer advanced techniques
- **Chord Simplification:** Reduce complex chord voicings
- **Open String Focus:** Prefer open chord positions

### Medium Pro Guitar/Bass
**Philosophy:** Basic chord progressions and simple patterns

**Technical Specs:**
- **Fret Limitation:** First 7 frets primarily
- **Open Chord Focus:** Standard open chord shapes
- **Technique Limitation:** Basic techniques only
- **Rhythm Simplification:** Quarter and eighth note patterns

### Easy Pro Guitar/Bass
**Philosophy:** Single notes and very basic chords

**Technical Specs:**
- **Fret Limitation:** First 5 frets maximum
- **Single Notes:** Emphasis on single-note patterns
- **Basic Chords:** Simple major/minor open chords only
- **Rhythm Focus:** Simple strumming patterns

## Conversion Considerations

### From Rock Band Format
1. **Extract string data:** Parse MIDI notes by difficulty base ranges
2. **Decode fret positions:** Convert velocity values to fret numbers
3. **Identify techniques:** Map MIDI channels to playing techniques
4. **Parse chord markers:** Extract root notes (4-15), slash markers (16), hidden markers (17)
5. **Reconstruct chord names:** Apply chord naming algorithm based on root notes
6. **Reconstruct timing:** Preserve note-on/note-off timing for sustains
7. **Difficulty separation:** Process each difficulty track independently

### To Rock Band Format
1. **String assignment:** Map guitar/bass strings to appropriate MIDI notes
2. **Fret encoding:** Convert fret positions to velocity values (100 + fret)
3. **Technique mapping:** Assign MIDI channels based on playing technique
4. **Chord analysis:** Generate root note indicators and chord markers
5. **Chord naming:** Place appropriate slash/hidden chord markers
6. **Difficulty adaptation:** Reduce complexity for lower difficulties
7. **Timing precision:** Ensure accurate note timing and sustain durations

### Critical Technical Points

- **Velocity Range:** 100-117 (standard) or 100-122 (extended) for fret positions
- **Channel Separation:** Different techniques must use correct MIDI channels
- **String Limits:** 6 strings (guitar) or 4 strings (bass) maximum
- **Open String Encoding:** Always velocity 100 regardless of tuning
- **Sustain Handling:** Note-off events determine when to release fret
- **Chord Voicing:** Consider physical playability on real instrument
- **Chord Marker Velocity:** Root notes (4-15), slash markers (16), hidden markers (17) must use velocity 100
- **Muted Chord Handling:** Single muted string makes entire chord muted
- **Chord Name Inheritance:** Hidden chord markers use last placed root note for naming

This system enables Rock Band to provide authentic guitar and bass gameplay that closely mirrors real instrument technique while maintaining game accessibility through progressive difficulty levels. The chord authoring system adds visual feedback and music theory integration to enhance the learning experience.