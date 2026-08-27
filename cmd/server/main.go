package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/app/docs"
	"centrachannel/internal/app/observability"
	"centrachannel/internal/app/registration"
	"centrachannel/internal/app/webhook"
	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/src/action"
	"centrachannel/internal/src/auth"
	"centrachannel/internal/src/campaign"
	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/conversation_tag"
	"centrachannel/internal/src/dashboard"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/note"
	"centrachannel/internal/src/tag"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/src/upload"
	"centrachannel/internal/src/user"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/ws"
)

func main() {
	c, err := di.New()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer func() {
		if cerr := c.Close(); cerr != nil {
			log.Printf("Error closing container: %v", cerr)
		}
	}()

	c.StartHub()

	cfg := c.Config

	app := fiber.New(fiber.Config{
		AppName: "CentraChannel API v1.0.0",
	})

	app.Use(middleware.CORSMiddleware())
	app.Use(middleware.LogMiddleware(c.Logger, cfg.Env))

	obsHandler := observability.NewObservabilityHandler()
	observability.RegisterRoutes(app, obsHandler)

	docs.RegisterRoutes(app)

	webhookHandler := webhook.NewWebhookHandler(c, cfg, ws.NewHubNotifier(c.Hub))
	webhook.RegisterRoutes(app, webhookHandler)

	regHandler := registration.NewRegistrationHandler(c)
	registration.RegisterRoutes(app, regHandler)

	wsHandler := ws.NewWSHandler(c.Hub, cfg.JWTSecret, c.Logger)
	app.Get("/ws", wsHandler.Handle)

	// Tenant Source
	app.Use(middleware.TenantMiddleware(c.Redis, c.DB))

	authMw := middleware.AuthMiddleware(cfg, c.Redis)
	adminOrAbove := middleware.RequireRole("super-admin", "admin")
	agentOrAbove := middleware.RequireRole("super-admin", "admin", "agent")

	auth.RegisterRoutes(app, "/api/auth", c)
	user.RegisterRoutes(app, "/api/user", c, authMw, adminOrAbove)
	campaign.RegisterRoutes(app, "/api/campaigns", c, authMw, agentOrAbove)
	channel.RegisterRoutes(app, "/api/channels", c, authMw, agentOrAbove)
	contact.RegisterRoutes(app, "/api/contacts", c, authMw, agentOrAbove)
	conversation.RegisterRoutes(app, "/api/conversations", c, authMw, agentOrAbove)
	conversation_tag.RegisterRoutes(app, "/api/conversations", c, authMw, agentOrAbove)
	message.RegisterRoutes(app, "/api/messages", c, authMw, agentOrAbove)
	tag.RegisterRoutes(app, "/api/tags", c, authMw, agentOrAbove)
	note.RegisterRoutes(app, "/api/note", c, authMw, agentOrAbove)
	whatsapp_device.RegisterRoutes(app, "/api/whatsapp-devices", c, authMw, agentOrAbove)
	action.RegisterRoutes(app, "/api/action", c, authMw, agentOrAbove)
	upload.RegisterRoutes(app, "/api/upload", c, authMw, agentOrAbove)
	dashboard.RegisterRoutes(app, "/api/dashboard", c, authMw, agentOrAbove)
	tenant.RegisterRoutes(app, "/api/tenant", c, authMw, adminOrAbove)

	c.Logger.Info("Starting server on port %d", cfg.Port)

	if err := app.Listen(":" + fmt.Sprintf("%d", cfg.Port)); err != nil {
		c.Logger.Fatal("Server error: %v", err)
	}
}
