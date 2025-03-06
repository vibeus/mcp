package mcp

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/vibeus/mcp/jsonrpc2"
)

type ClientProvider interface {
	jsonrpc2.Handler
	Start(*Client)
	Capabilities() ClientCapabilities
}

// ClientImpl is the implementation of the [ClientProvider] interface.
type ClientImpl struct {
	client *Client
	CapRootsProvider
	CapSamplingProvider

	once sync.Once
}

func (h *ClientImpl) Capabilities() ClientCapabilities {
	var caps ClientCapabilities
	if h.CapRootsProvider != nil {
		caps.Roots = h.CapRootsProvider.Roots_Capability()
	}
	if h.CapSamplingProvider != nil {
		caps.Sampling = h.CapSamplingProvider.Sampling_Capability()
	}
	return caps
}

func (h *ClientImpl) Start(client *Client) {
	h.client = client
	h.once.Do(func() {
		var wg sync.WaitGroup
		if h.CapRootsProvider != nil {
			if ch := h.CapRootsProvider.Roots_ListChanged(); ch != nil {
				wg.Add(1)
				go func() {
					s := client.ctx.GetSession()
					logger := s.GetLogger()
					if logger != nil {
						logger.Info("StartNotifier", "method", kMethodRootsListChanged)
					}
					wg.Done()
					context.AfterFunc(client.ctx, func() {
						close(ch)
					})
					for range ch {
						client.NotifyRootsListChanged(client.ctx)
					}
				}()
			}
		}
		wg.Wait()
	})
}

func (h *ClientImpl) HandleRequest(w jsonrpc2.ResponseWriter, req jsonrpc2.Request) error {
	switch req.Method {
	case kMethodRootsList:
		if h.CapRootsProvider != nil {
			roots := h.CapRootsProvider.Roots_ListChanged()
			w.WriteResponse(roots)
		} else {
			w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
		}
		return nil
	case kMethodSamplingCreateMessage:
		if h.CapSamplingProvider != nil {
			var msg SamplingMessage
			err := json.Unmarshal(*req.Params, &msg)
			if err != nil {
				w.WriteError(jsonrpc2.ErrObjInvalidParams)
			}
			go func() {
				resChan, errChan := h.CapSamplingProvider.Sampling_OnCreateMessage(msg)
				select {
				case res := <-resChan:
					w.WriteResponse(res)
				case err := <-errChan:
					w.WriteError(err)
				}
			}()
		} else {
			w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
		}
	default:
		w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
	}
	return nil
}
