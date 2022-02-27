package main

import (
	"context"
	"database/sql"
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
	"weezel/budget/shortlivedpage"
	"weezel/budget/telegramhandler"
	"weezel/budget/utils"
	"weezel/budget/web"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func connectAndInitDb(dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Fatal(err)
	}
	if exists, _ := utils.PathExists(dbPath); !exists {
		dbengine.CreateSchema(db)
	}

	return db
}

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
	var err error

	if len(os.Args) < 2 {
		fmt.Println("ERROR: Give config file as an argument")
		os.Exit(1)
	}

	configFile, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Panic(err)
	}
	conf := confighandler.LoadConfig(configFile)

	var cwd string = setWorkingDirectory(conf.TeleConfig.WorkingDir)

	err = logger.SetLoggingToFile(filepath.Join(cwd, "budget.log"))
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		logger.CloseLogFile()
	}()

	// protector.Protect(filepath.Join(cwd, "/"))

	db := connectAndInitDb(filepath.Join(cwd, "budget.db"))
	dbengine.UpdateDBReference(db)

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

	shortlivedpage.InitScheduler()

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
