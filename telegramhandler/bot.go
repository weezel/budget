package telegramhandler

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"weezel/budget/dbengine"
	"weezel/budget/external"
	"weezel/budget/logger"
	"weezel/budget/outputs"
	"weezel/budget/shortlivedpage"
	"weezel/budget/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var splitPath = regexp.MustCompile(`\s+`)

func displayHelp(username string, channelId int64, bot *tgbotapi.BotAPI) {
	logger.Infof("Help requested by %s", username)
	helpMsg := "Tunnistan seuraavat komennot:\n\n"
	helpMsg += "kulutus\n\n"
	helpMsg += "**osto** paikka [vapaaehtoinen pvm muodossa kk-vvvv] xx.xx\n\n"
	helpMsg += "**ostot** kk-vvvv kk-vvvv (mistä mihin)\n\n"
	helpMsg += "**palkka** kk-vvvv xxxx.xx (nettona)\r\n"
	helpMsg += "**palkat** kk-vvvv\r\n"
	helpMsg += "**poista** osto ID\r\n"
	helpMsg += "**tilastot** kk-vvvv kk-vvvv\r\n"
	helpMsg += "**velat** tai **velkaa** kk-vvvv\n\n"
	outMsg := tgbotapi.NewMessage(channelId, helpMsg)
	if _, err := bot.Send(outMsg); err != nil {
		logger.Errorf("sending to channel failed: %s", err)
	}
}

func handlePurchase(
	shopName string,
	lastElem string,
	username string,
	tokenized []string,
) string {
	category := utils.GetCategory(tokenized)
	purchaseDate := utils.GetDate(tokenized, "01-2006")
	price, err := strconv.ParseFloat(lastElem, 64)
	if err != nil {
		return "Virhe, hinta täytyy olla komennon viimeinen elementti ja muodossa x,xx tai x.xx"
	}

	err = dbengine.InsertPurchase(username, shopName, category, purchaseDate, price)
	if err != nil {
		logger.Error(err)
		return "Ostotapahtuman kirjaus epäonnistui"
	}

	logger.Infof("Purchased from %s [%s] with price %.2f by %s on %s",
		shopName,
		category,
		price,
		username,
		purchaseDate.Format("01-2006"))

	return fmt.Sprintf("Ostosi on kirjattu, %s. Kiitos!", username)
}

func getPurchasesData(username string, hostname string, tokenized []string) string {
	startMonth := utils.GetDate(tokenized[1:], "01-2006")
	endMonth := utils.GetDate(tokenized[2:], "01-2006")

	spending, err := dbengine.GetMonthlyPurchasesByUser(
		username, startMonth, endMonth)
	if err != nil {
		logger.Error(err)
		return "Kulutuksen hakemisessa ongelmaa"
	}

	var spendings external.SpendingHTMLOutput = external.SpendingHTMLOutput{
		From:      startMonth,
		To:        endMonth,
		Spendings: spending,
	}
	htmlPage, err := outputs.HTML(spendings, outputs.MontlySpendingsTemplate)
	if err != nil {
		logger.Errorf("Couldn't generate HTML results for spendings: %s", err)
		return "Kulutuksen näyttämisessä ongelmaa"
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
		logger.Infof("Added shortlived spendings page %s with end time %s",
			htmlPageHash, endTime)
	}

	return fmt.Sprintf("Kulutustiedot saatavilla 10min ajan täällä: https://%s/spendings?page_hash=%s",
		hostname,
		htmlPageHash)
}

func getStatsTimeSpan(hostname string, tokenized []string) string {
	startMonth := utils.GetDate(tokenized[1:], "01-2006")
	endMonth := utils.GetDate(tokenized[2:], "01-2006")

	monthlyStats, err := dbengine.GetMonthlyData(startMonth, endMonth)
	if err != nil {
		logger.Error(err)
		return "Tilastojen hakemisessa ongelmaa"
	}

	var spendings external.SpendingHTMLOutput = external.SpendingHTMLOutput{
		From:      startMonth,
		To:        endMonth,
		Spendings: monthlyStats,
	}
	htmlPage, err := outputs.HTML(spendings, outputs.MonthlyDataTemplate)
	if err != nil {
		logger.Infof("Couldn't generate HTML results for statistics: %s", err)
		return "Sivun näyttämisessä ongelmaa"
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
		logger.Infof("Added shortlived data page %s with end time %s",
			htmlPageHash, endTime)
	}

	return fmt.Sprintf("Tilastot saatavilla 10min ajan täällä: https://%s/statistics?page_hash=%s",
		hostname,
		htmlPageHash)
}

func handleRemovePurchase(username string, tokenized []string) (string, error) {
	switch tokenized[1] {
	case "osto":
		bid, err := strconv.ParseInt(tokenized[2], 10, 64)
		if err != nil {
			logger.Error(err)
			return "Oston ID parsinta epäonnistui", nil
		}

		row, err := dbengine.GetSpendingRowByID(bid, username)
		if err != nil {
			logger.Error(err)
			return fmt.Sprintf("Oston hakeminen ID:n (%d) perusteella epäonnistui", bid),
				nil
		}

		err = dbengine.DeleteSpendingByID(bid, username)
		if err != nil {
			logger.Error(err)
			return fmt.Sprintf("Oston ID (%d) poisto epäonnistui", bid), nil
		}
		logger.Infof("Removed item (ID %d) %s %.2f€ [%s] by %s",
			row.ID, row.Shopname, row.Price, row.Purchasedate, username)
		return fmt.Sprintf("Poistettu tapahtuma (ID %d) %s %.2f€ [%s]",
			row.ID, row.Shopname, row.Price, row.Purchasedate), nil
	}

	return "", errors.New("unknown operation")
}

func handleGetSalaries(tokenized []string) string {
	forMonth := utils.GetDate(tokenized, "01-2006")
	if reflect.DeepEqual(forMonth, time.Time{}) {
		logger.Errorf("Couldn't parse date for salaries: %#v", tokenized)
		return "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
	}

	halfYearAgo := forMonth.AddDate(0, -6, 0)
	salaries, err := dbengine.GetSalariesByMonthRange(
		halfYearAgo,
		forMonth,
	)
	if err != nil {
		logger.Error(err)
		return "Voi ei, ei saatu palkkatietoja."
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

	return strings.Join(finalMsg, "\n")
}

func handleSalaryInsert(
	username string,
	lastElem string,
	tokenized []string,
) string {
	salaryDate := utils.GetDate(tokenized, "01-2006")
	salary, err := strconv.ParseFloat(lastElem, 64)
	if err != nil {
		logger.Errorf("couldn't parse salary: %v", err)
		return "Virhe palkan parsinnassa. Palkan oltava viimeisenä ja muodossa x.xx tai x,xx"
	}

	dbengine.InsertSalary(username, salary, salaryDate)

	logger.Infof("Inserted salary amount of %.2f by %s on %s",
		salary,
		username,
		salaryDate.Format("01-2006"))
	return fmt.Sprintf("Palkka kirjattu, %s. Kiitos!", username)
}

func handleVelat(tokenized []string) string {
	forMonth := utils.GetDate(tokenized, "01-2006")
	if reflect.DeepEqual(forMonth, time.Time{}) {
		logger.Errorf("couldn't parse date for debts")
		return "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
	}

	debts, err := dbengine.GetSalaryCompensatedDebts(forMonth)
	if err != nil {
		logger.Error(err)
		return fmt.Sprintf("Voi ei, ei saatu velkatietoja: %s", err)
	}

	var s strings.Builder
	for _, user := range debts {
		msg := fmt.Sprintf("%s: %s on velkaa %.2f\n",
			forMonth.Format("01-2006"),
			user.Username,
			user.Owes,
		)
		s.WriteString(msg)
	}

	return s.String()
}

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
		return fmt.Errorf("sending to channel failed %s", err)
	}
	return nil
}

func ConnectionHandler(bot *tgbotapi.BotAPI, channelId int64, hostname string) {
	var command string
	var lastElem string
	var err error

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		logger.Errorf("Failed to receive updates: %#v", err)
	}

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
		logger.Infof("Tokenized: %v", tokenized)
		command = strings.ToLower(tokenized[0])

		switch command {
		case "osto":
			if len(tokenized) < 3 {
				displayHelp(username, channelId, bot)
				continue
			}

			shopName := tokenized[1]
			msg := handlePurchase(shopName, lastElem, username, tokenized)
			outMsg := tgbotapi.NewMessage(channelId, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "ostot":
			if len(tokenized) < 3 {
				displayHelp(username, channelId, bot)
				continue
			}

			msg := getPurchasesData(username, hostname, tokenized)
			outMsg := tgbotapi.NewMessage(channelId, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "tilastot":
			if len(tokenized) < 3 {
				displayHelp(username, channelId, bot)
				continue
			}

			msg := getStatsTimeSpan(hostname, tokenized)
			outMsg := tgbotapi.NewMessage(channelId, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "palkka":
			if len(tokenized) < 3 {
				displayHelp(username, channelId, bot)
				continue
			}

			msg := handleSalaryInsert(username, lastElem, tokenized)
			outMsg := tgbotapi.NewMessage(channelId, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "palkat":
			if len(tokenized) < 2 {
				displayHelp(username, channelId, bot)
				continue
			}

			msg := handleGetSalaries(tokenized)
			outMsg := tgbotapi.NewMessage(channelId, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "poista":
			if len(tokenized) < 3 {
				displayHelp(username, channelId, bot)
				continue
			}

			msg, err := handleRemovePurchase(username, tokenized)
			if err != nil {
				displayHelp(username, channelId, bot)
				continue
			}
			outMsg := tgbotapi.NewMessage(channelId, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "velat", "velkaa":
			if len(tokenized) < 2 {
				displayHelp(username, channelId, bot)
				continue
			}

			msg := handleVelat(tokenized)
			outMsg := tgbotapi.NewMessage(channelId, msg)
			if err = SendTelegram(bot, outMsg, false); err != nil {
				logger.Error(err)
			}
		case "help", "apua":
			displayHelp(username, channelId, bot)
			continue
		}

	}
}
