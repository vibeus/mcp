package jsonrpc2

import (
	"context"
	"encoding/json"
	"io"
	"sync"
)

type result struct {
	Value any
	Error error
}

type pendingRequest struct {
	response chan result
}

type endpoint struct {
	ctx    context.Context
	conn   io.ReadWriteCloser
	framer Framer

	frameReadChan  chan []byte
	frameWriteChan chan []byte
	cancelFunc     context.CancelFunc

	once sync.Once
}

type Client struct {
	endpoint
	pending map[ID]pendingRequest
}

type Server struct {
	endpoint
	handler Handler
}

func NewClient(pctx context.Context, conn io.ReadWriteCloser, framer Framer) *Client {
	pending := make(map[ID]pendingRequest)
	ctx, cancelFunc := context.WithCancel(pctx)
	context.AfterFunc(ctx, func() {
		conn.Close()
	})
	return &Client{
		endpoint: endpoint{
			ctx:            ctx,
			conn:           conn,
			framer:         framer,
			frameReadChan:  make(chan []byte, 1),
			frameWriteChan: make(chan []byte, 1),
			cancelFunc:     cancelFunc,
		},
		pending: pending,
	}
}

func (c *endpoint) readFrame() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			frame, err := c.framer.ReadFrame()
			if err != nil && err != io.EOF {
				c.cancelFunc()
				return
			}
			c.frameReadChan <- frame
		}
	}
}

func (c *endpoint) writeFrame() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case frame := <-c.frameWriteChan:
			err := c.framer.WriteFrame(frame)
			if err != nil {
				c.cancelFunc()
				return
			}
		}
	}
}

// read from c.frameRead and send it to the appropriate channel
func (c *Client) handleResponse(id ID) {
	c.once.Do(func() {
		go c.readFrame()
		go c.writeFrame()
	})

	select {
	case <-c.ctx.Done():
		return
	case frame := <-c.frameReadChan:
		var err error
		res := &responseUnion{}
		err = json.Unmarshal(frame, res)
		if err != nil {
			c.pending[id].response <- result{Error: err}
			return
		}
		if res.Error != nil {
			reserr := &ResponseError{
				Error: *res.Error,
			}
			c.pending[id].response <- result{Value: reserr}
			return
		}
		if res.Result != nil {
			resok := &ResponseOk{
				Result: res.Result,
			}
			c.pending[id].response <- result{Value: resok}
			return
		}
		c.pending[id].response <- result{Error: ErrInvalidContent}
	}
}

func (c *Client) Call(id ID, method string, params any, output *result) error {
	req := requestUnion{
		ID:     &id,
		Method: method,
		Params: params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return RPCError{err}
	}

	// prepare wait for response
	c.pending[id] = pendingRequest{
		response: make(chan result),
	}
	defer func() {
		close(c.pending[id].response)
		delete(c.pending, id)
	}()
	go c.handleResponse(id)

	// send frame
	select {
	case <-c.ctx.Done():
		return RPCError{ErrContextCancel}
	case c.frameWriteChan <- data:
	}

	// wait for response
	select {
	case <-c.ctx.Done():
		return ErrContextCancel
	case res := <-c.pending[id].response:
		if res.Error != nil {
			return res.Error
		}

		switch v := res.Value.(type) {
		case *ResponseOk:
			output.Value = v.Result
		case *ResponseError:
			output.Error = v.Error
		}
		return nil
	}
}

func (c *Client) Notify(method string, params any) error {
	req := requestUnion{
		Method: method,
		Params: params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return RPCError{err}
	}

	select {
	case <-c.ctx.Done():
		return RPCError{ErrContextCancel}
	case c.frameWriteChan <- data:
		return nil
	}
}

type ResponseWriter interface {
	WriteResponse(ResponseOk) error
	WriteError(ResponseError) error
}

type Handler interface {
	HandleRequest(ResponseWriter, Request) error
	HandleNotification(Request) error
}

func NewServer(pctx context.Context, conn io.ReadWriteCloser, framer Framer, handler Handler) *Server {
	ctx, cancelFunc := context.WithCancel(pctx)
	context.AfterFunc(ctx, func() {
		conn.Close()
	})
	return &Server{
		endpoint: endpoint{
			ctx:            ctx,
			conn:           conn,
			framer:         framer,
			frameReadChan:  make(chan []byte, 1),
			frameWriteChan: make(chan []byte, 1),
			cancelFunc:     cancelFunc,
		},
		handler: handler,
	}
}

type responseWriter struct {
	id     *ID
	output chan []byte
}

func (w *responseWriter) WriteResponse(res ResponseOk) error {
	r := responseUnion{
		Version: JSONRPC2Version,
		Result:  res.Result,
		ID:      w.id,
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	w.output <- data
	return nil
}

func (w *responseWriter) WriteError(res ResponseError) error {
	r := responseUnion{
		Version: JSONRPC2Version,
		Error:   &res.Error,
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	w.output <- data
	return nil
}

// Serve handles incoming requests and notifications. It reads frames from the
// connection, returns if the context is canceled, or if there are errors during
// the process.
func (s *Server) Serve() error {
	s.once.Do(func() {
		go s.readFrame()
		go s.writeFrame()
	})

	for {
		select {
		case <-s.ctx.Done():
			return nil
		case frame := <-s.frameReadChan:
			var req requestUnion
			writer := responseWriter{
				output: s.frameWriteChan,
			}
			err := json.Unmarshal(frame, &req)
			if err != nil {
				err = writer.WriteError(ResponseError{Error: ErrorObject{Code: JSONRPC2ErrorParseError, Message: "Invalid JSON"}})
				if err != nil {
					return err
				}
				continue
			}

			// now we has detected id
			writer.id = req.ID
			if req.ID == nil {
				err = s.handler.HandleNotification(Request{Method: req.Method, Params: req.Params})
			} else {
				err = s.handler.HandleRequest(&writer, Request{Method: req.Method, Params: req.Params, id: req.ID})
			}
			if err != nil {
				return err
			}
		}
	}
}
