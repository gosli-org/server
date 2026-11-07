package terminal

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// Manager owns all terminal sessions until their processes exit or Close is called.
type Manager struct {
	dir      string
	mu       sync.Mutex
	sessions map[*Session]struct{}
	closed   bool
	wg       sync.WaitGroup
}

type Session struct {
	cmd      *exec.Cmd
	pty      *os.File
	output   chan []byte
	stop     chan struct{}
	readDone chan struct{}
	exited   chan struct{}
	detached chan struct{}
	detach   sync.Once
}

func New(dir string) *Manager {
	return &Manager{dir: dir, sessions: make(map[*Session]struct{})}
}

func (m *Manager) Open() (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, errors.New("terminal manager is closed")
	}
	cmd := exec.Command("bash", "--noprofile", "--norc", "-i")
	cmd.Dir = m.dir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	file, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		return nil, err
	}
	// Make Close interrupt a PTY read, as required by creack/pty.
	if err := syscall.SetNonblock(int(file.Fd()), true); err != nil {
		file.Close()
		cmd.Process.Kill()
		cmd.Wait()
		return nil, err
	}
	s := &Session{cmd: cmd, pty: file, output: make(chan []byte), stop: make(chan struct{}), readDone: make(chan struct{}), exited: make(chan struct{}), detached: make(chan struct{})}
	m.sessions[s] = struct{}{}
	m.wg.Add(2)
	go func() {
		defer m.wg.Done()
		defer close(s.output)
		defer close(s.readDone)
		buf := make([]byte, 4096)
		for {
			n, err := file.Read(buf)
			if n > 0 {
				select {
				case s.output <- append([]byte(nil), buf[:n]...):
				case <-s.detached:
					// The session keeps consuming PTY output without retaining it.
				case <-s.stop:
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	go func() {
		defer m.wg.Done()
		cmd.Wait()
		close(s.exited)
		// Let the reader deliver the last PTY bytes before releasing the file.
		<-s.readDone
		file.Close()
		m.mu.Lock()
		delete(m.sessions, s)
		m.mu.Unlock()
	}()
	return s, nil
}

func (s *Session) Input(data string) error {
	_, err := s.pty.WriteString(data)
	return err
}

func (s *Session) Output() <-chan []byte { return s.output }

// Detach releases the output consumer without ending the terminal session.
func (s *Session) Detach() { s.detach.Do(func() { close(s.detached) }) }

func (m *Manager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		m.wg.Wait()
		return
	}
	m.closed = true
	sessions := make([]*Session, 0, len(m.sessions))
	for s := range m.sessions {
		sessions = append(sessions, s)
		close(s.stop)
		// Give bash an opportunity to hang up its jobs before forcing exit.
		syscall.Kill(-s.cmd.Process.Pid, syscall.SIGHUP)
		// Closing the controlling PTY also hangs up the foreground job.
		s.pty.Close()
	}
	m.mu.Unlock()
	for _, s := range sessions {
		select {
		case <-s.exited:
		case <-time.After(time.Second):
			syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
		}
	}
	m.wg.Wait()
}
