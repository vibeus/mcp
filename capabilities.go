package mcp

import (
	"sync"

	"github.com/vibeus/mcp/jsonrpc2"
)

type Root struct {
	URI  string `json:"uri"`
	Name string `json:"name,omitempty"`
}

type CapRootsProvider interface {
	Roots_Started() *sync.Once
	Roots_Capability() *CapRoots
	Roots_OnList() []Root
	Roots_ListChanged() chan struct{}
}

type CapSamplingProvider interface {
	Sampling_Capability() *CapSampling
	// Sampling needs the peer to be a JSON-RPC2 server.
	jsonrpc2.HandlerOf[SamplingMessage, SamplingResponse]
}
