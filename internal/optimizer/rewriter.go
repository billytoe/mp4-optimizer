package optimizer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"mp4-optimizer/pkg/atomic"
)

// ProgressCallback is a function type for progress updates
type ProgressCallback func(progress float64, message string)

// Optimize rearranges the MP4 atoms to move 'moov' to the front.
func Optimize(path string, callback ...ProgressCallback) error {
	var progressFn ProgressCallback
	if len(callback) > 0 && callback[0] != nil {
		progressFn = callback[0]
	}

	reportProgress := func(p float64, msg string) {
		if progressFn != nil {
			progressFn(p, msg)
		}
	}

	reportProgress(0, "开始处理...")

	// 1. Open original file for reading
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	reportProgress(10, "解析文件结构...")

	// 2. Parse atoms to find moov and its size
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

	reportProgress(20, "读取元数据...")

	// 3. Read the whole moov into memory
	if _, err := in.Seek(moovAtom.Offset, io.SeekStart); err != nil {
		return err
	}
	moovBuf := make([]byte, moovAtom.Size)
	if _, err := io.ReadFull(in, moovBuf); err != nil {
		return err
	}

	reportProgress(30, "处理元数据...")

	displacement := moovAtom.Size

	// 4. Parse/Patch moov
	if err := PatchMoov(moovBuf, displacement); err != nil {
		return fmt.Errorf("failed to patch moov: %w", err)
	}

	reportProgress(40, "创建临时文件...")

	// 5. Create temporary file in the same directory
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := filepath.Base(path)
	nameWithoutExt := base[:len(base)-len(ext)]

	tmpFile, err := os.CreateTemp(dir, nameWithoutExt+"_tmp_*"+ext)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		tmpFile.Close()
		if !success {
			os.Remove(tmpPath)
		}
	}()

	reportProgress(50, "写入优化文件...")

	// 6. Writing to temp file
	// 1. Write ftyp
	if foundFtyp {
		if _, err := in.Seek(ftypAtom.Offset, io.SeekStart); err != nil {
			return err
		}
		if _, err := io.CopyN(tmpFile, in, ftypAtom.Size); err != nil {
			return err
		}
	}

	reportProgress(60, "写入元数据...")

	// 2. Write patched moov
	if _, err := tmpFile.Write(moovBuf); err != nil {
		return err
	}

	reportProgress(70, "写入视频数据...")

	// 3. Write others (mdat, etc), skipping ftyp and moov
	totalAtoms := len(atoms)
	processedAtoms := 0
	for _, a := range atoms {
		if a.Type == "ftyp" || a.Type == "moov" {
			continue
		}

		if _, err := in.Seek(a.Offset, io.SeekStart); err != nil {
			return err
		}

		// Use CopyN or Copy depending on size
		if a.Size > 0 {
			if _, err := io.CopyN(tmpFile, in, a.Size); err != nil {
				return err
			}
		} else {
			// Size 0 means "until EOF"
			if _, err := io.Copy(tmpFile, in); err != nil {
				return err
			}
		}

		processedAtoms++
		progress := 70 + float64(processedAtoms)/float64(totalAtoms)*25
		reportProgress(progress, fmt.Sprintf("写入视频数据... %d/%d", processedAtoms, totalAtoms-2))
	}

	// 7. Sync and close temp file
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	tmpFile.Close()

	reportProgress(95, "完成...")

	// 8. Atomically replace original file with temp file
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to replace file: %w", err)
	}

	success = true
	reportProgress(100, "完成！")
	return nil
}
