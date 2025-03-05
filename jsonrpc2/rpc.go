// Package jsonrpc2 is a JSON-RPC 2.0 client and server. It provides methods for sending requests and receiving responses.
package jsonrpc2

var (
	ErrObjMethodNotSupported = ErrorObject{
		Code:    JSONRPC2ErrorMethodNotFound,
		Message: "The method does not exist on the server.",
	}
	ErrObjParseError = ErrorObject{
		Code:    JSONRPC2ErrorParseError,
		Message: "Could not decode JSON object.",
	}
	ErrObjInvalidRequest = ErrorObject{Code: JSONRPC2ErrorInvalidRequest, Message: "Invalid request."}
	ErrObjInvalidParams  = ErrorObject{Code: JSONRPC2ErrorInvalidParams, Message: "Invalid parameters."}
)

// Handler is an interface for handling JSON-RPC requests. The HandleRequest
// method should be implemented to handle incoming requests and write responses
type Handler interface {
	HandleRequest(ResponseWriter, Request) error
}
