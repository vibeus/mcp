package mcp

import (
	"context"
	"log/slog"

	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
)

type MCPVersionNegotiator interface {
	// given a client version string, return the server's version string.
	NegotiateMCPVersion(clientVersion string) string
}

type Server struct {
	ctx               SessionContext
	versionNegotiator MCPVersionNegotiator
}

func NewServer(conn jsonrpc2.Conn) *Server {
	server := new(Server)
	s := session{}
	s.serverInfo = &ServerInfo{Name: "unnamed server", Version: "0"}
	server.ctx = s.Init(context.Background(), conn)
	return server
}

func (c Server) SetLogger(logger *slog.Logger) {
	c.ctx.GetSession().SetLogger(logger)
}

func (c Server) SetMCPVersion(version string) {
	s := c.ctx.GetSession()
	s.SetProtocolVersion(version)
}

func (c Server) SetCapabilities(sc ServerCapabilities) {
	s := c.ctx.GetSession()
	s.SetServerCapabilities(&sc)
}

// impl jsonrpc2.StreamServer
func (c Server) ServeStream(ctx context.Context, conn jsonrpc2.Conn) error {
	s := c.ctx.GetSession()
	for {
		switch s.GetMCPState() {
		case MCPState_Start:
			go jsonrpc2.HandlerServer(startHandler(c)).ServeStream(ctx, conn)
			return nil
		case MCPState_Initializing:
			go jsonrpc2.HandlerServer(initializingHandler(c)).ServeStream(ctx, conn)
			return nil
		case MCPState_Initialized:
			return nil
		}
	}
}

func startHandler(c Server) jsonrpc2.Handler {
	return func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		s := c.ctx.GetSession()
		logger := s.GetLogger()
		if logger != nil {
			logger.Debug("startHandler", "method", req.Method(), "params", req.Params())
		}

		switch req.Method() {
		case kMethodPing:
			return reply(ctx, nil, nil)
		case kMethodInitialize:
			ci := new(ClientInitializeInfo)
			err := json.Unmarshal(req.Params(), ci)
			if err != nil {
				return jsonrpc2.Errorf(jsonrpc2.ParseError, "can't parse ClientInitializeInfo")
			}
			s.SetMCPState(MCPState_Initializing)
			s.SetClientCapabilities(&ci.Capabilities)
			s.SetClientInfo(&ci.ClientInfo)

			version := c.versionNegotiator.NegotiateMCPVersion(ci.ProtocolVersion)
			s.SetProtocolVersion(version)

			si := new(ServerInitializeInfo)
			si.ProtocolVersion = version
			si.Capabilities = *s.GetServerCapabilities()
			si.ServerInfo = *s.GetServerInfo()
			result, _ := json.MarshalIndent(si, "", " ")
			return reply(ctx, result, nil)
		default:
			return jsonrpc2.MethodNotFoundHandler(ctx, reply, req)
		}
	}
}

func initializingHandler(c Server) jsonrpc2.Handler {
	return func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		switch req.Method() {
		case kMethodPing:
			return reply(ctx, nil, nil)
		case kMethodInitialized:
			s := c.ctx.GetSession()
			s.SetMCPState(MCPState_Initialized)
			return nil
		default:
			return jsonrpc2.MethodNotFoundHandler(ctx, reply, req)
		}
	}
}
