Adapted from: <http://docs.c3universe.com/rbndocs/index.php?title=Mix_and_MIDI_Setup>

# Rock Band MIDI File Structure and Setup Guide

This document provides a comprehensive overview of Rock Band MIDI file structure, track organization, and technical requirements for conversion to other software formats.

## File Overview

Rock Band MIDI files are Standard MIDI Format (SMF) files that contain:
- Instrument chart data (notes and difficulties)
- Tempo mapping and time signatures
- Animation and visual cues
- Camera and lighting information
- Game flow control events

## Required Track Structure

### Minimum Track Requirements

| Track Name | Required Text Events | Required MIDI Notes | Description |
|------------|---------------------|-------------------|-------------|
| `PART DRUMS` | Drum mix events 0-3 | 1 gem (all difficulties) | Drum kit gameplay |
| `PART BASS` | None | 1 gem (all difficulties) | Bass guitar gameplay |
| `PART GUITAR` | None | 1 gem (all difficulties) | Lead guitar gameplay |
| `PART KEYS` | None | 1 gem (all difficulties) | Keyboard gameplay |
| `PART REAL_KEYS_X` | None | 1 gem | Pro Keys Expert |
| `PART REAL_KEYS_H` | None | 1 gem | Pro Keys Hard |
| `PART REAL_KEYS_M` | None | 1 gem | Pro Keys Medium |
| `PART REAL_KEYS_E` | None | 1 gem | Pro Keys Easy |
| `PART REAL_KEYS_ANIM_RH` | None | None | Pro Keys Right Hand Animation |
| `PART REAL_KEYS_ANIM_LH` | None | None | Pro Keys Left Hand Animation |
| `PART VOCALS` | 1 lyric (aligned with note) | 1 Note tube, 1 Phrase Marker | Lead vocals |
| `HARM1` | 1 lyric (aligned with note) | 1 Note tube, 1 Phrase Marker | Harmony 1 |
| `HARM2` | None | None | Harmony 2 |
| `HARM3` | None | None | Harmony 3 |
| `EVENTS` | `[music_start]`, `[music_end]`, `[end]` | None | Game flow control |
| `BEAT` | None | C-1 Downbeat, C#-1 Other Beats | Character animation timing |
| `VENUE` | None | None | Camera and lighting |

*Note: At least one instrument track (or all KEYS tracks) is required for a valid file.*

## Track Details

### Instrument Tracks

**Standard Instrument Tracks:**
- `PART DRUMS`, `PART BASS`, `PART GUITAR`, `PART KEYS`
- Support multiple difficulty levels within the same track
- Use standardized MIDI note ranges for different instruments

**Pro Keys Tracks:**
- `PART REAL_KEYS_X` (Expert), `PART REAL_KEYS_H` (Hard), `PART REAL_KEYS_M` (Medium), `PART REAL_KEYS_E` (Easy)
- Separate tracks for each difficulty level
- Animation tracks for left/right hand movements

### Vocal Tracks

**Main Vocals:** `PART VOCALS`
- Contains lead vocal melody
- Each MIDI note requires corresponding lyric event
- Uses phrase markers for scoring sections

**Harmony Vocals:** `HARM1`, `HARM2`, `HARM3`
- Additional vocal parts for multi-part harmonies
- Follow same structure as main vocals
- Not all harmony tracks required

### Control Tracks

**EVENTS Track:**
- Contains only text events (no MIDI notes)
- Controls game flow and crowd behavior
- Key events:
  - `[music_start]` - Transitions crowd from intro to play state
  - `[music_end]` - Transitions crowd to outro state  
  - `[end]` - Stops playback and calculates scores (MUST be last MIDI item)
  - `[coda]` - Marks beginning of Big Rock Ending

**BEAT Track:**
- Contains timing for character animations
- Uses specific MIDI notes:
  - `C-1 (12)` - Downbeat (beat 1)
  - `C#-1 (13)` - All other beats
- Last note must occur one beat before `[end]` event
- Pattern example (4/4): C-1, C#-1, C#-1, C#-1 per measure

**VENUE Track:**
- Controls cameras and lighting
- Can be left empty for auto-generation
- Requires practice sections in EVENTS track for best auto-generation

## Tempo and Time Signature Handling

### Tempo Mapping
- Essential for accurate gameplay timing
- One tempo marker per measure recommended (maximum 1 per beat)
- Must be extremely accurate as it affects all gameplay
- Count-in tempo should match first measure tempo

### Time Signature Changes
- Supported throughout the song
- BEAT track must be adjusted for meter changes
- Affects character animations and crowd timing

## Text Events and Animation Markers

### Crowd Control Events (EVENTS Track)
| Text Event | Effect |
|------------|--------|
| `[crowd_realtime]` | Non-beat-based animations (default) |
| `[crowd_intense]` | Maximum intensity crowd reactions |
| `[crowd_normal]` | Moderate crowd energy |
| `[crowd_mellow]` | Low-key crowd swaying |
| `[crowd_clap]` | Enables clapping sound effects (default) |
| `[crowd_noclap]` | Disables clapping sound effects |

### Practice Mode Timekeeping (EVENTS Track)
When stems are available, these MIDI notes provide click track in practice mode:
- `D0 (26)` - Hi-hat sample
- `C#0 (25)` - Snare sample  
- `C0 (24)` - Kick drum sample

## Audio Stem Integration

### Required Audio Files
| Stem File | Content | Format |
|-----------|---------|--------|
| `K` | Kick Drum | Mono, 16-bit, 44.1kHz |
| `SN` | Snare Drum | Stereo, 16-bit, 44.1kHz |
| `CYM` | Kit Mix (overheads + toms) | Stereo, 16-bit, 44.1kHz |
| `BASS` | Playable Bass Part | Mono/Stereo, 16-bit, 44.1kHz |
| `GTR` | Playable Guitar Part | Stereo, 16-bit, 44.1kHz |
| `KEYS` | Playable Keyboard Part | Stereo, 16-bit, 44.1kHz |
| `VOX` | Playable Vocals | Stereo, 16-bit, 44.1kHz |
| `TRKS` | All Other Instruments | Stereo, 16-bit, 44.1kHz |

### Reference Files
- **Dry Vocals:** Mono, 16kHz (for lip sync and scoring)
- **Dry Harmony:** Mono, 16kHz (for harmony scoring)
- **CD Reference Mix:** Stereo (quality reference)

## Special Gameplay Features

### Big Rock Ending (BRE)
- Triggered by `[coda]` event in EVENTS track
- Changes scoring system for final section
- Requires special BRE lanes in instrument tracks

### Overdrive/Star Power
- Automatic deploy sections created with 600ms+ gaps between vocal phrases
- Instrument-specific overdrive authoring in individual tracks

### Count-In Requirements
- 2 measures standard (3 for fast songs 160+ BPM, 4 for 210+ BPM)
- First gameplay notes must appear after 2.5 seconds minimum
- 3-second rule for gamertag display
- Pattern: "One, two, One two three four"

## Conversion Considerations

### For Converting TO Other Formats:
1. **Extract Tempo Map:** Parse tempo changes and time signatures
2. **Note Data:** Convert MIDI note numbers to appropriate scales/ranges
3. **Lyric Events:** Extract and clean text events from vocal tracks
4. **Track Separation:** Separate instruments maintain their distinct parts
5. **Timing Precision:** Preserve non-quantized timing (especially vocals)

### For Converting FROM Other Formats:
1. **Create Required Tracks:** Ensure all minimum required tracks exist
2. **Tempo Mapping:** Accurate tempo map is critical
3. **BEAT Track Generation:** Create proper downbeat/upbeat pattern
4. **Event Placement:** Add required `[music_start]`, `[music_end]`, `[end]` events
5. **Audio Alignment:** Ensure MIDI timing matches audio stems

### Key Technical Points:
- **File Format:** Standard MIDI Format (SMF)
- **Timing Resolution:** High precision required (1/64 or 1/128 grid recommended)
- **Event Ordering:** `[end]` event must be absolute last MIDI item
- **Track Dependencies:** BEAT track timing affects all animations
- **Stem Coordination:** MIDI timing must perfectly match audio stem timing

This structure enables Rock Band's complex gameplay mechanics including real-time scoring, dynamic audio mixing, character animation, and visual effects synchronized to the music.
