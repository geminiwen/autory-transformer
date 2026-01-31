package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/geminiwen/anthropic-to-ark/config"
	"github.com/geminiwen/anthropic-to-ark/internal/handler"
)

func main() {
	// Load configuration
	if err := config.Load(); err != nil {
		hlog.Fatalf("Failed to load config: %v", err)
	}

	cfg := config.Get()
	hlog.Infof("Starting Anthropic to Ark proxy on port %s", cfg.Port)

	// Start Anthropic passthrough proxy on port 10091
	// go startAnthropicProxy() // Disabled for now

	// Initialize Hertz server for Ark proxy
	h := server.Default(server.WithHostPorts(fmt.Sprintf(":%s", cfg.Port)))

	// Setup routes
	setupRoutes(h)

	// Start server
	h.Spin()
}

func startAnthropicProxy() {
	hlog.Infof("Starting Anthropic passthrough proxy on port 10091")

	h := server.New(server.WithHostPorts(":10091"))

	anthropicHandler := handler.NewAnthropicProxyHandler()

	// Health check
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, map[string]string{
			"status": "ok",
			"proxy":  "anthropic-passthrough",
		})
	})

	// Messages API - passthrough to Anthropic
	h.POST("/v1/messages", anthropicHandler.Handle)

	h.Spin()
}

func setupRoutes(h *server.Hertz) {
	messagesHandler := handler.NewMessagesHandler()

	// Health check
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, map[string]string{
			"status": "ok",
		})
	})

	// Messages API
	h.POST("/v1/messages", messagesHandler.Handle)
}
