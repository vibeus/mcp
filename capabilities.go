package mcp

import (
	"context"
	"sync"

	"github.com/vibeus/mcp/jsonrpc2"
)

type CapRootsProvider interface {
	Roots_Started() *sync.Once
	Roots_Capability() *CapRoots
	Roots_OnList() []Root
	Roots_ListChanged() chan struct{}
}

type CapSamplingProvider interface {
	Sampling_Capability() *CapSampling
	jsonrpc2.HandlerOf[SamplingMessage, SamplingResponse]
}

type CapPromptsProvider interface {
	Prompts_Started() *sync.Once
	Prompts_Capability() *CapPrompts
	Prompts_OnList(cursor string) []ListPromptsResponse
	Prompts_OnGet(name string) (PromptGetResponse, *jsonrpc2.ErrorObject)
	Prompts_ListChanged() chan struct{}
}

type CapToolsProvider interface {
	Tools_Started() *sync.Once
	Tools_Capability() *CapTools
	Tools_OnList(cursor string) []ListToolsResonponse
	Tools_OnCall(name string, args map[string]string) (ToolCallResponse, *jsonrpc2.ErrorObject)
	Tools_ListChanged() chan struct{}
}

type CapResourcesProvider interface {
	Resources_Started() *sync.Once
	Resources_Capability() *CapResources
	Resources_OnList(cursor string) []ResourceSpec
	Resources_OnTemplatesList() []ResourceTemplateSpec
	Resources_OnRead(uri string) []ResourceContentUnion
	Resources_ListChanged() chan struct{}
}

type Root struct {
	URI  string `json:"uri"`
	Name string `json:"name,omitempty"`
}

type PagedRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

type ListPromptsResponse struct {
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
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments"`
}

type MessageWithRole struct {
	Role    string          `json:"role"`
	Content ContentTextOnly `json:"content"`
}

type ContentTextOnly struct {
	Type string `json:"type"` // text
	Text string `json:"text"`
}

type PromptGetResponse struct {
	Description string            `json:"description"`
	Messages    []MessageWithRole `json:"messages"`
}

type ListToolsResonponse struct {
	Tools      []ToolSpec `json:"tools"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

type ToolSpec struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	InputSchema ToolSchema `json:"input_schema"`
}

type ToolSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]ParamSchema `json:"properties"`
	Required   []string               `json:"required"`
}

type ParamSchema struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolCallRequest struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments"`
}

type ToolCallResponse struct {
	Content []ToolCallContentUnion `json:"content"`
	IsError bool                   `json:"isError"`
}

// Text Content
//
//	{
//	  "type": "text",
//	  "text": "Tool result text"
//	}
//
// Image Content
//
//	{
//		"type": "image",
//		"data": "base64-encoded-data",
//		"mimeType": "image/png"
//	}
//
// Audio Content
//
//	{
//	  "type": "audio",
//	  "data": "base64-encoded-audio-data",
//	  "mimeType": "audio/wav"
//	}
//
// EmbeddedResource
//
//	{
//		"type": "resource",
//		"resource": {
//		  "uri": "resource://example",
//		  "mimeType": "text/plain",
//		  "text": "Resource content"
//		}
//	}
type ToolCallContentUnion struct {
	Type     string                `json:"type"` // text | image | audio | resource
	Text     string                `json:"text,omitempty"`
	MimeType string                `json:"mimeType,omitempty"`
	Data     string                `json:"data,omitempty"` // base64 encoded image | audio data
	Resource *ResourceContentUnion `json:"resource,omitempty"`
}

type ResourcesListResponse struct {
	Resources  []ResourceSpec `json:"resources"`
	NextCursor string         `json:"nextCursor,omitempty"`
}

type ResourcesTemplatesListResponse struct {
	ResourceTemplates []ResourceTemplateSpec `json:"resourceTemplates"`
}

type ResourcesReadRequest struct {
	URI string `json:"uri"`
}
type ResourcesReadResponse struct {
	Content []ResourceContentUnion `json:"content"`
}

type ResourceSpec struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Size        int64  `json:"size,omitempty"` // size in bytes
}

// Text Content
//
//	 {
//		"uri": "file:///example.txt",
//		"mimeType": "text/plain",
//		"text": "Resource content"
//	 }
//
// Binary Content
//
//	{
//		"uri": "file:///example.png",
//		"mimeType": "image/png",
//		"blob": "base64-encoded-data"
//	}
type ResourceContentUnion struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64 encoded binary resource data
}

type ResourceTemplateSpec struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
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

func startCapTools(server *ServerState, tools CapToolsProvider) {
	once := tools.Tools_Started()
	if once == nil {
		return
	}

	once.Do(func() {
		var wg sync.WaitGroup
		if ch := tools.Tools_ListChanged(); ch != nil {
			wg.Add(1)
			go func() {
				s := server.ctx.GetSession()
				logger := s.GetLogger()
				if logger != nil {
					logger.Info("StartNotifier", "method", kMethodToolsListChanged)
				}
				wg.Done()

				context.AfterFunc(server.ctx, func() {
					close(ch)
				})
				for range ch {
					server.NotifyToolsListChanged(server.ctx)
				}
			}()
		}
		wg.Wait()
	})
}

func startCapResources(server *ServerState, resources CapResourcesProvider) {
	once := resources.Resources_Started()
	if once == nil {
		return
	}

	once.Do(func() {
		var wg sync.WaitGroup
		if ch := resources.Resources_ListChanged(); ch != nil {
			wg.Add(1)
			go func() {
				s := server.ctx.GetSession()
				logger := s.GetLogger()
				if logger != nil {
					logger.Info("StartNotifier", "method", kMethodResourcesListChanged)
				}
				wg.Done()

				context.AfterFunc(server.ctx, func() {
					close(ch)
				})
				for range ch {
					server.NotifyResourcesListChanged(server.ctx)
				}
			}()
		}
		wg.Wait()
	})
}
