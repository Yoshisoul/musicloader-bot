package telegram

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	bot     *tgbotapi.BotAPI
	updates tgbotapi.UpdatesChannel
}

func NewBot(bot *tgbotapi.BotAPI) *Bot {
	return &Bot{bot: bot}
}

func (b *Bot) Start() {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	b.updates = b.bot.GetUpdatesChan(u)
	for update := range b.updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			if update.Message.IsCommand() {
				err := b.handleCommand(update.Message)
				if err != nil {
					log.Printf("error when handling the command: %s\n", err)
					b.bot.Send(tgbotapi.NewMessage(update.
						Message.Chat.ID, "Error when handling the command, try again: "+err.Error()))
				}
				continue
			}

			err := b.handleMessage(update.Message)
			if err != nil {
				log.Printf("error when handling the message: %s\n", err)
				b.bot.Send(tgbotapi.NewMessage(update.
					Message.Chat.ID, "Error when handling the message, try again: "+err.Error()))
			}
		}
	}
}
