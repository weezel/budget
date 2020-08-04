package telegramhandler

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"weezel/budget/dbengine"
	"weezel/budget/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var splitPath = regexp.MustCompile(`\s+`)

func displayHelp(username string, channelId int64, bot *tgbotapi.BotAPI) {
	log.Printf("Help requested by %s", username)
	helpMsg := "Tunnistan seuraavat komennot:\n"
	helpMsg += "osto paikka [vapaaehtoinen pvm muodossa kk-vvvv] xx.xx\n"
	helpMsg += "palkka kk-vvvv xxxx.xx (nettona)\n"
	helpMsg += "velat, velkaa kk-vvvv\n"
	outMsg := tgbotapi.NewMessage(channelId, helpMsg)
	bot.Send(outMsg)
}

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

		switch command {
		case "osto":
			if len(tokenized) < 3 {
				displayHelp(username, channelId, bot)
				continue
			}

			shopName = tokenized[1]
			category := utils.GetCategory(tokenized)
			purchaseDate := utils.GetDate(tokenized, "01-2006")
			price, err = strconv.ParseFloat(lastElem, 64)
			if err != nil {
				log.Printf("ERROR: price wasn't the last item: %v", tokenized)
				helpMsg := "Virhe, hinta täytyy olla komennon viimeinen elementti"
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				bot.Send(outMsg)
				continue
			}

			dbengine.InsertShopping(username, shopName, category, purchaseDate, price)

			log.Printf("Purchased from %s [%s] with price %.2f by %s on %s",
				shopName,
				category,
				price,
				username,
				purchaseDate.Format("01-2006"))
			thxMsg := fmt.Sprintf("Ostosi on kirjattu, %s. Kiitos!", username)
			outMsg := tgbotapi.NewMessage(channelId, thxMsg)
			bot.Send(outMsg)
			continue
		case "palkka":
			if len(tokenized) < 3 {
				displayHelp(username, channelId, bot)
				continue
			}

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
		case "velat", "velkaa":
			forMonth := utils.GetDate(tokenized, "01-2006")
			if reflect.DeepEqual(forMonth, time.Time{}) {
				log.Printf("ERROR: couldn't parse date for debts: %v", err)
				helpMsg := "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				bot.Send(outMsg)
				continue
			}

			debts, err := dbengine.GetSalaryCompensatedDebts(forMonth)
			if err != nil {
				log.Print(err)
				helpMsg := "Voi ei, ei saatu velkatietoja."
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				bot.Send(outMsg)
				continue
			}

			for _, user := range debts {
				msg := fmt.Sprintf("%s: %s on velkaa %.2f",
					forMonth.Format("01-2006"),
					user.Username,
					user.Owes,
				)
				outMsg := tgbotapi.NewMessage(channelId, msg)
				bot.Send(outMsg)
			}
		case "help", "apua":
			displayHelp(username, channelId, bot)
			continue
		default:
			continue
		}
	}
}
