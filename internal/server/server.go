package server

import (
	"aktel/internal/bot"
	"aktel/internal/database"
	"github.com/rs/zerolog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type FiberServer struct {
	*fiber.App

	log   zerolog.Logger
	db    *database.Database
	tgBot *bot.TGBot
	wpBot *bot.WPBot
}

func New() (*FiberServer, func()) {
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	db, cleanup, err := database.NewDatabase()
	if err != nil {
		logger.Err(err).Msg("database connection failed")
	}

	tgBot, err := bot.NewTGBot(logger, db)
	if err != nil {
		logger.Err(err).Msg("telegram bot creation failed")
	}

	wpBot, err := bot.NewWPBot()
	if err != nil {
		logger.Err(err).Msg("whatsapp bot creation failed")
	}

	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "aktel",
			AppName:      "aktel",
		}),
		log:   logger,
		db:    db,
		tgBot: tgBot,
		wpBot: wpBot,
	}

	return server, cleanup
}
