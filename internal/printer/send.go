package printer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// printRawBytes hands the ESC/POS bytes to the OS to print, without any
// native printing library: on macOS/Linux via CUPS (`lp -o raw`), on
// Windows via a locally shared printer (`copy /b`) — the exact same
// approach src/printer.js used, so the one-time printer setup in the
// README (share the printer, note the share name) still applies unchanged.
func printRawBytes(printerName string, buffer []byte) error {
	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		return err
	}
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("pdv-print-%s.bin", hex.EncodeToString(suffix)))

	if err := os.WriteFile(tempFile, buffer, 0o644); err != nil {
		return err
	}
	defer os.Remove(tempFile)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "copy", "/b", tempFile, `\\localhost\`+printerName)
	} else {
		cmd = exec.Command("lp", "-d", printerName, "-o", "raw", tempFile)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(output)
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}
