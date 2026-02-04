package optimizer

import (
	"encoding/binary"
	"fmt"
)

// PatchMoov updates the chunk offsets in the moov atom by the given displacement.
// It searches for 'stco' and 'co64' boxes and adjusts their values.
func PatchMoov(moov []byte, displacement int64) error {
	// We scan the moov byte slice for "stco" and "co64"
	// This is a heuristic scan, but safe enough if we validate box sizes.

	// A strictly correct parser would traverse the tree.
	// Given we have the whole moov, we can just search for the 4-byte tags.
	// But we must ensure 4-byte alignment relative to the box structure or at least check valid sizes.

	// Quick implementation: scan every byte?
	// To be safer and faster: parse the tree.
	// But implementing a full tree parser is heavy.
	// Let's do a linear scan but verify box structure.

	if displacement == 0 {
		return nil
	}

	for i := 0; i < len(moov)-8; i++ {
		// Look for boxes
		// Size (4), Type (4)
		// We only care about stco and co64

		// Check for stco
		if string(moov[i+4:i+8]) == "stco" {
			size := binary.BigEndian.Uint32(moov[i : i+4])
			// Verify size fits in buffer
			if int(size) > len(moov)-i {
				continue // False positive or truncated
			}
			if err := patchStco(moov[i:i+int(size)], displacement); err != nil {
				return err
			}
			// Skip this box
			i += int(size) - 1
			continue
		}

		// Check for co64
		if string(moov[i+4:i+8]) == "co64" {
			size := binary.BigEndian.Uint32(moov[i : i+4])
			// Verify size fits in buffer
			if int(size) > len(moov)-i {
				continue
			}
			if err := patchCo64(moov[i:i+int(size)], displacement); err != nil {
				return err
			}
			i += int(size) - 1
			continue
		}
	}
	return nil
}

func patchStco(box []byte, displacement int64) error {
	// Header: Size(4) Type(4) Version(1) Flags(3) Count(4)
	if len(box) < 16 {
		return fmt.Errorf("stco box too small")
	}
	count := binary.BigEndian.Uint32(box[12:16])

	// Entries start at 16, each 4 bytes
	if len(box) < 16+int(count)*4 {
		return fmt.Errorf("stco box truncated")
	}

	for j := 0; j < int(count); j++ {
		offset := 16 + j*4
		val := binary.BigEndian.Uint32(box[offset : offset+4])
		newVal := int64(val) + displacement
		if newVal > 0xFFFFFFFF {
			return fmt.Errorf("displacement causes stco overflow, needs co64 upgrade")
		}
		binary.BigEndian.PutUint32(box[offset:offset+4], uint32(newVal))
	}
	return nil
}

func patchCo64(box []byte, displacement int64) error {
	// Header: Size(4) Type(4) Version(1) Flags(3) Count(4)
	if len(box) < 16 {
		return fmt.Errorf("co64 box too small")
	}
	count := binary.BigEndian.Uint32(box[12:16])

	// Entries start at 16, each 8 bytes
	if len(box) < 16+int(count)*8 {
		return fmt.Errorf("co64 box truncated")
	}

	for j := 0; j < int(count); j++ {
		offset := 16 + j*8
		val := binary.BigEndian.Uint64(box[offset : offset+8])
		newVal := int64(val) + displacement
		// Check for negative? displacement can be negative if we move logic changes, but here it's likely positive.
		binary.BigEndian.PutUint64(box[offset:offset+8], uint64(newVal))
	}
	return nil
}
