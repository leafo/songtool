Adapted from: <http://docs.c3universe.com/rbndocs/index.php?title=Pro_Keyboard_Authoring>

# Rock Band Pro Keys MIDI Authoring Guide

This document provides a comprehensive overview of how Pro Keys (Pro Keyboard) parts are authored in Rock Band MIDI files, including track structure, special metadata notes, and difficulty-specific requirements.

## Overview

Pro Keys in Rock Band represents actual keyboard gameplay using real piano keys rather than the simplified 5-button system. The authoring philosophy is to create an exact transcription of the right-hand keyboard part, then adapt it to fit within the game's constraints using lane shifts, wrapping, and voicing adjustments.

## Track Structure

### Required Pro Keys Tracks

| Track Name | Difficulty | MIDI Note Range | Description |
|------------|------------|----------------|-------------|
| `PART REAL_KEYS_X` | Expert | C2 (48) - C4 (72) | Most complex, exact transcription |
| `PART REAL_KEYS_H` | Hard | C2 (48) - C4 (72) | Reduced complexity from Expert |
| `PART REAL_KEYS_M` | Medium | C2 (48) - C4 (72) | Basic rhythmic core |
| `PART REAL_KEYS_E` | Easy | C2 (48) - C4 (72) | Single notes only |

### Animation Tracks

| Track Name | Description | MIDI Note Range |
|------------|-------------|----------------|
| `PART REAL_KEYS_ANIM_RH` | Right hand animation | C2 (48) - C4 (72) |
| `PART REAL_KEYS_ANIM_LH` | Left hand animation | C2 (48) - C4 (72) |

*Note: Animation tracks control on-screen character hand movements and don't affect gameplay.*

## Lane Shift System

### Range Display Limitation
- Pro Keys track displays only a 10th (16 semitones) at a time
- Example: C2-E3 shows 16 keys on screen
- Lane shifts are required to navigate between different octave ranges

### Lane Shift Markers (Metadata Notes)

| MIDI Note | Range Displayed | Common Usage |
|-----------|----------------|--------------|
| `0` | C2-E3 | Primary range (recommended) |
| `2` | D2-F3 | Intermediate |
| `4` | E2-G3 | Intermediate |
| `5` | F2-A3 | Primary range (recommended) |
| `7` | G2-B3 | Intermediate |
| `9` | A2-C4 | Primary range (recommended) |

### Lane Shift Rules
- **Only white notes** can be used for lane shifts (no black keys)
- **Required:** Each difficulty must have one range marker at song beginning
- **Timing:** Place shift approximately one bar before first out-of-range note
- **Placement:** Put shifts on shared notes between current and target ranges
- **Frequency:** Minimize shifts; avoid shifting for single notes
- **Black keys forbidden:** Never use MIDI notes 1, 3, 6, 8, 10 for lane shifts

### Lane Shift Restrictions by Difficulty
- **Expert/Hard:** Lane shifts allowed and encouraged
- **Medium/Easy:** Lane shifts **NOT ALLOWED** - must fit within chosen 10th range

## Special Metadata Notes

### Solo Sections
- **MIDI Note:** `G7 (115)`
- **Purpose:** Marks keyboard solo sections for scoring multiplier
- **Timing:** Start at first note (note-on), end at last note (note-off)
- **Usage:** Only for obvious solos, avoid short fills

### Glissando Lanes
- **MIDI Note:** `F#8 (126)`
- **Difficulty:** Expert only
- **Purpose:** Creates "free play" zones for glissandos
- **Function:** Disables precise scoring - player hits any keys within marker
- **Requirements:** 
  - Quarter note minimum length
  - White notes only, evenly spaced
  - Usually excludes first note to encourage actual playing

### Trill Markers
- **MIDI Note:** `G8 (127)`
- **Purpose:** Enables alternating two-note patterns
- **Mechanics:** 
  - Player alternates between two notes at 160ms or faster
  - Falls back to exact rhythm if too slow
  - **Limitation:** Only works for two-note trills (three+ breaks game)

### Overdrive Sections
- Must be placed in both 5-lane (`PART KEYS`) and Expert Pro Keys (`PART REAL_KEYS_X`)
- Sections must match exactly between tracks
- Standard overdrive authoring rules apply

## Difficulty-Specific Requirements

### Expert Pro Keys (`PART REAL_KEYS_X`)
**Philosophy:** Exact melodic, harmonic, and rhythmic transcription of right-hand part

**Technical Specs:**
- **Chord limit:** Up to 4-note chords
- **Chord span:** Maximum one octave (12 semitones)
- **Sustain spacing:** 
  - Simple transitions: 1/16th note gap
  - Complex transitions: 1/8th note gap
- **Lane shifts:** Allowed but minimize short shifts
- **Left hand:** Can include if playable with one hand

### Hard Pro Keys (`PART REAL_KEYS_H`)
**Philosophy:** Reasonable reduction of Expert part

**Technical Specs:**
- **Chord limit:** 2-3 note chords only
- **Chord span:** Maximum 7th interval (11 semitones)
- **Interval jumps:** Avoid jumps larger than 7th
- **Sustains:** Same as Expert unless reduced for spacing
- **Lane shifts:** Allowed, remove unnecessary shifts from Expert
- **Cleanup:** Remove text events, solo/glissando/trill markers from Expert

### Medium Pro Keys (`PART REAL_KEYS_M`)
**Philosophy:** Basic rhythmic core with mandatory spacing

**Technical Specs:**
- **Spacing requirement:** 1/4 note between all gems
- **Chord limit:** 2-note chords only
- **Chord span:** Maximum 6th interval (9 semitones)
- **Sustains:** Pull back to leave 1/4 note gaps (minimum 3/16ths duration)
- **Interval jumps:** Avoid larger than 6th
- **Lane shifts:** **FORBIDDEN** - must fit in chosen range
- **Chord reduction:** If impossible to fit, reduce to single prominent note

### Easy Pro Keys (`PART REAL_KEYS_E`)
**Philosophy:** Simplified single-note melody

**Technical Specs:**
- **Spacing requirement:** Half note between gems
- **Chords:** **FORBIDDEN** - single notes only
- **Sustains:** Same as Medium unless reduced
- **Interval jumps:** Avoid larger than 5th (7 semitones)
- **Lane shifts:** **FORBIDDEN** - must match Medium's range
- **Note selection:** Most prominent and musically sensible notes

## Advanced Techniques

### Wrapping
- **Purpose:** Fit long melodic lines within range constraints
- **Method:** Similar to 5-button guitar wrapping
- **Rules:** Wrap at rhythmically and melodically logical points
- **Consideration:** Observe interval jumping rules for each difficulty

### Overlapping Gems (Broken Chords)
**Allowed overlaps by difficulty:**
- **Expert/Hard Pro Keys:** Up to 4 overlapping notes
- **Medium Pro Keys:** Generally none (rare exceptions: 2 notes, 1/4+ spacing)
- **Easy Pro Keys:** None allowed

**Technical limits:**
- **Maximum span:** One octave
- **Note limit:** 4 simultaneous notes maximum
- **Duration:** First note must end before adding note outside octave range

### Octave Displacement
- **When:** To avoid excessive lane shifts
- **Method:** Move notes/chords down an octave to fit current range
- **Rule:** Must still feel natural to play, avoid giant jumps
- **Alternative:** Change chord voicings if only top notes are out of range

## Animation Track Authoring

### Character Animation Range
- **Limitation:** Character has only 3 octaves on screen keyboard
- **Shared octave:** C2-C3 (RH) overlaps with C3-C4 (LH)
- **Solution:** Cheat RH higher, LH lower to avoid visual conflicts

### Animation Track Content
- **Right Hand (`PART REAL_KEYS_ANIM_RH`):** Copy from Expert Pro Keys
- **Left Hand (`PART REAL_KEYS_ANIM_LH`):** Broad strokes, rhythm-aligned
- **Workflow:** Transcribe full keyboard part initially for both gameplay and animation

## Conversion Considerations

### From Rock Band Format
1. **Extract gameplay notes:** C2-C4 range from difficulty tracks
2. **Parse lane shifts:** MIDI notes 0, 2, 4, 5, 7, 9 indicate range changes
3. **Identify special sections:** 
   - Solos: MIDI note 115
   - Glissandos: MIDI note 126
   - Trills: MIDI note 127
4. **Difficulty mapping:** Separate tracks for each difficulty level
5. **Animation data:** Extract from ANIM_RH/ANIM_LH tracks

### To Rock Band Format
1. **Difficulty reduction:** Start with Expert transcription, reduce complexity
2. **Lane shift planning:** Identify range changes, place markers appropriately
3. **Chord limitations:** Enforce difficulty-specific chord and interval rules
4. **Spacing requirements:** Apply minimum note spacing for Medium/Easy
5. **Special sections:** Add metadata notes for solos, glissandos, trills
6. **Animation creation:** Generate hand movement data for visual display

### Critical Technical Points
- **Note range:** All gameplay limited to C2 (48) - C4 (72)
- **Lane shift timing:** Critical for playability - must anticipate range changes
- **Metadata separation:** Special markers only on Expert difficulty
- **Track coordination:** Overdrive must match between 5-lane and Pro Keys
- **Difficulty cascade:** Each difficulty reduces complexity from the level above

This system allows Rock Band to provide realistic piano gameplay while managing the complexity through progressive difficulty levels and intelligent range management.