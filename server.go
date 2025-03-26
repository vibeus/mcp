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

type ServerState struct {
	ctx               SessionContext
	rpc               *jsonrpc2.Peer
	versionNegotiator MCPVersionNegotiator
}

func NewServer(conn io.ReadWriteCloser) *ServerState {
	server := new(ServerState)
	s := session{}
	s.serverInfo = &ServerInfo{Name: "unnamed server", Version: "0"}
	server.ctx = s.Init(context.Background(), conn)
	server.rpc = jsonrpc2.NewPeer(server.ctx, jsonrpc2.NewLineFramer(conn), &serverHandler{server})
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

func (c *ServerState) Serve() error {
	c.rpc.Start()
	return nil
}

func (c *ServerState) SamplingCreateMessage(msg SamplingMessage) error {
	res, err := c.rpc.Call(kMethodSamplingCreateMessage, msg)
	if err != nil {
		return err
	}
	var response SamplingResponse
	err = res.RecvResponse(&response)
	if err != nil {
		return err
	}
	return nil
}
