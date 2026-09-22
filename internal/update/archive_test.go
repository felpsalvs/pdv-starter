package update

import (
	"os"
	"path/filepath"
	"testing"
)

// These tests exercise verifyChecksum/extractBinary against real
// goreleaser output (dist-release/, produced by `goreleaser release
// --snapshot --clean --skip=publish`), so they're skipped when that
// directory hasn't been built locally rather than failing the suite.
func distReleaseDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "dist-release")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("dist-release/ not built — run `goreleaser release --snapshot --clean --skip=publish` first")
	}
	return dir
}

func TestExtractAndVerifyRealArchives(t *testing.T) {
	dir := distReleaseDir(t)
	checksums, err := os.ReadFile(filepath.Join(dir, "checksums.txt"))
	if err != nil {
		t.Fatal(err)
	}

	for _, goos := range []string{"linux", "darwin", "windows"} {
		name := ArchiveName(goos, "amd64")
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if err := verifyChecksum(checksums, name, data); err != nil {
			t.Errorf("%s: checksum: %v", name, err)
		}

		bin, err := extractBinary(goos, data)
		if err != nil {
			t.Fatalf("%s: extract: %v", name, err)
		}
		if len(bin) == 0 {
			t.Errorf("%s: extracted binary is empty", name)
		}
	}
}

func TestVerifyChecksumRejectsTamperedData(t *testing.T) {
	dir := distReleaseDir(t)
	checksums, err := os.ReadFile(filepath.Join(dir, "checksums.txt"))
	if err != nil {
		t.Fatal(err)
	}

	name := ArchiveName("linux", "amd64")
	if err := verifyChecksum(checksums, name, []byte("not the real archive")); err == nil {
		t.Error("expected checksum mismatch error, got nil")
	}
}
