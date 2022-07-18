package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"weezel/budget/confighandler"
	"weezel/budget/dbengine"
	"weezel/budget/logger"
	"weezel/budget/outputs"
	"weezel/budget/shortlivedpage"
	"weezel/budget/telegramhandler"
	"weezel/budget/utils"
	"weezel/budget/web"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	localRun       bool
	configFileName string
)

// setWorkingDirectory changes working directory to same where
// the executable is
func setWorkingDirectory(workdirPath string) string {
	absPath, err := filepath.Abs(workdirPath)
	if err != nil {
		logger.Fatal(err)
	}
	cdwPath := path.Dir(absPath + "/")
	if err := os.Chdir(cdwPath); err != nil {
		logger.Fatal(err)
	}
	log.Printf("Working directory set to %s\n", cdwPath)

	trimmed := strings.TrimRight(cdwPath, "/")
	return trimmed + "/"
}

func main() {
	ctx := context.Background()
	var err error

	flag.BoolVar(&localRun, "l", false, "Local run")
	flag.StringVar(&configFileName, "f", "", "Config file name")
	flag.Parse()

	if configFileName == "" {
		fmt.Println("ERROR: Give config file as an argument")
		os.Exit(1)
	}

	configFile, err := ioutil.ReadFile(filepath.Clean(configFileName))
	if err != nil {
		log.Panic(err)
	}
	conf := confighandler.LoadConfig(configFile)

	cwd := setWorkingDirectory(conf.TeleConfig.WorkingDir)

	err = logger.SetLoggingToFile(filepath.Join(cwd, "budget.log"))
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		logger.CloseLogFile()
	}()

	// protector.Protect(filepath.Join(cwd, "/"))

	_, err = dbengine.New(filepath.Join(cwd, "budget.db"))
	if err != nil {
		logger.Fatal(err)
	}

	shortlivedpage.InitScheduler()

	if !localRun {
		bot, err := tgbotapi.NewBotAPI(conf.TeleConfig.ApiKey)
		if err != nil {
			logger.Fatalf("Couldn't create a new bot: %s", err)
		}
		bot.Debug = false
		logger.Infof("Using sername: %s", bot.Self.UserName)
		go telegramhandler.ConnectionHandler(
			bot,
			conf.TeleConfig.ChannelId,
			conf.WebserverConfig.Hostname)
	} else {
		// Run locally, hence without Telegram
		startMonth := utils.GetDate([]string{"03-2022"}, "01-2006")
		endMonth := utils.GetDate([]string{"03-2022"}, "01-2006")

		monthlyStats, err := dbengine.GetMonthlyPurchases(ctx, startMonth, endMonth)
		if err != nil {
			logger.Fatal("Tilastojen hakemisessa ongelmaa")
		}

		spendings := dbengine.SpendingHTMLOutput{
			From:      startMonth,
			To:        endMonth,
			Spendings: monthlyStats,
		}
		htmlPage, err := outputs.HTML(spendings, outputs.MontlySpendingsTemplate)
		if err != nil {
			logger.Fatalf("Sivun näyttämisessä ongelmaa: %s", err)
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

		fmt.Printf("Tilastot saatavilla 10min ajan täällä: http://127.0.0.1:8111/statistics?page_hash=%s",
			htmlPageHash)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", web.ApiHandler)
	httpServ := &http.Server{
		Addr:    conf.WebserverConfig.HttpPort,
		Handler: mux,
	}

	go func() {
		logger.Info(httpServ.ListenAndServe())
	}()
	logger.Infof("Listening on port %s", conf.WebserverConfig.HttpPort)

	// Graceful shutdown for HTTP server
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	logger.Infof("HTTP server stopping")
	defer cancel()
	logger.Fatal(httpServ.Shutdown(ctx))
}
