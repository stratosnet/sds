package setting

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

var IpcEndpoint string

func SetIPCEndpoint(homePath string) {
	IpcEndpoint = DefaultIPCEndpoint(homePath)
}

// DefaultIPCEndpoint returns the IPC path used by default.
func DefaultIPCEndpoint(homePath string) string {
	return IPCEndpoint(DefaultDataDir(homePath), "sds.ipc")
}

// IPCEndpoint resolves an IPC endpoint based on a configured value, taking into
// account the set data folders as well as the designated platform we're currently
// running on.
func IPCEndpoint(dataDir string, ipcPath string) string {
	// On windows we can only use plain top-level pipes
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(ipcPath, `\\.\pipe\`) {
			return ipcPath
		}
		return `\\.\pipe\` + ipcPath
	}
	// Resolve names into the data directory full paths otherwise
	if filepath.Base(ipcPath) == ipcPath {
		if dataDir == "" {
			return filepath.Join(os.TempDir(), ipcPath)
		}
		return filepath.Join(dataDir, ipcPath)
	}
	return ipcPath
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir(homePath string) string {
	// Try to place the data folder in the user's home dir
	home := homePath
	if homePath == "" {
		home = homeDir()
	}
	if home != "" {
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(home, "Library", "Stratos_sds")
		case "windows":
			// We used to put everything in %HOME%\AppData\Roaming, but this caused
			// problems with non-typical setups. If this fallback location exists and
			// is non-empty, use it, otherwise DTRT and check %LOCALAPPDATA%.
			fallback := filepath.Join(home, "AppData", "Roaming", "Stratos_sds")
			appdata := windowsAppData()
			if appdata == "" || isNonEmptyDir(fallback) {
				return fallback
			}
			return filepath.Join(appdata, "Stratos_sds")
		default:
			return filepath.Join(home, ".stratos_sds")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

func windowsAppData() string {
	v := os.Getenv("LOCALAPPDATA")
	if v == "" {
		// Windows XP and below don't have LocalAppData. Crash here because
		// we don't support Windows XP and undefining the variable will cause
		// other issues.
		panic("environment variable LocalAppData is undefined")
	}
	return v
}

func isNonEmptyDir(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		return false
	}
	names, _ := f.Readdir(1)
	f.Close()
	return len(names) > 0
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
