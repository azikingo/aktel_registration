package bot

import (
	"context"
	"errors"
	"os"

	"aktel/internal/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
)

type TGBot struct {
	Bot *tgbotapi.BotAPI

	log zerolog.Logger
	db  *database.Database
}

func NewTGBot(logger zerolog.Logger, db *database.Database) (*TGBot, error) {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return nil, errors.New("missing TELEGRAM_TOKEN in environment")
	}

	// Start Telegram bot
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	// Start handle updates from Telegram
	logger.Info().Msgf("Bot authorized as: @%s", bot.Self.UserName)

	return &TGBot{
		Bot: bot,
		log: logger,
		db:  db,
	}, nil
}

func (b *TGBot) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.Bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		user := update.Message.From
		chatId := update.Message.Chat.ID

		// Save general user info
		if err := b.db.SaveUserFromTelegram(ctx, user); err != nil {
			b.log.Err(err).Msg("save user from telegram failed")
		}

		// Command handling
		if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(chatId, "❓ Unknown command.")

			switch update.Message.Command() {
			case "start":
				msg = tgbotapi.NewMessage(chatId, "👋 Welcome to AKTEL Tournament bot!\nPress /help to use.")

			case "teams":
			//args := update.Message.CommandArguments()
			//if args != "" {
			//	angel, err := GetAngelByPhone(phone)
			//	errHistory := saveSearchHistory(user, angel, SPhone, args)
			//	if errHistory != nil {
			//		log.Printf("error on saving history: %s", errHistory)
			//	}
			//	if err != nil {
			//		log.Printf("DB error while fetching angel: %v", err)
			//		bot.Send(tgbotapi.NewMessage(chatId, "❌ Could not retrieve your angel."))
			//		continue
			//	}
			//
			//	if angel == nil {
			//		bot.Send(tgbotapi.NewMessage(chatId, "🚫 Angel not found."))
			//	} else {
			//		msg := tgbotapi.NewMessage(chatId, formatAngel(angel))
			//		msg.ParseMode = tgbotapi.ModeHTML
			//		bot.Send(msg)
			//	}
			//}

			case "help":
				msg = tgbotapi.NewMessage(chatId, "I did a list to you here for commands that I can do:\n"+
					"/start - Start the bot\n"+
					"/teams - List registered teams\n"+
					"/help - Show this message\n")

			}

			_, err := b.Bot.Send(msg)
			if err != nil {
				b.log.Err(err).Msg("telegram message reply for command failed")
			}
			continue
		}

		// Handle regular messages (non-commands)
		msg := tgbotapi.NewMessage(chatId, "I only respond to commands like /start or /phone.")
		_, err := b.Bot.Send(msg)
		if err != nil {
			b.log.Err(err).Msg("telegram message sending failed")
		}
	}
}

func (b *TGBot) Stop(ctx context.Context) {
	b.Bot.StopReceivingUpdates()
}

func (b *TGBot) SendMessageToChannel(username, message string, mode *string) (*tgbotapi.Message, error) {
	msg := tgbotapi.NewMessageToChannel(username, message)

	switch {
	case mode == nil:
		break
	case *mode == tgbotapi.ModeHTML:
		msg.ParseMode = tgbotapi.ModeHTML
	case *mode == tgbotapi.ModeMarkdown:
		msg.ParseMode = tgbotapi.ModeMarkdown
	case *mode == tgbotapi.ModeMarkdownV2:
		msg.ParseMode = tgbotapi.ModeMarkdownV2
	default:
		return nil, errors.New("message mode is invalid")
	}

	sentMessage, err := b.Bot.Send(msg)
	if err != nil {
		return nil, err
	}

	return &sentMessage, nil
}
