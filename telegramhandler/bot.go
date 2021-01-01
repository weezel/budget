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
	"weezel/budget/external"
	"weezel/budget/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var splitPath = regexp.MustCompile(`\s+`)

func displayHelp(username string, channelId int64, bot *tgbotapi.BotAPI) {
	log.Printf("Help requested by %s", username)
	helpMsg := "Tunnistan seuraavat komennot:\n\n"
	helpMsg += "kulutus\n\n"
	helpMsg += "osto paikka [vapaaehtoinen pvm muodossa kk-vvvv] xx.xx\n\n"
	helpMsg += "ostot [kk-vvvv]\n\n"
	helpMsg += "palkka kk-vvvv xxxx.xx (nettona)\r\n"
	helpMsg += "palkat kk-vvvv\r\n"
	helpMsg += "velat, velkaa kk-vvvv\n\n"
	outMsg := tgbotapi.NewMessage(channelId, helpMsg)
	if _, err := bot.Send(outMsg); err != nil {
		log.Printf("ERROR: sending to channel failed: %s", err)
	}
}

// SendTelegram returns true if message sending succeeds and false otherwise
func SendTelegram(
	bot *tgbotapi.BotAPI,
	msg tgbotapi.MessageConfig,
	sectionName string,
	markdown bool) bool {
	if markdown {
		msg.ParseMode = tgbotapi.ModeMarkdown
	}
	if _, err := bot.Send(msg); err != nil {
		log.Printf("ERROR: %s: sending to channel failed %s", sectionName, err)
		return false
	}
	return true
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
			",",
			".",
			-1)
		log.Printf("Tokenized: %v", tokenized)
		command = strings.ToLower(tokenized[0])

		switch command {
		case "kulutus":
			log.Printf("Spending report requested by %s", username)
			// spendingData, err := dbengine.GetMonthlySpending()
			// if err != nil {
			// 	log.Print(err)
			// 	continue
			// }
			// spendingImg, err := plotters.LineHistogramOfAnnualSpending(spendingData)
			// if err != nil {
			// 	log.Print(err)
			// 	continue
			// }
			// imgSum := utils.CalcSha256Sum(spendingImg)
			// if imgSum == "" {
			// 	log.Print("ERROR: plotting image checksum was zero")
			// 	continue
			// }
			// photoUpload := tgbotapi.NewPhotoUpload(
			// 	channelId,
			// 	tgbotapi.FileBytes{
			// 		Name:  imgSum + ".png",
			// 		Bytes: spendingImg,
			// 	},
			// )
			// _, err = bot.Send(photoUpload)
			// if err != nil {
			// 	log.Printf("ERROR: upload spending img failed: %v", err)
			// }

			// log.Print("Spending report generated")
			continue
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
				if SendTelegram(bot, outMsg, "osto1", false) == false {
					continue
				}
				continue
			}

			err = dbengine.InsertShopping(username, shopName, category, purchaseDate, price)
			if err != nil {
				log.Println(err)
				outMsg := tgbotapi.NewMessage(channelId, err.Error())
				_ = SendTelegram(bot, outMsg, "osto2", false)
				continue
			}

			log.Printf("Purchased from %s [%s] with price %.2f by %s on %s",
				shopName,
				category,
				price,
				username,
				purchaseDate.Format("01-2006"))
			thxMsg := fmt.Sprintf("Ostosi on kirjattu, %s. Kiitos!", username)
			outMsg := tgbotapi.NewMessage(channelId, thxMsg)
			if SendTelegram(bot, outMsg, "osto3", false) == false {
				continue
			}
			continue
		case "ostot":
			if len(tokenized) < 2 {
				displayHelp(username, channelId, bot)
				continue
			}

			month, err := time.Parse("01-2006", tokenized[1])
			if err != nil {
				helpMsg := "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				if SendTelegram(bot, outMsg, "ostot1", false) == false {
					continue
				}
			}

			spending, err := dbengine.GetMonthlyPurchasesByUser(username, month)
			if err != nil {
				log.Println(err)
				outMsg := tgbotapi.NewMessage(
					channelId,
					"Kulutuksen hakemisessa ongelmaa")
				_ = SendTelegram(bot, outMsg, "purchByUser", false)
				continue
			}
			if reflect.DeepEqual(spending, []external.SpendingHistory{}) {
				outMsg := tgbotapi.NewMessage(
					channelId,
					"Ei ostoja tässä kuussa")
				_ = SendTelegram(bot, outMsg, "noPurchasesByUser", false)
				continue
			}

			var finalMsg []string = make([]string, len(spending))
			for i, s := range spending {
				cleanedEvent := strings.ReplaceAll(s.EventName, "_", " ")
				msg := fmt.Sprintf("%s  %s  %.2f",
					s.MonthYear.Format("01-2006"),
					cleanedEvent,
					s.Spending)
				finalMsg[i] = msg
			}
			outMsg := tgbotapi.NewMessage(channelId, strings.Join(finalMsg, "\n"))
			if SendTelegram(bot, outMsg, "ostot2", false) == false {
				continue
			}

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
				if SendTelegram(bot, outMsg, "palkka1", false) == false {
					continue
				}
				continue
			}

			dbengine.InsertSalary(username, salary, salaryDate)

			log.Printf("Salary amount of %.2f by %s on %s",
				salary,
				username,
				salaryDate.Format("01-2006"))
			thxMsg := fmt.Sprintf("Palkka kirjattu, %s. Kiitos!", username)
			outMsg := tgbotapi.NewMessage(channelId, thxMsg)
			if SendTelegram(bot, outMsg, "palkka2", false) == false {
				continue
			}
		case "palkat":
			forMonth := utils.GetDate(tokenized, "01-2006")
			if reflect.DeepEqual(forMonth, time.Time{}) {
				log.Printf("ERROR: couldn't parse date for salaries: %v", err)
				helpMsg := "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				if SendTelegram(bot, outMsg, "palkat1", false) == false {
					continue
				}
				continue
			}

			halfYearAgo := forMonth.AddDate(0, -6, 0)
			salaries, err := dbengine.GetSalariesByMonthRange(
				halfYearAgo,
				forMonth,
			)
			if err != nil {
				log.Print(err)
				helpMsg := "Voi ei, ei saatu palkkatietoja."
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				if SendTelegram(bot, outMsg, "palkat2", false) == false {
					continue
				}
				continue
			}

			var finalMsg []string = make([]string, len(salaries))
			for i, user := range salaries {
				var salarySet string = "\u274C"
				if user.Salary > 0 {
					salarySet = "\u2714"
				}
				msg := fmt.Sprintf("%s  %s  %s",
					user.Username,
					user.Date,
					salarySet,
				)
				finalMsg[i] = msg
			}
			outMsg := tgbotapi.NewMessage(channelId, strings.Join(finalMsg, "\n"))
			if SendTelegram(bot, outMsg, "palkat3", false) == false {
				continue
			}
		case "velat", "velkaa":
			forMonth := utils.GetDate(tokenized, "01-2006")
			if reflect.DeepEqual(forMonth, time.Time{}) {
				log.Printf("ERROR: couldn't parse date for debts: %v", err)
				helpMsg := "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				if SendTelegram(bot, outMsg, "velat1", false) == false {
					continue
				}
				continue
			}

			debts, err := dbengine.GetSalaryCompensatedDebts(forMonth)
			if err != nil {
				log.Print(err)
				helpMsg := "Voi ei, ei saatu velkatietoja."
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				if SendTelegram(bot, outMsg, "velat2", false) == false {
					continue
				}
				continue
			}

			for _, user := range debts {
				msg := fmt.Sprintf("%s: %s on velkaa %.2f",
					forMonth.Format("01-2006"),
					user.Username,
					user.Owes,
				)
				outMsg := tgbotapi.NewMessage(channelId, msg)
				if SendTelegram(bot, outMsg, "velat3", false) == false {
					continue
				}
			}
		case "help", "apua":
			displayHelp(username, channelId, bot)
			continue
		default:
			continue
		}
	}
}
