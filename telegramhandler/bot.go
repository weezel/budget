package telegramhandler

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"weezel/budget/dbengine"
	"weezel/budget/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var splitPath = regexp.MustCompile(`\s+`)

func ConnectionHandler(apikey string, channelId int64, debug bool) {
	bot, err := tgbotapi.NewBotAPI(apikey)
	if err != nil {
		log.Panicf("Possible error in config file: %s", err)
	}

	bot.Debug = debug

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	var command string
	var shopName string
	var lastElem string
	var price float64
	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		username := update.Message.From.String()
		msg := update.Message.Text
		tokenized := splitPath.Split(msg, -1)
		lastElem = strings.Replace(
			tokenized[len(tokenized)-1],
			",", ".", -1)
		log.Printf("Tokenized: %v", tokenized)
		command = tokenized[0]

		if len(tokenized) < 3 {
			log.Printf("Help requested by %s", username)
			helpMsg := "Tunnistan seuraavat komennot:\n"
			helpMsg += "osto\n"
			helpMsg += "palkka\n"
			helpMsg += "velat, velkaa\n"
			outMsg := tgbotapi.NewMessage(channelId, helpMsg)
			bot.Send(outMsg)
			continue
		}

		switch command {
		case "osto":
			shopName = tokenized[1]
			purchaseDate := utils.GetDate(tokenized, "01-2006")
			price, err = strconv.ParseFloat(lastElem, 64)
			if err != nil {
				log.Printf("ERROR: price wasn't the last item: %v", tokenized)
				helpMsg := "Virhe, hinta täytyy olla komennon viimeinen elementti"
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				bot.Send(outMsg)
				continue
			}

			dbengine.InsertShopping(username, shopName, purchaseDate, price)

			log.Printf("Purchased from %s with price %.2f by %s on %s",
				shopName,
				price,
				username,
				purchaseDate.Format("01-2006"))
			thxMsg := fmt.Sprintf("Ostosi on kirjattu, %s. Kiitos!", username)
			outMsg := tgbotapi.NewMessage(channelId, thxMsg)
			bot.Send(outMsg)
			continue
		case "palkka":
			salaryDate := utils.GetDate(tokenized, "01-2006")
			salary, err := strconv.ParseFloat(lastElem, 64)
			if err != nil {
				log.Printf("ERROR: couldn't parse salary: %v", err)
				helpMsg := "Virhe palkan parsinnassa. Palkan oltava viimeisenä ja muodossa x.xx tai x,xx"
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				bot.Send(outMsg)
				continue
			}

			dbengine.InsertSalary(username, salary, salaryDate)

			log.Printf("Salary amount of %.2f by %s on %s",
				salary,
				username,
				salaryDate.Format("01-2006"))
			thxMsg := fmt.Sprintf("Palkka kirjattu, %s. Kiitos!", username)
			outMsg := tgbotapi.NewMessage(channelId, thxMsg)
			bot.Send(outMsg)
		case "velat":
		case "velkaa":
			thxMsg := fmt.Sprint("Velkaa ollaan seuraavasti:")
			outMsg := tgbotapi.NewMessage(channelId, thxMsg)
			bot.Send(outMsg)
		default:
			continue
		}
	}
}
