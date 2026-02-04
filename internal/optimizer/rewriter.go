package optimizer

import (
	"fmt"
	"io"
	"os"

	"mp4-optimizer/pkg/atomic"
)

// Optimize rearranges the MP4 atoms to move 'moov' to the front.
func Optimize(path string) error {
	// 1. Rename to .bak
	bakPath := path + ".bak"
	if err := os.Rename(path, bakPath); err != nil {
		return fmt.Errorf("failed to backup file: %w", err)
	}

	success := false
	defer func() {
		if !success {
			// Restore backup
			os.Rename(bakPath, path)
		} else {
			// Remove backup
			os.Remove(bakPath)
		}
	}()

	// 2. Open .bak for reading
	in, err := os.Open(bakPath)
	if err != nil {
		return err
	}
	defer in.Close()

	// 3. Parse atoms to find moov and its size
	atoms, err := atomic.FindAtoms(in)
	if err != nil {
		return fmt.Errorf("failed to parse atoms: %w", err)
	}

	var moovAtom atomic.Atom
	var ftypAtom atomic.Atom
	foundMoov := false
	foundFtyp := false

	for _, a := range atoms {
		if a.Type == "moov" {
			moovAtom = a
			foundMoov = true
		}
		if a.Type == "ftyp" {
			ftypAtom = a
			foundFtyp = true
		}
	}

	if !foundMoov {
		return fmt.Errorf("no moov atom found")
	}

	// 4. Create new file
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	// 5. Strategy:
	// Write ftyp (if any)
	// Write moov (patched)
	// Write everything else (except ftyp and moov)

	// Calculate displacement
	// Original layout: [ftyp] ... [moov]
	// New layout:      [ftyp] [moov] ...
	// The content before moov (like mdat) is shifted by +moov_size
	// BUT wait.
	// If original was: [ftyp] [mdat] [moov]
	// mdat offset was: len(ftyp)
	// New mdat offset: len(ftyp) + len(moov)
	// So displacement = len(moov).

	// Validating this assumption:
	// Anything that WAS before moov is now pushed AFTER moov.
	// So its offset increases by len(moov).
	// Anything that WAS after moov (if any) ... ?
	// Usually moov is at the end.

	// We read the whole moov into memory
	if _, err := in.Seek(moovAtom.Offset, io.SeekStart); err != nil {
		return err
	}
	moovBuf := make([]byte, moovAtom.Size)
	if _, err := io.ReadFull(in, moovBuf); err != nil {
		return err
	}

	displacement := moovAtom.Size

	// Parse/Patch moov
	if err := PatchMoov(moovBuf, displacement); err != nil {
		return fmt.Errorf("failed to patch moov: %w", err)
	}

	// Writing
	// 1. Write ftyp
	if foundFtyp {
		if _, err := in.Seek(ftypAtom.Offset, io.SeekStart); err != nil {
			return err
		}
		if _, err := io.CopyN(out, in, ftypAtom.Size); err != nil {
			return err
		}
	}

	// 2. Write patched moov
	if _, err := out.Write(moovBuf); err != nil {
		return err
	}

	// 3. Write others (mdat, etc), skipping ftyp and moov
	for _, a := range atoms {
		if a.Type == "ftyp" || a.Type == "moov" {
			continue
		}

		if _, err := in.Seek(a.Offset, io.SeekStart); err != nil {
			return err
		}

		// Use CopyN or Copy depending on size
		if a.Size > 0 {
			if _, err := io.CopyN(out, in, a.Size); err != nil {
				return err
			}
		} else {
			// Size 0 means "until EOF"
			if _, err := io.Copy(out, in); err != nil {
				return err
			}
		}
	}

	success = true
	return nil
}
