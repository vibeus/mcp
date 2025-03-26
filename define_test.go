package mcp

import (
	"sync"

	"github.com/vibeus/mcp/jsonrpc2"
)

type testServerImpl struct {
	prompts_ListChanged   chan struct{}
	prompts_started       sync.Once
	resources_ListChanged chan struct{}
	resources_started     sync.Once
	tools_ListChanged     chan struct{}
	tools_started         sync.Once
}

func NewTestServerImpl() *testServerImpl {
	return &testServerImpl{
		prompts_ListChanged:   make(chan struct{}),
		resources_ListChanged: make(chan struct{}),
		tools_ListChanged:     make(chan struct{}),
	}
}

func (*testServerImpl) NegotiateMCPVersion(client string) string {
	return LatestMCPVersion
}

// CapPromptsProvider implementation
func (c *testServerImpl) Prompts_Started() *sync.Once {
	return &c.prompts_started
}
func (c *testServerImpl) Prompts_Capability() *CapPrompts {
	return &CapPrompts{ListChanged: c.prompts_ListChanged != nil}
}
func (c *testServerImpl) Prompts_OnList(cursor string) []ListPromptsResponse {
	return []ListPromptsResponse{
		{
			Prompts: []PromptSpec{
				{
					Name:        "test_prompt",
					Description: "Test prompt for demonstration",
					Arguments: []ArgumentSpec{
						{
							Name:        "question",
							Description: "Question to ask",
							Required:    true,
						},
					},
				},
			},
		},
	}
}
func (c *testServerImpl) Prompts_ListChanged() chan struct{} { return c.prompts_ListChanged }
func (c *testServerImpl) Prompts_OnGet(name string) (PromptGetResponse, *jsonrpc2.ErrorObject) {
	if name == "test_prompt" {
		return PromptGetResponse{
			Description: "Test prompt for demonstration",
			Messages: []MessageWithRole{
				{
					Role: "assistant",
					Content: ContentTextOnly{
						Type: "text",
						Text: "Question to ask",
					},
				},
			},
		}, nil
	}
	return PromptGetResponse{}, &jsonrpc2.ErrorObject{Code: -32601, Message: "Prompt not found"}
}

// Add resources capability to test server implementation
func (c *testServerImpl) Resources_Started() *sync.Once {
	return &c.resources_started
}

func (c *testServerImpl) Resources_Capability() *CapResources {
	return &CapResources{
		ListChanged: c.resources_ListChanged != nil,
		Subscribe:   true,
	}
}

func (c *testServerImpl) Resources_OnList(cursor string) []ResourceSpec {
	return []ResourceSpec{
		{
			URI:      "resource://test",
			Name:     "Test Resource",
			MimeType: "text/plain",
		},
	}
}

func (c *testServerImpl) Resources_OnTemplatesList() []ResourceTemplateSpec {
	return []ResourceTemplateSpec{
		{
			URITemplate: "resource://test/{id}",
			Name:        "Test Template",
			MimeType:    "text/plain",
		},
	}
}

func (c *testServerImpl) Resources_OnRead(uri string) []ResourceContentUnion {
	if uri == "resource://test/0" {
		return []ResourceContentUnion{
			{
				URI:  uri,
				Text: "Test resource content",
			},
		}
	}
	return nil
}

func (c *testServerImpl) Resources_ListChanged() chan struct{} {
	return c.resources_ListChanged
}

// CapToolsProvider implementation
func (c *testServerImpl) Tools_Started() *sync.Once {
	return &c.tools_started
}
func (c *testServerImpl) Tools_Capability() *CapTools {
	return &CapTools{ListChanged: c.tools_ListChanged != nil}
}
func (c *testServerImpl) Tools_OnList(cursor string) []ListToolsResonponse {
	return []ListToolsResonponse{
		{
			Tools: []ToolSpec{
				{
					Name:        "test_tool",
					Description: "Test tool for demonstration",
					InputSchema: ToolSchema{
						Type: "object",
						Properties: map[string]ParamSchema{
							"param1": {
								Type:        "string",
								Description: "Test parameter",
							},
						},
						Required: []string{"param1"},
					},
				},
			},
		},
	}
}
func (c *testServerImpl) Tools_ListChanged() chan struct{} { return c.tools_ListChanged }
func (c *testServerImpl) Tools_OnCall(name string, args map[string]string) (ToolCallResponse, *jsonrpc2.ErrorObject) {
	if name == "test_tool" {
		return ToolCallResponse{
			Content: []ToolCallContentUnion{
				{
					Type: "text",
					Text: "Tool executed successfully",
				},
			},
			IsError: false,
		}, nil
	}
	return ToolCallResponse{}, &jsonrpc2.ErrorObject{Code: -32601, Message: "Method not found"}
}

type testClientImpl struct {
	roots_ListChanged chan struct{}
	roots_started     sync.Once
}

func (c *testClientImpl) Roots_Started() *sync.Once {
	return &c.roots_started
}

func (c *testClientImpl) Roots_Capability() *CapRoots {
	return &CapRoots{ListChanged: c.roots_ListChanged != nil}
}

func (c *testClientImpl) Roots_OnList() []Root {
	return []Root{
		{URI: "file://myfile", Name: "Example Root"},
	}
}

func (c *testClientImpl) Roots_ListChanged() chan struct{} {
	return c.roots_ListChanged
}

func (c *testClientImpl) Sampling_Capability() *CapSampling {
	return new(CapSampling)
}

func (c *testClientImpl) HandleRequest(w jsonrpc2.ResponseWriterOf[SamplingResponse], msg SamplingMessage) error {
	res := SamplingResponse{}
	return w.WriteResponse(res)
}
