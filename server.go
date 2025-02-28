package mcp

import (
	"context"
	"io"
	"log/slog"

	"github.com/vibeus/mcp/jsonrpc2"
)

type MCPVersionNegotiator interface {
	// given a client version string, return the server's version string.
	NegotiateMCPVersion(clientVersion string) string
}

type Server struct {
	ctx               SessionContext
	rpcServer         *jsonrpc2.Server
	versionNegotiator MCPVersionNegotiator
}

func NewServer(conn io.ReadWriteCloser) *Server {
	server := new(Server)
	s := session{}
	s.serverInfo = &ServerInfo{Name: "unnamed server", Version: "0"}
	server.ctx = s.Init(context.Background(), conn)
	server.rpcServer = jsonrpc2.NewServer(server.ctx, conn, jsonrpc2.NewLineFramer(conn), &serverHandler{server})
	return server
}

func (c *Server) SetLogger(logger *slog.Logger) {
	c.ctx.GetSession().SetLogger(logger)
}

func (c *Server) SetMCPVersion(version string) {
	s := c.ctx.GetSession()
	s.SetProtocolVersion(version)
}

func (c *Server) SetCapabilities(sc ServerCapabilities) {
	s := c.ctx.GetSession()
	s.SetServerCapabilities(&sc)
}

func (c *Server) Serve() error {
	for {
		err := c.rpcServer.Serve()
		if err != nil {
			c.ctx.GetSession().Close()
			return err
		}
	}
}
