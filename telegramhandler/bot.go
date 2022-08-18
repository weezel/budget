package telegramhandler

import (
	"context"
	"regexp"
	"strings"
	"weezel/budget/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var splitPath = regexp.MustCompile(`\s+`)

// SendTelegram returns true if message sending succeeds and false otherwise
func SendTelegram(
	bot *tgbotapi.BotAPI,
	msg tgbotapi.MessageConfig,
	markdown bool,
) error {
	if markdown {
		msg.ParseMode = tgbotapi.ModeMarkdown
	}
	if _, err := bot.Send(msg); err != nil {
		return err
	}
	return nil
}

func ConnectionHandler(bot *tgbotapi.BotAPI, channelID int64, hostname string) {
	var err error

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	ctx := context.Background()

	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		username := update.Message.From.String()
		msg := update.Message.Text
		tokenized := splitPath.Split(msg, -1)
		lastElem := strings.ReplaceAll(tokenized[len(tokenized)-1], ",", ".")
		logger.Infof("Tokenized: %v", tokenized)
		command := strings.ToLower(tokenized[0])

		switch command {
		case "osto":
			if len(tokenized) < 3 {
				displayHelp(username, channelID, bot)
				continue
			}

			shopName := tokenized[1]
			msg = handlePurchase(ctx, shopName, lastElem, username, tokenized)
			outMsg := tgbotapi.NewMessage(channelID, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "ostot":
			if len(tokenized) < 3 {
				displayHelp(username, channelID, bot)
				continue
			}

			msg = getStatsByTimeSpan(ctx, username, hostname, tokenized)
			outMsg := tgbotapi.NewMessage(channelID, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "tilastot":
			if len(tokenized) < 3 {
				displayHelp(username, channelID, bot)
				continue
			}

			msg = getStatsTimeSpan(ctx, hostname, tokenized)
			outMsg := tgbotapi.NewMessage(channelID, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "palkka":
			if len(tokenized) < 3 {
				displayHelp(username, channelID, bot)
				continue
			}

			msg = handleSalaryInsert(ctx, username, lastElem, tokenized)
			outMsg := tgbotapi.NewMessage(channelID, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "poista":
			if len(tokenized) < 4 {
				displayHelp(username, channelID, bot)
				continue
			}

			msg := handleRemovePurchase(ctx, username, tokenized)
			outMsg := tgbotapi.NewMessage(channelID, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "velat", "velkaa":
			if len(tokenized) < 2 {
				displayHelp(username, channelID, bot)
				continue
			}

			msg := handleDebts(ctx, tokenized)
			outMsg := tgbotapi.NewMessage(channelID, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "help", "apua":
			displayHelp(username, channelID, bot)
			continue
		}

	}
}
