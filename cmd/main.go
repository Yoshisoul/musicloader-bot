package main

import (
	"log"
	"youtubeToMp3/pkg/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	bot, err := tgbotapi.NewBotAPI("7926179976:AAG6t0n5SWOQ-B_HHRjwcsMARBJhPQFZTKo") // поменять
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	botStruct := telegram.NewBot(bot)
	botStruct.Start()
}

// почистить логи

// в данный момент после создания чата каналы будут вечно работать,
//нужно удалять их со временем, если чат не активен

// может быть можно придумать способ разделить калбеки и обычные сообщения
