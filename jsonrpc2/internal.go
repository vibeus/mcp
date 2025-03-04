package jsonrpc2

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
)

type endpoint struct {
	ctx    context.Context
	framer Framer

	frameReadChan  chan []byte
	frameWriteChan chan []byte
	cancelFunc     context.CancelFunc

	logger *slog.Logger
	once   sync.Once
}

func (c *endpoint) readFrame() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			frame, err := c.framer.ReadFrame()
			if err != nil && err != io.EOF {
				if c.logger != nil {
					c.logger.Error("error reading frame", "error", err)
				}
				c.cancelFunc()
				return
			}
			if c.logger != nil {
				c.logger.Debug(">>", "frame", string(frame))
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
			if c.logger != nil {
				c.logger.Debug("<<", "frame", string(frame))
			}
			err := c.framer.WriteFrame(frame)
			if err != nil {
				if c.logger != nil {
					c.logger.Error("error writing frame", "error", err)
				}
				c.cancelFunc()
				return
			}
		}
	}
}

// responseWriter implements ResponseWriter interface
type responseWriter struct {
	id     *ID
	output chan []byte
}

func (w *responseWriter) WriteResponse(res any) error {
	var encoded_res json.RawMessage
	var err error
	encoded_res, err = json.Marshal(res)
	if err != nil {
		return err
	}

	r := responseUnion{
		Version: JSONRPC2Version,
		Result:  &encoded_res,
		ID:      w.id,
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	w.output <- data
	return nil
}

func (w *responseWriter) WriteError(res ErrorObject) error {
	r := responseUnion{
		Version: JSONRPC2Version,
		Error:   &res,
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	w.output <- data
	return nil
}
