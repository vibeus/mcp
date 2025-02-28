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

type Client struct {
	ctx       SessionContext
	rpcClient *jsonrpc2.Client

	timeoutConfig ClientTimeout
}

func NewClient(conn io.ReadWriteCloser) *Client {
	client := new(Client)
	client.timeoutConfig = ClientTimeout{
		PingTimeout: DefaultClientPingTimeout,
		RPCTimeout:  DefaultClientRPCTimeout,
	}
	s := new(session)
	s.clientInfo = &ClientInfo{Name: "unnamed client", Version: "0"}
	s.conn = conn
	client.ctx = s.Init(context.Background(), conn)
	client.rpcClient = jsonrpc2.NewClient(client.ctx, conn, jsonrpc2.NewLineFramer(conn))
	return client
}

func (c *Client) SetLogger(logger *slog.Logger) {
	s := c.ctx.GetSession()
	s.SetLogger(logger)
}

func (c *Client) SetMCPVersion(version string) {
	s := c.ctx.GetSession()
	s.SetProtocolVersion(version)
}

func (c *Client) SetCapabilities(cc ClientCapabilities) {
	s := c.ctx.GetSession()
	s.SetClientCapabilities(&cc)
}

func (c *Client) Ping(ctx context.Context) error {
	to_ctx, _ := context.WithTimeout(ctx, c.timeoutConfig.PingTimeout)
	select {
	case <-to_ctx.Done():
		return to_ctx.Err()
	default:
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug(kMethodPing)
		}
		id := s.NextID()
		err := c.rpcClient.Call(id, kMethodPing, nil, nil)

		if err != nil {
			s.SetMCPState(MCPState_End)
			return err
		}

		if logger != nil {
			logger.Info(kMethodPing, "id", id.String())
		}
	}
	return nil
}

// Initialize is called by the client to negotiate MCPVersion and Capabilities
// with the server.
func (c Client) Initialize(ctx context.Context) error {
	to_ctx, _ := context.WithTimeout(ctx, c.timeoutConfig.RPCTimeout)

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
		id := s.NextID()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug("Call", "method", kMethodInitialize, "id", id.String(), "client", ci)
		}
		err := c.rpcClient.Call(id, kMethodInitialize, ci, &si)
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
		if logger != nil {
			logger.Info(kMethodInitialize, "id", id, "clientInitInfo", ci, "serverInitInfo", si)
		}
	}

	return nil
}

// Initialized is called to notify the server that it has finished negotiating MCPVersion and Capabilities.
func (c Client) Initialized(ctx context.Context) error {
	s := c.ctx.GetSession()
	logger := s.GetLogger()
	if logger != nil {
		logger.Debug("Notify", "method", kMethodInitialized)
	}
	err := c.rpcClient.Notify(kMethodInitialized, nil)
	if err != nil {
		return err
	}

	if logger != nil {
		logger.Info(kMethodInitialized)
	}
	s.SetMCPState(MCPState_Initialized)
	return nil
}
