package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"runtime"
	"strings"
)

const binaryName = "pdv"

// ArchiveName is the release asset name for the running platform, matching
// the name_template in .goreleaser.yaml exactly: "pdv_<os>_<arch>.<ext>".
func ArchiveName(goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("pdv_%s_%s.%s", goos, goarch, ext)
}

func binaryFileName(goos string) string {
	if goos == "windows" {
		return binaryName + ".exe"
	}
	return binaryName
}

// verifyChecksum checks data against the "<sha256>  <name>" line for name
// inside checksums.txt's contents (goreleaser's default checksum file
// format), so a corrupted or tampered download is rejected before it ever
// touches disk as the new executable.
func verifyChecksum(checksums []byte, name string, data []byte) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])

	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[1] == name {
			if fields[0] != got {
				return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", name, fields[0], got)
			}
			return nil
		}
	}
	return fmt.Errorf("no checksum entry found for %s", name)
}

// extractBinary pulls the platform binary out of the downloaded archive
// (.zip on Windows, .tar.gz elsewhere).
func extractBinary(goos string, archiveData []byte) ([]byte, error) {
	want := binaryFileName(goos)
	if goos == "windows" {
		return extractFromZip(archiveData, want)
	}
	return extractFromTarGz(archiveData, want)
}

func extractFromZip(data []byte, want string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		if f.Name == want {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", want)
}

func extractFromTarGz(data []byte, want string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == want {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", want)
}

// CurrentArchiveName is ArchiveName() for the platform this binary is
// actually running on.
func CurrentArchiveName() string {
	return ArchiveName(runtime.GOOS, runtime.GOARCH)
}
