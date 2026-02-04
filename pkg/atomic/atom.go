package atomic

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Atom represents an MP4 box header and its offset in the file.
type Atom struct {
	Offset int64
	Size   int64
	Type   string
}

// ReadAtomHeader reads the next atom header from the reader.
// It returns the atom (with scanned size and type) and the number of bytes read for the header.
// Does NOT read the body.
func ReadAtomHeader(r io.Reader) (Atom, int64, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Atom{}, 0, err
	}

	size := int64(binary.BigEndian.Uint32(header[0:4]))
	typ := string(header[4:8])
	bytesRead := int64(8)

	if size == 1 {
		// Extended size (64-bit)
		var extended [8]byte
		if _, err := io.ReadFull(r, extended[:]); err != nil {
			return Atom{Size: size, Type: typ}, bytesRead, err
		}
		size = int64(binary.BigEndian.Uint64(extended[:]))
		bytesRead += 8
	}

	return Atom{Size: size, Type: typ}, bytesRead, nil
}

// FindAtoms scans the top-level atoms in the file.
func FindAtoms(rs io.ReadSeeker) ([]Atom, error) {
	var atoms []Atom
	
	// Ensure we are at the beginning
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	for {
		offset, err := rs.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}

		atom, headerLen, err := ReadAtomHeader(rs)
		if err == io.EOF {
			break
		}
		if err != nil {
			return atoms, err
		}

		atom.Offset = offset
		atoms = append(atoms, atom)

		// Seek to next atom
		// Size includes the header.
		// So we seek forward by Size - headerLen
		// Note: size=0 means "until end of file", typical for mdat as last atom.
		if atom.Size == 0 {
			// This is the last atom
			break
		}

		nextOffset := atom.Size - headerLen
		if nextOffset < 0 {
			return atoms, fmt.Errorf("invalid atom size %d at offset %d", atom.Size, offset)
		}
		
		if _, err := rs.Seek(nextOffset, io.SeekCurrent); err != nil {
			// If we can't seek (EOF?), maybe we are done or file is truncated
			if err == io.EOF {
				break
			}
			return atoms, err
		}
	}

	return atoms, nil
}
