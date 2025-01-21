package telegram

import (
	"context"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"
	"youtubeToMp3/pkg/downloader"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const commandStart = "/start"
const youtubeLinkPattern = `^(https?\:\/\/)?(www\.youtube\.com|youtu\.?be)\/.+$`

func (b *Bot) handleCommand(msg *tgbotapi.Message) error {
	botMsg := tgbotapi.NewMessage(msg.Chat.ID, "Unknown commmand, please enter /start")

	switch msg.Command() {
	case commandStart:
		botMsg.Text = "Hello, send youtube link you want to download in mp3"
	}

	_, err := b.bot.Send(botMsg)
	return err
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) error {
	botMsg := tgbotapi.NewMessage(msg.Chat.ID, "Bad link. Please send link on YouTube")

	isYoutubeLink, err := regexp.MatchString(youtubeLinkPattern, msg.Text)
	if err != nil {
		return err
	}

	if isYoutubeLink {
		if b.activeChoice[msg.Chat.ID] {
			botMsg.Text = "Please make previous choice or cancel it and send link again"
			_, err := b.bot.Send(botMsg)
			if err != nil {
				return err
			}
			return nil
		}

		msgCallback, err := b.handleItag(msg)
		if err != nil {
			return err
		}
		if msgCallback == nil {
			return nil
		}

		botMsg.ReplyMarkup = nil

		choiceCh := make(chan string, 5)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer func() {
			close(choiceCh)
			cancel()
		}()

		go b.handleChoice(ctx, choiceCh, msg.Chat.ID, msgCallback.MessageID)
		select {
		case choice := <-choiceCh:
			if choice == "cancel" {
				msgCallback.Text = "Canceled"
				_, err = b.bot.Request(tgbotapi.NewEditMessageText(msg.Chat.ID, msgCallback.MessageID, "Canceled"))
				if err != nil {
					return err
				}

				log.Printf("Choice canceled: username = %s, chat ID = %d \n", msg.From.UserName, msg.Chat.ID)
				ctx.Done()
				return nil
			}

			choiceInt, err := strconv.Atoi(choice)
			if err != nil {
				return err
			}

			_, err = b.bot.Request(tgbotapi.NewEditMessageText(msg.Chat.ID, msgCallback.MessageID, "Downloading video..."))
			if err != nil {
				return err
			}

			// установить здесь таймер
			// timerCtx, cancelTimer := context.WithTimeout(context.Background(), 15*time.Second)
			// defer cancelTimer()

			// var audioFile *os.File

			// go func() {
			// 	audioFile, err := downloader.DownloadMp3(msg.Text, choiceInt, "mp3", timerCtx)
			// }()

			audioFile, err := downloader.DownloadMp3(msg.Text, choiceInt, "mp3")

			if err != nil {
				return err
			}

			_, err = b.bot.Request(tgbotapi.NewEditMessageText(msg.Chat.ID, msgCallback.MessageID, "Mp3 downloaded, sending to you..."))
			if err != nil {
				return err
			}

			audioFile, err = os.Open(audioFile.Name())
			if err != nil {
				return err
			}
			defer audioFile.Close()

			audioBytes, err := io.ReadAll(audioFile)
			if err != nil {
				return err
			}

			audio := tgbotapi.FileBytes{Name: audioFile.Name(), Bytes: audioBytes}
			audioMsg := tgbotapi.NewAudio(msg.Chat.ID, audio)
			audioMsg.ReplyToMessageID = msgCallback.MessageID
			_, err = b.bot.Send(audioMsg)
			if err != nil {
				return err
			}

		case <-ctx.Done():
			_, err = b.bot.Request(tgbotapi.NewEditMessageText(msg.Chat.ID, msgCallback.MessageID, "Timeout! No choice was made. Send link again"))
			if err != nil {
				return err
			}
		}

	} else {
		botMsg.Text = "Bad link. Please send link on YouTube"
		_, err := b.bot.Send(botMsg)
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

// itags:
// 139 - audio M4A 48kbps
// 140 - audio MP3 128kbps
// 171 - audio MP3 192kbps
// 141 - audio MP3 256kbps
func (b *Bot) handleItag(msg *tgbotapi.Message) (*tgbotapi.Message, error) {
	videoInfo, _, err := downloader.GetVideoInfo(msg.Text)
	if err != nil {
		return nil, err
	}

	log.Println("Available itags:")
	for _, format := range videoInfo.Formats {
		log.Printf("%d, ", format.ItagNo)
	}
	log.Println()

	keyboard := tgbotapi.NewInlineKeyboardMarkup()

	for _, format := range videoInfo.Formats {
		var button tgbotapi.InlineKeyboardButton

		switch format.ItagNo {
		case 139:
			button = tgbotapi.NewInlineKeyboardButtonData("MP3 48kbps", "139")
		case 140:
			button = tgbotapi.NewInlineKeyboardButtonData("MP3 128kbps (Standard)", "140")
		case 171:
			button = tgbotapi.NewInlineKeyboardButtonData("MP3 192kbps", "171")
		case 141:
			button = tgbotapi.NewInlineKeyboardButtonData("MP3 256kbps", "141")
		default:
			continue
		}
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(button))
	}

	button := tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel")
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(button))

	if len(keyboard.InlineKeyboard) == 0 {
		botMsg := tgbotapi.NewMessage(msg.Chat.ID, "Can't find audio format for this video")
		_, err := b.bot.Send(botMsg)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	botMsg := tgbotapi.NewMessage(msg.Chat.ID, "Available quality for this video:")
	botMsg.ReplyToMessageID = msg.MessageID
	botMsg.ReplyMarkup = keyboard
	msgInfo, err := b.bot.Send(botMsg)
	if err != nil {
		return nil, err
	}

	return &msgInfo, nil
}

func (b *Bot) handleChoice(ctx context.Context, choice chan<- string, chatID int64, messageID int) {
	b.activeChoice[chatID] = true
	for {
		log.Printf("Waiting for button update: chat ID = %d \n", chatID)
		select {
		// if channel doesn't exist, it will block here
		case update := <-b.callbackUpdates[chatID]: // we can only take (!read) value from channel
			log.Printf("Button update receive: chat ID = %d \n", chatID)
			if update.CallbackQuery.Message.MessageID == messageID {
				b.removeButtonsFromMessage(chatID, messageID)

				choice <- update.CallbackQuery.Data
				b.activeChoice[chatID] = false
				return
			} else {
				log.Printf("Update from another message: %v\n", update.CallbackQuery)
				b.removeButtonsFromMessage(chatID, messageID)
			}
		case <-ctx.Done():
			b.removeButtonsFromMessage(chatID, messageID)
			b.activeChoice[chatID] = false
			return
		}
	}
}

func (b *Bot) removeButtonsFromMessage(chatID int64, messageID int) {
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
	})
	_, err := b.bot.Request(edit)
	if err != nil {
		log.Printf("Failed to remove buttons: %s\n", err)
	}
}
