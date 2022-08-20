package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
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
	"weezel/budget/web"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pressly/goose/v3"
)

var configFileName string

//go:embed schemas/*.sql
var sqlMigrations embed.FS

var schemasDir = "schemas"

// setWorkingDirectory changes working directory to same where
// the executable is
func setWorkingDirectory(workdirPath string) string {
	absPath, err := filepath.Abs(workdirPath)
	if err != nil {
		logger.Fatal(err)
	}
	cdwPath := path.Dir(absPath)
	if err := os.Chdir(cdwPath); err != nil {
		logger.Fatal(err)
	}
	log.Printf("Working directory set to %s\n", cdwPath)

	trimmed := strings.TrimRight(cdwPath, "/")
	return trimmed + "/"
}

func dbMigrations(conf confighandler.TomlConfig) error {
	dbConn, err := dbengine.DBConnForMigrations(conf)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	goose.SetBaseFS(sqlMigrations)

	// Do the DB Migrations
	// goose.SetLogger(&logrus.Logger{}) // FIXME
	if err := goose.Status(dbConn, schemasDir); err != nil {
		return err
	}
	if err := goose.Up(dbConn, schemasDir); err != nil {
		return err
	}

	return nil
}

func main() {
	ctx := context.Background()

	flag.StringVar(&configFileName, "f", "", "Config file name")
	flag.Parse()

	if configFileName == "" {
		fmt.Println("ERROR: Give config file as an argument")
		os.Exit(1)
	}

	wd, _ := os.Getwd()

	configFile, err := os.ReadFile(filepath.Join(wd, configFileName))
	if err != nil {
		log.Panic(err)
	}
	conf, err := confighandler.LoadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	cwd := setWorkingDirectory(conf.General.WorkingDir)

	err = logger.SetLoggingToFile(filepath.Join(cwd, "budget.log"))
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		logger.CloseLogFile()
	}()

	// Perform database migrations
	err = dbMigrations(conf)
	if err != nil {
		logger.Fatal(err)
	}

	// protector.Protect(filepath.Join(cwd, "/"))

	_, err = dbengine.New(ctx, conf.Postgres)
	if err != nil {
		logger.Fatal(err)
	}

	shortlivedpage.InitScheduler()

	bot, err := tgbotapi.NewBotAPI(conf.Telegram.APIKey)
	if err != nil {
		logger.Fatalf("Couldn't create a new bot: %s", err)
	}
	bot.Debug = false
	logger.Infof("Using sername: %s", bot.Self.UserName)
	go telegramhandler.ConnectionHandler(
		bot,
		conf.Telegram.ChannelID,
		conf.Webserver.Hostname)

	mux := http.NewServeMux()
	mux.HandleFunc("/", web.APIHandler)
	httpServ := &http.Server{
		Addr:    conf.Webserver.HTTPPort,
		Handler: mux,
	}

	go func() {
		logger.Info(httpServ.ListenAndServe())
	}()
	logger.Infof("Listening on port %s", conf.Webserver.HTTPPort)

	// Graceful shutdown for HTTP server
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	logger.Infof("HTTP server stopping")
	defer cancel()
	logger.Fatal(httpServ.Shutdown(ctx))
}
