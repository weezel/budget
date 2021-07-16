package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"weezel/budget/confighandler"
	"weezel/budget/dbengine"
	"weezel/budget/shortlivedpage"
	"weezel/budget/telegramhandler"
	"weezel/budget/utils"
	"weezel/budget/web"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	logFileName string = "budget.log"
)

var (
	loggingFileHandle *os.File
)

func connectAndInitDb(dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	if exists, _ := utils.PathExists(dbPath); exists == false {
		dbengine.CreateSchema(db)
	}

	return db
}

// setWorkingDirectory changes working directory to same where
// the executable is
func setWorkingDirectory(workdirPath string) string {
	absPath, err := filepath.Abs(workdirPath)
	if err != nil {
		log.Fatal(err)
	}
	cdwPath := path.Dir(absPath + "/")
	if err := os.Chdir(cdwPath); err != nil {
		log.Fatal(err)
	}
	log.Printf("Working directory set to %s\n", cdwPath)

	trimmed := strings.TrimRight(cdwPath, "/")
	return trimmed + "/"
}

func logToFile(logDir string) *os.File {
	loggingFileAbsPath := path.Join(logDir, logFileName)
	log.Printf("Logging to file %s\n", loggingFileAbsPath)
	log.SetFlags(log.Ldate | log.Ltime)
	/* #nosec */
	f, err := os.OpenFile(
		loggingFileAbsPath,
		os.O_APPEND|os.O_CREATE|os.O_RDWR,
		0600)
	if err != nil {
		log.Fatalf("Error opening file %v\n", err)
	}
	log.SetOutput(f)
	return f
}

func main() {
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

	// This will initialize loggingFileHandle variable
	loggingFileHandle = logToFile(cwd)
	defer func() {
		if err := loggingFileHandle.Close(); err != nil {
			log.Printf("ERROR: couldn't close file: %s", err)
		}
	}()

	// protector.Protect(filepath.Join(cwd, "/"))

	db := connectAndInitDb(filepath.Join(cwd, "budget.db"))
	dbengine.UpdateDBReference(db)
	db = nil // GC variable

	bot, err := tgbotapi.NewBotAPI(conf.TeleConfig.ApiKey)
	if err != nil {
		log.Fatalf("Couldn't create a new bot: %s", err)
	}
	bot.Debug = false
	log.Printf("Using sername: %s", bot.Self.UserName)
	go telegramhandler.ConnectionHandler(
		bot,
		conf.TeleConfig.ChannelId,
		conf.WebserverConfig.Hostname)

	shortlivedpage.InitScheduler()

	mux := http.NewServeMux()
	mux.HandleFunc("/", web.ApiHandler)
	log.Printf("Listening on port %q\n", conf.WebserverConfig.HttpPort)
	err = http.ListenAndServe(conf.WebserverConfig.HttpPort, mux)
	if err != nil {
		log.Fatalf("Cannot listen on port %q: %q",
			conf.WebserverConfig.HttpPort,
			err)
	}

}
