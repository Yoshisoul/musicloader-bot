package telegram

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"
	"youtubeToMp3/pkg/downloader"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	commandStart       = "start"
	youtubeLinkPattern = `^(https?\:\/\/)?(www\.youtube\.com|youtu\.?be)\/.+$`
	startMsg           = "Hello, send youtube link you want to download in mp3"
	// unkownMsg          = "Unknown commmand, please enter /start"
	// badlinkMsg         = "Bad link. Please send link on YouTube"
	// makeChoiceMsg      = "Please make previous choice or cancel it and send link again"
	// cancelMsg          = "Canceled"
	dwnldMsg = "Downloading video..."
	// timeoutMsg         = "Timeout! Can't download video. Send link again"
	sendMsg = "Mp3 downloaded, sending to you..."
	// noChoiceMsg        = "Timeout! No choice was made. Send link again"
	// cantFindMsg        = "Can't find audio format for this video"
	qualityMsg   = "Available quality for this video:"
	choiceBuffer = 5
	timerWait    = 10 * time.Second
	choiceWait   = 10 * time.Second
)

func (b *Bot) handleCommand(msg *tgbotapi.Message) error {
	botMsg := tgbotapi.NewMessage(msg.Chat.ID, unkownMsg)

	switch msg.Command() {
	case commandStart:
		botMsg.Text = startMsg
	}

	_, err := b.bot.Send(botMsg)
	return err
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) error {
	botMsg := tgbotapi.NewMessage(msg.Chat.ID, badlinkMsg)

	isYoutubeLink, err := regexp.MatchString(youtubeLinkPattern, msg.Text)
	if err != nil {
		return err
	}

	if isYoutubeLink {
		if b.activeChoice[msg.Chat.ID] {
			botMsg.Text = makeChoiceMsg
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
			return fmt.Errorf("can't find audio format for this video")
		}

		botMsg.ReplyMarkup = nil

		choiceCh := make(chan string, choiceBuffer)
		ctx, cancel := context.WithTimeout(context.Background(), choiceWait)
		defer func() {
			close(choiceCh)
			cancel()
		}()

		go b.waitResponse(ctx, choiceCh, msg.Chat.ID, msgCallback.MessageID)

		err = b.handleChoice(ctx, choiceCh, msg, msgCallback)
		if err != nil {
			return err
		}

	} else {
		botMsg.Text = badlinkMsg
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
func (b *Bot) handleItag(linkMsg *tgbotapi.Message) (*tgbotapi.Message, error) {
	videoInfo, _, err := downloader.GetVideoInfo(linkMsg.Text)
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
		botMsg := tgbotapi.NewMessage(linkMsg.Chat.ID, cantFindMsg)
		_, err := b.bot.Send(botMsg)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	botMsg := tgbotapi.NewMessage(linkMsg.Chat.ID, qualityMsg)
	botMsg.ReplyToMessageID = linkMsg.MessageID
	botMsg.ReplyMarkup = keyboard
	msgInfo, err := b.bot.Send(botMsg)
	if err != nil {
		return nil, err
	}

	return &msgInfo, nil
}

func (b *Bot) waitResponse(ctx context.Context, choice chan<- string, chatID int64, messageID int) {
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

func (b *Bot) handleChoice(ctx context.Context, choiceCh chan string, linkMsg *tgbotapi.Message, callbackMsg *tgbotapi.Message) error {
	var err error
	select {
	case choice := <-choiceCh:
		if choice == "cancel" {
			callbackMsg.Text = "Canceled"
			_, err = b.bot.Request(tgbotapi.NewEditMessageText(linkMsg.Chat.ID, callbackMsg.MessageID, cancelMsg))
			if err != nil {
				return err
			}

			log.Printf("Choice canceled: username = %s, chat ID = %d \n", linkMsg.From.UserName, linkMsg.Chat.ID)
			ctx.Done()
			return nil
		}

		choiceInt, err := strconv.Atoi(choice)
		if err != nil {
			return err
		}

		_, err = b.bot.Request(tgbotapi.NewEditMessageText(linkMsg.Chat.ID, callbackMsg.MessageID, dwnldMsg))
		if err != nil {
			return err
		}

		timerCtx, timerDone := context.WithTimeout(context.Background(), timerWait)
		var audioFile *os.File
		done := make(chan bool)
		errCh := make(chan error)

		go func() {
			defer func() {
				timerDone()
				close(done)
				close(errCh) // if err but ch is closed, then will be panic. So closing is here
			}()
			audioFile, err = downloader.DownloadMp3(timerCtx, linkMsg.Text, choiceInt, "mp3")

			if err != nil {
				errCh <- err
			}
			done <- true
		}()

		select {
		case <-timerCtx.Done():
			_, err = b.bot.Request(tgbotapi.NewEditMessageText(linkMsg.Chat.ID, callbackMsg.MessageID, timeoutMsg))
			if err != nil {
				return err
			}

			return nil
		case <-done:
			log.Printf("Mp3 downloaded, sending...: username = %s, chat ID = %d \n", linkMsg.From.UserName, linkMsg.Chat.ID)
			_, err = b.bot.Request(tgbotapi.NewEditMessageText(linkMsg.Chat.ID, callbackMsg.MessageID, sendMsg))
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
			audioMsg := tgbotapi.NewAudio(linkMsg.Chat.ID, audio)
			audioMsg.ReplyToMessageID = callbackMsg.MessageID
			_, err = b.bot.Send(audioMsg)
			if err != nil {
				return err
			}
		case err := <-errCh:
			if err != nil {
				return err
			}
		}

	case <-ctx.Done():
		_, err = b.bot.Request(tgbotapi.NewEditMessageText(linkMsg.Chat.ID, callbackMsg.MessageID, noChoiceMsg))
		if err != nil {
			return err
		}
	}

	return nil
}
