package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
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
	"centrachannel/internal/src/webhook"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/utils/response"
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

	cfg := c.Config

	app := fiber.New(fiber.Config{
		AppName: "CentraChannel API v1.0.0",
	})

	app.Use(middleware.TenantMiddleware(c.Redis, c.DB))
	app.Use(middleware.LoggerMiddleware())
	app.Use(middleware.CORSMiddleware(cfg.CORSAllowedOrigins))

	app.Get("/", func(c fiber.Ctx) error {
		return response.OK(c, "Welcome to CentraChannel API", fiber.Map{
			"version": "1.0.0",
			"status":  "running",
		})
	})

	app.Get("/health", func(c fiber.Ctx) error {
		return response.OK(c, "healthy", fiber.Map{"status": "ok"})
	})
	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("pong")
	})
	app.Get("/version", func(c fiber.Ctx) error {
		return response.OK(c, "ok", fiber.Map{"version": "1.0.0", "app": "CentraChannel API"})
	})

	// API Documentation
	app.Get("/docs/*", static.New("./docs"))

	app.Get("/docs", func(c fiber.Ctx) error {
		c.Type("html", "utf-8")
		return c.SendString(`<!DOCTYPE html>
<html>
<head>
  <title>CentraChannel API - Redoc</title>
  <meta charset="utf-8"/>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</head>
<body>
  <div id="redoc-container"></div>
  <script>
    Redoc.init('/docs/api/openapi.json', {}, document.getElementById('redoc-container'))
  </script>
</body>
</html>`)
	})

	app.Get("/swagger", func(c fiber.Ctx) error {
		c.Type("html", "utf-8")
		return c.SendString(`<!DOCTYPE html>
<html>
<head>
  <title>CentraChannel API - Swagger UI</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: '/docs/api/openapi.json',
      dom_id: '#swagger-ui'
    })
  </script>
</body>
</html>`)
	})

	hub := ws.NewHub()
	go hub.Run()

	webhookHandler := webhook.NewWebhookHandler(c, cfg, hub)
	webhook.RegisterRoutes(app, webhookHandler)

	authHandler := auth.NewAuthHandler(c)
	auth.RegisterRoutes(app, authHandler)

	tenantHandler := tenant.NewTenantHandler(c)
	tenant.RegisterPublicRoutes(app, tenantHandler)

	// Protected routes
	authMw := middleware.AuthMiddleware(cfg)
	adminOrAbove := middleware.RequireRole("super-admin", "admin")
	agentOrAbove := middleware.RequireRole("super-admin", "admin", "agent")

	wsHandler := ws.NewHandler(hub, cfg)
	app.Get("/ws", authMw, middleware.Tenant(wsHandler.Handle))

	userGroup := app.Group("/api/user", authMw, adminOrAbove)
	userHandler := user.NewUserHandler(c)
	user.RegisterRoutesByGroup(userGroup, userHandler)

	campaignGroup := app.Group("/api/campaigns", authMw, agentOrAbove)
	campaignHandler := campaign.NewCampaignHandler(c)
	campaign.RegisterRoutes(campaignGroup, campaignHandler)

	channelGroup := app.Group("/api/channels", authMw, agentOrAbove)
	channelHandler := channel.NewChannelHandler(c)
	channel.RegisterRoutes(channelGroup, channelHandler)

	contactGroup := app.Group("/api/contacts", authMw, agentOrAbove)
	contactHandler := contact.NewContactHandler(c)
	contact.RegisterRoutes(contactGroup, contactHandler)

	convGroup := app.Group("/api/conversations", authMw, agentOrAbove)
	conversationHandler := conversation.NewConversationHandler(c, hub)
	conversation.RegisterRoutes(convGroup, conversationHandler)

	messageHandler := message.NewMessageHandler(c, hub)
	message.RegisterConversationRoutes(convGroup, messageHandler)
	msgGroup := app.Group("/api/messages", authMw, agentOrAbove)
	message.RegisterRoutes(msgGroup, messageHandler)

	ctHandler := conversation_tag.NewConversationTagHandler(c)
	conversation_tag.RegisterRoutes(convGroup, ctHandler)

	tagGroup := app.Group("/api/tags", authMw, agentOrAbove)
	tagHandler := tag.NewTagHandler(c)
	tag.RegisterRoutes(tagGroup, tagHandler)

	noteHandler := note.NewNoteHandler(c)
	note.RegisterConversationRoutes(convGroup, noteHandler)
	notesGroup := app.Group("/api/notes", authMw, agentOrAbove)
	note.RegisterRoutes(notesGroup, noteHandler)

	wdGroup := app.Group("/api/whatsapp-devices", authMw, agentOrAbove)
	wdHandler := whatsapp_device.NewWhatsAppDeviceHandler(c)
	whatsapp_device.RegisterRoutes(wdGroup, wdHandler)

	uploadGroup := app.Group("/api/upload", authMw, agentOrAbove)
	uploadHandler := upload.NewUploadHandler(c)
	upload.RegisterRoutes(uploadGroup, uploadHandler)

	dashGroup := app.Group("/api/dashboard", authMw, agentOrAbove)
	dashHandler := dashboard.NewDashboardHandler(c)
	dashboard.RegisterRoutes(dashGroup, dashHandler)

	tenantGroup := app.Group("/api/tenants", authMw, adminOrAbove)
	tenant.RegisterProtectedRoutes(tenantGroup, tenantHandler)

	c.Logger.Info("Starting server on port %d", cfg.Port)

	if err := app.Listen(":" + fmt.Sprintf("%d", cfg.Port)); err != nil {
		c.Logger.Fatal("Server error: %v", err)
	}
}
