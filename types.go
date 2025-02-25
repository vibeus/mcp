package mcp

var (
	kMethodPing        = "ping"
	kMethodInitialize  = "initialize"
	kMethodInitialized = "initialized"
	LatestMCPVersion   = "2024-11-05"
)

type CapRoot struct {
	ListChanged bool `json:"listChanged"`
}

type CapSampling struct{}

type CapLogging struct{}

type CapPrompts struct {
	ListChanged bool `json:"listChanged"`
}

type CapResources struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

type CapTools struct {
	ListChanged bool `json:"listChanged"`
}

type ClientCapabilities struct {
	Root     *CapRoot     `json:"root,omitempty"`
	Sampling *CapSampling `json:"sampling,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ClientInitializeInfo struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
	Logging   *CapLogging   `json:"logging,omitempty"`
	Prompts   *CapPrompts   `json:"prompts,omitempty"`
	Resources *CapResources `json:"resources,omitempty"`
	Tools     *CapTools     `json:"omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerInitializeInfo struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
}
