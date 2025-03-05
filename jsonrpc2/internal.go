package jsonrpc2

import (
	"context"
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
		frame, err := c.framer.ReadFrame()
		if err != nil {
			if err != io.EOF {
				if c.logger != nil {
					c.logger.Debug("error reading frame", "error", err)
				}
			}
			c.cancelFunc()
			return
		}
		select {
		case <-c.ctx.Done():
			return
		case c.frameReadChan <- frame:
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
				if c.logger != nil {
					c.logger.Error("error writing frame", "error", err)
				}
				c.cancelFunc()
				return
			}
		}
	}
}
