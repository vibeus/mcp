package mcp

import "github.com/vibeus/mcp/jsonrpc2"

type Root struct {
	URI  string `json:"uri"`
	Name string `json:"name,omitempty"`
}

type CapRootsProvider interface {
	Roots_Capability() *CapRoots
	Roots_OnList() []Root
	Roots_ListChanged() chan struct{}
}

type CapSamplingProvider interface {
	Sampling_Capability() *CapSampling
	Sampling_OnCreateMessage(SamplingMessage) (<-chan SamplingResponse, <-chan jsonrpc2.ErrorObject)
}
