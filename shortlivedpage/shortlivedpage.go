package shortlivedpage

import (
	"log"
	"sync"
	"time"

	"github.com/prprprus/scheduler"
)

var (
	lock            sync.RWMutex
	shortLivedPages map[string]ShortLivedPage
	cleanerSchedule *scheduler.Scheduler
)

type ShortLivedPage struct {
	TimeToLiveSeconds int64
	StartTime         time.Time
	HtmlPage          *[]byte
}

func cleaner() {
	var removableHashes []string = []string{}

	log.Printf("Starting scheduled cleaning for short lived pages")
	for pageHash, page := range shortLivedPages {
		endTime := page.StartTime.Add(
			time.Duration(
				int64(time.Second) * page.TimeToLiveSeconds))
		if time.Now().After(endTime) {
			log.Printf("Removing page due timeout: %v [%s]",
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
	log.Printf("Stopping scheduled cleaning for short lived pages")
}

func InitScheduler() {
	shortLivedPages = make(map[string]ShortLivedPage, 0)

	cleanerSchedule, err := scheduler.NewScheduler(1000)
	if err != nil {
		log.Fatalf("ERROR while initializing scheduler: %s", err)
	}
	log.Printf("Cleaner scheduler started")
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
