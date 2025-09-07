// Package main provides functionality for reading SNG (Song Package) files.
//
// SNG files are binary container formats used in music games to store complete
// song packages including chart files, audio stems, images, and metadata.
// All file data is XOR-masked for simple obfuscation.
//
// Basic Usage:
//
//	// Open an SNG file
//	sng, err := OpenSngFile("song.sng")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer sng.Close()
//
//	// Get metadata
//	metadata := sng.GetMetadata()
//	fmt.Printf("Song: %s by %s\n", metadata["name"], metadata["artist"])
//
//	// List contained files
//	files := sng.ListFiles()
//	for _, filename := range files {
//		fmt.Println("Contains:", filename)
//	}
//
//	// Read a specific file
//	midiData, err := sng.ReadFile("notes.mid")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("MIDI file size: %d bytes\n", len(midiData))
//
// File Format:
//
// SNG files contain four main sections:
//   - Header: File identifier, version, and XOR mask
//   - Metadata: Key-value pairs with song information
//   - File Index: List of contained files with sizes and offsets
//   - File Data: XOR-masked file contents
//
// The XOR masking uses a lookup table approach where each byte is masked
// based on its position within the individual file and a 16-byte mask
// from the header.
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// SngFileIdentifier is the magic bytes that identify an SNG file
	SngFileIdentifier = "SNGPKG"
	// SngHeaderSize is the size of the SNG file header in bytes
	SngHeaderSize = 26
)

// SngHeader represents the SNG file header containing identification and XOR mask
type SngHeader struct {
	Identifier [6]byte  // Must be "SNGPKG"
	Version    uint32   // Format version (currently 1)
	XorMask    [16]byte // 16-byte mask for XOR operations
}

// SngMetadata represents key-value pairs of song metadata
type SngMetadata map[string]string

// SngFileEntry represents a file contained within the SNG package
type SngFileEntry struct {
	Filename string // Name of the file
	Size     uint64 // Size of the file data in bytes
	Offset   uint64 // Absolute offset to the file data within the SNG file
}

// MergedAudio represents a temporary merged audio file with cleanup capabilities
type MergedAudio struct {
	FilePath string       // Path to the temporary merged audio file
	cleanup  func() error // Cleanup function to remove temp files
}

// Close removes temporary files and cleans up resources
func (ma *MergedAudio) Close() error {
	if ma.cleanup != nil {
		return ma.cleanup()
	}
	return nil
}

// SngFile represents an opened SNG file with its header, metadata, file index, and reader
type SngFile struct {
	Header   SngHeader      // SNG file header
	Metadata SngMetadata    // Song metadata key-value pairs
	Files    []SngFileEntry // Index of contained files
	reader   *os.File       // File reader for accessing file data
}

// OpenSngFile opens an SNG file for reading and parses its header, metadata, and file index.
// The returned SngFile must be closed with Close() when finished.
//
// Returns an error if the file cannot be opened, is not a valid SNG file,
// or if any section cannot be parsed correctly.
func OpenSngFile(filename string) (*SngFile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	sng := &SngFile{
		reader:   file,
		Metadata: make(SngMetadata),
	}

	if err := sng.readHeader(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	if err := sng.readMetadata(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	if err := sng.readFileIndex(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to read file index: %w", err)
	}

	return sng, nil
}

// Close closes the underlying file reader. It should be called when finished
// with the SngFile to free system resources.
func (s *SngFile) Close() error {
	if s.reader != nil {
		return s.reader.Close()
	}
	return nil
}

func (s *SngFile) readHeader() error {
	if _, err := s.reader.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if err := binary.Read(s.reader, binary.LittleEndian, &s.Header); err != nil {
		return err
	}

	if string(s.Header.Identifier[:]) != SngFileIdentifier {
		return fmt.Errorf("invalid file identifier: %s", string(s.Header.Identifier[:]))
	}

	return nil
}

func (s *SngFile) readMetadata() error {
	var metadataLength uint64
	if err := binary.Read(s.reader, binary.LittleEndian, &metadataLength); err != nil {
		return err
	}

	var metadataCount uint64
	if err := binary.Read(s.reader, binary.LittleEndian, &metadataCount); err != nil {
		return err
	}

	for i := uint64(0); i < metadataCount; i++ {
		var keyLen int32
		if err := binary.Read(s.reader, binary.LittleEndian, &keyLen); err != nil {
			return err
		}

		if keyLen < 0 || keyLen > 1024 {
			return fmt.Errorf("invalid key length: %d", keyLen)
		}

		key := make([]byte, keyLen)
		if _, err := io.ReadFull(s.reader, key); err != nil {
			return err
		}

		var valueLen int32
		if err := binary.Read(s.reader, binary.LittleEndian, &valueLen); err != nil {
			return err
		}

		if valueLen < 0 || valueLen > 10240 {
			return fmt.Errorf("invalid value length: %d", valueLen)
		}

		value := make([]byte, valueLen)
		if _, err := io.ReadFull(s.reader, value); err != nil {
			return err
		}

		s.Metadata[string(key)] = string(value)
	}

	return nil
}

func (s *SngFile) readFileIndex() error {
	var indexLength uint64
	if err := binary.Read(s.reader, binary.LittleEndian, &indexLength); err != nil {
		return err
	}

	var fileCount uint64
	if err := binary.Read(s.reader, binary.LittleEndian, &fileCount); err != nil {
		return err
	}

	for i := uint64(0); i < fileCount; i++ {
		var filenameLen uint8
		if err := binary.Read(s.reader, binary.LittleEndian, &filenameLen); err != nil {
			return err
		}

		filename := make([]byte, filenameLen)
		if _, err := io.ReadFull(s.reader, filename); err != nil {
			return err
		}

		var fileSize uint64
		if err := binary.Read(s.reader, binary.LittleEndian, &fileSize); err != nil {
			return err
		}

		var fileOffset uint64
		if err := binary.Read(s.reader, binary.LittleEndian, &fileOffset); err != nil {
			return err
		}

		entry := SngFileEntry{
			Filename: string(filename),
			Size:     fileSize,
			Offset:   fileOffset,
		}

		s.Files = append(s.Files, entry)
	}

	return nil
}

// ListFiles returns a slice containing the filenames of all files stored in the SNG package.
// The order matches the order in the file index.
func (s *SngFile) ListFiles() []string {
	files := make([]string, len(s.Files))
	for i, entry := range s.Files {
		files[i] = entry.Filename
	}
	return files
}

// ReadFile extracts and returns the contents of the specified file from the SNG package.
// The file data is automatically unmasked using the XOR algorithm.
//
// Returns an error if the file is not found in the package or if there's an I/O error.
//
// Common files found in SNG packages include:
//   - "notes.mid" - MIDI chart data
//   - "song.opus", "guitar.opus", etc. - Audio stems
//   - "album.jpg" - Album artwork
//   - "song.ini" - Additional metadata
func (s *SngFile) ReadFile(filename string) ([]byte, error) {
	var entry *SngFileEntry
	for i := range s.Files {
		if s.Files[i].Filename == filename {
			entry = &s.Files[i]
			break
		}
	}

	if entry == nil {
		return nil, fmt.Errorf("file not found: %s", filename)
	}

	if _, err := s.reader.Seek(int64(entry.Offset), io.SeekStart); err != nil {
		return nil, err
	}

	maskedData := make([]byte, entry.Size)
	if _, err := io.ReadFull(s.reader, maskedData); err != nil {
		return nil, err
	}

	return s.unmaskData(maskedData), nil
}

// unmaskData applies a XOR unmasking algorithm to decode file data.
// The algorithm uses a 256-byte lookup table created from the 16-byte XOR mask
// in the header. Each byte is unmasked based on its position within the file.
func (s *SngFile) unmaskData(maskedData []byte) []byte {
	lookup := make([]byte, 256)
	for i := 0; i < 256; i++ {
		lookup[i] = byte(i) ^ s.Header.XorMask[i&0x0F]
	}

	unmaskedData := make([]byte, len(maskedData))
	for i, maskedByte := range maskedData {
		unmaskedData[i] = maskedByte ^ lookup[i&0xFF]
	}

	return unmaskedData
}

// GetMetadata returns a copy of all metadata key-value pairs from the SNG file.
// The returned map is safe to modify without affecting the original SngFile.
//
// Common metadata keys include:
//   - "name" - Song title
//   - "artist" - Artist name
//   - "album" - Album name
//   - "genre" - Musical genre
//   - "year" - Release year
//   - "charter" - Chart creator
//   - "song_length" - Duration in milliseconds
//   - "diff_guitar", "diff_bass", etc. - Difficulty ratings (0-7)
//   - "preview_start_time" - Preview start time in milliseconds
func (s *SngFile) GetMetadata() SngMetadata {
	result := make(SngMetadata)
	for k, v := range s.Metadata {
		result[k] = v
	}
	return result
}

// GetMergedAudio processes all opus files in the SNG and returns a merged audio file.
// Returns error if no opus files found or if merge fails - no fallback.
func (s *SngFile) GetMergedAudio() (*MergedAudio, error) {
	// Find all opus files in the SNG
	var opusFiles []string
	files := s.ListFiles()
	for _, filename := range files {
		if strings.HasSuffix(filename, ".opus") {
			opusFiles = append(opusFiles, filename)
		}
	}

	if len(opusFiles) == 0 {
		return nil, fmt.Errorf("no opus files found in SNG")
	}

	log.Printf("Found %d opus files to merge: %v", len(opusFiles), opusFiles)

	// Create temporary directory for conversion
	tempDir, err := os.MkdirTemp("", "sng-audio-merge-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Extract all opus files to temp directory
	var inputPaths []string
	for i, filename := range opusFiles {
		audioData, err := s.ReadFile(filename)
		if err != nil {
			os.RemoveAll(tempDir)
			return nil, fmt.Errorf("failed to read %s: %w", filename, err)
		}

		inputPath := filepath.Join(tempDir, fmt.Sprintf("input_%d.opus", i))
		if err := os.WriteFile(inputPath, audioData, 0644); err != nil {
			os.RemoveAll(tempDir)
			return nil, fmt.Errorf("failed to write temp file for %s: %w", filename, err)
		}
		inputPaths = append(inputPaths, inputPath)
	}

	// Create output path
	outputPath := filepath.Join(tempDir, "output.ogg")

	// Build ffmpeg command to merge all audio files
	args := []string{}

	// Add all input files
	for _, inputPath := range inputPaths {
		args = append(args, "-i", inputPath)
	}

	// Build the amerge filter complex string
	if len(inputPaths) > 1 {
		// Create filter complex for merging multiple inputs
		filterInputs := ""
		for i := range inputPaths {
			filterInputs += fmt.Sprintf("[%d:a]", i)
		}
		filterComplex := fmt.Sprintf("%samerge=inputs=%d[aout]", filterInputs, len(inputPaths))

		args = append(args,
			"-filter_complex", filterComplex,
			"-map", "[aout]",
		)
	} else {
		// Single file, just map it directly
		args = append(args, "-map", "0:a")
	}

	// Add output parameters
	args = append(args,
		"-ac", "2", // Stereo (2 channels)
		"-ar", "44100", // 44100 Hz sample rate
		"-c:a", "libvorbis", // Use Vorbis codec
		"-b:a", "128k", // ~128000 bps bitrate
		"-y", // Overwrite output file
		outputPath,
	)

	// Run ffmpeg to merge and convert
	log.Printf("Running ffmpeg to merge %d audio files", len(inputPaths))
	cmd := exec.Command("ffmpeg", args...)

	// Capture any error output
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("ffmpeg merge failed: %w", err)
	}

	log.Printf("Audio merge completed successfully")

	// Return MergedAudio with cleanup function
	mergedAudio := &MergedAudio{
		FilePath: outputPath,
		cleanup: func() error {
			return os.RemoveAll(tempDir)
		},
	}

	return mergedAudio, nil
}

