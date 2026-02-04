package analyzer

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

// Helper to create valid atom header
func makeAtom(typ string, size uint32) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[0:4], size)
	copy(buf[4:8], []byte(typ))
	return buf
}

func TestCheckFastStart(t *testing.T) {
	// 1. Create a tmp file for Fast Start (moov before mdat)
	// Structure: [ftyp 12] [moov 8] [mdat 8]
	buf := bytes.Buffer{}
	buf.Write(makeAtom("ftyp", 12)) // 8 header + 4 data
	buf.Write([]byte{0, 0, 0, 0})
	buf.Write(makeAtom("moov", 8))
	buf.Write(makeAtom("mdat", 8))

	tmpFast, _ := os.CreateTemp("", "fast*.mp4")
	defer os.Remove(tmpFast.Name())
	tmpFast.Write(buf.Bytes())
	tmpFast.Close()

	isFast, err := CheckFastStart(tmpFast.Name())
	if err != nil {
		t.Fatalf("CheckFastStart failed: %v", err)
	}
	if !isFast {
		t.Errorf("Expected fast start, got false")
	}

	// 2. Create a tmp file for Slow Start (mdat before moov)
	// Structure: [ftyp 12] [mdat 8] [moov 8]
	buf2 := bytes.Buffer{}
	buf2.Write(makeAtom("ftyp", 12))
	buf2.Write([]byte{0, 0, 0, 0})
	buf2.Write(makeAtom("mdat", 8))
	buf2.Write(makeAtom("moov", 8))

	tmpSlow, _ := os.CreateTemp("", "slow*.mp4")
	defer os.Remove(tmpSlow.Name())
	tmpSlow.Write(buf2.Bytes())
	tmpSlow.Close()

	isFast, err = CheckFastStart(tmpSlow.Name())
	if err != nil {
		t.Fatalf("CheckFastStart failed: %v", err)
	}
	if isFast {
		t.Errorf("Expected slow start, got true")
	}
}
