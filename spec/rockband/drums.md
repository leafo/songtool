Adapted from: <http://docs.c3universe.com/rbndocs/index.php?title=Drum_Authoring>

# Rock Band Drums MIDI Authoring Guide

This document provides a comprehensive overview of how drum parts are encoded in Rock Band MIDI files, including track structure, special metadata notes, Pro Drums features, and difficulty-specific requirements.

## Overview

Rock Band drum authoring represents drum kit performance using a 5-pad system (4 pads + kick pedal) with support for Pro Drums that distinguishes cymbals from toms. The authoring philosophy is rhythmically literal transcription of the actual drum performance, then adapted to fit the Rock Band kit constraints.

## Track Structure

### Main Drum Track
- **Track Name:** `PART DRUMS`
- **Contains:** All difficulty levels in octaves
- **Philosophy:** Exact rhythmic transcription adapted for Rock Band kit

### 2x Bass Pedal Support
- **Track Name:** `PART DRUMS_2X`
- **Purpose:** Alternative version for double bass pedal players
- **Creation:** Duplicate main track, rename text event to `PART DRUMS_2X`

## MIDI Note Mapping by Difficulty

### Expert Drums
- **MIDI Range:** 96 (C6) - 100 (E6)
- **Philosophy:** Literal transcription of drum performance

| MIDI Note | Drum Kit Element |
|-----------|------------------|
| 96 (C6) | Kick Drum |
| 97 (C#6) | Snare Drum |
| 98 (D6) | Hi-Hat / Rack Tom 1 |
| 99 (D#6) | Ride Cymbal / Rack Tom 2 |
| 100 (E6) | Crash Cymbal / Floor Tom |

### Hard Drums
- **MIDI Range:** 84 (C5) - 88 (E5)
- **Mapping:** Same as Expert, one octave lower
- **Reduction:** Remove complexity while maintaining core patterns

### Medium Drums  
- **MIDI Range:** 72 (C4) - 76 (E4)
- **Mapping:** Same as Hard, one octave lower
- **Focus:** Basic rock drumming fundamentals

### Easy Drums
- **MIDI Range:** 60 (C3) - 64 (E3)
- **Mapping:** Same as Medium, one octave lower
- **Limitation:** Never require all 3 limbs simultaneously

## Pro Drums Features

### Tom vs Cymbal Distinction
Pro Drums uses modifier notes to change cymbal gems into tom gems:

| MIDI Note | Function |
|-----------|----------|
| 110 (D7) | Changes Yellow gems to TOM for note duration |
| 111 (D#7) | Changes Blue gems to TOM for note duration |
| 112 (E7) | Changes Green gems to TOM for note duration |

**Default Behavior:** Yellow, Blue, Green display as cymbals without modifiers

### Drum Rolls
Two types of drum rolls available:

#### Standard Drum Rolls
- **MIDI Note:** 126 (F#8)
- **Usage:** Fast single-pad rolls (quarter note minimum)
- **Function:** Player hits any notes within marker for scoring

#### Special Drum Rolls (Two-Lane)
- **MIDI Note:** 127 (G8)
- **Usage:** Cymbal swells between two pads
- **Limitation:** Only works for two-note patterns
- **Note:** Can be difficult to combo; consider single-lane alternative

### Drum Fills
- **MIDI Range:** 120 (C8) - 124 (E8)
- **Function:** Free-form sections where player can improvise
- **Timing:** Cannot overlap with Solo Sections
- **Strategy:** End fills at musical transitions, not rigidly every 4 bars

## Special Metadata and Text Events

### Drum Mix Events
Control which audio streams are muted/unmuted by each pad:

**Format:** `[mix <difficulty> drums<configuration>]`
- `<difficulty>`: 0=Easy, 1=Medium, 2=Hard, 3=Expert
- `<configuration>`: Audio stream setup

#### Common Configurations
| Event | Description |
|-------|-------------|
| `drums0` | Single stereo mix for entire kit |
| `drums0d` | Disco flip - Yellow=Snare, Red=Hi-Hat |
| `drums1` | Mono kick, mono snare, stereo else |
| `drums2` | Mono kick, stereo snare, stereo else |
| `drums3` | Stereo kick, stereo snare, stereo else |

#### Pro Drums Disco Flip
- `drums0dnoflip`, `drums1dnoflip`, etc.
- Maintains disco flip scoring without switching pads in Pro Drums
- Allows proper hi-hat attachment usage

### Character Animation Events
| Text Event | Animation |
|------------|-----------|
| `[idle_realtime]` | Non-beat-synced idling |
| `[idle]` | Normal idling |
| `[idle_intense]` | Intense idling |
| `[play]` | Standard playing |
| `[mellow]` | Mellow playing style |
| `[intense]` | Intense playing style |
| `[ride_side_true]` | Max Weinberg side-swipe ride technique |
| `[ride_side_false]` | Normal ride hits |

## Animation Track (Optional)

### Hand/Foot Animation Notes
Detailed animation system using MIDI notes 24-51:

| MIDI Note | Animation |
|-----------|-----------|
| 24 (C0) | Kick hit with right foot |
| 25 (C#0) | Hi-hat pedal open (duration-based) |
| 26 (D0) | Snare hit with left hand |
| 27 (D#0) | Snare hit with right hand |
| 28 (E0) | Soft snare hit with left hand |
| 29 (F0) | Soft snare hit with right hand |
| 30 (F#0) | Hi-hat hit with left hand |
| 31 (G0) | Hi-hat hit with right hand |
| 32 (G#0) | Percussion with right hand |
| 34 (A#0) | Crash1 hard hit with left hand |
| 35 (B0) | Crash1 soft hit with left hand |
| 36 (C1) | Crash1 hard hit with right hand |
| 37 (C#1) | Crash1 soft hit with right hand |
| 38 (D1) | Crash2 hard hit with right hand |
| 39 (D#1) | Crash2 soft hit with right hand |
| 40-41 (E1-F1) | Crash chokes |
| 42 (F#1) | Ride hit with right hand |
| 43 (G1) | Ride hit with left hand |
| 44-45 (G#1-A1) | Crash2 left hand variations |
| 46-51 | Tom hits (left/right hand combinations) |

## Difficulty-Specific Guidelines

### Expert
- **Approach:** Literal rhythmic transcription
- **Adaptations:** Adjust for Rock Band kit limitations
- **Complex Techniques:** 
  - Flams: Simultaneous red+yellow for snare flams
  - Double crashes: Green+yellow simultaneously
  - Open hi-hats: Blue for distinct open sounds

### Hard
- **Philosophy:** Remove complexity while teaching fundamentals
- **Key Concepts:**
  - Complete limb independence
  - Alternating hands for rolls
  - Kick/crash pairing
  - Fast timekeeping (up to 170 BPM 8th notes)
- **Reductions:**
  - Thin out adjacent kicks
  - Remove fill kicks
  - Eliminate hand crossovers
  - Un-flip disco beats

### Medium
- **Philosophy:** Basic rock drumming skeleton
- **Limitations:**
  - No kicks/snares between right-hand timekeeping
  - No 3-limb simultaneous hits
  - Right hand 8th notes only below 140 BPM
  - One kick per measure above 170 BPM
- **Focus:** Crash/kick pairing on downbeats

### Easy
- **Philosophy:** Never require all 3 limbs
- **Approach:** Either 2-hand beat OR kick+snare beat
- **Guidelines:**
  - No kick/crash pairing (save for Medium)
  - Reduce fills to quarter notes
  - Extra space after crashes at fast tempos

## Conversion Considerations

### From Rock Band Format
1. **Extract difficulty levels:** Parse octave-separated note ranges
2. **Identify Pro Drums features:** 
   - Tom markers: MIDI notes 110-112
   - Roll lanes: MIDI notes 126-127
   - Fill lanes: MIDI notes 120-124
3. **Parse mix events:** Extract audio routing information
4. **Animation data:** Optional detailed hand/foot movements
5. **Map to standard drums:** Convert 5-pad system to full kit

### To Rock Band Format
1. **Create difficulty cascade:** Start with complex transcription, reduce complexity
2. **Add Pro Drums markers:** Place tom modifiers where appropriate
3. **Configure mix events:** Set up audio stream routing
4. **Plan fill sections:** Identify improvisation opportunities
5. **Add special techniques:** Mark rolls, flams, disco flips

### Critical Technical Points
- **Octave separation:** Each difficulty occupies different MIDI octave
- **Modifier system:** Tom markers change visual display only
- **Mix event dependency:** Required for proper audio muting
- **Animation independence:** Visual layer separate from gameplay
- **2x bass support:** Optional double-kick track for advanced players

### Drum Kit Mapping Strategy
**Standard Template:**
- Green: Crash Cymbal / Floor Tom
- Blue: Ride Cymbal / Rack Tom 2  
- Yellow: Hi-Hat / Rack Tom 1
- Red: Snare Drum
- Orange: Kick Drum

**Common Exceptions:**
- Open hi-hats on blue for sonic distinction
- Hi-hats on green for complex left-hand patterns
- Disco flip for fast 16th note hi-hat patterns

This system enables Rock Band to represent complex drum performances across skill levels while maintaining playability on the simplified 5-pad controller system, with Pro Drums adding realistic cymbal/tom distinction for advanced players.
