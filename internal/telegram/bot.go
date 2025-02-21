package telegram

import (
	"log"
	"sync"
	"time"
	"youtubeToMp3/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	tickerDuration        = 1 * time.Minute
	inactiveTime          = 5 * time.Minute
	updateTimeout         = 120
	callbackUpdatesBuffer = 5
)

type Bot struct {
	bot             *tgbotapi.BotAPI
	updates         tgbotapi.UpdatesChannel
	mu              sync.Mutex
	callbackUpdates map[int64]chan tgbotapi.Update // key for chatID
	activeChoice    map[int64]bool
	lastActivity    map[int64]time.Time
	messages        config.Messages
}

func NewBot(bot *tgbotapi.BotAPI, messages config.Messages) *Bot {
	return &Bot{
		bot:             bot,
		activeChoice:    make(map[int64]bool),
		callbackUpdates: make(map[int64]chan tgbotapi.Update),
		lastActivity:    make(map[int64]time.Time),
		messages:        messages,
	}
}

func (b *Bot) Start() {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = updateTimeout
	b.updates = b.bot.GetUpdatesChan(u)

	go b.setСleanupInactiveChats()

	for update := range b.updates {
		log.Printf("General update received: username = %s, chat ID = %d \n",
			update.SentFrom().UserName, update.FromChat().ID)

		// no race condition here, because chat is unique and sequential
		b.lastActivity[update.FromChat().ID] = time.Now()
		if _, ok := b.callbackUpdates[update.FromChat().ID]; !ok {
			b.callbackUpdates[update.FromChat().ID] = make(chan tgbotapi.Update, callbackUpdatesBuffer)
			log.Printf("Callback channel created for chat ID: %v\n", update.FromChat().ID)
		}

		go b.handleChatUpdate(update)
	}
}

func (b *Bot) handleChatUpdate(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		log.Printf("Callback update received, username: %s, chatID: %v\n",
			update.SentFrom().UserName, update.FromChat().ID)

		b.callbackUpdates[update.FromChat().ID] <- update
		return
	}

	if update.Message.IsCommand() {
		log.Printf("Command update received, username: %s, chatID: %v\n",
			update.SentFrom().UserName, update.FromChat().ID)

		err := b.handleCommand(update.Message)
		if err != nil {
			log.Printf("Error when handling the command: %s\n", err)
			b.bot.Send(tgbotapi.NewMessage(update.
				Message.Chat.ID, "Error when handling the command, try again: "+err.Error()))
		}
		return
	}

	if update.Message != nil {
		log.Printf("Message update received, username: %s, chatID: %v\n",
			update.SentFrom().UserName, update.FromChat().ID)

		err := b.handleMessage(update.Message)
		if err != nil {
			log.Printf("Error when handling the message: %s\n", err)
			b.handleError(update.FromChat().ID, err)
		}
		return
	}

	log.Printf("Update type not recognized, username: %s, chatID: %v\n",
		update.SentFrom().UserName, update.FromChat().ID)
}

func (b *Bot) setСleanupInactiveChats() {
	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()

	for range ticker.C {
		for chatID, lastActive := range b.lastActivity {
			if time.Since(lastActive) > inactiveTime {
				b.mu.Lock() // to avoid updates during deletion
				close(b.callbackUpdates[chatID])
				delete(b.callbackUpdates, chatID)
				delete(b.lastActivity, chatID)
				b.mu.Unlock()
				log.Printf("Removed inactive chat: %d\n", chatID)
			}
		}
	}
}
