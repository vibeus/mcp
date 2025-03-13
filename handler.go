package mcp

import (
	"encoding/json"

	"github.com/vibeus/mcp/jsonrpc2"
)

type serverHandler struct {
	server *Server
}

func (h *serverHandler) HandleNotification(req jsonrpc2.Request) error {
	return nil
}

func (h *serverHandler) HandleRequest(w *jsonrpc2.ResponseWriter, req jsonrpc2.Request) error {
	s := h.server.ctx.GetSession()
	switch s.GetMCPState() {
	case MCPState_Start:
		err := h.handleStart(w, req)
		if err != nil {
			return err
		}
	case MCPState_Initializing:
		err := h.handleInitializing(w, req)
		if err != nil {
			return err
		}
	case MCPState_Initialized:
	}
	return nil
}

func (h *serverHandler) handleStart(w *jsonrpc2.ResponseWriter, req jsonrpc2.Request) error {
	s := h.server.ctx.GetSession()
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

		version := h.server.versionNegotiator.NegotiateMCPVersion(ci.ProtocolVersion)
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

func (h *serverHandler) handleInitializing(w *jsonrpc2.ResponseWriter, req jsonrpc2.Request) error {
	switch req.Method {
	case kMethodPing:
		return w.WriteResponse(nil)
	case kMethodInitialized:
		if !req.IsNotification() {
			return w.WriteError(jsonrpc2.ErrObjInvalidRequest)
		}
		s := h.server.ctx.GetSession()
		s.SetMCPState(MCPState_Initialized)
		return nil
	default:
		return w.WriteError(jsonrpc2.ErrObjMethodNotSupported)
	}
}
