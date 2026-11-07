package dispatcher

import (
	"errors"

	"github.com/gosli-org/server/internal/terminal"
)

type Dispatcher struct {
	terminals *terminal.Manager
}

func New(terminals *terminal.Manager) *Dispatcher {
	return &Dispatcher{terminals: terminals}
}

// Dispatch routes requests for the single terminal session of a connection.
// The returned session is owned by terminal, independently of its connection.
func (d *Dispatcher) Dispatch(session *terminal.Session, kind, data string) (*terminal.Session, error) {
	switch kind {
	case "open":
		if session != nil {
			return session, errors.New("connection already has a terminal session")
		}
		return d.terminals.Open()
	case "input":
		if session == nil {
			return nil, errors.New("open a terminal session before sending input")
		}
		return session, session.Input(data)
	default:
		return session, errors.New("unknown request type")
	}
}
