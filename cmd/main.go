package main

import (
	"log"
	"youtubeToMp3/pkg/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	bot, err := tgbotapi.NewBotAPI("7926179976:AAG6t0n5SWOQ-B_HHRjwcsMARBJhPQFZTKo")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	botStruct := telegram.NewBot(bot)
	botStruct.Start()
}

// сделать работоспособным для многопоточности
