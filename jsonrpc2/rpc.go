package jsonrpc2

var (
	ErrObjMethodNotSupported = ErrorObject{
		Code:    JSONRPC2ErrorMethodNotFound,
		Message: "The method does not exist on the server.",
	}

	ErrObjInvalidRequest = ErrorObject{Code: JSONRPC2ErrorInvalidRequest, Message: "Invalid request."}
)

type ResponseWriter interface {
	WriteResponse(any) error
	WriteError(ErrorObject) error
}

type Handler interface {
	HandleRequest(ResponseWriter, Request) error
}
