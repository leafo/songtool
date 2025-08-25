package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	SngFileIdentifier = "SNGPKG"
	SngHeaderSize     = 26
)

type SngHeader struct {
	Identifier [6]byte
	Version    uint32
	XorMask    [16]byte
}

type SngMetadata map[string]string

type SngFileEntry struct {
	Filename string
	Size     uint64
	Offset   uint64
}

type SngFile struct {
	Header   SngHeader
	Metadata SngMetadata
	Files    []SngFileEntry
	reader   *os.File
}

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

func (s *SngFile) ListFiles() []string {
	files := make([]string, len(s.Files))
	for i, entry := range s.Files {
		files[i] = entry.Filename
	}
	return files
}

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

func (s *SngFile) GetMetadata() SngMetadata {
	result := make(SngMetadata)
	for k, v := range s.Metadata {
		result[k] = v
	}
	return result
}
