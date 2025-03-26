package mcp

import (
	"context"
	"encoding/json"
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

type CapPromptsProvider interface {
	Prompts_Started() *sync.Once
	Prompts_Capability() *CapPrompts
	Prompts_OnList() []PagedPrompts
	Prompts_OnGet(name string) (PromptGetResponse, *jsonrpc2.ErrorObject)
	Prompts_ListChanged() chan struct{}
}

type PagedPrompts struct {
	Prompts    []PromptSpec `json:"prompts"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

type PromptSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Arguments   []ArgumentSpec `json:"arguments"`
}

type ArgumentSpec struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

type PromptGetRequest struct {
	Name      string `json:"name"`
	Arguments []json.RawMessage
}

type MessageWithRole struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type PromptGetResponse struct {
	Description string            `json:"description"`
	Messages    []MessageWithRole `json:"messages"`
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

func startCapPrompts(server *ServerState, prompts CapPromptsProvider) {
	once := prompts.Prompts_Started()
	if once == nil {
		return
	}

	once.Do(func() {
		var wg sync.WaitGroup
		if ch := prompts.Prompts_ListChanged(); ch != nil {
			wg.Add(1)
			go func() {
				s := server.ctx.GetSession()
				logger := s.GetLogger()
				if logger != nil {
					logger.Info("StartNotifier", "method", kMethodPromptsListChanged)
				}
				wg.Done()

				context.AfterFunc(server.ctx, func() {
					close(ch)
				})
				for range ch {
					server.NotifyPromptsListChanged(server.ctx)
				}
			}()
		}
		wg.Wait()
	})
}
