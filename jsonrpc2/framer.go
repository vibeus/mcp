package jsonrpc2

import (
	"bufio"
	"bytes"
	"io"
	"unicode/utf8"
)

// Framer is an interface for framing JSON-RPC2 messages over a wire. It provides methods to read and write frames of data.
type Framer interface {
	ReadFrame() ([]byte, error)
	WriteFrame([]byte) error
}

// LineFramer implements a simple line-based JSON-RPC2 framing. It reads and writes lines of text from the underlying wire.
type LineFramer struct {
	wire    io.ReadWriteCloser
	scanner *bufio.Scanner
}

func NewLineFramer(w io.ReadWriteCloser) *LineFramer {
	return &LineFramer{
		wire:    w,
		scanner: bufio.NewScanner(w),
	}
}

func (c *LineFramer) ReadFrame() ([]byte, error) {
	if c.scanner.Scan() {
		return []byte(c.scanner.Bytes()), nil
	}
	return []byte(c.scanner.Bytes()), c.scanner.Err()
}

func (c LineFramer) WriteFrame(input []byte) error {
	var buf bytes.Buffer
	b := input

	// filter invalid utf8 characters
	for {
		r, size := utf8.DecodeRune(b)
		if size == 0 {
			buf.WriteRune('\n')
			break
		}
		if r == '\n' {
			return ErrInvalidContent
		}
		_, err := buf.WriteRune(r)
		if err != nil {
			return err
		}
		b = b[size:]
	}

	frame := buf.Bytes()
	total := 0
	for {
		n, err := c.wire.Write(frame)
		if err != nil {
			return err
		}
		total += n
		if total >= len(frame) {
			break
		}
	}
	return nil
}
