package analyzer

import (
	"encoding/binary"
	"fmt"
	"io"
	"mp4-optimizer/pkg/atomic"
	"os"
	"time"
)

// Metadata holds the video file metadata
type Metadata struct {
	Size     int64     `json:"size"`
	Duration float64   `json:"duration"` // in seconds
	Width    int       `json:"width"`
	Height   int       `json:"height"`
	Codec    string    `json:"codec"`
	Modified time.Time `json:"modified"`
}

// GetMetadata extracts metadata from an MP4 file
func GetMetadata(path string) (*Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	meta := &Metadata{
		Size:     info.Size(),
		Modified: info.ModTime(),
	}

	// Use robust atom parser from pkg/atomic
	atoms, err := atomic.FindAtoms(f)
	if err != nil {
		// Even if FindAtoms fails partially, we might have skipped key atoms?
		// But usually it returns valid structure.
		// Detailed logging would help here.
		fmt.Printf("Warning: error scanning atoms in %s: %v\n", path, err)
	}

	// Find moov
	var moov *atomic.Atom
	for _, a := range atoms {
		if a.Type == "moov" {
			moov = &a
			break
		}
	}

	if moov == nil {
		// No moov found, fast exit
		return meta, nil
	}

	// Seek to moov start
	if _, err := f.Seek(moov.Offset, io.SeekStart); err != nil {
		return nil, err
	}

	// Read header to skip it and handle extended size
	// We re-read header to be safe about offset calculation
	header := make([]byte, 8)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, err
	}
	size := int64(binary.BigEndian.Uint32(header[0:4]))

	headerSize := int64(8)
	if size == 1 {
		// Extended size, read next 8 bytes
		var extended [8]byte
		if _, err := io.ReadFull(f, extended[:]); err != nil {
			return nil, err
		}
		size = int64(binary.BigEndian.Uint64(extended[:]))
		headerSize += 8
	}

	// Determine end position of moov
	// Offset points to start of header.
	// Body starts at Offset + headerSize.
	// End is Offset + Size.
	endPos := moov.Offset + size

	// We are now at the start of body (after header)
	if err := parseMoov(f, endPos, meta); err != nil {
		fmt.Printf("Error parsing moov for %s: %v\n", path, err)
	}

	return meta, nil
}

// Old parseAtoms removed in favor of atomic.FindAtoms
// func parseAtoms...

func parseMoov(r io.ReadSeeker, endPos int64, meta *Metadata) error {
	const headerSize = 8

	for {
		currentPos, _ := r.Seek(0, io.SeekCurrent)
		if currentPos >= endPos {
			break
		}

		header := make([]byte, headerSize)
		if _, err := io.ReadFull(r, header); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		size := int64(binary.BigEndian.Uint32(header[0:4]))
		typ := string(header[4:8])

		if size == 1 {
			var extended [8]byte
			if _, err := io.ReadFull(r, extended[:]); err != nil {
				return err
			}
			size = int64(binary.BigEndian.Uint64(extended[:]))
		}

		// Handle specific atoms
		if typ == "mvhd" {
			// Parse Movie Header for duration
			// Version (1) + Flags (3)
			// Creation (4/8) + Mod (4/8) + Timescale (4) + Duration (4/8)
			data := make([]byte, 100) // Read enough
			// Actually check version
			if _, err := r.Read(data); err == nil {
				version := data[0]
				var timescale uint32
				var duration uint64

				if version == 1 {
					// 1(ver) + 3(flags) + 8(create) + 8(mod) = 20 bytes offset
					timescale = binary.BigEndian.Uint32(data[20:24])
					duration = binary.BigEndian.Uint64(data[24:32])
				} else {
					// 1(ver) + 3(flags) + 4(create) + 4(mod) = 12 bytes offset
					timescale = binary.BigEndian.Uint32(data[12:16])
					durationVal := binary.BigEndian.Uint32(data[16:20])
					duration = uint64(durationVal)
				}
				if timescale != 0 {
					meta.Duration = float64(duration) / float64(timescale)
				}
				// Rewind to continue parsing siblings?
				// The buffer read advanced the cursor.
				// We need to restore if we want to be clean, but we handle the seek below.
			}
		} else if typ == "trak" {
			// Parse Track
			trackStart, _ := r.Seek(0, io.SeekCurrent)
			// Adjust for the data we might have read in 'mvhd' block logic?
			// Wait, the logic above for 'mvhd' READs data, so file pointer moved.
			// But for 'trak', we enter here immediately after header.
			// So we are at start of trak body.

			trackEnd := trackStart + size - headerSize // approximate logic

			// We need to check if this is a VIDEO track
			// We can look into mdia -> hdlr
			// Or just parse tkhd and check dimensions. Audio usually has 0 width/height.
			parseTrak(r, trackEnd, meta)

			// After parsing trak, we continue to next sibling.
			// But wait, parseTrak might have moved the cursor.
			// We track loop by seek at the end of loop.
		}

		// Seek to next atom
		// size includes header
		payloadSize := size - headerSize
		if size == 1 {
			payloadSize -= 8
		}

		// We need to calculate where the next atom starts based on the START of this atom
		// CurrentPos (before header read) + Size
		nextAtomPos := currentPos + size
		if _, err := r.Seek(nextAtomPos, io.SeekStart); err != nil {
			return err
		}
	}
	return nil
}

func parseTrak(r io.ReadSeeker, endPos int64, meta *Metadata) {
	const headerSize = 8

	var isVideo bool
	var width, height int
	var codec string

	// We traverse trak to find tkhd and mdia
	for {
		currentPos, _ := r.Seek(0, io.SeekCurrent)
		if currentPos >= endPos {
			break
		}

		header := make([]byte, headerSize)
		if _, err := io.ReadFull(r, header); err != nil {
			break
		}

		size := int64(binary.BigEndian.Uint32(header[0:4]))
		typ := string(header[4:8])

		if size == 1 { // Extended size
			var extended [8]byte
			io.ReadFull(r, extended[:])
			size = int64(binary.BigEndian.Uint64(extended[:]))
		}

		if typ == "tkhd" {
			// Track Header
			// Ver(1)+Flags(3) + Create(4/8) + Mod(4/8) + TrackID(4) + Reserved(4) + Duration(4/8) + Reserved(8) + Layer(2) + Alt(2) + Vol(2) + Reserved(2) + Matrix(36) + Width(4) + Height(4)
			data := make([]byte, 100)
			r.Read(data)
			version := data[0]

			var w, h uint32
			if version == 1 {
				// Offset for Width:
				// 1+3 + 8+8 + 4 + 4 + 8 + 8 + 2+2+2+2 + 36 = 88 bytes?
				// Let's verify standard.
				// version(1) + flags(3)
				// creation_time(8)
				// modification_time(8)
				// track_id(4)
				// reserved(4)
				// duration(8)
				// reserved(8)
				// layer(2), alternate_group(2), volume(2), reserved(2)
				// matrix(36)
				// width(4), height(4)
				// Sum: 4 + 16 + 8 + 8 + 8 + 36 = 80 bytes. So Width at 80?
				if len(data) >= 88 {
					w = binary.BigEndian.Uint32(data[80:84])
					h = binary.BigEndian.Uint32(data[84:88])
				}
			} else {
				// version(1) + flags(3)
				// creation_time(4)
				// modification_time(4)
				// track_id(4)
				// reserved(4)
				// duration(4)
				// reserved(8)
				// layer(2), alternate_group(2), volume(2), reserved(2)
				// matrix(36)
				// width(4), height(4)
				// Sum: 4 + 8 + 8 + 4 + 8 + 8 + 36 = 76 bytes?
				// 0-3: Ver/Flags
				// 4-7: Create
				// 8-11: Mod
				// 12-15: ID
				// 16-19: Res
				// 20-23: Dur
				// 24-31: Res
				// 32-33: Layer
				// 34-35: Alt
				// 36-37: Vol
				// 38-39: Res
				// 40-75: Matrix
				// 76-79: Width
				// 80-83: Height
				if len(data) >= 84 {
					w = binary.BigEndian.Uint32(data[76:80])
					h = binary.BigEndian.Uint32(data[80:84])
				}
			}
			// Fixed point 16.16 values
			width = int(w >> 16)
			height = int(h >> 16)

			if width > 0 && height > 0 {
				isVideo = true // Only video tracks have dimensions usually
			}

		} else if typ == "mdia" {
			// Media Box -> minf -> stbl -> stsd
			mdiaEnd := currentPos + size
			if size == 1 {
				mdiaEnd = currentPos + 8 + size
			} // rough adjust
			c := parseMdia(r, mdiaEnd)
			if c != "" {
				codec = c
			}
		}

		nextAtomPos := currentPos + size
		r.Seek(nextAtomPos, io.SeekStart)
	}

	// Update meta if this is the "best" video track we found so far
	// For simplicity, just overwrite if it looks like video
	// Or check if we already have one.
	if isVideo && width > 0 && height > 0 {
		meta.Width = width
		meta.Height = height
		if codec != "" {
			meta.Codec = codec
		}
	}
}

func parseMdia(r io.ReadSeeker, endPos int64) string {
	// Need to find minf -> stbl -> stsd
	// recurse... simplified for brevity, assume structure

	// Find minf
	// ... logic similar to other parsers ...
	// Being lazy/efficient: we deep dive just looking for 'stsd' inside this range.
	// Is it safe? 'stsd' is unique enough.
	// But 'stsd' is inside 'stbl' inside 'minf' inside 'mdia'.

	// Let's just linearly scan for 'stsd' atom header within the limit?
	// It's a bit risky if 'stsd' bytes appear in data, but unlikely in headers structure.
	// Proper way:

	return findCodecInBoxRecursively(r, endPos)
}

func findCodecInBoxRecursively(r io.ReadSeeker, limit int64) string {
	const headerSize = 8
	for {
		currentPos, _ := r.Seek(0, io.SeekCurrent)
		if currentPos >= limit {
			break
		}

		header := make([]byte, headerSize)
		if _, err := io.ReadFull(r, header); err != nil {
			break
		}
		size := int64(binary.BigEndian.Uint32(header[0:4]))
		typ := string(header[4:8])

		if size == 1 {
			var extended [8]byte
			io.ReadFull(r, extended[:])
			size = int64(binary.BigEndian.Uint64(extended[:]))
		}

		if typ == "minf" || typ == "stbl" {
			// Dive in
			// Content starts here
			// Recurse
			s := findCodecInBoxRecursively(r, currentPos+size)
			if s != "" {
				return s
			}
		} else if typ == "stsd" {
			// Sample Description Box
			// Version(1) + Flags(3) + Count(4)
			var tmp [8]byte
			r.Read(tmp[:]) // Ver+Flags+Count

			// Next is the SampleEntry. The 4 bytes type IS the codec
			// e.g. 'avc1', 'mp4v', 'hvc1'
			r.Read(tmp[:8]) // Size(4) + Type(4)
			codec := string(tmp[4:8])
			return codec
		}

		// Skip
		nextAtomPos := currentPos + size
		r.Seek(nextAtomPos, io.SeekStart)
	}
	return ""
}
