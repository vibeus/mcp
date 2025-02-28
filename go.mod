module github.com/vibeus/mcp

go 1.23.4

require go.lsp.dev/jsonrpc2 v0.10.0

require (
	github.com/google/uuid v1.6.0
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.3.4
	golang.org/x/sys v0.0.0-20211110154304-99a53858aa08 // indirect
)

replace github.com/vibeus/mcp/jsonrpc2 => ./jsonrpc2
