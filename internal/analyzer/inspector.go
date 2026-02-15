package analyzer

import (
	"fmt"
	"os"

	"mp4-optimizer/pkg/atomic"
)

// CheckFastStart returns true if the MP4 file at path has 'moov' atom before 'mdat' atom.
// It also returns an error if the structure is invalid or atoms are missing.
func CheckFastStart(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	atoms, err := atomic.FindAtoms(f)
	if err != nil {
		return false, fmt.Errorf("parse atoms: %w", err)
	}

	var moovIndex, mdatIndex = -1, -1

	for i, a := range atoms {
		if a.Type == "moov" {
			moovIndex = i
		}
		if a.Type == "mdat" {
			mdatIndex = i
		}
	}

	if moovIndex == -1 {
		return false, fmt.Errorf("no moov atom found")
	}
	if mdatIndex == -1 {
		// If no mdat, technically it might be valid metadata-only or fragmented,
		// but usually we expect mdat.
		// However, if moov exists and no mdat, it's fast-start compatible (just metadata).
		// Let's assume true if moov is present.
		// But in typical video files, we want moov < mdat.
		return true, nil
	}

	return moovIndex < mdatIndex, nil
}

// ValidateFile checks if the MP4 file at path is complete and not truncated.
// Returns true if the file appears to be complete, false if truncated.
func ValidateFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	return atomic.ValidateFile(f)
}
