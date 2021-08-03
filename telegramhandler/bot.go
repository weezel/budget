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
	"weezel/budget/outputs"
	"weezel/budget/shortlivedpage"
	"weezel/budget/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var splitPath = regexp.MustCompile(`\s+`)

func displayHelp(username string, channelId int64, bot *tgbotapi.BotAPI) {
	log.Printf("Help requested by %s", username)
	helpMsg := "Tunnistan seuraavat komennot:\n\n"
	helpMsg += "kulutus\n\n"
	helpMsg += "**osto** paikka [vapaaehtoinen pvm muodossa kk-vvvv] xx.xx\n\n"
	helpMsg += "**ostot** kk-vvvv kk-vvvv (mistä mihin)\n\n"
	helpMsg += "**palkka** kk-vvvv xxxx.xx (nettona)\r\n"
	helpMsg += "**palkat** kk-vvvv\r\n"
	helpMsg += "**poista** osto ID\r\n"
	helpMsg += "**velat** tai **velkaa** kk-vvvv\n\n"
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

func ConnectionHandler(bot *tgbotapi.BotAPI, channelId int64, hostname string) {
	var command string
	var shopName string
	var lastElem string
	var price float64

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)

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
			if len(tokenized) < 3 {
				displayHelp(username, channelId, bot)
				continue
			}

			startMonth, err := time.Parse("01-2006", tokenized[1])
			if err != nil {
				helpMsg := "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				if SendTelegram(bot, outMsg, "ostot-startmonth", false) == false {
					continue
				}
			}

			endMonth, err := time.Parse("01-2006", tokenized[2])
			if err != nil {
				helpMsg := "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				if SendTelegram(bot, outMsg, "ostot-endmonth", false) == false {
					continue
				}
			}

			spending, err := dbengine.GetMonthlyPurchasesByUser(
				username, startMonth, endMonth)
			if err != nil {
				log.Println(err)
				outMsg := tgbotapi.NewMessage(
					channelId,
					"Kulutuksen hakemisessa ongelmaa")
				_ = SendTelegram(bot, outMsg, "purchByUser", false)
				continue
			}

			var spendings external.SpendingHTMLOutput = external.SpendingHTMLOutput{
				From:      startMonth,
				To:        endMonth,
				Spendings: spending,
			}
			htmlPage, err := outputs.HTML(spendings, outputs.MontlySpendingsTemplate)
			if err != nil {
				log.Printf("Couldn't generate HTML results for spendings: %s", err)
				continue
			}

			htmlPageHash := utils.CalcSha256Sum(htmlPage)
			shortlivedPage := shortlivedpage.ShortLivedPage{
				TimeToLiveSeconds: 600,
				StartTime:         time.Now(),
				HtmlPage:          &htmlPage,
			}
			addOk := shortlivedpage.Add(htmlPageHash, shortlivedPage)
			if addOk {
				endTime := shortlivedPage.StartTime.Add(
					time.Duration(shortlivedPage.TimeToLiveSeconds))
				log.Printf("Added shortlived spendings page %s with end time %s",
					htmlPageHash, endTime)
			}

			urlBase := fmt.Sprintf("Kulutustiedot saatavilla 10min ajan täällä: https://%s/spendings?page_hash=%s",
				hostname,
				htmlPageHash)
			outMsg := tgbotapi.NewMessage(channelId, urlBase)
			if SendTelegram(bot, outMsg, "ostot2", false) == false {
				continue
			}
			continue
		case "tilastot":
			if len(tokenized) < 3 {
				displayHelp(username, channelId, bot)
				continue
			}

			startMonth, err := time.Parse("01-2006", tokenized[1])
			if err != nil {
				helpMsg := "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				if !SendTelegram(bot, outMsg, "tilastot-startmonth", false) {
					continue
				}
			}

			endMonth, err := time.Parse("01-2006", tokenized[2])
			if err != nil {
				helpMsg := "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
				outMsg := tgbotapi.NewMessage(channelId, helpMsg)
				if !SendTelegram(bot, outMsg, "tilastot-endmonth", false) {
					continue
				}
			}

			monthlyStats, err := dbengine.GetMonthlyData(startMonth, endMonth)
			if err != nil {
				log.Println(err)
				outMsg := tgbotapi.NewMessage(
					channelId,
					"Tilastojen hakemisessa ongelmaa")
				_ = SendTelegram(bot, outMsg, "monthlyStats", false)
				continue
			}

			var spendings external.SpendingHTMLOutput = external.SpendingHTMLOutput{
				From:      startMonth,
				To:        endMonth,
				Spendings: monthlyStats,
			}
			htmlPage, err := outputs.HTML(spendings, outputs.MonthlyDataTemplate)
			if err != nil {
				log.Printf("Couldn't generate HTML results for statistics: %s", err)
				continue
			}

			htmlPageHash := utils.CalcSha256Sum(htmlPage)
			shortlivedPage := shortlivedpage.ShortLivedPage{
				TimeToLiveSeconds: 600,
				StartTime:         time.Now(),
				HtmlPage:          &htmlPage,
			}
			addOk := shortlivedpage.Add(htmlPageHash, shortlivedPage)
			if addOk {
				endTime := shortlivedPage.StartTime.Add(
					time.Duration(shortlivedPage.TimeToLiveSeconds))
				log.Printf("Added shortlived data page %s with end time %s",
					htmlPageHash, endTime)
			}

			urlBase := fmt.Sprintf("Tilastot saatavilla 10min ajan täällä: https://%s/statistics?page_hash=%s",
				hostname,
				htmlPageHash)
			outMsg := tgbotapi.NewMessage(channelId, urlBase)
			if !SendTelegram(bot, outMsg, "tilastot2", false) {
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
		case "poista":
			if len(tokenized) < 3 {
				displayHelp(username, channelId, bot)
				continue
			}

			switch tokenized[1] {
			case "osto":
				bid, err := strconv.ParseInt(tokenized[2], 10, 64)
				if err != nil {
					outMsg := tgbotapi.NewMessage(channelId, "Oston ID parsinta epäonnistui")
					if !SendTelegram(bot, outMsg, "poista1", false) {
						continue
					}
				}

				row, err := dbengine.GetSpendingRowByID(bid, username)
				if err != nil {
					errMsg := fmt.Sprintf("Oston hakeminen ID:n (%d) perusteella epäonnistui", bid)
					outMsg := tgbotapi.NewMessage(channelId, errMsg)
					if !SendTelegram(bot, outMsg, "poista2", false) {
						continue
					}
					continue
				}

				err = dbengine.DeleteSpendingByID(bid, username)
				if err != nil {
					errMsg := fmt.Sprintf("Oston ID (%d) poisto epäonnistui", bid)
					outMsg := tgbotapi.NewMessage(channelId, errMsg)
					if !SendTelegram(bot, outMsg, "poista3", false) {
						continue
					}
					continue
				}
				deletedEntry := fmt.Sprintf("Poistettu tapahtuma (ID %d) %s %.2f€ [%s]",
					row.ID, row.Shopname, row.Price, row.Purchasedate)
				outMsg := tgbotapi.NewMessage(channelId, deletedEntry)
				if !SendTelegram(bot, outMsg, "poista4", false) {
					continue
				}
			default:
				displayHelp(username, channelId, bot)
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
				helpMsg := fmt.Sprintf("Voi ei, ei saatu velkatietoja: %s", err)
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
