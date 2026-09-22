package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

func normalize(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	if v[0] != 'v' {
		return "v" + v
	}
	return v
}

// CheckLatest reports whether a newer release than currentVersion is
// published. A "dev" build (the default when the binary wasn't built with
// -ldflags -X main.version=...) never updates, since there's no
// meaningful version to compare against.
func CheckLatest(ctx context.Context, currentVersion string) (release *Release, hasUpdate bool, err error) {
	if currentVersion == "" || currentVersion == "dev" {
		return nil, false, nil
	}

	release, err = FetchLatest(ctx)
	if err != nil {
		return nil, false, err
	}

	current, latest := normalize(currentVersion), normalize(release.TagName)
	if !semver.IsValid(current) || !semver.IsValid(latest) {
		return release, false, nil
	}
	return release, semver.Compare(latest, current) > 0, nil
}

// DownloadAndApply fetches the release's archive for the running platform,
// verifies it against checksums.txt, and replaces execPath with the new
// binary. The caller is responsible for restarting the process afterwards
// (see Restart) and for backing up any state first.
func DownloadAndApply(ctx context.Context, release *Release, execPath string) error {
	archiveAsset, ok := release.Asset(CurrentArchiveName())
	if !ok {
		return fmt.Errorf("release %s has no asset for %s/%s", release.TagName, runtime.GOOS, runtime.GOARCH)
	}
	checksumsAsset, ok := release.Asset("checksums.txt")
	if !ok {
		return fmt.Errorf("release %s has no checksums.txt", release.TagName)
	}

	archiveData, err := downloadAll(ctx, archiveAsset.URL)
	if err != nil {
		return fmt.Errorf("baixando %s: %w", archiveAsset.Name, err)
	}
	checksums, err := downloadAll(ctx, checksumsAsset.URL)
	if err != nil {
		return fmt.Errorf("baixando checksums.txt: %w", err)
	}
	if err := verifyChecksum(checksums, archiveAsset.Name, archiveData); err != nil {
		return err
	}

	newBinary, err := extractBinary(runtime.GOOS, archiveData)
	if err != nil {
		return err
	}

	return swapExecutable(execPath, newBinary)
}

// swapExecutable replaces the file at execPath with newData. Even on
// Windows, renaming a running executable is allowed (the OS keeps the
// running process attached to the old file); only deleting or overwriting
// its bytes in place is not, so the old file is renamed out of the way
// first and cleaned up on a best-effort basis (CleanupPrevious removes it
// on the next start once the old process is gone).
func swapExecutable(execPath string, newData []byte) error {
	oldPath := execPath + ".old"
	_ = os.Remove(oldPath)

	if err := os.Rename(execPath, oldPath); err != nil {
		return fmt.Errorf("renomeando executável atual: %w", err)
	}
	if err := os.WriteFile(execPath, newData, 0o755); err != nil {
		_ = os.Rename(oldPath, execPath) // best-effort rollback
		return fmt.Errorf("escrevendo novo executável: %w", err)
	}
	_ = os.Remove(oldPath) // fails harmlessly while the old process is still exiting
	return nil
}

// CleanupPrevious removes a leftover "<exe>.old" from a previous update
// that couldn't be deleted while the old process was still shutting down.
func CleanupPrevious(execPath string) {
	_ = os.Remove(execPath + ".old")
}

// Restart launches a new instance of execPath with the same arguments and
// exits the current process. The caller must have already released any
// exclusive resources (the DB connection, the HTTP listener) it wants the
// new instance to reacquire cleanly.
func Restart(execPath string, args []string) error {
	cmd := exec.Command(execPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
