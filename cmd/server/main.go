package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/src/auth"
	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/conversation_tag"
	"centrachannel/internal/src/dashboard"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/note"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tag"
	"centrachannel/internal/src/upload"
	"centrachannel/internal/src/user"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/utils/response"
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

	authHandler := auth.NewAuthHandler(c)
	auth.RegisterRoutes(app, authHandler)

	// Protected routes
	authMw := middleware.AuthMiddleware(cfg)

	userGroup := app.Group("/api/user", authMw)
	userHandler := user.NewUserHandler(c)
	user.RegisterRoutesByGroup(userGroup, userHandler)

	channelGroup := app.Group("/api/channels", authMw)
	channelHandler := channel.NewChannelHandler(c)
	channel.RegisterRoutes(channelGroup, channelHandler)

	contactGroup := app.Group("/api/contacts", authMw)
	contactHandler := contact.NewContactHandler(c)
	contact.RegisterRoutes(contactGroup, contactHandler)

	profileGroup := app.Group("/api/profiles", authMw)
	profileHandler := profile.NewProfileHandler(c)
	profile.RegisterRoutes(profileGroup, profileHandler)

	convGroup := app.Group("/api/conversations", authMw)
	conversationHandler := conversation.NewConversationHandler(c)
	conversation.RegisterRoutes(convGroup, conversationHandler)

	messageHandler := message.NewMessageHandler(c)
	message.RegisterConversationRoutes(convGroup, messageHandler)
	msgGroup := app.Group("/api/messages", authMw)
	message.RegisterRoutes(msgGroup, messageHandler)

	ctHandler := conversation_tag.NewConversationTagHandler(c)
	conversation_tag.RegisterRoutes(convGroup, ctHandler)

	tagGroup := app.Group("/api/tags", authMw)
	tagHandler := tag.NewTagHandler(c)
	tag.RegisterRoutes(tagGroup, tagHandler)

	noteHandler := note.NewNoteHandler(c)
	note.RegisterConversationRoutes(convGroup, noteHandler)
	notesGroup := app.Group("/api/notes", authMw)
	note.RegisterRoutes(notesGroup, noteHandler)

	wdGroup := app.Group("/api/whatsapp-devices", authMw)
	wdHandler := whatsapp_device.NewWhatsAppDeviceHandler(c)
	whatsapp_device.RegisterRoutes(wdGroup, wdHandler)

	uploadGroup := app.Group("/api/upload", authMw)
	uploadHandler := upload.NewUploadHandler(c)
	upload.RegisterRoutes(uploadGroup, uploadHandler)

	dashGroup := app.Group("/api/dashboard", authMw)
	dashHandler := dashboard.NewDashboardHandler(c)
	dashboard.RegisterRoutes(dashGroup, dashHandler)

	c.Logger.Info("Starting server on port %d", cfg.Port)

	if err := app.Listen(":" + fmt.Sprintf("%d", cfg.Port)); err != nil {
		c.Logger.Fatal("Server error: %v", err)
	}
}
