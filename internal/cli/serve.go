package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sprite-ai/agrev/internal/api"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	Long: `Start an HTTP server exposing the agrev analysis engine.

Endpoints:
  GET  /health       — Health check
  POST /api/analyze  — Run analysis on a diff
  POST /api/parse    — Parse a diff into structured files
  POST /api/summary  — Generate summary from agent trace
  GET  /api/ws       — WebSocket for interactive review sessions`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringP("addr", "a", "127.0.0.1", "address to listen on")
	serveCmd.Flags().IntP("port", "p", 6142, "port to listen on")
}

func runServe(cmd *cobra.Command, args []string) error {
	addr, _ := cmd.Flags().GetString("addr")
	port, _ := cmd.Flags().GetInt("port")

	listen := fmt.Sprintf("%s:%d", addr, port)
	srv := api.New(listen)
	return srv.ListenAndServe()
}
