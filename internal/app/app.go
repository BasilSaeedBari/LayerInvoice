package app

import (
	"embed"
	"layerinvoice/internal/auth"
	"layerinvoice/internal/config"
	"layerinvoice/internal/db"
)

// App holds application state and resources shared by handlers.
type App struct {
	Config       *config.Config
	DB           *db.DB
	SessionStore *auth.SessionStore
	TemplatesFS  embed.FS // Embedded HTML templates
}

// New creates and wires up a new App context.
func New(cfg *config.Config, database *db.DB, sessionStore *auth.SessionStore, templatesFS embed.FS) *App {
	return &App{
		Config:       cfg,
		DB:           database,
		SessionStore: sessionStore,
		TemplatesFS:  templatesFS,
	}
}
