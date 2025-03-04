package jsonrpc2

import (
	"context"
	"encoding/json"
	"log/slog"
)

type Peer struct {
	endpoint
	handler Handler
}

func NewPeer(pctx context.Context, framer Framer, handler Handler) *Peer {
	ctx, cancelFunc := context.WithCancel(pctx)
	context.AfterFunc(ctx, func() {
		framer.Close()
	})
	return &Peer{
		endpoint: endpoint{
			ctx:            ctx,
			framer:         framer,
			frameReadChan:  make(chan []byte, 1),
			frameWriteChan: make(chan []byte, 1),
			cancelFunc:     cancelFunc,
		},
		handler: handler,
	}
}

func (p *Peer) SetLogger(logger *slog.Logger) {
	p.logger = logger
}

func (p *Peer) Start() {
	p.once.Do(func() {
		go p.readFrame()
		go p.writeFrame()
	})
}

// Serve starts a goroutine to read frames from the framer and another goroutine
// to write frames to the framer. The Serve method blocks until the context is
// done. It returns an error if the handler is not provided or responseWriter
// can't be written.
func (p *Peer) Serve() error {
	if p.handler == nil {
		return RPCError{ErrNoHandler}
	}

	p.Start()

	for {
		select {
		case <-p.ctx.Done():
			return nil
		case frame := <-p.frameReadChan:
			var req requestUnion
			writer := responseWriter{
				output: p.frameWriteChan,
			}
			err := json.Unmarshal(frame, &req)
			if err != nil {
				err = writer.WriteError(ErrorObject{Code: JSONRPC2ErrorParseError, Message: "Invalid JSON"})
				if err != nil {
					return RPCError{err}
				}
				continue
			}

			writer.id = req.ID
			err = p.handler.HandleRequest(&writer, Request{Method: req.Method, Params: req.Params, id: req.ID})
			if err != nil {
				return RPCError{err}
			}
		}
	}
}

func (p *Peer) Call(id ID, method string, params any, output any) error {
	var encoded_param json.RawMessage
	var err error
	encoded_param, err = json.Marshal(params)
	if err != nil {
		return RPCError{err}
	}

	req := requestUnion{
		ID:     &id,
		Method: method,
		Params: &encoded_param,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return RPCError{err}
	}

	p.Start()

	select {
	case <-p.ctx.Done():
		return RPCError{ErrContextCancel}
	case p.frameWriteChan <- data:
	}

	select {
	case <-p.ctx.Done():
		return RPCError{ErrContextCancel}
	case frame := <-p.frameReadChan:
		response := &responseUnion{}
		err := json.Unmarshal(frame, response)
		if err != nil {
			return RPCError{err}
		}
		if response.Result != nil {
			err := json.Unmarshal(*response.Result, output)
			if err != nil {
				return RPCError{err}
			}
			return nil
		}
		if response.Error != nil {
			return RPCError{response.Error}
		}
		return RPCError{ErrInvalidContent}
	}
}

func (p *Peer) Notify(method string, params any) error {
	var encoded_param json.RawMessage
	var err error
	encoded_param, err = json.Marshal(params)

	if err != nil {
		return RPCError{err}
	}
	req := requestUnion{
		Method: method,
		Params: &encoded_param,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return RPCError{err}
	}

	select {
	case <-p.ctx.Done():
		return RPCError{ErrContextCancel}
	case p.frameWriteChan <- data:
		return nil
	}
}
