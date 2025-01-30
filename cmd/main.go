package main

import (
	"log"
	"youtubeToMp3/pkg/config"
	"youtubeToMp3/pkg/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/subosito/gotenv"
)

func main() {
	if err := gotenv.Load(); err != nil {
		log.Println("no .env file")
	}

	cfg, err := config.Init()
	if err != nil {
		log.Fatal(err)
	}

	log.Println(cfg)

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	botStruct := telegram.NewBot(bot, cfg.Messages)
	botStruct.Start()
}
