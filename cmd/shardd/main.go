// Command shardd is the desktop daemon: it serves the Shard gRPC
// API to the app, mounts the vault as a folder when unlocked and syncs in
// the background.
//
// The app finds the daemon through endpoint.json in the config dir:
// {"address": "unix:///run/user/1000/selfhold-shard.sock", "token": "..."}.
// The file is readable only by the user; every call must carry the token.
//
// Other apps (docs/CONNECTOR.md) reach the Connector API on a socket of its
// own, found through connector.json in the same dir ({"address": ...}, no
// secret): it serves only app records, to apps the user paired.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/winfsp/cgofuse/fuse"
	"google.golang.org/grpc"

	"github.com/SelfHold/shard/core/connector"
	cpb "github.com/SelfHold/shard/core/connector/pb"
	"github.com/SelfHold/shard/core/crypto"
	"github.com/SelfHold/shard/core/engine"
	"github.com/SelfHold/shard/core/eventlog"
	"github.com/SelfHold/shard/core/mount"
	"github.com/SelfHold/shard/core/rpc"
	"github.com/SelfHold/shard/core/rpc/pb"
	"github.com/SelfHold/shard/core/vault"
)

func main() {
	cfgDir, _ := os.UserConfigDir()
	home, _ := os.UserHomeDir()
	base := filepath.Join(cfgDir, "selfhold", "shard")
	vaultPath := flag.String("vault", filepath.Join(base, "vault.shard"), "vault file")
	dataRoot := flag.String("data", base, "device data root")
	mountDir := flag.String("mount", filepath.Join(home, "Shard"), "where to show the vault (empty: do not mount)")
	listen := flag.String("listen", defaultListen(base), "unix:PATH or tcp:HOST:PORT")
	endpoint := flag.String("endpoint", filepath.Join(base, "endpoint.json"), "where to write address and token for the app")
	connListen := flag.String("connector-listen", connector.DefaultListen(base), "Connector API for other apps: unix:PATH or tcp:HOST:PORT (empty: off)")
	connEndpoint := flag.String("connector-endpoint", filepath.Join(base, "connector.json"), "where to write the Connector API address")
	device := flag.String("device", "", "device name (default: hostname)")
	logFile := flag.String("log", filepath.Join(base, "shardd.log"), "log file (empty: stderr only)")
	flag.Parse()

	out := io.Writer(os.Stderr)
	if *logFile != "" {
		if f, err := openLog(*logFile); err == nil {
			defer f.Close()
			out = io.MultiWriter(os.Stderr, f)
		}
	}
	logger := log.New(out, "shardd: ", log.LstdFlags)
	svc := rpc.NewService(*vaultPath, *dataRoot, runtime.GOOS)
	svc.Log = eventlog.New(1000, out)

	// The folder chosen in the app wins over the default, not over -mount.
	settings := filepath.Join(base, "daemon.json")
	dir := *mountDir
	explicit := false
	flag.Visit(func(f *flag.Flag) { explicit = explicit || f.Name == "mount" })
	if saved, ok := loadMountDir(settings); ok && !explicit {
		dir = saved
	}
	m := &mounter{dir: dir, staging: *dataRoot, log: logger}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	svc.Hooks = rpc.Hooks{
		OnUnlock: m.mount, OnLock: m.unmount, OnSync: m.retry,
		MountDir: m.getDir,
		SetMountDir: func(d string) error {
			m.setDir(d)
			return saveMountDir(settings, d)
		},
		Shutdown: stop, // "Stop" in the app: the same as SIGTERM
	}

	lis, err := listenOn(*listen)
	if err != nil {
		logger.Fatal(err)
	}
	token := crypto.RandomID(32)
	if err := writeEndpoint(*endpoint, lis, token); err != nil {
		logger.Fatal(err)
	}
	srv := grpc.NewServer(rpc.TokenAuth(token)...)
	pb.RegisterShardServer(srv, svc)

	var conn *grpc.Server
	if *connListen != "" {
		clis, err := listenOn(*connListen)
		if err != nil {
			logger.Fatal(err)
		}
		conn = grpc.NewServer()
		cpb.RegisterConnectorServer(conn, svc.Connector())
		if err := connector.WriteEndpoint(*connEndpoint, address(clis)); err != nil {
			logger.Fatal(err)
		}
		go func() {
			logger.Printf("connector on %s", clis.Addr())
			if err := conn.Serve(clis); err != nil {
				logger.Print(err)
			}
		}()
	}

	if pw := os.Getenv("SHARD_PASSWORD"); pw != "" {
		os.Unsetenv("SHARD_PASSWORD")
		v, err := vault.Open(*vaultPath, []byte(pw))
		if err != nil {
			logger.Fatal(err)
		}
		if err := svc.OpenWith(ctx, v, *device); err != nil {
			logger.Fatal(err)
		}
		logger.Printf("vault unlocked")
	}

	go func() {
		logger.Printf("listening on %s", lis.Addr())
		if err := srv.Serve(lis); err != nil {
			logger.Print(err)
		}
	}()
	<-ctx.Done()
	logger.Print("shutting down")
	svc.Close(context.Background())
	// Open Watch streams would hold GracefulStop forever.
	stopped := make(chan struct{})
	go func() { srv.GracefulStop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		srv.Stop()
	}
	os.Remove(*endpoint)
	if conn != nil {
		conn.Stop() // app Watch streams end; apps reconnect to the next run
		os.Remove(*connEndpoint)
		if p, ok := strings.CutPrefix(*connListen, "unix:"); ok {
			os.Remove(p)
		}
	}
}

func defaultListen(base string) string {
	if runtime.GOOS == "windows" {
		return "tcp:127.0.0.1:0"
	}
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = base
	}
	return "unix:" + filepath.Join(dir, "selfhold-shard.sock")
}

func listenOn(addr string) (net.Listener, error) {
	switch {
	case strings.HasPrefix(addr, "unix:"):
		p := strings.TrimPrefix(addr, "unix:")
		if len(p) > 100 {
			return nil, fmt.Errorf("socket path too long for unix sockets (%d bytes): %s; pass a shorter -listen", len(p), p)
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			return nil, err
		}
		if c, err := net.Dial("unix", p); err == nil {
			c.Close()
			return nil, fmt.Errorf("another shardd is running on %s", p)
		}
		os.Remove(p)
		l, err := net.Listen("unix", p)
		if err == nil {
			os.Chmod(p, 0o600)
		}
		return l, err
	case strings.HasPrefix(addr, "tcp:"):
		return net.Listen("tcp", strings.TrimPrefix(addr, "tcp:"))
	}
	return nil, fmt.Errorf("bad listen address %q", addr)
}

// address is a listener's address as endpoint files carry it.
func address(lis net.Listener) string {
	addr := lis.Addr().String()
	if lis.Addr().Network() == "unix" {
		addr = "unix://" + addr
	}
	return addr
}

func writeEndpoint(path string, lis net.Listener, token string) error {
	data, _ := json.MarshalIndent(map[string]string{"address": address(lis), "token": token}, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// openLog opens the log for appending; a log over 5 MB starts over.
func openLog(path string) (*os.File, error) {
	if fi, err := os.Stat(path); err == nil && fi.Size() > 5<<20 {
		os.Rename(path, path+".1")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
}

func loadMountDir(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var s struct {
		Mount *string `json:"mount"`
	}
	if json.Unmarshal(data, &s) != nil || s.Mount == nil {
		return "", false
	}
	return *s.Mount, true
}

func saveMountDir(path, dir string) error {
	data, _ := json.MarshalIndent(map[string]string{"mount": dir}, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// mounter mounts the vault while it is unlocked.
type mounter struct {
	dir, staging string
	log          *log.Logger

	mu   sync.Mutex
	host *fuse.FileSystemHost
	fs   *mount.FS
	done <-chan bool
}

func (m *mounter) getDir() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dir
}

func (m *mounter) setDir(d string) {
	m.mu.Lock()
	m.dir = d
	m.mu.Unlock()
}

func (m *mounter) mount(e *engine.Engine) (string, error) {
	dir := m.getDir()
	if dir == "" {
		return "", nil
	}
	if runtime.GOOS != "windows" { // WinFsp wants a path that does not exist
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	f, err := mount.New(e, filepath.Join(m.staging, "staging-"+e.Config().VaultID[:8]))
	if err != nil {
		return "", err
	}
	f.Log = m.log
	host, done := mount.Mount(f, dir)
	select {
	case ok := <-done:
		if !ok {
			return "", errors.New("mount failed (is FUSE / WinFsp installed, and is " + dir + " empty?)")
		}
		return "", errors.New("mount ended right away")
	case <-time.After(1500 * time.Millisecond):
	}
	m.mu.Lock()
	m.host, m.fs, m.done = host, f, done
	m.mu.Unlock()
	m.log.Printf("mounted at %s", dir)
	f.RetryQueued()
	return dir, nil
}

func (m *mounter) unmount() {
	m.mu.Lock()
	host, done := m.host, m.done
	m.host, m.fs = nil, nil
	m.mu.Unlock()
	if host == nil {
		return
	}
	select {
	case <-done: // already unmounted (cgofuse handles SIGINT itself)
		return
	default:
	}
	host.Unmount()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		m.log.Print("unmount timed out (folder busy?)")
	}
}

func (m *mounter) retry(*engine.Engine) {
	m.mu.Lock()
	f := m.fs
	m.mu.Unlock()
	if f != nil {
		if left := f.RetryQueued(); left > 0 {
			m.log.Printf("%d edited files still waiting for upload", left)
		}
	}
}
