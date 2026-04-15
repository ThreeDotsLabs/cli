package mcp

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

const DefaultPort = 39131

// Server wraps an MCP server that exposes training loop controls.
type Server struct {
	state      *LoopState
	port       int
	httpServer *server.StreamableHTTPServer
}

func NewServer(state *LoopState, port int) *Server {
	hooks := &server.Hooks{}
	hooks.AddAfterInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest, result *mcp.InitializeResult) {
		state.SetMCPClientInfo(message.Params.ClientInfo.Name, message.Params.ClientInfo.Version)
		logrus.WithFields(logrus.Fields{
			"client_name":    message.Params.ClientInfo.Name,
			"client_version": message.Params.ClientInfo.Version,
		}).Info("MCP client connected")
	})

	mcpServer := server.NewMCPServer(
		"tdl-training",
		"1.0.0",
		server.WithHooks(hooks),
	)

	registerTools(mcpServer, state)

	httpServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithStateLess(true),
	)

	return &Server{
		state:      state,
		port:       port,
		httpServer: httpServer,
	}
}

// Start begins serving in a goroutine. Returns immediately.
// If the port is in use, logs a warning and returns nil (MCP is optional).
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)

	// Check port availability early to give a clear message
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logrus.WithError(err).WithField("addr", addr).Warn("MCP server: port unavailable, running without MCP")
		return nil
	}
	ln.Close()

	go func() {
		logrus.WithField("addr", addr).Info("MCP server starting")
		if err := s.httpServer.Start(addr); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Warn("MCP server stopped with error")
		}
	}()

	// Shut down when context is cancelled
	go func() {
		<-ctx.Done()
		s.httpServer.Shutdown(context.Background())
	}()

	return nil
}

// Addr returns the address the server is configured to listen on.
func (s *Server) Addr() string {
	return fmt.Sprintf("127.0.0.1:%d", s.port)
}
