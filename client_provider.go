package mcp

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/vibeus/mcp/jsonrpc2"
)

type ClientProvider interface {
	Start(*ClientState)
	Capabilities() ClientCapabilities
	Handler() jsonrpc2.Handler
}

// ClientImpl is the implementation of the [ClientProvider] interface.
type ClientImpl struct {
	client *ClientState
	// Provider Roots Capability, can be nil if not supported.
	CapRootsProvider
	// Provider Sampling Capability, can be nil if not supported.
	CapSamplingProvider

	once sync.Once
}

func startCapRoots(client *ClientState, roots CapRootsProvider) {
	once := roots.Roots_Started()
	if once == nil {
		return
	}

	once.Do(func() {
		var wg sync.WaitGroup
		if ch := roots.Roots_ListChanged(); ch != nil {
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
		wg.Wait()
	})
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

func (h *ClientImpl) Start(client *ClientState) {
	h.client = client
	h.once.Do(func() {
		if h.CapRootsProvider != nil {
			startCapRoots(client, h.CapRootsProvider)
		}
	})
}

func (h *ClientImpl) Handler() jsonrpc2.Handler {
	return h
}

func (h *ClientImpl) HandleRequest(w *jsonrpc2.ResponseWriter, req jsonrpc2.Request) error {
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
			return h.CapSamplingProvider.HandleRequest(jsonrpc2.MakeResponseWriterOf[SamplingResponse](w), msg)
		} else {
			w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
		}
	default:
		w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
	}
	return nil
}
