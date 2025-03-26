package mcp

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/vibeus/mcp/jsonrpc2"
)

var (
	DefaultServerPingTimeout time.Duration = 5 * time.Second
	DefaultServerRPCTimeout  time.Duration = 10 * time.Second
)

type ServerTimeout struct {
	PingTimeout time.Duration
	RPCTimeout  time.Duration
}

var (
	DefaultServerTimeout = ServerTimeout{
		DefaultServerPingTimeout,
		DefaultServerRPCTimeout,
	}
)

type ServerState struct {
	ctx  SessionContext
	impl ServerProvider
	rpc  *jsonrpc2.Peer

	timeoutConfig ServerTimeout
}

func NewServer(conn io.ReadWriteCloser) *ServerState {
	server := new(ServerState)
	server.timeoutConfig = DefaultServerTimeout

	s := session{}
	s.serverInfo = &ServerInfo{Name: "unnamed server", Version: "0"}
	server.ctx = s.Init(context.Background(), conn)
	return server
}

func (c *ServerState) SetLogger(logger *slog.Logger) {
	c.rpc.SetLogger(logger)
	c.ctx.GetSession().SetLogger(logger)
}

func (c *ServerState) SetMCPVersion(version string) {
	s := c.ctx.GetSession()
	s.SetProtocolVersion(version)
}

func (c *ServerState) SetCapabilities(sc ServerCapabilities) {
	s := c.ctx.GetSession()
	s.SetServerCapabilities(&sc)
}

func (c *ServerState) Setup(impl ServerProvider) {
	c.impl = impl
	c.rpc = jsonrpc2.NewPeer(c.ctx, jsonrpc2.NewLineFramer(c.ctx.GetSession().GetConn()), impl)
}

func (c *ServerState) Serve() error {
	c.impl.BindState(c)
	c.rpc.Start()
	return nil
}

func (c *ServerState) NotifyPromptsListChanged(ctx context.Context) error {
	to_ctx, cancel := context.WithTimeout(ctx, c.timeoutConfig.PingTimeout)
	defer cancel()

	select {
	case <-to_ctx.Done():
		return to_ctx.Err()
	default:
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug("Notify", "method", kMethodPromptsListChanged)
		}
		return c.rpc.Notify(kMethodPromptsListChanged, nil)
	}
}
