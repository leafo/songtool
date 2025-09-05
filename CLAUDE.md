# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Songtool is a Go command-line utility for analyzing and processing music game files, specifically:
- **SNG files**: Binary song package format containing MIDI charts, audio stems, metadata, and artwork used by CloneHero and YARG to package RockBand and Guitar hero songs
- **MIDI files**: Standard MIDI files with Rock Band/Guitar Hero specific track conventions
- **ToneLib format**: Support for writing ToneLib Jam song files

## Architecture

### Core Components

- **main.go**: CLI entry point with flag parsing and file format detection
- **drums.go**: Rock Band drum chart processing and General MIDI conversion
- **general_midi.go**: General MIDI instrument mappings and utilities
- **gm_export.go**: General MIDI export functionality for multi-track conversion
- **lyrics.go**: MIDI lyric event extraction utilities
- **pro_bass.go**: Rock Band Pro Bass chart processing and conversion
- **sngfile.go**: Complete SNG file format implementation with XOR unmasking
- **timeline.go**: BEAT track analysis for extracting measure/tempo information
- **tonelib.go**: ToneLib format export including XML generation and ZIP archive creation
- **vocals.go**: Rock Band vocal chart processing and melody extraction

### File Format Support

The tool auto-detects file formats by extension:
- `.sng` files are processed as SNG packages
- Other files are treated as standard MIDI files

## Development Commands

### Building

```bash
go build -o songtool
```

### Dependencies
```bash
go mod tidy         # Clean up dependencies
```

## Key Features

## Usage

```
Usage: ./songtool [flags] <file> [output]
  -export-gm
    	Export drums, vocals, and bass to single General MIDI file
  -export-gm-bass
    	Export pro bass to General MIDI file
  -export-gm-drums
    	Export drum patterns to General MIDI file
  -export-gm-vocals
    	Export vocal melody to General MIDI file
  -export-tonelib-song
    	Create complete ToneLib .song file (ZIP archive)
  -export-tonelib-xml
    	Export to ToneLib the_song.dat XML format
  -extract-file string
    	Extract and print contents of specified file from SNG package to stdout
  -filter-track string
    	Filter to show only tracks whose name contains this string (case-insensitive)
  -json
    	Output information as JSON (supported with: default analysis, --timeline)
  -timeline
    	Print beat timeline from BEAT track
```

### Common Usage Examples

**Basic file analysis:**
```bash
./songtool song.sng                     # Analyze SNG package and embedded MIDI
./songtool notes.mid                    # Analyze standalone MIDI file (typically should be a special Rockband notes.mid file)
./songtool -json song.sng               # Print to stdout JSON dump of entire song and MIDI file as json, including every event
```

**Track filtering and debugging:**
```bash
./songtool -filter-track "PART DRUMS" notes.mid    # Show only drum tracks
./songtool -timeline notes.mid                     # Extract measures/beats timeline
./songtool -timeline -json notes.mid               # Print to stdout the measures/beats timeline
```

**Export operations:**
```bash
./songtool -export-gm-drums notes.mid drums.mid      # Export drums to General MIDI
./songtool -export-gm song.sng complete.mid          # Export all parts to GM
./songtool -export-tonelib-xml notes.mid [song.xml]  # Export to ToneLib XML (either write to file or print to stdout)
./songtool -export-tonelib-song song.sng out.song    # Create complete ToneLib archive
```

### SNG File Processing
- Extracts metadata (song title, artist, difficulty ratings)
- Lists contained files (MIDI charts, audio stems, artwork)
- Common files: `notes.mid`, `song.opus`, `album.jpg`, `song.ini`

## Specification Files

Detailed format documentation is in `spec/`:

### Core Format Specifications
- `spec/sngfile.md`: Complete SNG binary format specification including file structure, XOR masking, metadata, and file extraction
- `spec/tonelib.md`: ToneLib Jam .song file format specification covering ZIP archive structure and the_song.dat XML format

### Rock Band MIDI Specifications
- `spec/rockband/midi.md`: Rock Band MIDI file structure and setup guide covering track organization and technical requirements
- `spec/rockband/drums.md`: Comprehensive drum authoring guide including 5-pad system, Pro Drums, MIDI key mappings, and difficulty encoding
- `spec/rockband/vocals.md`: Vocal encoding guide covering lyric synchronization, note representation, and chart simplification
- `spec/rockband/pro-keys.md`: Pro Keys (keyboard) authoring covering real piano key gameplay, lane shifts, and voicing adaptations
- `spec/rockband/pro-guitar.md`: Pro Guitar/Bass authoring guide including string mapping, fret positioning, MIDI channels, and chord systems
