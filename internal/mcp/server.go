package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// Server is an MCP server that communicates over stdio.
// It reads JSON-RPC messages from stdin, dispatches them to the Handler,
// and writes responses to stdout. All logging goes to stderr.
type Server struct {
	handler *Handler
	reader  io.Reader
	writer  io.Writer
}

// NewServer creates a Server that reads from stdin and writes to stdout.
func NewServer(handler *Handler) *Server {
	return &Server{
		handler: handler,
		reader:  os.Stdin,
		writer:  os.Stdout,
	}
}

// Run starts the server's main loop. It blocks until stdin returns EOF or
// an unrecoverable error occurs.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(s.reader)
	// Increase buffer size to handle large tool results.
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		resp, err := s.handler.ProcessMessage(line)
		if err != nil {
			slog.Error("fatal error processing message", "error", err)
			return fmt.Errorf("process message: %w", err)
		}

		// nil response means it was a notification — nothing to send back.
		if resp == nil {
			continue
		}

		if err := s.writeResponse(resp); err != nil {
			// If we can't write to stdout the client is gone — exit cleanly.
			slog.Debug("failed to write response, client may have disconnected", "error", err)
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	slog.Info("stdin closed, shutting down")
	return nil
}

func (s *Server) writeResponse(data []byte) error {
	// MCP stdio transport: one JSON object per line, terminated by \n.
	enc := json.NewEncoder(s.writer)
	enc.SetEscapeHTML(false)
	return enc.Encode(json.RawMessage(data))
}
