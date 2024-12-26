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
const youtubeLinkPattern = `^(https?\:\/\/)?(www\.youtube\.com|youtu\.?be)\/.+$`

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
		b.handleItag(msg)

		choiceCh := make(chan string)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
			audioFile, err := downloader.DownloadMp3(msg.Text, choiceInt, "mp3")

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

// itags:
// 139 - audio M4A 48kbps
// 140 - audio MP3 128kbps
// 171 - audio MP3 192kbps
// 141 - audio MP3 256kbps
func (b *Bot) handleItag(msg *tgbotapi.Message) error {
	videoInfo, _, err := downloader.GetVideoInfo(msg.Text)
	if err != nil {
		return err
	}

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

	if len(keyboard.InlineKeyboard) == 0 {
		botMsg := tgbotapi.NewMessage(msg.Chat.ID, "Can't find audio format for this video")
		_, err := b.bot.Send(botMsg)
		if err != nil {
			return err
		}
	}

	botMsg := tgbotapi.NewMessage(msg.Chat.ID, "Available quality for this video:")
	botMsg.ReplyMarkup = keyboard
	_, err = b.bot.Send(botMsg)
	if err != nil {
		return err
	}

	return nil
}
