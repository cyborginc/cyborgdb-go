package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// Error codes from cyborgdb-tests' protocol (schema/protocol.schema.json).
const (
	protocolVersion = 1
	codeOpFailed    = -32000 // the native call failed; data.kind is set
	codeUnsupported = -32601 // not expressible in this API
	codeBadParams   = -32602 // params can't be mapped to a native call
	codeAdapter     = -32603 // the adapter itself failed: never a product error
	codeParse       = -32700
)

type rpcError struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    *errData `json:"data,omitempty"`
}

type errData struct {
	Kind   string `json:"kind"`
	Native string `json:"native"`
}

func (e *rpcError) Error() string { return e.Message }

// failf aborts the current op; dispatch turns the panic back into the response's error.
func failf(code int, format string, args ...any) {
	panic(&rpcError{Code: code, Message: fmt.Sprintf(format, args...)})
}

// The driver's pipes: requests in on fd 3, responses out on fd 4.
const (
	fdRequests  = 3
	fdResponses = 4
)

// A request line starts in a 1 MiB buffer and may grow to 64 MiB; bulk data travels as $tensor files.
const (
	lineBuffer  = 1 << 20
	maxLineSize = 64 << 20
)

var errNoDriver = errors.New("fd 3/4 not open; run under the cyborgdb-tests driver")

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// serveFDs reads requests from fd 3 and writes responses to fd 4, one per line, until fd 3 closes.
func serveFDs(h *handler) error {
	in, out := os.NewFile(fdRequests, "protocol-in"), os.NewFile(fdResponses, "protocol-out")
	if in == nil || out == nil {
		return errNoDriver
	}
	defer out.Close()
	return serve(in, out, h)
}

func serve(r io.Reader, w io.Writer, h *handler) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, lineBuffer), maxLineSize)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		b, err := json.Marshal(h.respond(line))
		if err != nil {
			b, err = json.Marshal(response{JSONRPC: "2.0", ID: json.RawMessage("null"),
				Error: &rpcError{Code: codeAdapter, Message: "encode response: " + err.Error()}})
			if err != nil {
				return fmt.Errorf("encode error response: %w", err)
			}
		}
		if _, err := w.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return sc.Err()
}

func (h *handler) respond(line []byte) (resp response) {
	resp = response{JSONRPC: "2.0", ID: json.RawMessage("null")}
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		resp.Error = &rpcError{Code: codeParse, Message: "unparseable request: " + err.Error()}
		return resp
	}
	if len(req.ID) > 0 {
		resp.ID = req.ID
	}
	result, err := h.handle(req.Method, req.Params)
	if err != nil {
		resp.Error = err
		return resp
	}
	if result == nil {
		result = json.RawMessage("null")
	}
	resp.Result = result
	return resp
}

// handle runs one op. An *rpcError panic is the op's error; any other panic is an adapter bug.
func (h *handler) handle(method string, raw json.RawMessage) (result any, rerr *rpcError) {
	defer func() {
		if p := recover(); p != nil {
			if e, ok := p.(*rpcError); ok {
				result, rerr = nil, e
				return
			}
			result, rerr = nil, &rpcError{Code: codeAdapter, Message: fmt.Sprintf("adapter error: %v", p)}
		}
	}()
	if method == "hello" {
		return h.hello(), nil
	}
	op, ok := ops[method]
	if !ok {
		return nil, &rpcError{Code: codeUnsupported, Message: "cyborgdb-go does not implement " + method}
	}
	p := params{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, &rpcError{Code: codeBadParams, Message: "params must be an object"}
		}
	}
	return op(h, p), nil
}
