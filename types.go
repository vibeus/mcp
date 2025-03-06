package mcp

var (
	kMethodPing                  = "ping"
	kMethodInitialize            = "initialize"
	kMethodInitialized           = "initialized"
	kMethodRootsList             = "roots/list"
	kMethodRootsListChanged      = "notifications/roots/list_changed"
	kMethodSamplingCreateMessage = "sampling/createMessage"
	LatestMCPVersion             = "2024-11-05"
)

type CapRoots struct {
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
	Roots    *CapRoots    `json:"roots,omitempty"`
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
	Tools     *CapTools     `json:"tools,omitempty"`
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

type SamplingMessageContent struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

type SamplingMessageItem struct {
	Role    string                 `json:"role"`
	Content SamplingMessageContent `json:"content"`
}

type SamplingMessageModelHint struct {
	Name string `json:"name"`
}

type SamplingMessageModelPreference struct {
	Hints                []SamplingMessageModelHint `json:"hints"`
	CostPriority         float32                    `json:"costPriority"`
	CpeedPriority        float32                    `json:"speedPriority"`
	IntelligencePriority float32                    `json:"intelligencePriority"`
}

type SamplingMessage struct {
	Messages        []SamplingMessageItem          `json:"messages"`
	ModelPreference SamplingMessageModelPreference `json:"modelPreference"`
	SystemPrompt    string                         `json:"systemPrompt"`
	MaxTokens       int64                          `json:"maxTokens"`
}

type SamplingResponse struct {
	Role       string                 `json:"role"`
	Content    SamplingMessageContent `json:"content"`
	Model      string                 `json:"model"`
	StopReason string                 `json:"stopReason"`
}
