package shared

import (
	"fmt"
	"os/exec"
	"runtime"
)

var getRuntime = func() string { return runtime.GOOS }

// OpenBrowser opens the default system browser to the specified URL.
//
// Supports macOS, Linux, and Windows platforms.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	rt := getRuntime()
	switch rt {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", rt)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	return nil
}
