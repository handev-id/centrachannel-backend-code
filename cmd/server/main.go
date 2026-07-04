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

	app.Use(middleware.CORSMiddleware())
	app.Use(middleware.LogMiddleware(c.Logger, cfg.Env))

	obsHandler := observability.NewObservabilityHandler()
	observability.RegisterRoutes(app, obsHandler)

	docs.RegisterRoutes(app)

	webhookHandler := webhook.NewWebhookHandler(c, cfg)
	webhook.RegisterRoutes(app, webhookHandler)

	regHandler := registration.NewRegistrationHandler(c)
	registration.RegisterRoutes(app, regHandler)

	tenantHandler := tenant.NewTenantHandler(c)

	// Tenant Source
	app.Use(middleware.TenantMiddleware(c.Redis, c.DB))

	authHandler := auth.NewAuthHandler(c)
	auth.RegisterRoutes(app, authHandler)

	authMw := middleware.AuthMiddleware(cfg, c.Redis)
	adminOrAbove := middleware.RequireRole("super-admin", "admin")
	agentOrAbove := middleware.RequireRole("super-admin", "admin", "agent")

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
	conversationHandler := conversation.NewConversationHandler(c)
	conversation.RegisterRoutes(convGroup, conversationHandler)

	messageHandler := message.NewMessageHandler(c)
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
