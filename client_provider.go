package mcp

import (
	"encoding/json"
	"sync"

	"github.com/vibeus/mcp/jsonrpc2"
)

type ClientProvider interface {
	jsonrpc2.Handler
	BindState(*ClientState)
	StartClientProvider()
	Capabilities() ClientCapabilities
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

func (c *ClientImpl) Capabilities() ClientCapabilities {
	var caps ClientCapabilities
	if c.CapRootsProvider != nil {
		caps.Roots = c.CapRootsProvider.Roots_Capability()
	}
	if c.CapSamplingProvider != nil {
		caps.Sampling = c.CapSamplingProvider.Sampling_Capability()
	}
	return caps
}

func (c *ClientImpl) BindState(client *ClientState) {
	c.client = client
}

func (c *ClientImpl) StartClientProvider() {
	if c.client == nil {
		panic("must call BindState before StartClientProvider")
	}
	c.once.Do(func() {
		if c.CapRootsProvider != nil {
			startCapRoots(c.client, c.CapRootsProvider)
		}
	})
}

func (c *ClientImpl) HandleRequest(w *jsonrpc2.ResponseWriter, req jsonrpc2.Request) error {
	switch req.Method {
	case kMethodRootsList:
		if c.CapRootsProvider != nil {
			roots := c.CapRootsProvider.Roots_ListChanged()
			w.WriteResponse(roots)
		} else {
			w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
		}
		return nil
	case kMethodSamplingCreateMessage:
		if c.CapSamplingProvider != nil {
			var msg SamplingMessage
			err := json.Unmarshal(*req.Params, &msg)
			if err != nil {
				w.WriteError(jsonrpc2.ErrObjInvalidParams)
			}
			return c.CapSamplingProvider.HandleRequest(jsonrpc2.MakeResponseWriterOf[SamplingResponse](w), msg)
		} else {
			w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
		}
	default:
		w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
	}
	return nil
}
