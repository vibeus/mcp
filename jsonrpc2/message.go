package jsonrpc2

import (
	"encoding/json"
	"errors"
	"fmt"
)

type ID struct {
	name   string
	number int32
}

func MakeStringID(s string) ID {
	return ID{name: s}
}

func MakeNumberID(n int32) ID {
	return ID{number: n}
}

// MarshalJSON implements json.Marshaler.
func (id *ID) MarshalJSON() ([]byte, error) {
	if id.name != "" {
		return json.Marshal(id.name)
	}
	return json.Marshal(id.number)
}

func (id *ID) String() string {
	if id.name != "" {
		return id.name
	}
	return fmt.Sprintf("%d", id.number)
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *ID) UnmarshalJSON(data []byte) error {
	*id = ID{}
	if err := json.Unmarshal(data, &id.number); err == nil {
		return nil
	}
	return json.Unmarshal(data, &id.name)
}

type requestUnion struct {
	Version string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  *json.RawMessage `json:"params,omitempty"`
	ID      *ID              `json:"id,omitempty"`
}

type Request struct {
	Method string
	Params *json.RawMessage
	id     *ID // nil if is notification
}

func (r Request) GetID() *ID {
	return r.id
}

func (r Request) IsNotification() bool {
	return r.id == nil
}

type ErrorObject struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    *json.RawMessage `json:"data,omitempty"`
}

func (o ErrorObject) Error() string {
	data, _ := json.Marshal(o.Data)
	return fmt.Sprintf("jsonrpc2 error code %d: %s\n %s", o.Code, o.Message, data)
}

type responseUnion struct {
	Version string           `json:"jsonrpc"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject     `json:"error,omitempty"`
	ID      *ID              `json:"id"`
}

const (
	JSONRPC2Version             = "2.0"
	JSONRPC2ErrorParseError     = -32700
	JSONRPC2ErrorInvalidRequest = -32600
	JSONRPC2ErrorMethodNotFound = -32601
	JSONRPC2ErrorInvalidParams  = -32602
	JSONRPC2ErrorInternalError  = -32603
)

type RPCError struct {
	child error
}

func (e RPCError) Unwrap() error {
	return e.child
}

func (e RPCError) Error() string {
	return fmt.Sprintf("RPC error: %v", e.child)
}

var (
	ErrInvalidID      = errors.New("invalid ID")
	ErrInvalidContent = errors.New("invalid content")
	ErrContextCancel  = errors.New("context canceled")
	ErrNoHandler      = errors.New("no handler provided")
)
