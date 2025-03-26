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
