package telegram

import (
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	errUnknownCommand = errors.New("unknown commmand, please enter /start")
	errBadLink        = errors.New("bad link. Please send link on YouTube")
	errMakeChoice     = errors.New("please make previous choice or cancel it and send link again")
	errCantFind       = errors.New("can't find audio format for this video")
)

func (b *Bot) handleError(chatID int64, err error) {
	msg := tgbotapi.NewMessage(chatID, b.messages.Internal)

	switch err {
	case errUnknownCommand:
		msg.Text = b.messages.UnknownCommand
	case errBadLink:
		msg.Text = b.messages.InvalidLink
	case errMakeChoice:
		msg.Text = b.messages.MakeChoice
	case errCantFind:
		msg.Text = b.messages.NoFormat
	}

	b.bot.Send(msg)
}
