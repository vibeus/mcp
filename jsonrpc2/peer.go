package jsonrpc2

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
)

// Peer is a struct that represents a JSON-RPC 2.0 client and server. It
// provides methods for sending requests and receiving responses.
type Peer struct {
	endpoint
	pendingRequests map[ID]PendingRequest
	handler         Handler

	// as a client, how many requests we have sent
	requestCount int32

	mutex sync.Mutex
}

// NewPeer creates a new Peer instance with the given context, framer, and
// handler.
//   - If the peer is used as a server, the handler must be provided to handle
//     incoming requests and [Peer.Start] must be called to start serving.
//   - If the peer is used as a client, no handler is required and it can be nil.
//     [Peer.Start] will be called with [Peer.Call] and [Peer.Notify].
func NewPeer(pctx context.Context, framer Framer, handler Handler) *Peer {
	ctx, cancelFunc := context.WithCancel(pctx)
	peer := &Peer{
		endpoint: endpoint{
			ctx:            ctx,
			framer:         framer,
			frameReadChan:  make(chan []byte, 1),
			frameWriteChan: make(chan []byte, 1),
			cancelFunc:     cancelFunc,
		},
		pendingRequests: make(map[ID]PendingRequest),
		handler:         handler,
	}
	context.AfterFunc(ctx, func() {
		if peer.logger != nil {
			peer.logger.Debug("shutting down peer", "reason", "context done")
		}
		framer.Close()
	})
	return peer
}

func (p *Peer) SetLogger(logger *slog.Logger) {
	p.logger = logger
}

// Start starts the peer and begins serving incoming requests. It must be called
// once after the peer is setup. It is automatically called with [Peer.Call] and
// [Peer.Notify].
func (p *Peer) Start() {
	p.once.Do(func() {
		go p.serve()
		go p.readFrame()
		go p.writeFrame()
	})
}

func (p *Peer) serve() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case frame := <-p.frameReadChan:
			err := p.handleFrame(frame)
			if err != nil {
				p.cancelFunc()
				if p.logger != nil {
					p.logger.Error("error handling frame", "error", err)
				}
				return
			}
		}
	}
}

// Notify makes a notification to the remote peer without waiting for a response.
func (p *Peer) Notify(method string, params any) error {
	p.Start()

	var encoded_param json.RawMessage
	var err error
	encoded_param, err = json.Marshal(params)

	if err != nil {
		return RPCError{err}
	}
	req := requestData{
		Version: JSONRPC2Version,
		Method:  method,
		Params:  &encoded_param,
	}

	err = p.sendRequestOrNotification(p.ctx, req)
	if err != nil {
		return RPCError{err}
	}
	return nil
}

// Call makes a call to the remote peer. The method is called with the given
// parameters. The response is returned by calling [Peer.RecvResponse] on the
// returned [PendingRequest] object.
//
// When Call returns without an error, the request is sent to framer.
//
// NOTE: The [PendingRequest] object MUST be passed to either
// [PendingRequest.Cancel] or [PendingRequest.RecvResponse], otherwise the call
// will hang indefinitely.
func (p *Peer) Call(method string, params any) (*PendingRequest, error) {
	p.Start()

	var encoded_param json.RawMessage
	var err error
	encoded_param, err = json.Marshal(params)
	if err != nil {
		return nil, RPCError{err}
	}

	p.requestCount++
	id := makeNumberID(p.requestCount)

	req := requestData{
		Version: JSONRPC2Version,
		ID:      &id,
		Method:  method,
		Params:  &encoded_param,
	}

	p.mutex.Lock()
	ctx, cancelFunc := context.WithCancel(p.ctx)
	channel := make(chan responseData, 1)
	request := PendingRequest{id: id, ctx: ctx, cancelFunc: cancelFunc, channel: channel}
	p.pendingRequests[id] = request
	context.AfterFunc(ctx, func() {
		p.mutex.Lock()
		delete(p.pendingRequests, id)
		p.mutex.Unlock()
		close(channel)
	})
	p.mutex.Unlock()

	err = p.sendRequestOrNotification(ctx, req)
	if err != nil {
		return nil, RPCError{err}
	}
	return &request, nil
}

func (p *Peer) handleFrame(frame []byte) error {
	var wireData wireUnion
	err := json.Unmarshal(frame, &wireData)
	if err != nil {
		erro := ErrObjParseError
		erro.Data = &json.RawMessage{}
		erro.Data.UnmarshalJSON(frame)
		return erro
	}
	if wireData.IsResponse() || wireData.IsError() {
		if wireData.ID == nil {
			p.logger.Error("received a error without an ID", "frame", string(frame))
			response := responseData{
				Error: wireData.Error,
			}
			p.mutex.Lock()
			var requestChannels []chan<- responseData
			for _, req := range p.pendingRequests {
				requestChannels = append(requestChannels, req.channel)
			}
			p.mutex.Unlock()

			for _, ch := range requestChannels {
				ch <- response
			}
		} else {
			id := *wireData.ID
			p.mutex.Lock()
			request, ok := p.pendingRequests[id]
			p.mutex.Unlock()
			if ok {
				response := responseData{
					Result: wireData.Result,
					Error:  wireData.Error,
					ID:     wireData.ID,
				}
				select {
				case <-request.ctx.Done():
				case request.channel <- response:
				}
			}
		}
		return nil
	}
	// receive a request from the server
	if p.handler != nil {
		writer := ResponseWriter{
			output: p.frameWriteChan,
			id:     wireData.ID,
		}
		return p.handler.HandleRequest(&writer, Request{Method: wireData.Method, Params: wireData.Params, id: wireData.ID})
	} else {
		return ErrNoHandler
	}
}

func (p *Peer) sendRequestOrNotification(ctx context.Context, req requestData) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ErrContextCancel
	case p.frameWriteChan <- data:
		return nil
	}
}
