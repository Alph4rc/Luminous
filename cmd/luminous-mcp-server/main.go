// Command luminous-mcp-server runs an MCP (Model Context Protocol) server over
// stdio that exposes Luminous API tools for AI agents like Claude Code.
//
// Usage:
//
//	go run ./cmd/luminous-mcp-server
//
// Environment variables:
//
//	API_BASE_URL          – Luminous API base URL (default: http://localhost:8080)
//	API_TOKEN             – optional Bearer token for authenticated endpoints
//	HTTP_TIMEOUT_SECONDS  – request timeout in seconds (default: 10)
//	LOG_LEVEL             – debug|info|warn|error (default: info)
package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"luminous/internal/httpclient"
	"luminous/internal/mcp"
	"luminous/internal/skill"
	"luminous/internal/skills"
)

func main() {
	// ── logging ──────────────────────────────────────────────────
	logLevel := parseLogLevel(getEnv("LOG_LEVEL", "info"))
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))
	// After this point all slog calls go to stderr — stdout is reserved
	// for MCP JSON-RPC messages.

	// ── config ───────────────────────────────────────────────────
	baseURL := getEnv("API_BASE_URL", "http://localhost:8080")
	token := getEnv("API_TOKEN", "")
	timeoutSec := getEnvInt("HTTP_TIMEOUT_SECONDS", 10)

	// Safety: never log the actual token.
	if token != "" {
		slog.Info("API_TOKEN configured", "length", len(token))
	} else {
		slog.Info("API_TOKEN not set — requests will be sent without Authorization header")
	}

	slog.Info("starting Luminous MCP server",
		"baseURL", baseURL,
		"timeoutSec", timeoutSec,
		"logLevel", logLevel.String(),
	)

	// ── HTTP client ──────────────────────────────────────────────
	httpClient := httpclient.NewClient(baseURL, token, time.Duration(timeoutSec)*time.Second)

	// ── tool registry ────────────────────────────────────────────
	registry := skill.NewRegistry()

	if err := registry.Register(skills.NewQuerySchool(httpClient)); err != nil {
		slog.Error("failed to register query_school tool", "error", err)
		os.Exit(1)
	}

	slog.Info("registered tools", "count", len(registry.List()))

	// ── MCP server ───────────────────────────────────────────────
	handler := mcp.NewHandler(registry, "luminous-mcp-server", "1.0.0")
	server := mcp.NewServer(handler)

	if err := server.Run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}

	slog.Info("server shut down cleanly")
}

// ── helpers ─────────────────────────────────────────────────────

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		fmt.Fprintf(os.Stderr, "WARNING: invalid integer for %s=%q, using default %d\n", key, v, fallback)
	}
	return fallback
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		fmt.Fprintf(os.Stderr, "WARNING: unknown LOG_LEVEL %q, falling back to info\n", s)
		return slog.LevelInfo
	}
}
