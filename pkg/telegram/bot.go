package telegram

import (
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	bot             *tgbotapi.BotAPI
	updates         tgbotapi.UpdatesChannel
	chatUpdates     map[int64]chan tgbotapi.Update
	callbackUpdates map[int64]chan tgbotapi.Update
	mu              sync.Mutex
}

func NewBot(bot *tgbotapi.BotAPI) *Bot {
	return &Bot{
		bot:             bot,
		chatUpdates:     make(map[int64]chan tgbotapi.Update),
		callbackUpdates: make(map[int64]chan tgbotapi.Update),
	}
}

func (b *Bot) Start() {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 120
	b.updates = b.bot.GetUpdatesChan(u)

	for update := range b.updates {
		log.Println("General update received")
		b.mu.Lock()
		if ch, ok := b.chatUpdates[update.FromChat().ChatConfig().ChatID]; ok {
			ch <- update
		} else {
			ch := make(chan tgbotapi.Update, 5) // создаём канал для чата, если такого нет
			b.chatUpdates[update.FromChat().ChatConfig().ChatID] = ch
			b.callbackUpdates[update.FromChat().ChatConfig().ChatID] = make(chan tgbotapi.Update, 5)
			ch <- update
		}
		b.mu.Unlock()
		go b.handleChatUpdate(<-b.chatUpdates[update.FromChat().ChatConfig().ChatID])
	}
}

func (b *Bot) handleChatUpdate(update tgbotapi.Update) {
	log.Printf("username: %s, chatID: %v\n", update.SentFrom().UserName, update.FromChat().ID)
	log.Println("Chat update received")

	if update.CallbackQuery != nil {
		log.Println("Callback update received, chat ID:", update.CallbackQuery.Message.Chat.ID)
		b.callbackUpdates[update.CallbackQuery.Message.Chat.ID] <- update
		return
	}

	// подробнее в handlers.go
	// if update.CallbackQuery != nil {
	// 	log.Println("Callback update received")
	// 	err := b.handleCallbackQuery(update.CallbackQuery)
	// 	if err != nil {
	// 		log.Printf("error when handling the callback query: %s\n", err)
	// 		b.bot.Send(tgbotapi.NewMessage(update.
	// 			CallbackQuery.Message.Chat.ID, "Error when handling the callback query, try again: "+err.Error()))
	// 	}
	// }

	if update.Message.IsCommand() {
		log.Println("Command update received")
		err := b.handleCommand(update.Message)
		if err != nil {
			log.Printf("error when handling the command: %s\n", err)
			b.bot.Send(tgbotapi.NewMessage(update.
				Message.Chat.ID, "Error when handling the command, try again: "+err.Error()))
		}
		return
	}

	log.Println("Message update received")
	err := b.handleMessage(update.Message)
	if err != nil {
		log.Printf("error when handling the message: %s\n", err)
		b.bot.Send(tgbotapi.NewMessage(update.
			Message.Chat.ID, "Error when handling the message, try again: "+err.Error()))
	}
}
