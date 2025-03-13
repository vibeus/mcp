package jsonrpc2

import (
	"context"
	"encoding/json"
)

type PendingRequest struct {
	id         ID
	ctx        context.Context
	channel    chan responseData
	cancelFunc context.CancelFunc
}

func (p PendingRequest) GetID() ID {
	return p.id
}

// Cancel stops receiving further calls from the given request.
func (p PendingRequest) Cancel() {
	p.cancelFunc()
	// wait for channel to close
	<-p.channel
}

// RecvResponse receives a response from the given request. The output parameter
// is used to store the result of the call.
// It will be blocked until a response is received or the context is canceled.
//
// RecvResponse will return an [ErrContextCancel] wrapped in [RPCError] when the
// context is canceled.
func (p PendingRequest) RecvResponse(output any) error {
	response, ok := <-p.channel
	if ok {
		defer p.cancelFunc()
		if response.Result != nil {
			err := json.Unmarshal(*response.Result, output)
			if err != nil {
				return RPCError{err}
			}
			return nil
		}
		if response.Error != nil {
			return response.Error
		}
		return nil
	}
	return RPCError{ErrContextCancel}
}

// ResponseWriter writes the response of a request. It is used to send responses back to the client.
type ResponseWriter struct {
	id     *ID
	output chan []byte
}

func (w *ResponseWriter) WriteResponse(res any) error {
	var encoded_res json.RawMessage
	var err error
	encoded_res, err = json.Marshal(res)
	if err != nil {
		return err
	}

	r := responseData{
		Version: JSONRPC2Version,
		Result:  &encoded_res,
		ID:      w.id,
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	w.output <- data
	return nil
}

func (w *ResponseWriter) WriteError(res ErrorObject) error {
	r := responseData{
		Version: JSONRPC2Version,
		Error:   &res,
		ID:      w.id,
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	w.output <- data
	return nil
}

type responseWriterOf[T any] struct {
	w *ResponseWriter
}

func (r responseWriterOf[T]) WriteResponse(res T) error {
	return r.w.WriteResponse(res)
}

func (r responseWriterOf[T]) WriteError(err ErrorObject) error {
	return r.w.WriteError(err)
}

// MakeResponseWriterOf[T] wraps over the ResponseWriter interface and provides a generic implementation of WriteResponseOf[T].
func MakeResponseWriterOf[T any](w *ResponseWriter) ResponseWriterOf[T] {
	return responseWriterOf[T]{w}
}
