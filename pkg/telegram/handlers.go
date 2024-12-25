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
		b.HandleItag(msg)

		choiceCh := make(chan string)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer func() {
			close(choiceCh)
			cancel()
		}()

		go b.HandleChoice(ctx, choiceCh)
		botMsg.ReplyMarkup = nil
		select {
		case choice := <-choiceCh:
			botMsg.Text = "Downloading video..."
			_, err = b.bot.Send(botMsg)
			if err != nil {
				return err
			}

			choiceInt, err := strconv.Atoi(choice)
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

		case <-ctx.Done():
			botMsg.Text = "Timeout! No choice was made. Send link again"
			_, err := b.bot.Send(botMsg)
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

func (b *Bot) HandleChoice(ctx context.Context, choice chan<- string) {
	for {
		select {
		case update := <-b.updates:
			if update.CallbackQuery != nil {
				chatID := update.CallbackQuery.Message.Chat.ID
				messageID := update.CallbackQuery.Message.MessageID

				edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{
					InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
				})
				_, err := b.bot.Request(edit)
				if err != nil {
					log.Printf("Failed to remove buttons: %s\n", err)
				}

				choice <- update.CallbackQuery.Data
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// 139 - audio M4A 48kbps
// 140 - audio MP3 128kbps
// 171 - audio MP3 192kbps
// 141 - audio MP3 256kbps
func (b *Bot) HandleItag(msg *tgbotapi.Message) error {
	button48 := tgbotapi.NewInlineKeyboardButtonData("MP3 48kbps", "139")
	button128 := tgbotapi.NewInlineKeyboardButtonData("MP3 128kbps(def)", "140")

	button192 := tgbotapi.NewInlineKeyboardButtonData("MP3 192kbps", "171")
	button256 := tgbotapi.NewInlineKeyboardButtonData("MP3 256kbps", "141")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(button128, button192, button256),
	)

	botMsg := tgbotapi.NewMessage(msg.Chat.ID, "Choose the quality:")
	botMsg.ReplyMarkup = keyboard
	_, err := b.bot.Send(botMsg)
	if err != nil {
		return err
	}

	return nil
}
