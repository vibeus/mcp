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

// MarshalJSON implements [pkg/encoding/json.Marshaler].
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

// UnmarshalJSON implements [pkg/encoding/json.Unmarshaler].
func (id *ID) UnmarshalJSON(data []byte) error {
	*id = ID{}
	if err := json.Unmarshal(data, &id.number); err == nil {
		return nil
	}
	return json.Unmarshal(data, &id.name)
}

type wireUnion struct {
	Version string           `json:"jsonrpc"`
	Method  string           `json:"method,omitempty"` // for request
	Params  *json.RawMessage `json:"params,omitempty"` // for request

	Result *json.RawMessage `json:"result,omitempty"` // for normal response
	Error  *ErrorObject     `json:"error,omitempty"`  // for error response
	ID     *ID              `json:"id,omitempty"`     // for request and response
}

func (d wireUnion) IsResponse() bool {
	return d.ID != nil && d.Result != nil
}

func (d wireUnion) IsError() bool {
	return d.Error != nil
}

type requestData struct {
	Version string           `json:"jsonrpc"`
	Method  string           `json:"method,omitempty"`
	Params  *json.RawMessage `json:"params,omitempty"`
	ID      *ID              `json:"id,omitempty"`
}

type responseData struct {
	Version string           `json:"jsonrpc"`
	Result  *json.RawMessage `json:"result,omitempty"` // for normal response
	Error   *ErrorObject     `json:"error,omitempty"`  // for error response
	ID      *ID              `json:"id"`               // must be null, can't be omitted
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
	ErrInvalidContent = errors.New("jsonrpc2: invalid content")
	ErrContextCancel  = errors.New("jsonrpc2: context canceled")
	// When a request is received and the handler cannot be found, this error will be returned.
	ErrNoHandler = errors.New("jsonrpc2: no handler provided")
)
