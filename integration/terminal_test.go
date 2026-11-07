package integration_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/coder/websocket"
)

var serverBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "goslid-integration-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	serverBinary = filepath.Join(dir, "goslid")
	build := exec.Command("go", "build", "-o", serverBinary, "./cmd/goslid")
	build.Dir = ".."
	if output, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build goslid: %v\n%s", err, output)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type server struct {
	cmd     *exec.Cmd
	done    chan error
	url     string
	stopped bool
}

func startServer(t *testing.T, projectDir string) *server {
	t.Helper()
	log, err := os.CreateTemp(t.TempDir(), "goslid-stderr-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		log.Close()
		if t.Failed() {
			contents, _ := os.ReadFile(log.Name())
			t.Logf("goslid stderr: %s", contents)
		}
	})
	cmd := exec.Command(serverBinary, "-project-dir", projectDir, "-listen", "127.0.0.1:0")
	cmd.Stderr = log
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	s := &server{cmd: cmd, done: make(chan error, 1)}
	go func() { s.done <- cmd.Wait() }()
	t.Cleanup(func() { s.stop(t) })
	ready := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			ready <- scanner.Text()
		} else {
			ready <- ""
		}
	}()
	select {
	case line := <-ready:
		if !strings.HasPrefix(line, "listening ws://") {
			t.Fatalf("unexpected startup output %q", line)
		}
		s.url = strings.TrimPrefix(line, "listening ")
	case <-time.After(5 * time.Second):
		t.Fatal("goslid did not start")
	}
	return s
}

func (s *server) stop(t *testing.T) {
	t.Helper()
	if s.stopped {
		return
	}
	s.stopped = true
	select {
	case err := <-s.done:
		if err != nil {
			t.Errorf("goslid exited: %v", err)
		}
		return
	default:
	}
	if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-s.done:
		if err != nil {
			t.Errorf("goslid shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		s.cmd.Process.Kill()
		<-s.done
		t.Error("goslid did not stop")
	}
}

type client struct {
	conn *websocket.Conn
	ctx  context.Context
}

func connect(t *testing.T, s *server) *client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	conn, _, err := websocket.Dial(ctx, s.url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	c := &client{conn: conn, ctx: ctx}
	c.send(t, map[string]string{"type": "open"})
	return c
}

func (c *client) send(t *testing.T, message any) {
	t.Helper()
	data, err := json.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.conn.Write(c.ctx, websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
}

func (c *client) input(t *testing.T, data string) {
	t.Helper()
	c.send(t, map[string]string{"type": "input", "data": data})
}

func (c *client) until(t *testing.T, expected string) string {
	t.Helper()
	var output strings.Builder
	for !strings.Contains(output.String(), expected) {
		kind, data, err := c.conn.Read(c.ctx)
		if err != nil {
			t.Fatalf("read output waiting for %q: %v; received %q", expected, err, output.String())
		}
		if kind != websocket.MessageText {
			t.Fatalf("output is not a text message: %v", kind)
		}
		var message struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}
		if err := json.Unmarshal(data, &message); err != nil {
			t.Fatal(err)
		}
		if message.Type != "output" {
			t.Fatalf("unexpected message: %s", data)
		}
		output.WriteString(message.Data)
	}
	return output.String()
}

func TestTerminalStartsInConfiguredProject(t *testing.T) {
	project := t.TempDir()
	s := startServer(t, project)
	c := connect(t, s)
	c.input(t, "printf 'cwd:%s\\n' \"$PWD\"\r")
	c.until(t, "cwd:"+project+"\r\n")
}

func TestTerminalRetainsShellStateAndControlSequences(t *testing.T) {
	s := startServer(t, t.TempDir())
	c := connect(t, s)
	c.input(t, "gosli_value=from-first-input\r")
	c.input(t, "printf 'state:%s\\n' \"$gosli_value\"\r")
	c.until(t, "state:from-first-input\r\n")
	c.input(t, "sleep 0.05; printf '\\033[31mred\\033[0m\\n'\r")
	c.until(t, "\x1b[31mred\x1b[0m\r\n")
	c.input(t, "stty size\r")
	c.until(t, "24 80\r\n")
}

func TestTerminalDeliversFinalOutputBeforeShellExit(t *testing.T) {
	project := t.TempDir()
	s := startServer(t, project)
	c := connect(t, s)
	pid := shellPID(t, c, project)
	c.input(t, "printf '%20000s' ''; printf '\\146inal-marker\\n'; exit\r")
	c.until(t, "final-marker\r\n")
	waitProcessExit(t, pid)
}

func TestServerShutdownTerminatesAttachedAndDetachedShells(t *testing.T) {
	project := t.TempDir()
	s := startServer(t, project)
	c := connect(t, s)
	pid := shellPID(t, c, project)
	detached := connect(t, s)
	detachedPID := shellPID(t, detached, project)
	if err := detached.conn.Close(websocket.StatusNormalClosure, "leave session running"); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(detachedPID, 0); err != nil {
		t.Fatalf("detached shell stopped before server shutdown: %v", err)
	}
	s.stop(t)
	waitProcessExit(t, pid)
	waitProcessExit(t, detachedPID)
}

func TestDetachedTerminalDrainsOutputAndExits(t *testing.T) {
	project := t.TempDir()
	s := startServer(t, project)
	c := connect(t, s)
	pid := shellPID(t, c, project)
	c.input(t, "while [ ! -e continue ]; do sleep 0.01; done; printf '%1048576s' ''; printf completed > finished; exit\r")
	if err := c.conn.Close(websocket.StatusNormalClosure, "disconnect before continuing work"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "finished")); !os.IsNotExist(err) {
		t.Fatalf("work completed before releasing the post-disconnect barrier: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "continue"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		data, err := os.ReadFile(filepath.Join(project, "finished"))
		if err == nil && string(data) == "completed" {
			break
		}
		select {
		case <-deadline:
			t.Fatal("detached terminal did not drain output and finish its work")
		case <-tick.C:
		}
	}
	waitProcessExit(t, pid)
}

func shellPID(t *testing.T, c *client, project string) int {
	t.Helper()
	c.input(t, "printf '%s' \"$$\" > shell.pid; printf 'shell-%s\\n' ready\r")
	c.until(t, "shell-ready\r\n")
	data, err := os.ReadFile(filepath.Join(project, "shell.pid"))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatal(err)
	}
	return pid
}

func waitProcessExit(t *testing.T, pid int) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("process %d still exists", pid)
		case <-tick.C:
		}
	}
}

func TestServerRejectsNonLoopbackListener(t *testing.T) {
	cmd := exec.Command(serverBinary, "-listen", "0.0.0.0:8080")
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "loopback") {
		t.Fatalf("non-loopback listener was not rejected: %v, %s", err, output)
	}
}
