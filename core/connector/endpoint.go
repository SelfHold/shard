package connector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// EndpointFile is where apps find the desktop connector: <user config
// dir>/cloudsync/connector.json (~/.config/cloudsync, %AppData%\cloudsync,
// ~/Library/Application Support/cloudsync). It holds no secret.
func EndpointFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "cloudsync", "connector.json")
}

// DefaultListen is the desktop connector socket:
// $XDG_RUNTIME_DIR/cloudsync-connector.sock (else next to base), a
// localhost port on Windows.
func DefaultListen(base string) string {
	if runtime.GOOS == "windows" {
		return "tcp:127.0.0.1:0"
	}
	d := os.Getenv("XDG_RUNTIME_DIR")
	if d == "" {
		d = base
	}
	return "unix:" + filepath.Join(d, "cloudsync-connector.sock")
}

// Endpoint is the content of connector.json.
type Endpoint struct {
	Address    string `json:"address"` // unix:///path or host:port
	APIVersion int    `json:"api_version"`
}

// WriteEndpoint writes connector.json for address.
func WriteEndpoint(path, address string) error {
	data, _ := json.MarshalIndent(Endpoint{Address: address, APIVersion: APIVersion}, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	// Readable by the user only, like the rest of the config dir.
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
