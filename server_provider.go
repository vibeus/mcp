package mcp

import (
	"encoding/json"
	"sync"

	"github.com/vibeus/mcp/jsonrpc2"
)

type MCPVersionNegotiator interface {
	// given a client version string, return the server's version string.
	NegotiateMCPVersion(clientVersion string) string
}

type ServerProvider interface {
	jsonrpc2.Handler
	BindState(*ServerState)
	Capabilities() ServerCapabilities
	StartServerProvider()
}

type ServerImpl struct {
	server *ServerState
	MCPVersionNegotiator
	CapPromptsProvider
	CapToolsProvider
	CapResourcesProvider // Add resources capability provider

	once sync.Once
}

func (c *ServerImpl) BindState(server *ServerState) {
	c.server = server
}

func (c *ServerImpl) StartServerProvider() {
	if c.server == nil {
		panic("must call BindState before StartServerProvider")
	}
	c.once.Do(func() {
		if c.CapPromptsProvider != nil {
			startCapPrompts(c.server, c.CapPromptsProvider)
		}
		if c.CapToolsProvider != nil {
			startCapTools(c.server, c.CapToolsProvider)
		}
		if c.CapResourcesProvider != nil { // Start resources capability
			startCapResources(c.server, c.CapResourcesProvider)
		}
	})
}

func (c *ServerImpl) Capabilities() ServerCapabilities {
	cap := ServerCapabilities{}
	if c.CapPromptsProvider != nil {
		cap.Prompts = c.CapPromptsProvider.Prompts_Capability()
	}
	if c.CapToolsProvider != nil {
		cap.Tools = c.CapToolsProvider.Tools_Capability()
	}
	if c.CapResourcesProvider != nil { // Add resources capability
		cap.Resources = c.CapResourcesProvider.Resources_Capability()
	}
	return cap
}

func (c *ServerImpl) HandleNotification(req jsonrpc2.Request) error {
	return nil
}

func (c *ServerImpl) HandleRequest(w *jsonrpc2.ResponseWriter, req jsonrpc2.Request) error {
	s := c.server.ctx.GetSession()
	switch s.GetMCPState() {
	case MCPState_Start:
		err := c.handleStart(w, req)
		if err != nil {
			return err
		}
	case MCPState_Initializing:
		err := c.handleInitializing(w, req)
		if err != nil {
			return err
		}
	case MCPState_Initialized:
		switch req.Method {
		case kMethodPromptsList:
			if c.CapPromptsProvider != nil {
				var msg PagedRequest
				err := json.Unmarshal(*req.Params, &msg)
				if err != nil {
					w.WriteError(jsonrpc2.ErrObjInvalidParams)
					return nil
				}
				prompts := c.CapPromptsProvider.Prompts_OnList(msg.Cursor)
				w.WriteResponse(prompts)
			} else {
				w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
			}
			return nil
		case kMethodPromptsGet:
			if c.CapPromptsProvider != nil {
				var msg PromptGetRequest
				err := json.Unmarshal(*req.Params, &msg)
				if err != nil {
					w.WriteError(jsonrpc2.ErrObjInvalidParams)
					return nil
				}
				promptName := msg.Name
				response, erro := c.CapPromptsProvider.Prompts_OnGet(promptName)
				if erro != nil {
					w.WriteError(*erro)
					return nil
				}
				w.WriteResponse(response)
			} else {
				w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
			}
			return nil
		case kMethodToolsList:
			if c.CapToolsProvider != nil {
				var msg PagedRequest
				err := json.Unmarshal(*req.Params, &msg)
				if err != nil {
					w.WriteError(jsonrpc2.ErrObjInvalidParams)
					return nil
				}
				tools := c.CapToolsProvider.Tools_OnList(msg.Cursor)
				w.WriteResponse(tools)
			} else {
				w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
			}
			return nil
		case kMethodToolsCall:
			if c.CapToolsProvider != nil {
				var msg ToolCallRequest
				err := json.Unmarshal(*req.Params, &msg)
				if err != nil {
					w.WriteError(jsonrpc2.ErrObjInvalidParams)
					return nil
				}
				response, erro := c.CapToolsProvider.Tools_OnCall(msg.Name, msg.Arguments)
				if erro != nil {
					w.WriteError(*erro)
					return nil
				}
				w.WriteResponse(response)
			} else {
				w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
			}
			return nil
		case kMethodResourcesList:
			if c.CapResourcesProvider != nil { // Add resources capability
				var msg PagedRequest
				err := json.Unmarshal(*req.Params, &msg)
				if err != nil {
					w.WriteError(jsonrpc2.ErrObjInvalidParams)
					return nil
				}
				resources := c.CapResourcesProvider.Resources_OnList(msg.Cursor)
				response := ResourcesListResponse{}
				response.Resources = resources
				w.WriteResponse(response)
			} else {
				w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
			}
			return nil
		case kMethodResourcesTemplatesList:
			if c.CapResourcesProvider != nil { // Add resources capability
				resources := c.CapResourcesProvider.Resources_OnTemplatesList()
				response := ResourcesTemplatesListResponse{}
				response.ResourceTemplates = resources
				w.WriteResponse(response)
			} else {
				w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
			}
			return nil
		case kMethodResourcesRead:
			if c.CapResourcesProvider != nil { // Add resources capability
				var msg ResourcesReadRequest
				err := json.Unmarshal(*req.Params, &msg)
				if err != nil {
					w.WriteError(jsonrpc2.ErrObjInvalidParams)
					return nil
				}
				uri := msg.URI
				response := ResourcesReadResponse{}
				response.Content = c.CapResourcesProvider.Resources_OnRead(uri)
				if len(response.Content) == 0 {
					obj := kErrObjResourceNotFound
					var data struct {
						Uri string `json:"uri"`
					}
					data.Uri = uri
					datajson, _ := json.Marshal(data)
					obj.Data.UnmarshalJSON(datajson)
					w.WriteError(obj)
				} else {
					w.WriteResponse(response)
				}
			} else {
				w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
			}
			return nil
		}
	}
	return nil
}

func (c *ServerImpl) handleStart(w *jsonrpc2.ResponseWriter, req jsonrpc2.Request) error {
	s := c.server.ctx.GetSession()
	logger := s.GetLogger()
	if logger != nil {
		logger.Debug("startHandler", "method", req.Method, "params", req.Params)
	}

	switch req.Method {
	case kMethodPing:
		return w.WriteResponse(nil)
	case kMethodInitialize:
		ci := new(ClientInitializeInfo)

		err := json.Unmarshal(*req.Params, ci)
		if err != nil {
			return err
		}

		s.SetMCPState(MCPState_Initializing)
		s.SetClientCapabilities(&ci.Capabilities)
		s.SetClientInfo(&ci.ClientInfo)

		version := c.MCPVersionNegotiator.NegotiateMCPVersion(ci.ProtocolVersion)
		s.SetProtocolVersion(version)

		si := new(ServerInitializeInfo)
		si.ProtocolVersion = version
		si.Capabilities = *s.GetServerCapabilities()
		si.ServerInfo = *s.GetServerInfo()
		return w.WriteResponse(si)
	default:
		return w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
	}
}

func (c *ServerImpl) handleInitializing(w *jsonrpc2.ResponseWriter, req jsonrpc2.Request) error {
	switch req.Method {
	case kMethodPing:
		return w.WriteResponse(nil)
	case kMethodInitialized:
		if !req.IsNotification() {
			return w.WriteError(jsonrpc2.ErrObjInvalidRequest)
		}
		s := c.server.ctx.GetSession()
		c.StartServerProvider()
		s.SetMCPState(MCPState_Initialized)
		return nil
	default:
		return w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
	}
}
