package mcp

import (
	"context"
	"io"

	"log/slog"

	"github.com/google/uuid"
	"github.com/vibeus/mcp/jsonrpc2"
)

func init() {
	uuid.EnableRandPool()
}

type sessionContextKey struct{}

type SessionContext struct {
	context.Context
}

func (c SessionContext) GetSession() Session {
	return c.Value(sessionContextKey{}).(Session)
}

type Session interface {
	// Init a new session.
	Init(ctx context.Context, conn io.ReadWriteCloser) SessionContext
	// Close the session.
	Close()

	SessionID() string

	// Get the connection.
	GetConn() io.ReadWriteCloser

	NextID() jsonrpc2.ID

	// Get/Set the logger.
	GetLogger() *slog.Logger
	SetLogger(*slog.Logger)

	// Get/Set the protocol version
	GetProtocolVersion() string
	SetProtocolVersion(string)

	// Get/Set the server info.
	GetServerInfo() *ServerInfo
	SetServerInfo(*ServerInfo)

	// Get/Set the client info.
	GetClientInfo() *ClientInfo
	SetClientInfo(*ClientInfo)

	// Get/Set the server capabilities.
	GetServerCapabilities() *ServerCapabilities
	SetServerCapabilities(*ServerCapabilities)

	// Get/Set the client capabilities.
	GetClientCapabilities() *ClientCapabilities
	SetClientCapabilities(*ClientCapabilities)

	// Get/Set the server state.
	GetMCPState() MCPState
	SetMCPState(MCPState)
}

type MCPState int

const (
	MCPState_Start MCPState = iota
	MCPState_Initializing
	MCPState_Initialized
	MCPState_End
)

func (s MCPState) String() string {
	switch s {
	case MCPState_Start:
		return "start"
	case MCPState_Initializing:
		return "initializing"
	case MCPState_Initialized:
		return "initialized"
	default:
		return "unknown"
	}
}

// Impl Session
type session struct {
	id              string
	conn            io.ReadWriteCloser
	logger          *slog.Logger
	protocolVersion string
	serverInfo      *ServerInfo
	clientInfo      *ClientInfo
	serverCaps      *ServerCapabilities
	clientCaps      *ClientCapabilities
	cancel          context.CancelFunc
	mcpState        MCPState
	requestID       int32
}

func (s *session) Init(ctx context.Context, conn io.ReadWriteCloser) SessionContext {
	s.id = uuid.New().String()
	s.conn = conn

	var pctx context.Context
	pctx, s.cancel = context.WithCancel(ctx)
	// close the connection after session is closed
	context.AfterFunc(pctx, func() {
		if s.conn != nil {
			s.conn.Close()
		}
	})

	return SessionContext{context.WithValue(pctx, sessionContextKey{}, s)}
}

func (s *session) Close() {
	s.cancel()
}

func (s session) SessionID() string {
	return s.id
}

func (s session) GetConn() io.ReadWriteCloser {
	return s.conn
}

func (s *session) NextID() jsonrpc2.ID {
	id := jsonrpc2.MakeNumberID(s.requestID)
	s.requestID++
	return id
}

func (s session) GetLogger() *slog.Logger {
	return s.logger
}

func (s *session) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

func (s session) GetProtocolVersion() string {
	return s.protocolVersion
}

func (s *session) SetProtocolVersion(version string) {
	s.protocolVersion = version
}

func (s session) GetServerInfo() *ServerInfo {
	return s.serverInfo
}

func (s *session) SetServerInfo(si *ServerInfo) {
	s.serverInfo = si
}

func (s session) GetClientInfo() *ClientInfo {
	return s.clientInfo
}

func (s *session) SetClientInfo(ci *ClientInfo) {
	s.clientInfo = ci
}

func (s session) GetServerCapabilities() *ServerCapabilities {
	return s.serverCaps
}

func (s *session) SetServerCapabilities(sc *ServerCapabilities) {
	s.serverCaps = sc
}

func (s session) GetClientCapabilities() *ClientCapabilities {
	return s.clientCaps
}

func (s *session) SetClientCapabilities(cc *ClientCapabilities) {
	s.clientCaps = cc
}

func (s session) GetMCPState() MCPState {
	return s.mcpState
}

func (s *session) SetMCPState(ms MCPState) {
	if s.logger != nil {
		s.logger.Debug("Setting MCP state", "state", ms.String())
	}
	s.mcpState = ms
}
