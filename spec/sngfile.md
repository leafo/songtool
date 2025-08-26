# SNG File Format Specification

Adapted from: https://github.com/trojannemo/Nautilus/tree/master/Nautilus/SngLib

## Overview

The SNG (Song Package) file format is a binary container format designed for storing song data, metadata, and associated files for music games. This format stores complete song packages including chart files, audio stems, images, and metadata with file data masked using XOR operations.

## File Structure

All multi-byte integers are stored in little-endian byte order unless otherwise specified.

### Header Section

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| 0x00 | 6 bytes | File Identifier | ASCII string "SNGPKG" |
| 0x06 | 4 bytes | Version | uint32 version number (current: 1) |
| 0x0A | 16 bytes | XOR Mask | 16-byte mask for file data XOR operations |

### Metadata Section

| Offset | Size | Field | Description |
|--------|------|-------|-------------|
| 0x1A | 8 bytes | Metadata Length | uint64 total size of metadata section |
| 0x22 | 8 bytes | Metadata Count | uint64 number of metadata key-value pairs |

Following the metadata header are `Metadata Count` entries with the following structure:

| Field | Size | Description |
|-------|------|-------------|
| Key Length | 4 bytes | int32 length of key string |
| Key Data | Variable | UTF-8 encoded key string |
| Value Length | 4 bytes | int32 length of value string |
| Value Data | Variable | UTF-8 encoded value string |

### File Index Section

| Field | Size | Description |
|-------|------|-------------|
| File Index Length | 8 bytes | uint64 total size of file index section |
| File Count | 8 bytes | uint64 number of files in the package |

Following the file index header are `File Count` entries with the following structure:

| Field | Size | Description |
|-------|------|-------------|
| Filename Length | 1 byte | uint8 length of filename (max 255 chars) |
| Filename | Variable | UTF-8 encoded filename |
| File Size | 8 bytes | uint64 size of file data in bytes |
| File Offset | 8 bytes | uint64 absolute offset to file data |

### File Data Section

| Field | Size | Description |
|-------|------|-------------|
| File Section Length | 8 bytes | uint64 total size of all file data |
| File Data | Variable | Concatenated XOR-masked file contents |

## Data Masking

All file data is XOR-masked using a position-dependent operation with the 16-byte XOR mask from the header.

### Masking Algorithm

The masking uses a lookup table approach:

1. Create a 256-byte lookup table where `lookup[i] = i ^ mask[i & 0x0F]`
2. For each byte at file position `pos`, XOR with `lookup[pos & 0xFF]`

### Implementation Details

```
For each byte at position filePos within the individual file:
    maskedByte = originalByte ^ lookupTable[filePos & 0xFF]

Where:
    lookupTable[i] = i ^ xorMask[i & 0x0F]
    filePos starts at 0 for each individual file
```

## Metadata Keys

The format supports a comprehensive set of metadata keys for song information:

### Basic Song Information
- `name` - Song title
- `artist` - Artist name  
- `album` - Album name
- `genre` - Musical genre
- `sub_genre` - Sub-genre classification
- `year` - Release year
- `charter` - Chart creator
- `version` - Chart version

### Difficulty Ratings
Difficulty values are integers typically ranging 0-7:
- `diff_guitar` - Lead guitar difficulty
- `diff_bass` - Bass difficulty  
- `diff_drums` - Drums difficulty
- `diff_vocals` - Vocals difficulty
- `diff_keys` - Keyboard difficulty
- Additional instrument-specific difficulties (e.g., `diff_guitar_real`, `diff_bass_real_22`)

### Timing and Preview
- `song_length` - Song duration in milliseconds
- `delay` - Audio delay/offset in milliseconds
- `preview_start_time` - Preview start time in milliseconds
- `preview_end_time` - Preview end time in milliseconds
- `video_start_time` - Video start time in milliseconds
- `video_end_time` - Video end time in milliseconds

### Visual Assets
- `background` - Background image filename
- `video` - Background video filename
- `album` - Album artwork filename
- `icon` - Song icon filename

### Advanced Features
- `pro_drums` - Boolean for pro drums support
- `five_lane_drums` - Boolean for 5-lane drums
- `modchart` - Boolean for modified charts
- `end_events` - Boolean for end events support
- Various instrument tunings and specialized options

## Supported File Types

### Audio Files
Supported audio formats with standard stem naming:
- `song.*` - Full mix/master track
- `guitar.*` - Guitar stem
- `bass.*` - Bass stem  
- `drums.*` - Drums stem
- `vocals.*` - Vocals stem
- `keys.*` - Keyboard stem
- `crowd.*` - Crowd audio
- `preview.*` - Preview audio

Supported extensions: `.wav`, `.ogg`, `.opus`, `.mp3`

### Chart Files
- `notes.chart` - Clone Hero chart format
- `notes.mid` - MIDI chart format

### Visual Assets  
- `background.*` - Background images (`.png`, `.jpg`, `.jpeg`)
- `album.*` - Album artwork
- `highway.*` - Highway textures
- `video.*` - Background videos (`.mp4`, `.avi`, `.webm`, `.vp8`, `.ogv`, `.mpeg`)

### Configuration
- `song.ini` - Song metadata in INI format

## Reading SNG Files

### Basic Reading Process

1. **Verify Header**
   - Read 6-byte identifier, verify it equals "SNGPKG"
   - Read version (4 bytes), verify it equals 1
   - Read 16-byte XOR mask for unmasking

2. **Read Metadata**
   - Read metadata section length and count
   - For each metadata entry:
     - Read key length and key string
     - Read value length and value string
     - Store key-value pair

3. **Read File Index**
   - Read file index section length and file count
   - For each file entry:
     - Read filename length and filename
     - Read file size and offset
     - Store file metadata

4. **Read File Data**
   - Read file section length
   - For each file:
     - Seek to file offset
     - Read XOR-masked file data
     - Unmask using XOR mask
     - Store unmasked file content

### Unmasking Example (C#)

```csharp
private static void UnmaskFileData(Span<byte> data, byte[] xorMask, long filePos)
{
    // Create lookup table
    byte[] lookup = new byte[256];
    for (int i = 0; i < 256; i++)
    {
        lookup[i] = (byte)(i ^ xorMask[i & 0x0F]);
    }
    
    // Unmask each byte
    for (int i = 0; i < data.Length; i++)
    {
        data[i] ^= lookup[(filePos + i) & 0xFF];
    }
}
```

## Writing SNG Files

### Basic Writing Process

1. **Calculate Sizes**
   - Determine header size (26 bytes + metadata + file index)
   - Calculate total metadata size
   - Calculate file index size
   - Calculate total file data size

2. **Write Header**
   - Write "SNGPKG" identifier
   - Write version (1)  
   - Write randomly generated 16-byte XOR mask

3. **Write Metadata**
   - Write metadata section length and count
   - For each key-value pair:
     - Write key length and UTF-8 key
     - Write value length and UTF-8 value

4. **Write File Index**
   - Write file index length and file count
   - For each file:
     - Write filename length and UTF-8 filename
     - Write file size and calculated offset

5. **Write File Data**
   - Write file section total length
   - For each file:
     - Mask file data using XOR mask
     - Write masked data

### Masking for Writing

File data must be masked with the same XOR algorithm before writing:

```csharp
private static void MaskFileData(Span<byte> data, byte[] xorMask, long filePos)
{
    // Same algorithm as unmasking - XOR is symmetric
    UnmaskFileData(data, xorMask, filePos);
}
```

## Error Handling

### Common Validation Checks

- Verify file identifier matches "SNGPKG"
- Ensure version is supported (currently only version 1)
- Validate that declared lengths don't exceed file boundaries
- Check that metadata key/value lengths are non-negative
- Verify file offsets and sizes are within bounds
- Ensure filenames are valid UTF-8 and reasonable length

### Recovery Strategies

- Skip invalid metadata entries rather than failing completely
- Log warnings for unknown metadata keys
- Handle files larger than expected by truncating
- Gracefully handle missing or corrupted file data

## Implementation Notes

### Performance Considerations

- The masking algorithm can be vectorized for better performance
- Pre-compute lookup tables for frequently accessed files
- Use memory-mapped files for large SNG files
- Stream processing for files that don't fit in memory

### Compatibility

- The format is designed to be forward-compatible
- Unknown metadata keys should be preserved
- Files with unrecognized extensions should be included
- Version field allows for future format evolution

### XOR Masking

- XOR masking is a simple bitwise operation
- Each file has a unique random XOR mask
- The same operation is used for both masking and unmasking

## Example File Structure

```
Offset   Size    Content
------   ----    -------
0x0000   6       "SNGPKG"
0x0006   4       Version: 1
0x000A   16      XOR Mask: [random bytes]
0x001A   8       Metadata Length: 150
0x0022   8       Metadata Count: 5
0x002A   ...     Metadata entries
0x00B8   8       File Index Length: 200  
0x00C0   8       File Count: 8
0x00C8   ...     File index entries
0x0190   8       File Section Length: 50000000
0x0198   ...     XOR-masked file data
```

This specification provides complete information for implementing SNG file readers and writers compatible with the Nautilus SngLib library.
