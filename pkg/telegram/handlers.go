package telegram

import (
	"io"
	"os"
	"regexp"
	"strconv"
	"youtubeToMp3/pkg/downloader"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const commandStart = "start"
const youtubeLinkPattern = `^(https?\:\/\/)?(www\.youtube\.com|youtu\.?be)\/.+$` // check links

func (b Bot) handleCommand(msg *tgbotapi.Message) error {
	botMsg := tgbotapi.NewMessage(msg.Chat.ID, "Unknown commmand, please enter /start")

	switch msg.Command() {
	case commandStart:
		botMsg.Text = "Send youtube link"
		_, err := b.bot.Send(botMsg)
		return err
	default:
		_, err := b.bot.Send(botMsg)
		return err
	}
}

func (b Bot) handleMessage(msg *tgbotapi.Message) error {
	botMsg := tgbotapi.NewMessage(msg.Chat.ID, "Bad link. Please send link on YouTube")

	isYoutubeLink, err := regexp.MatchString(youtubeLinkPattern, msg.Text)
	if err != nil {
		return err
	}

	if isYoutubeLink {
		// button128 := tgbotapi.NewInlineKeyboardButtonData("MP3 128kbps", "140")
		// button256 := tgbotapi.NewInlineKeyboardButtonData("MP3 256kbps", "141")

		// keyboard := tgbotapi.NewInlineKeyboardMarkup(
		// 	tgbotapi.NewInlineKeyboardRow(button128, button256),
		// )

		// botMsg.Text = "Choose the quality:"
		// botMsg.ReplyMarkup = keyboard
		// _, err := b.bot.Send(botMsg)
		// if err != nil {
		// 	return err
		// }

		// choiceCh := make(chan string)
		// go b.HandleChoice(choiceCh)

		// select {
		// case choice := <-choiceCh:
		botMsg.ReplyMarkup = nil
		botMsg.Text = "Downloading video..."
		_, err = b.bot.Send(botMsg)
		if err != nil {
			return err
		}

		choiceInt, err := strconv.Atoi("140")
		if err != nil {
			return err
		}
		audioFile, err := downloader.DownloadMp3(msg.Text, choiceInt)

		if err != nil {
			return err
		}

		botMsg.Text = "Mp3 downloaded, sending to you..."
		_, err = b.bot.Send(botMsg)
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
		_, err = b.bot.Send(audioMsg)
		if err != nil {
			return err
		}

		// case <-time.After(30 * time.Second):
		// 	botMsg.Text = "Timeout! No choice was made. Send link again"
		// 	botMsg.ReplyMarkup = nil
		// 	_, err := b.bot.Send(botMsg)
		// 	if err != nil {
		// 		return err
		// 	}
		// }

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

// func (b *Bot) HandleChoice(choice chan<- string, updates tgbotapi.UpdatesChannel) {
// 	updates := b.bot.GetUpdatesChan(tgbotapi.NewUpdate(0))
// 	for update := range updates {
// 		if update.CallbackQuery != nil {

// 			choice <- update.CallbackQuery.Data

// 			edit := tgbotapi.NewEditMessageReplyMarkup(
// 				update.CallbackQuery.Message.Chat.ID,
// 				update.CallbackQuery.Message.MessageID,
// 				tgbotapi.InlineKeyboardMarkup{},
// 			)
// 			b.bot.Send(edit)

// 			// b.bot.Send(tgbotapi.NewCallback(update.CallbackQuery.ID, "Your choice was processed."))
// 			return
// 		}
// 	}
// }
