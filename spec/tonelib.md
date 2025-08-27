# ToneLib Song File Format Specification

This specification file describes the file format used for [ToneLib Jam](https://tonelib.net/jam-overview.html)

## Overview

ToneLib `.song` files are ZIP archives containing musical composition data, audio tracks, graphics configurations, and plugin settings. The format supports multi-track arrangements with detailed notation, tempo changes, and instrument configurations.

This document provides comprehensive coverage of both the complete ToneLib `.song` archive format and the detailed structure of the primary component file `the_song.dat`, which contains all musical notation, timing, and track data. The `the_song.dat` file is the main musical score component within ToneLib `.song` files and is essential for understanding how musical content is encoded.

## File Structure

A `.song` file is a standard ZIP archive with the following structure:

```
song_file.song (ZIP archive)
├── version.info          # Version information (4 bytes)
├── the_song.dat          # Main musical score data (XML)
├── audio/
│   └── [hash].snd       # Audio track (Ogg Vorbis)
└── plg_set_list.dat     # Plugin settings and chain (XML)
```

## File Components

### 1. version.info
- **Format**: Binary data
- **Size**: 4 bytes
- **Content**: Exact bytes: `33 2e 31 00` (ASCII "3.1" + null terminator)
- **Purpose**: Identifies the ToneLib format version
- **Critical**: Must match exactly - bytes `[0x33, 0x2e, 0x31, 0x00]`

### 2. the_song.dat (Main Score)
- **Format**: XML (UTF-8, CRLF line endings)
- **Purpose**: Contains all musical notation, timing, and track data
- **Key Role**: This is the **primary element for synchronizing the transcription with an audio file**

#### Root Element: `<Score>`

The entire song is encapsulated within the `<Score>` element. It serves as the container for all other elements.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Score>
  <!-- All other elements go here -->
</Score>
```

#### Song Information: `<info>`

The `<info>` block contains metadata about the song. While many of these fields can be left empty, they are useful for organization.

```xml
<info>
  <!-- Metadata fields -->
  <name/>         <!-- The title of the song -->
  <artist/>       <!-- The performing artist -->
  <album/>        <!-- The album the song is from -->
  <author/>       <!-- The composer of the music -->
  <date/>         <!-- The date the file was created or the song was released -->
  <copyright/>    <!-- Copyright information -->
  <writer/>       <!-- The lyricist or songwriter -->
  <transcriber/>  <!-- The person who created the transcription -->
  <remarks/>      <!-- Any additional notes -->
  <show_remarks>no</show_remarks>
</info>
  
#### Tempo and Time Signature: `<BarIndex>`

The `<BarIndex>` defines the overall structure of the song in terms of bars (measures), tempo, and time signature. This is the **primary element for synchronizing the transcription with an audio file**.

```xml
<BarIndex>
  <!-- Tempo and time signature definitions -->
  <Bar id="1" tempo="121" jam_set="0">
    <time_sign numerator="4" duration="4"/>
    <label letter="" text="Intro"/>  <!-- Optional section label -->
  </Bar>
  <Bar id="2" jam_set="0"/>
  <Bar id="3" jam_set="0"/>
  <Bar id="4" tempo="125" jam_set="0"/>
  <Bar id="5" jam_set="0" tempo="121">
    <time_sign numerator="3" duration="4"/>
  </Bar>
  <!-- Bars inherit tempo/time signature from previous bars -->
</BarIndex>
```

**Bar Elements:**
- `<Bar>`: Each `<Bar>` element represents one measure of the song
  - `id`: A unique number for each bar, starting from 1
  - `tempo`: The tempo in beats per minute (BPM) for that bar. **Note:** The tempo is only specified when it changes. Subsequent bars will use the last specified tempo. Getting this tempo map correct is the key to proper audio sync.
  - `jam_set`: Always 0 (purpose unknown)
  - `<time_sign>`: This child element is only present if the time signature changes
    - `numerator`: The top number of the time signature (e.g., 4 for 4/4 time)
    - `duration`: The bottom number of the time signature (e.g., 4 for 4/4 time)
  - `<label>`: This optional child element is used to add a text marker for a specific bar, often to denote a section of the song like "Intro" or "Verse"
    - `letter`: A short-form letter for the section (e.g., "A", "B"). Can be left empty
    - `text`: The descriptive name of the section (e.g., "Intro")
  
#### Tracks and Notes: `<Tracks>`

The `<Tracks>` element contains one or more `<Track>` elements, each representing a different instrument or part.

```xml
<Tracks>
  <Track name="TrackName" color="ARGB" visible="1" collapse="0" lock="0"
         solo="0" mute="0" opt="0" vol_db="VOLUME" bank="BANK_NUM"
         program="PROGRAM_NUM" chorus="0" reverb="0" phaser="0" 
         tremolo="0" id="TRACK_ID" offset="OFFSET">
    <Strings>
      <String id="N" tuning="MIDI_OFFSET"/>
    </Strings>
    <Bars>
      <Bar id="BAR_ID">
        <Clef value="CLEF_TYPE"/>
        <KeySign value="KEY_SIGNATURE"/>
        <Beat duration="DURATION" dyn="DYNAMICS" dotted="1">
          <Note fret="FRET" string="STRING" tied="yes|no">
            <Effects ghost="yes">
              <Grace fret="FRET" duration="DURATION" dynamic="VELOCITY" transition="TYPE"/>
            </Effects>
          </Note>
          <Text value="LYRICS"/>
        </Beat>
        <Beats/>
      </Bar>
    </Bars>
  </Track>
</Tracks>
</Score>
```

### Understanding Pitch: Strings, Tuning, and Frets

The pitch of every note in `tonelib jam` is determined by three key components: the **string** it's played on, the **tuning** of that string, and the **fret** number.

**Final MIDI Note = `String tuning` (MIDI offset) + `fret` number**

This formula is fundamental to understanding how all notes are encoded in the ToneLib format.

#### Key Elements:

**BarIndex Key Points**:
- Defines global tempo and timing structure
- `tempo`: BPM (beats per minute) - only specified when changing tempo
- Bars without `tempo` attribute inherit the tempo from the previous bar with a tempo
- Getting the tempo map correct is the key to proper audio synchronization
- `jam_set`: Always 0 (purpose unknown)
- `time_sign`: Time signature with `numerator` and `duration` attributes
  - Only specified when time signature changes (e.g., 4/4 to 3/4)
  - Bars inherit time signature from previous bar that specified one
  - Examples: `numerator="4" duration="4"` (4/4), `numerator="3" duration="4"` (3/4)
- `label`: Optional section markers for song structure organization

**Track Attributes**:
- `name`: Track name (e.g., "Voice", "Drum")
- `color`: ARGB color value (hex format with "ff" prefix)
- `vol_db`: Volume in decibels
- `bank`: MIDI bank number (128 for drums)
- `program`: MIDI program number
- `id`: Unique track identifier
- `offset`: Time offset

#### `<Strings>` and Tuning Definition

Inside each `<Track>`, the `<Strings>` element defines the base pitch for each string.

- `<String>`: Represents a single string on the instrument
  - `id`: The identifier for the string (e.g., "1", "2", "3")
  - `tuning`: This is the **most important attribute for pitch**. It sets the open-string pitch as a **MIDI note number**. For example, in standard guitar tuning, the low E string is MIDI note 40.

**String Tuning Examples**:
- **Standard guitar**: [64, 59, 55, 50, 45, 40] (High E, B, G, D, A, Low E)
  ```xml
  <Strings>
    <String id="1" tuning="64"/> <!-- High E string -->
    <String id="2" tuning="59"/> <!-- B string -->
    <String id="3" tuning="55"/> <!-- G string -->
    <String id="4" tuning="50"/> <!-- D string -->
    <String id="5" tuning="45"/> <!-- A string -->
    <String id="6" tuning="40"/> <!-- Low E string -->
  </Strings>
  ```
- **Drums**: All set to 0 (so fret values directly represent MIDI drum notes)
  ```xml
  <Strings>
    <String id="1" tuning="0"/>
    <String id="2" tuning="0"/>
    <String id="3" tuning="0"/>
    <String id="4" tuning="0"/>
    <String id="5" tuning="0"/>
    <String id="6" tuning="0"/>
  </Strings>
  ```

#### `<Note>` Element and Effects

The `<Note>` element specifies which fret is played on which string. The final, absolute pitch of the note is calculated with the formula above.

**Attributes of `<Note>`:**
- `fret`: The fret number to be played. An open string is `fret="0"`. Each fret represents one semitone higher in pitch
- `string`: The `id` of the string being played. This tells the software which base `tuning` value to use for the calculation
- `tied="yes"`: If present, this note is tied to the previous note of the same pitch, extending its duration

**Child Element: `<Effects>`**

The `<Note>` element can contain an `<Effects>` child element to specify articulations and special playing techniques.

- **Ghost Note:** The `ghost="yes"` attribute designates the note as a "ghost note." Ghost notes are played much more softly than surrounding notes and are often more percussive in nature.
  ```xml
  <Note fret="36" string="1">
    <Effects ghost="yes"/>
  </Note>
  ```

- **Grace Note:** The `<Grace>` element represents a grace note that precedes the main note.
  - `fret`: The fret of the grace note itself
  - `duration`: The duration of the grace note
  - `dynamic`: The velocity or loudness of the grace note
  - `transition`: The type of grace note (e.g., a slide, bend)
  ```xml
  <Note fret="7" string="4">
    <Effects>
      <Grace fret="9" duration="3" dynamic="79" transition="0"/>
    </Effects>
  </Note>
  ```

**Real-world example from actual files:**
```xml
<Beat duration="8" dyn="mf">
  <Note fret="7" string="4">
    <Effects>
      <Grace fret="9" duration="3" dynamic="79" transition="0"/>
    </Effects>
  </Note>
</Beat>
```

#### `<Bars>` and `<Beat>` Elements

The `<Bars>` element within a `<Track>` contains the musical information for that instrument, broken down by bar.

- `<Bar id="...">`: Corresponds to the bars defined in the `<BarIndex>`
- `<Beat>`: Each `<Bar>` is divided into `<Beat>`s
  - `duration`: The duration of the beat. This is a denominator value (e.g., `4` for a quarter note, `8` for an eighth note, `16` for a sixteenth note). The sum of the durations of all `<Beat>` elements in a bar should equal the time signature
  - `dyn`: Dynamics (mf=mezzo-forte)
  - `dotted`: Optional attribute for dotted rhythms (dotted="1")
  - **Important**: Empty beats (rests) can be created with just `<Beat duration="N" dyn="mf"/>` without any child elements
- `<Note>`: Fret and string positions with optional effects
- `<Text>`: Lyrics or annotations (placed within the same `<Beat>` as the associated `<Note>`)
- `<Effects>`: Special note effects including grace notes and ghost notes
- `<Beats/>`: Required closing tag at the end of each `<Bar>`

### 3. Audio Files (audio/*.snd)
- **Format**: Ogg Vorbis
- **Naming**: SHA-256 hash with `.snd` extension
- **Properties**:
  - Sample Rate: 44.1 kHz
  - Channels: Stereo (2)
  - Bitrate: ~175-192 kbps
  - Encoding: Xiph.Org libVorbis

#### Audio File Linking: `<Backing_track1>`

This section within the `<Score>` element links an external audio file to the project. **Note:** In the complete ToneLib `.song` format, audio files are stored separately as `.snd` files within the ZIP archive's `audio/` directory. The actual synchronization of the notes to this audio is controlled by the tempo map in the `<BarIndex>` element.

- `<audio>`: Contains information about the audio file
  - `<name>`: The filename of the audio file (e.g., `song.ogg`). In the full ToneLib format, this references an audio file stored separately in the ZIP archive
  - `<time_offset>`: A value in seconds to shift the start of the audio. A negative value means the audio starts slightly before the first beat of the first bar. This is useful for aligning audio that has a pickup or lead-in
  - `<bars>`: **(Optional)** This element and its child `<beat>` elements are not used for playback synchronization. They are only used to store and display visual markers in the timeline if an automatic beat-detection process has been run on the audio file. For creating a file from scratch, this entire `<bars>` section can be omitted

**Example:**
```xml
<Backing_track1>
  <audio>
    <name>audio/mysong.ogg</name>
    <time_offset>0.0</time_offset>
  </audio>
</Backing_track1>
```

### Special Section: Encoding Vocal Melodies

Vocal tracks are encoded like any other melodic instrument. Here's how to interpret the "Voice" track for conversion:

- **Track Name:** The `<Track>` name should be "Voice"
- **Pitch:** The pitch of the vocal note is determined by the combination of the `string` and `fret` attributes in the `<Note>` element. You will need to map the MIDI note numbers from your source data (e.g., Rock Band) to a corresponding string and fret
- **Lyrics:** Lyrics are placed within a `<Text>` element inside a `<Beat>`. The lyric is associated with the `<Note>` that shares the same `<Beat>` parent

**Example:**
```xml
<Bar id="5">
  <Beat duration="8">
    <Text value="Take me down"/>
    <Note fret="7" string="4"/>
  </Beat>
  <Beat duration="4">
    <Note fret="9" string="4"/>
  </Beat>
  ...
</Bar>
```

Here, the words "Take me down" are sung on the note defined by `fret="7"` and `string="4"`.

**Real-world examples from actual ToneLib files:**
```xml
<Beat duration="8" dyn="mf">
  <Text value="Take me down"/>
  <Note fret="7" string="4"/>
</Beat>
<Beat duration="4" dyn="mf">
  <Note fret="9" string="4"/>
</Beat>
<Beat duration="8" dyn="mf">
  <Note fret="7" string="4"/>
</Beat>
```

**Complex lyric example with tied notes:**
```xml
<Beat duration="4" dyn="mf">
  <Note fret="7" string="4" tied="yes"/>
</Beat>
<Beat duration="4" dyn="mf"/>
<Beat duration="8" dyn="mf">
  <Text value="In a Tid-"/>
  <Note fret="5" string="4"/>
</Beat>
```

### Special Section: Encoding Drums

Drum tracks are handled differently from melodic tracks. Instead of pitch, the `fret` attribute directly represents a specific drum sound based on the General MIDI (GM) drum map.

- **Track Name:** The `<Track>` name should be "Drum"
- **MIDI Bank:** For drums, the `bank` attribute of the `<Track>` is typically set to `128` to select the drum kit sounds
- **String Setup for Drums:** For a drum track, the `<Strings>` element is configured in a specific way to simplify the mapping. All strings are defined with a `tuning` of `0`

- **Drum Sounds as "Frets":** Because the string `tuning` is `0`, the pitch calculation formula (`tuning + fret`) simplifies to just `fret`. This means the `fret` attribute in a `<Note>` element becomes the **direct MIDI note number** for the desired drum sound. Here are some common mappings from the GM drum map:
  - `fret="36"`: Bass Drum
  - `fret="38"`: Snare Drum
  - `fret="42"`: Closed Hi-Hat
  - `fret="46"`: Open Hi-Hat
  - `fret="49"`: Crash Cymbal
  - `fret="51"`: Ride Cymbal

- **Chords:** When multiple `<Note>` elements are inside the same `<Beat>`, it signifies that those drum pieces are hit simultaneously. The `string` attribute is used simply to keep the notes visually separated in the software's editor

**Example:**
```xml
<Bar id="25">
  <Beat duration="8">
    <Note fret="49" string="1"/> <!-- Crash Cymbal (MIDI note 49) -->
    <Note fret="42" string="2"/> <!-- Closed Hi-Hat (MIDI note 42) -->
    <Note fret="36" string="3"/> <!-- Bass Drum (MIDI note 36) -->
  </Beat>
  <Beat duration="8">
    <Note fret="42" string="1"/> <!-- Closed Hi-Hat -->
  </Beat>
  <Beat duration="8">
    <Note fret="42" string="1"/> <!-- Closed Hi-Hat -->
    <Note fret="38" string="2"/> <!-- Snare Drum -->
  </Beat>
  ...
</Bar>
```

In this example, the first beat of the bar is a crash, hi-hat, and bass drum hit all at once.

### 4. Track Definitions (Tracks Section)

The `<Tracks>` section within `the_song.dat` contains all musical track data. Each `<Track>` element represents a single instrument or voice in the composition.

#### Track Structure:
```xml
<Tracks>
  <Track name="TRACK_NAME" color="ARGB_HEX" visible="1" collapse="0" lock="0"
         solo="0" mute="0" opt="0" vol_db="VOLUME_DB" bank="MIDI_BANK"
         program="MIDI_PROGRAM" chorus="0" reverb="0" phaser="0" 
         tremolo="0" id="UNIQUE_ID" offset="TIME_OFFSET">
    <Strings>
      <String id="1" tuning="MIDI_OFFSET"/>
      <String id="2" tuning="MIDI_OFFSET"/>
      <String id="3" tuning="MIDI_OFFSET"/>
      <String id="4" tuning="MIDI_OFFSET"/>
      <String id="5" tuning="MIDI_OFFSET"/>
      <String id="6" tuning="MIDI_OFFSET"/>
    </Strings>
    <Bars>
      <Bar id="BAR_NUMBER">
        <Clef value="CLEF_TYPE"/>
        <KeySign value="KEY_SIGNATURE"/>
        <Beat duration="NOTE_DURATION" dyn="DYNAMICS" dotted="1">
          <Note fret="FRET_VALUE" string="STRING_NUMBER" tied="yes|no">
            <Effects ghost="yes">
              <Grace fret="FRET" duration="DURATION" dynamic="VELOCITY" transition="TYPE"/>
            </Effects>
          </Note>
          <Text value="LYRICS_OR_TEXT"/>
        </Beat>
        <Beats/>
      </Bar>
    </Bars>
  </Track>
</Tracks>
```

#### Track Attributes:
- **`name`**: Display name (e.g., "Voice", "Drum", "Guitar")
- **`color`**: ARGB color in hex format with "ff" prefix (e.g., "fff5a41c")
- **`visible`**: Track visibility (1=visible, 0=hidden)
- **`collapse`**: UI collapse state (1=collapsed, 0=expanded)
- **`lock`**: Edit protection (1=locked, 0=unlocked)
- **`solo`**: Solo playback (1=solo, 0=normal)
- **`mute`**: Mute state (1=muted, 0=unmuted)
- **`vol_db`**: Volume in decibels (can be negative values like "-0.1574783325195312" or "-23.70078659057617")
- **`bank`**: MIDI bank select (128 for percussion, 0 for melodic)
- **`program`**: MIDI program change (instrument selection)
- **`id`**: Unique track identifier
- **`offset`**: Time synchronization offset in ticks

#### Strings Section:
The ToneLib format is designed primarily for stringed instruments and requires all tracks to define strings with MIDI tuning values. Each string represents a distinct voice or pitch reference for notes played on that string.

**How String Tuning Works:**
- Each `<String>` element defines one string with a `tuning` attribute (MIDI note number)
- When a note is played on that string, the final MIDI pitch = `fret value + string tuning`
- This mimics real stringed instruments where fretting adds semitones to the open string
- Any number of strings can be defined (commonly 6 for guitar, but flexible)

**Examples:**
- Standard guitar tuning: String 6 (low E) = tuning="40" (MIDI note 40 = E2)
- Playing fret 3 on this string: 40 + 3 = 43 (MIDI note 43 = G2)
- For drums: All strings set to tuning="0", so fret values directly represent MIDI drum notes

#### Bars Section:
Contains the actual musical content organized by bar numbers:
- **`<Clef>`**: Staff notation type (1=treble, 5=percussion)
- **`<KeySign>`**: Key signature (0=C major)
- **`<Beat>`**: Individual rhythmic events with duration, dynamics, and optional dotted attribute
- **`<Note>`**: Musical notes with fret position and string assignment, optional tied attribute
- **`<Text>`**: Lyrics, chord names, or annotations
- **`<Effects>`**: Special note effects container with optional ghost attribute and/or Grace note sub-elements
  - `ghost="yes"`: Marks note as a ghost note (played softly/percussively)
- **`<Grace>`**: Grace note with fret, duration, dynamic, and transition attributes
- **`<Beats/>`**: Empty closing tag (required)

### 5. Plugin Settings (plg_set_list.dat)
- **Format**: XML (UTF-8, CRLF line endings)
- **Purpose**: Audio plugin chain and effect configurations

#### Structure:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<plg_set_list>
  <plg_set>
    <nodes>
      <node uid="ID" name="PLUGIN_NAME" x_pos="X" y_pos="Y">
        <PLUGIN name="NAME" descriptiveName="" format="FORMAT" category="CATEGORY"
                manufacturer="VENDOR" version="VERSION" file="FILE_ID"
                uniqueId="UNIQUE_ID" isInstrument="0" fileTime="0" infoUpdateTime="0"
                numInputs="N" numOutputs="N" isShell="0" hasARAExtension="0"
                uid="0"/>
        <state>PLUGIN_STATE_DATA</state>
      </node>
    </nodes>
  </plg_set>
</plg_set_list>
```

## Implementation Guidelines

### Creating a ToneLib Song File

1. **Prepare Components**:
   - Create XML score data (`the_song.dat`)
   - Encode audio as Ogg Vorbis (`.snd` file)
   - Create plugin settings XML (`plg_set_list.dat`)
   - Create version info (exact 4 bytes: `[0x33, 0x2e, 0x31, 0x00]`)

2. **File Naming**:
   - Use SHA-256 hashes for audio files
   - Keep original extensions (`.snd`)

3. **ZIP Archive**:
   - Create standard ZIP archive
   - Maintain directory structure (`audio/`)
   - Use compression method "deflate"

### Key Considerations

- **Tempo Changes**: Defined in BarIndex, can change at any bar (bars inherit tempo from previous tempo-specified bar)
- **Track Types**: Different clef values (1=treble, 5=percussion)
- **MIDI Integration**: Uses standard MIDI bank/program numbers
- **Time Signatures**: Defined in BarIndex, can change at any bar (bars inherit from previous time signature specification)
- **String Instruments**: Support 6-string configuration with MIDI tuning
- **Lyrics**: Embedded as Text elements within beats

## Additional Sections

Other elements can appear at the root of the `<Score>` element. These are generally for editor-specific features.

### Other Placeholders: `<Plugin_Host>`, `<Movies>`, and `<Play_along>`

These elements appear in the file but seem to be placeholders for features that are either not fully implemented or not used in this example. **Note:** In the complete ToneLib `.song` format, plugin configurations are handled by a separate `plg_set_list.dat` file within the ZIP archive. For conversion purposes, these elements can likely be included as empty tags or omitted entirely.

- `<Plugin_Host>`: Likely intended for hosting VST or other audio plugins. (Actual plugin settings are stored in `plg_set_list.dat` in the full format)
- `<Movies>`: Likely intended for synchronizing the score with a video file
- `<Play_along>`: Appears to be for editor-specific features, possibly related to sectioning or playback

## Drum Track Notation

All tracks in ToneLib must be represented as fretted instruments with six strings. For drums, this is achieved by setting all string tunings to 0, which allows the `fret` attribute in `<Note>` elements to directly represent MIDI note numbers (since fret + tuning = MIDI note, and tuning is 0).

### Drum Track Characteristics

**Track Configuration:**
```xml
<Track name="Drum" color="fffad11c" bank="128" program="0" 
       visible="1" collapse="0" lock="0" id="4" offset="0">
  <Strings>
    <String id="1" tuning="0"/>
    <String id="2" tuning="0"/>
    <String id="3" tuning="0"/>
    <String id="4" tuning="0"/>
    <String id="5" tuning="0"/>
    <String id="6" tuning="0"/>
  </Strings>
  <Bars>
    <Bar id="N">
      <Clef value="5"/>  <!-- Percussion clef -->
      <KeySign value="0"/>
      <!-- Drum notes here -->
    </Bar>
  </Bars>
</Track>
```

**Key Properties:**
- `bank="128"`: MIDI percussion bank
- `program="0"`: Standard drum kit
- `Clef value="5"`: Percussion clef (vs. `1` for treble)
- All string tunings set to `0`

### Drum Note Mapping

Common drum sounds found in the example file:

| Fret Value | MIDI Note | Drum Sound |
|------------|-----------|------------|
| 36 | C2 | Kick Drum |
| 38 | D2 | Acoustic Snare |
| 42 | F#2 | Closed Hi-Hat |
| 46 | A#2 | Open Hi-Hat |
| 49 | C#3 | Crash Cymbal 1 |

### Drum Notation Examples

**Basic Rock Beat:**
```xml
<Beat duration="8" dyn="mf">
  <Note fret="42" string="1"/>  <!-- Hi-hat -->
  <Note fret="36" string="2"/>  <!-- Kick -->
</Beat>
<Beat duration="8" dyn="mf">
  <Note fret="42" string="1"/>  <!-- Hi-hat -->
</Beat>
<Beat duration="8" dyn="mf">
  <Note fret="42" string="1"/>  <!-- Hi-hat -->
  <Note fret="38" string="2"/>  <!-- Snare -->
</Beat>
<Beat duration="8" dyn="mf">
  <Note fret="42" string="1"/>  <!-- Hi-hat -->
</Beat>
```

**Cymbal Accents:**
```xml
<Beat duration="8" dyn="mf">
  <Note fret="36" string="1"/>  <!-- Kick -->
  <Note fret="49" string="2"/>  <!-- Crash -->
</Beat>
```

### Implementation Notes

- Multiple drum sounds can be played simultaneously by including multiple `<Note>` elements within a single `<Beat>`
- The `string` attribute helps organize different drum voices (all strings have tuning="0" for drums)
- Drum patterns often use eighth notes (`duration="8"`) and sixteenth notes (`duration="16"`)
- Ghost notes can be notated with `<Effects ghost="yes"/>`
- **MIDI Note Calculation**: For any track, final MIDI note = fret value + string tuning value
  - Guitar example: fret="5" on string tuned to "40" = MIDI note 45
  - Drum example: fret="36" on string tuned to "0" = MIDI note 36 (kick drum)

**Critical Bar Structure Requirements**:
- Each `<Bar>` in a track must contain:
  1. Optional `<Clef value="1"/>` (treble clef, usually only in first bar)
  2. Optional `<KeySign value="0"/>` (key signature, usually only in first bar)
  3. One or more `<Beat>` elements
  4. Required trailing empty `<Beats/>` tag
- The sum of all `duration` values in `<Beat>` elements should equal the time signature
- For 4/4 time, beat durations should sum to represent 4 quarter note values

## Complete Example File

Here is a complete, minimal example of a `the_song.dat` XML file containing two measures with a vocal track and a drum track. This would be the content of the `the_song.dat` file within a ToneLib `.song` ZIP archive.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Score>
  <info>
    <name>Sample Song</name>
    <artist>The Examples</artist>
    <album/>
    <author/>
    <date/>
    <copyright/>
    <writer/>
    <transcriber/>
    <remarks/>
    <show_remarks>no</show_remarks>
  </info>

  <BarIndex>
    <Bar id="1" tempo="120" jam_set="0">
      <time_sign numerator="4" duration="4"/>
      <label text="Verse 1"/>
    </Bar>
    <Bar id="2" jam_set="0" />
    <Bar id="3" jam_set="0" tempo="121"/>
  </BarIndex>

  <Tracks>
    <Track name="Voice" color="fff5a41c" bank="0" program="27" id="1">
      <Strings>
        <String id="1" tuning="64"/>
        <String id="2" tuning="59"/>
        <String id="3" tuning="55"/>
        <String id="4" tuning="50"/>
        <String id="5" tuning="45"/>
        <String id="6" tuning="40"/>
      </Strings>
      <Bars>
        <Bar id="1">
          <Beat duration="4">
            <Text value="Hel-"/>
            <Note fret="7" string="4"/>
          </Beat>
          <Beat duration="4">
            <Text value="-lo"/>
            <Note fret="9" string="4"/>
          </Beat>
          <Beat duration="2"/>
        </Bar>
        <Bar id="2">
          <Beat duration="4">
            <Text value="World!"/>
            <Note fret="10" string="4"/>
          </Beat>
          <Beat duration="4"/>
          <Beat duration="2"/>
        </Bar>
      </Bars>
    </Track>
    <Track name="Drum" color="fffad11c" bank="128" program="0" id="2">
      <Strings>
        <String id="1" tuning="0"/>
        <String id="2" tuning="0"/>
        <String id="3" tuning="0"/>
        <String id="4" tuning="0"/>
        <String id="5" tuning="0"/>
        <String id="6" tuning="0"/>
      </Strings>
      <Bars>
        <Bar id="1">
          <Beat duration="8">
            <Note fret="36" string="1"/> <!-- Kick -->
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
          <Beat duration="8">
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
          <Beat duration="8">
            <Note fret="38" string="1"/> <!-- Snare -->
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
          <Beat duration="8">
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
          <Beat duration="8">
            <Note fret="36" string="1"/> <!-- Kick -->
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
          <Beat duration="8">
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
          <Beat duration="8">
            <Note fret="38" string="1"/> <!-- Snare -->
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
          <Beat duration="8">
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
        </Bar>
        <Bar id="2">
          <Beat duration="8">
            <Note fret="36" string="1"/> <!-- Kick -->
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
          <Beat duration="8">
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
          <Beat duration="8">
            <Note fret="38" string="1"/> <!-- Snare -->
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
          <Beat duration="8">
            <Note fret="42" string="2"/> <!-- Hi-hat -->
          </Beat>
          <Beat duration="8">
            <Note fret="36" string="1"/> <!-- Kick -->
            <Note fret="49" string="3"/> <!-- Crash -->
          </Beat>
          <Beat duration="8"/>
          <Beat duration="4"/>
        </Bar>
      </Bars>
    </Track>
  </Tracks>

  <Backing_track1>
    <audio>
      <name>audio/mysong.ogg</name>
      <time_offset>0.0</time_offset>
    </audio>
  </Backing_track1>
</Score>
```

## Relationship to Complete ToneLib Format

This XML content represents the `the_song.dat` file within a complete ToneLib `.song` format. The full ToneLib format is a ZIP archive containing:

- `the_song.dat` - The XML content described in this document
- `version.info` - Version information (4 bytes: `33 2e 31 00`)
- `audio/[hash].snd` - Audio files in Ogg Vorbis format
- `plg_set_list.dat` - Plugin settings and effect chains

## Example Minimal Song Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Score>
  <info>
    <name>My Song</name>
    <artist>Artist Name</artist>
  </info>
  <BarIndex>
    <Bar id="1" tempo="120" jam_set="0">
      <time_sign numerator="4" duration="4"/>
    </Bar>
  </BarIndex>
  <Tracks>
    <Track name="Guitar" color="fff5a41c" visible="1" id="1" offset="0">
      <Strings>
        <String id="1" tuning="64"/>
        <String id="2" tuning="59"/>
        <String id="3" tuning="55"/>
        <String id="4" tuning="50"/>
        <String id="5" tuning="45"/>
        <String id="6" tuning="40"/>
      </Strings>
      <Bars>
        <Bar id="1">
          <Clef value="1"/>
          <KeySign value="0"/>
          <Beat duration="1" dyn="mf"/>
          <Beats/>
        </Bar>
      </Bars>
    </Track>
  </Tracks>
</Score>
```

## Key Implementation Notes from Real Examples

1. **Tempo Precision**: Actual ToneLib files show very precise tempo changes (121, 124, 125, 127, etc.) that closely follow the original audio timing
2. **Complex Vocal Patterns**: Lyrics can span across multiple beats and include:
   - Tied notes that extend duration (`tied="yes"`)
   - Hyphenated words split across beats ("Tid-" and "-dal")
   - Grace notes for vocal embellishments
   - Dotted rhythms for natural speech patterns
3. **Empty Bars**: Many bars contain only rest beats with no notes or text - just `<Beat duration="1" dyn="mf"/>` for whole rests
4. **Consistent Structure**: Every bar ends with the `<Beats/>` closing tag
5. **Volume Precision**: Volume levels use high-precision decimal values (e.g., "-0.1574783325195312")
6. **Bar Structure**: Only the first bar typically contains `<Clef>` and `<KeySign>` elements; subsequent bars inherit these settings
