package server

import (
	"github.com/gofiber/fiber/v2"

	"aktel/internal/database"
)

type FiberServer struct {
	*fiber.App

	db database.Service
}

func New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "aktel",
			AppName:      "aktel",
		}),

		db: database.New(),
	}

	return server
}
