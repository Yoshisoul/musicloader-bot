package telegram

import (
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	bot             *tgbotapi.BotAPI
	updates         tgbotapi.UpdatesChannel
	mu              sync.Mutex
	callbackUpdates map[int64]chan tgbotapi.Update // key for chatID
	activeChoice    map[int64]bool
}

func NewBot(bot *tgbotapi.BotAPI) *Bot {
	return &Bot{
		bot:             bot,
		activeChoice:    make(map[int64]bool),
		callbackUpdates: make(map[int64]chan tgbotapi.Update),
	}
}

func (b *Bot) Start() {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 120
	b.updates = b.bot.GetUpdatesChan(u)

	for update := range b.updates {
		log.Printf("General update received: username = %s, chat ID = %d \n", update.SentFrom().UserName, update.FromChat().ID)

		if _, ok := b.callbackUpdates[update.FromChat().ID]; !ok {
			b.mu.Lock()
			b.callbackUpdates[update.FromChat().ID] = make(chan tgbotapi.Update, 5)
			b.mu.Unlock()
			log.Printf("Callback channel created for chat ID: %v\n", update.FromChat().ID)
		}

		go b.handleChatUpdate(update)
	}
}

func (b *Bot) handleChatUpdate(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		log.Printf("Callback update received, username: %s, chatID: %v\n", update.SentFrom().UserName, update.FromChat().ID)
		b.callbackUpdates[update.FromChat().ID] <- update
		return
	}

	if update.Message.IsCommand() {
		log.Printf("Command update received, username: %s, chatID: %v\n", update.SentFrom().UserName, update.FromChat().ID)
		err := b.handleCommand(update.Message)
		if err != nil {
			log.Printf("error when handling the command: %s\n", err)
			b.bot.Send(tgbotapi.NewMessage(update.
				Message.Chat.ID, "Error when handling the command, try again: "+err.Error()))
		}
		return
	}

	if update.Message != nil {
		log.Printf("Message update received, username: %s, chatID: %v\n", update.SentFrom().UserName, update.FromChat().ID)
		err := b.handleMessage(update.Message)
		if err != nil {
			log.Printf("error when handling the message: %s\n", err)
			b.bot.Send(tgbotapi.NewMessage(update.
				Message.Chat.ID, "Error when handling the message, try again: "+err.Error()))
		}
		return
	}

	log.Printf("Update type not recognized, username: %s, chatID: %v\n", update.SentFrom().UserName, update.FromChat().ID)
}
