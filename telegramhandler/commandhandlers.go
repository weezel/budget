package telegramhandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"weezel/budget/dbengine"
	"weezel/budget/logger"
	"weezel/budget/outputs"
	"weezel/budget/shortlivedpage"
	"weezel/budget/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func displayHelp(username string, channelID int64, bot *tgbotapi.BotAPI) {
	logger.Infof("Help requested by %s", username)
	helpMsg := "Tunnistan seuraavat komennot:\n\n"
	helpMsg += "**osto** paikka [vapaaehtoinen pvm muodossa kk-vvvv] xx.xx\n\n"
	helpMsg += "**palkka** kk-vvvv xxxx.xx (nettona)\r\n"
	helpMsg += "**poista** [osto TAI palkka] ID\r\n"
	helpMsg += "**tilastot** kk-vvvv kk-vvvv\r\n"
	helpMsg += "**velat** tai **velkaa** kk-vvvv\n\n"
	outMsg := tgbotapi.NewMessage(channelID, helpMsg)
	if _, err := bot.Send(outMsg); err != nil {
		logger.Errorf("sending to channel failed: %s", err)
	}
}

func handlePurchase(
	ctx context.Context,
	shopName string,
	rawPrice string,
	username string,
	tokenized []string,
) string {
	category := utils.GetCategory(tokenized)
	purchaseDate := utils.GetDate(tokenized, "01-2006")
	price, err := strconv.ParseFloat(rawPrice, 64)
	if err != nil {
		logger.Error(err)
		return "Virhe, hinta täytyy olla komennon viimeinen elementti ja muodossa x,xx tai x.xx"
	}

	pid, err := dbengine.AddExpense(ctx, username, shopName, category, purchaseDate, price)
	if err != nil {
		logger.Error(err)
		return "Ostotapahtuman kirjaus epäonnistui"
	}

	logger.Infof("Purchased from %s [%s] with price %.2f by %s on %s, ID=%d",
		shopName,
		category,
		price,
		username,
		purchaseDate.Format("01-2006"),
		pid)

	return fmt.Sprintf("Ostosi on kirjattu, %s. Kiitos!", username)
}

func getStatsByTimeSpan(ctx context.Context, username string, hostname string, tokenized []string) string {
	startMonth := utils.GetDate(tokenized[1:], "01-2006")
	endMonth := utils.GetDate(tokenized[2:], "01-2006")

	stats, err := dbengine.StatisticsByTimespan(ctx, startMonth, endMonth)
	if err != nil {
		logger.Error(err)
		return "Tilastojen hakemisessa ongelmaa"
	}

	detailedExpenses, err := dbengine.GetExpensesByTimespan(ctx, startMonth, endMonth)
	if err != nil {
		logger.Error(err)
		return "Kulutuksen hakemisessa ongelmaa"
	}

	statsHTMLVars := outputs.StatisticsVars{
		From:       startMonth,
		To:         endMonth,
		Statistics: stats,
		Detailed:   detailedExpenses,
	}
	htmlPage, err := outputs.RenderStatsHTML(statsHTMLVars)
	if err != nil {
		logger.Errorf("Couldn't generate HTML results for spendings: %s", err)
		return "Kulutuksen näyttämisessä ongelmaa"
	}

	htmlPageHash := utils.CalcSha256Sum(htmlPage)
	shortlivedPage := shortlivedpage.ShortLivedPage{
		TTLSeconds: 600,
		StartTime:  time.Now(),
		HTMLPage:   &htmlPage,
	}
	addOk := shortlivedpage.Add(htmlPageHash, shortlivedPage)
	if addOk {
		endTime := shortlivedPage.StartTime.Add(
			time.Duration(shortlivedPage.TTLSeconds))
		logger.Infof("Added shortlived spendings page %s with end time %s",
			htmlPageHash, endTime)
	}

	return fmt.Sprintf("Kulutustiedot saatavilla 10min ajan täällä: https://%s/spendings?page_hash=%s",
		hostname,
		htmlPageHash)
}

func getStatsTimeSpan(ctx context.Context, hostname string, tokenized []string) string {
	startMonth := utils.GetDate(tokenized[1:], "01-2006")
	endMonth := utils.GetDate(tokenized[2:], "01-2006")

	if startMonth.IsZero() || endMonth.IsZero() {
		logger.Errorf("couldn't parse date for stats, start=%#v, end=%#v",
			startMonth, endMonth)
		return "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
	}

	monthlyStats, err := dbengine.StatisticsByTimespan(ctx, startMonth, endMonth)
	if err != nil {
		logger.Error(err)
		return "Tilastojen hakemisessa ongelmaa"
	}

	statsVars := outputs.StatisticsVars{
		From:       startMonth,
		To:         endMonth,
		Statistics: monthlyStats,
	}
	htmlPage, err := outputs.RenderStatsHTML(statsVars)
	if err != nil {
		logger.Infof("Couldn't generate HTML results for statistics: %s", err)
		return "Sivun näyttämisessä ongelmaa"
	}

	htmlPageHash := utils.CalcSha256Sum(htmlPage)
	shortlivedPage := shortlivedpage.ShortLivedPage{
		TTLSeconds: 600,
		StartTime:  time.Now(),
		HTMLPage:   &htmlPage,
	}
	addOk := shortlivedpage.Add(htmlPageHash, shortlivedPage)
	if addOk {
		endTime := shortlivedPage.StartTime.Add(
			time.Duration(shortlivedPage.TTLSeconds))
		logger.Infof("Added shortlived data page %s with end time %s",
			htmlPageHash, endTime)
	}

	return fmt.Sprintf("Tilastot saatavilla 10min ajan täällä: https://%s/statistics?page_hash=%s",
		hostname,
		htmlPageHash)
}

func handleRemovePurchase(ctx context.Context, username string, tokenized []string) string {
	switch tokenized[1] {
	case "osto":
		pid, err := strconv.ParseInt(tokenized[2], 10, 32)
		if err != nil {
			logger.Error(err)
			return "Oston ID parsinta epäonnistui"
		}

		deletedID, err := dbengine.DeleteExpenseByID(ctx, int32(pid), username)
		if err != nil {
			logger.Error(err)
			return fmt.Sprintf("Oston ID (%d) poisto epäonnistui", pid)
		}
		logger.Infof("Removed expense item ID=%d %s %.2f€ [%s] by %s",
			deletedID.ID, deletedID.ShopName, deletedID.Price, deletedID.ExpenseDate, username)
		return fmt.Sprintf("Poistettu kulutapahtuma (ID %d) %s %.2f€ [%s] by %s",
			deletedID.ID, deletedID.ShopName, deletedID.Price, deletedID.ExpenseDate, username)
	case "palkka":
		pid, err := strconv.ParseInt(tokenized[2], 10, 32)
		if err != nil {
			logger.Error(err)
			return "Palkan ID parsinta epäonnistui"
		}

		deletedID, err := dbengine.DeleteSalaryByID(ctx, int32(pid), username)
		if err != nil {
			logger.Error(err)
			return fmt.Sprintf("Palkan ID (%d) poisto epäonnistui", pid)
		}
		logger.Infof("Removed salary item ID=%d %s %.2f by %s",
			deletedID.ID, deletedID.StoreDate, deletedID.Salary, username)
		return fmt.Sprintf("Poistettu palkkatapahtuma (ID %d) %s %.2f€ [%s]",
			deletedID.ID, deletedID.StoreDate, deletedID.Salary, username)
	}

	return "Vain 'osto' tai 'palkka' kelepaa"
}

func handleSalaryInsert(ctx context.Context, username string, lastElem string, tokenized []string) string {
	salaryDate := utils.GetDate(tokenized, "01-2006")
	salary, err := strconv.ParseFloat(lastElem, 64)
	if err != nil {
		logger.Errorf("couldn't parse salary: %v", err)
		return "Virhe palkan parsinnassa. Palkan oltava viimeisenä ja muodossa x.xx tai x,xx"
	}

	pid, err := dbengine.AddSalary(ctx, username, salary, salaryDate)
	if err != nil {
		logger.Errorf("couldn't insert salary: %v", err)
		return "Virhe palkan lisäämisessä, kysy apua"
	}

	logger.Infof("Inserted salary amount of %.2f by %s on %s, ID=%d",
		salary,
		username,
		salaryDate.Format("01-2006"),
		pid)
	return fmt.Sprintf("Palkka kirjattu, %s. Kiitos!", username)
}

func handleDebts(ctx context.Context, tokenized []string) string {
	month := utils.GetDate(tokenized, "01-2006")
	if month.IsZero() {
		logger.Errorf("couldn't parse date for debts")
		return "Virhe päivämäärän parsinnassa. Oltava muotoa kk-vvvv"
	}

	debts, err := dbengine.GetSalaryCompensatedDebts(ctx, month, month)
	if err != nil {
		logger.Error(err)
		return fmt.Sprintf("Bzzzt, ei saatu velkatietoja")
	}

	var s strings.Builder
	for _, user := range debts {
		msg := fmt.Sprintf("%s: %s on velkaa %.2f\n",
			month.Format("01-2006"),
			user.Username,
			user.Owes,
		)
		s.WriteString(msg)
	}

	return s.String()
}
