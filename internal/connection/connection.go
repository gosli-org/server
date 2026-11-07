package connection

import (
	"context"
	"encoding/json"
	"net/http"
	"unicode/utf8"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/gosli-org/server/internal/dispatcher"
	"github.com/gosli-org/server/internal/terminal"
)

type Handler struct {
	ctx        context.Context
	dispatcher *dispatcher.Dispatcher
}

func New(ctx context.Context, dispatcher *dispatcher.Dispatcher) *Handler {
	return &Handler{ctx: ctx, dispatcher: dispatcher}
}

type message struct {
	Type string  `json:"type"`
	Data *string `json:"data,omitempty"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	stop := context.AfterFunc(h.ctx, cancel)
	defer stop()
	var session *terminal.Session
	var outputDone chan struct{}
	defer func() {
		cancel()
		if session != nil {
			session.Detach()
		}
		conn.CloseNow()
		if outputDone != nil {
			<-outputDone
		}
	}()
	for {
		kind, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var request message
		if kind != websocket.MessageText || json.Unmarshal(data, &request) != nil ||
			(request.Type != "open" && request.Type != "input") ||
			(request.Type == "input" && request.Data == nil) {
			conn.Close(websocket.StatusInvalidFramePayloadData, "invalid request")
			return
		}
		input := ""
		if request.Data != nil {
			input = *request.Data
		}
		previous := session
		session, err = h.dispatcher.Dispatch(session, request.Type, input)
		if err != nil {
			conn.Close(websocket.StatusPolicyViolation, "request rejected")
			return
		}
		if previous == nil {
			outputDone = make(chan struct{})
			go func(session *terminal.Session) {
				defer close(outputDone)
				streamOutput(ctx, conn, session)
			}(session)
		}
	}
}

func streamOutput(ctx context.Context, conn *websocket.Conn, session *terminal.Session) {
	defer session.Detach()
	// Retain a partial UTF-8 character across PTY chunks so JSON encoding does
	// not replace a multibyte character split between reads.
	var pending []byte
	for {
		select {
		case <-ctx.Done():
			return
		case output, ok := <-session.Output():
			if !ok {
				return
			}
			pending = append(pending, output...)
			end := 0
			for end < len(pending) && utf8.FullRune(pending[end:]) {
				_, size := utf8.DecodeRune(pending[end:])
				end += size
			}
			if end == 0 {
				continue
			}
			data := string(pending[:end])
			if err := wsjson.Write(ctx, conn, message{Type: "output", Data: &data}); err != nil {
				return
			}
			pending = append(pending[:0], pending[end:]...)
		}
	}
}
