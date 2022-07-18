package shortlivedpage

import (
	"log"
	"sync"
	"time"
	"weezel/budget/logger"

	"github.com/prprprus/scheduler"
)

var (
	lock            sync.RWMutex
	shortLivedPages map[string]ShortLivedPage
)

type ShortLivedPage struct {
	StartTime  time.Time
	HTMLPage   *[]byte
	TTLSeconds int64
}

func cleaner() {
	removableHashes := []string{}

	logger.Debugf("Starting scheduled cleaning for short lived pages")
	for pageHash, page := range shortLivedPages {
		endTime := page.StartTime.Add(
			time.Duration(
				int64(time.Second) * page.TTLSeconds))
		if time.Now().After(endTime) {
			logger.Infof("Removing page due timeout: %v [%s]",
				page,
				endTime)
			removableHashes = append(removableHashes, pageHash)
		}
	}
	for _, pageHash := range removableHashes {
		lock.Lock()
		delete(shortLivedPages, pageHash)
		lock.Unlock()
	}
	logger.Debugf("Stopping scheduled cleaning for short lived pages")
}

func InitScheduler() {
	shortLivedPages = make(map[string]ShortLivedPage)

	cleanerSchedule, err := scheduler.NewScheduler(1000)
	if err != nil {
		log.Fatalf("Error while initializing scheduler: %s", err)
	}
	logger.Infof("Cleaner scheduler started")
	cleanerSchedule.Every().Second(0).Do(cleaner)
}

func Get(pageHash string) ShortLivedPage {
	lock.RLock()
	defer lock.RUnlock()

	if _, ok := shortLivedPages[pageHash]; ok {
		return shortLivedPages[pageHash]
	}

	return ShortLivedPage{}
}

func Add(pageHash string, page ShortLivedPage) bool {
	lock.Lock()
	defer lock.Unlock()

	if _, ok := shortLivedPages[pageHash]; !ok {
		shortLivedPages[pageHash] = page
		return true
	}
	return false
}

func Remove(pageHash string) bool {
	lock.Lock()
	defer lock.Unlock()

	if _, ok := shortLivedPages[pageHash]; ok {
		delete(shortLivedPages, pageHash)
		return true
	}
	return false
}
