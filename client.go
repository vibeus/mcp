package mcp

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/vibeus/mcp/jsonrpc2"
)

var (
	DefaultClientPingTimeout time.Duration = 5 * time.Second
	DefaultClientRPCTimeout  time.Duration = 10 * time.Second
)

type ClientTimeout struct {
	PingTimeout time.Duration
	RPCTimeout  time.Duration
}

var (
	DefaultClientTimeout = ClientTimeout{
		DefaultClientPingTimeout,
		DefaultClientRPCTimeout,
	}
)

type ClientState struct {
	ctx  SessionContext
	impl ClientProvider
	rpc  *jsonrpc2.Peer

	timeoutConfig ClientTimeout
}

func NewClient(conn io.ReadWriteCloser) *ClientState {
	client := new(ClientState)
	client.timeoutConfig = DefaultClientTimeout
	s := new(session)
	s.clientInfo = &ClientInfo{Name: "unnamed client", Version: "0"}
	s.conn = conn
	client.ctx = s.Init(context.Background(), conn)
	return client
}

func (c *ClientState) Setup(impl ClientProvider) {
	c.impl = impl
	c.rpc = jsonrpc2.NewPeer(c.ctx, jsonrpc2.NewLineFramer(c.ctx.GetSession().GetConn()), impl)
}

func (c *ClientState) SetLogger(logger *slog.Logger) {
	c.rpc.SetLogger(logger)
	s := c.ctx.GetSession()
	s.SetLogger(logger)
}

func (c *ClientState) SetMCPVersion(version string) {
	s := c.ctx.GetSession()
	s.SetProtocolVersion(version)
}

func (c *ClientState) SetCapabilities(cc ClientCapabilities) {
	s := c.ctx.GetSession()
	s.SetClientCapabilities(&cc)
}

func (c *ClientState) Ping(ctx context.Context) error {
	to_ctx, cancel := context.WithTimeout(ctx, c.timeoutConfig.PingTimeout)
	defer cancel()

	select {
	case <-to_ctx.Done():
		return to_ctx.Err()
	default:
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug(kMethodPing)
		}
		req, err := c.rpc.Call(kMethodPing, nil)
		if err == nil {
			err = req.RecvResponse(nil)
		}

		if err != nil {
			s.SetMCPState(MCPState_End)
			return err
		}
	}
	return nil
}

// Initialize is called by the client to negotiate MCPVersion and Capabilities
// with the server.
func (c *ClientState) Initialize(ctx context.Context) error {
	to_ctx, cancel := context.WithTimeout(ctx, c.timeoutConfig.RPCTimeout)
	defer cancel()

	select {
	case <-to_ctx.Done():
		return to_ctx.Err()
	default:
		s := c.ctx.GetSession()
		ci := new(ClientInitializeInfo)
		ci.ProtocolVersion = s.GetProtocolVersion()
		ci.ClientInfo = *s.GetClientInfo()
		ci.Capabilities = *s.GetClientCapabilities()

		si := new(ServerInitializeInfo)
		logger := s.GetLogger()
		req, err := c.rpc.Call(kMethodInitialize, ci)
		if err == nil {
			err = req.RecvResponse(&si)
		}
		if err != nil {
			s.SetMCPState(MCPState_End)
			return err
		}
		if logger != nil {
			logger.Debug("Call done", "method", kMethodInitialize, "server", si)
		}

		s.SetMCPState(MCPState_Initializing)
		s.SetProtocolVersion(si.ProtocolVersion)
		s.SetServerCapabilities(&si.Capabilities)
		s.SetServerInfo(&si.ServerInfo)
	}

	return nil
}

// Initialized is called to notify the server that it has finished negotiating MCPVersion and Capabilities.
func (c *ClientState) Initialized(ctx context.Context) error {
	s := c.ctx.GetSession()
	logger := s.GetLogger()
	if logger != nil {
		logger.Debug("Notify", "method", kMethodInitialized)
	}
	err := c.rpc.Notify(kMethodInitialized, nil)
	if err != nil {
		return err
	}

	c.impl.BindState(c)
	c.impl.StartClientProvider()
	s.SetMCPState(MCPState_Initialized)
	return nil
}

func (c *ClientState) NotifyRootsListChanged(ctx context.Context) error {
	to_ctx, cancel := context.WithTimeout(ctx, c.timeoutConfig.PingTimeout)
	defer cancel()

	select {
	case <-to_ctx.Done():
		return to_ctx.Err()
	default:
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug("Notify", "method", kMethodRootsListChanged)
		}
		return c.rpc.Notify(kMethodRootsListChanged, nil)
	}
}

func (c *ClientState) PromptsList(ctx context.Context, cursor string) ([]ListPromptsResponse, error) {
	s := c.ctx.GetSession()
	sc := s.GetServerCapabilities()
	if sc.Prompts == nil {
		return nil, jsonrpc2.ErrObjMethodNotSupported
	}

	to_ctx, cancel := context.WithTimeout(ctx, c.timeoutConfig.RPCTimeout)
	defer cancel()

	select {
	case <-to_ctx.Done():
		return nil, to_ctx.Err()
	default:
		params := PagedRequest{Cursor: cursor}
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug("Call", "method", kMethodPromptsList, "params", params)
		}
		var result []ListPromptsResponse
		req, err := c.rpc.Call(kMethodPromptsList, params)
		if err != nil {
			return nil, err
		}
		err = req.RecvResponse(&result)
		if logger != nil {
			logger.Debug("CallDone", "method", kMethodPromptsList, "result", result)
		}
		return result, err
	}
}

func (c *ClientState) PromptsGet(ctx context.Context, name string) (PromptGetResponse, error) {
	s := c.ctx.GetSession()
	sc := s.GetServerCapabilities()
	if sc.Prompts == nil {
		return PromptGetResponse{}, jsonrpc2.ErrObjMethodNotSupported
	}

	to_ctx, cancel := context.WithTimeout(ctx, c.timeoutConfig.RPCTimeout)
	defer cancel()

	select {
	case <-to_ctx.Done():
		return PromptGetResponse{}, to_ctx.Err()
	default:
		params := PromptGetRequest{Name: name}
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug("Call", "method", kMethodPromptsGet, "params", params)
		}
		var result PromptGetResponse
		req, err := c.rpc.Call(kMethodPromptsGet, params)
		if err != nil {
			return result, err
		}
		err = req.RecvResponse(&result)
		if logger != nil {
			logger.Debug("CallDone", "method", kMethodPromptsGet, "result", result)
		}
		return result, err
	}
}

func (c *ClientState) ToolsList(ctx context.Context, cursor string) ([]ListToolsResonponse, error) {
	s := c.ctx.GetSession()
	sc := s.GetServerCapabilities()
	if sc.Tools == nil {
		return nil, jsonrpc2.ErrObjMethodNotSupported
	}

	to_ctx, cancel := context.WithTimeout(ctx, c.timeoutConfig.RPCTimeout)
	defer cancel()

	select {
	case <-to_ctx.Done():
		return nil, to_ctx.Err()
	default:
		params := PagedRequest{Cursor: cursor}
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug("Call", "method", kMethodToolsList, "params", params)
		}
		var result []ListToolsResonponse
		req, err := c.rpc.Call(kMethodToolsList, params)
		if err != nil {
			return nil, err
		}
		err = req.RecvResponse(&result)
		if logger != nil {
			logger.Debug("CallDone", "method", kMethodToolsList, "result", result)
		}
		return result, err
	}
}

func (c *ClientState) ToolCall(ctx context.Context, name string, args map[string]string) (ToolCallResponse, error) {
	s := c.ctx.GetSession()
	sc := s.GetServerCapabilities()
	if sc.Tools == nil {
		return ToolCallResponse{}, jsonrpc2.ErrObjMethodNotSupported
	}

	to_ctx, cancel := context.WithTimeout(ctx, c.timeoutConfig.RPCTimeout)
	defer cancel()

	select {
	case <-to_ctx.Done():
		return ToolCallResponse{}, to_ctx.Err()
	default:
		params := ToolCallRequest{
			Name:      name,
			Arguments: args,
		}
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug("Call", "method", kMethodToolsCall, "params", params)
		}
		var result ToolCallResponse
		req, err := c.rpc.Call(kMethodToolsCall, params)
		if err != nil {
			return result, err
		}
		err = req.RecvResponse(&result)
		if logger != nil {
			logger.Debug("CallDone", "method", kMethodToolsCall, "result", result)
		}
		return result, err
	}
}

func (c *ClientState) ResourcesList(ctx context.Context, cursor string) (ResourcesListResponse, error) {
	s := c.ctx.GetSession()
	sc := s.GetServerCapabilities()
	if sc.Resources == nil {
		return ResourcesListResponse{}, jsonrpc2.ErrObjMethodNotSupported
	}

	to_ctx, cancel := context.WithTimeout(ctx, c.timeoutConfig.RPCTimeout)
	defer cancel()

	select {
	case <-to_ctx.Done():
		return ResourcesListResponse{}, to_ctx.Err()
	default:
		params := PagedRequest{Cursor: cursor}
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug("Call", "method", kMethodResourcesList, "params", params)
		}
		var result ResourcesListResponse
		req, err := c.rpc.Call(kMethodResourcesList, params)
		if err != nil {
			return ResourcesListResponse{}, err
		}
		err = req.RecvResponse(&result)
		if logger != nil {
			logger.Debug("CallDone", "method", kMethodResourcesList, "result", result)
		}
		return result, err
	}
}

func (c *ClientState) ResourcesTemplatesList(ctx context.Context) ([]ResourceTemplateSpec, error) {
	s := c.ctx.GetSession()
	sc := s.GetServerCapabilities()
	if sc.Resources == nil {
		return nil, jsonrpc2.ErrObjMethodNotSupported
	}

	to_ctx, cancel := context.WithTimeout(ctx, c.timeoutConfig.RPCTimeout)
	defer cancel()

	select {
	case <-to_ctx.Done():
		return nil, to_ctx.Err()
	default:
		params := PagedRequest{}
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug("Call", "method", kMethodResourcesTemplatesList, "params", params)
		}
		var result ResourcesTemplatesListResponse
		req, err := c.rpc.Call(kMethodResourcesTemplatesList, params)
		if err != nil {
			return nil, err
		}
		err = req.RecvResponse(&result)
		if logger != nil {
			logger.Debug("CallDone", "method", kMethodResourcesTemplatesList, "result", result)
		}
		return result.ResourceTemplates, err
	}
}

// ResourcesRead reads the content of a specific resource by URI
func (c *ClientState) ResourcesRead(ctx context.Context, uri string) ([]ResourceContentUnion, error) {
	s := c.ctx.GetSession()
	sc := s.GetServerCapabilities()
	if sc.Resources == nil {
		return nil, jsonrpc2.ErrObjMethodNotSupported
	}

	to_ctx, cancel := context.WithTimeout(ctx, c.timeoutConfig.RPCTimeout)
	defer cancel()

	select {
	case <-to_ctx.Done():
		return nil, to_ctx.Err()
	default:
		params := ResourcesReadRequest{URI: uri}
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug("Call", "method", kMethodResourcesRead, "params", params)
		}
		var result ResourcesReadResponse
		req, err := c.rpc.Call(kMethodResourcesRead, params)
		if err != nil {
			return nil, err
		}
		err = req.RecvResponse(&result)
		if logger != nil {
			logger.Debug("CallDone", "method", kMethodResourcesRead, "result", result)
		}
		return result.Content, err
	}
}
