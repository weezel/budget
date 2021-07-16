package web

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"text/template"
	"weezel/budget/shortlivedpage"
)

func LoadPage(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form\n")
		return nil
	}
	log.Printf("Received forms: %v\n", r.PostForm)
	receivedPageHash := template.HTMLEscapeString(r.FormValue("page_hash"))
	if len(receivedPageHash) < 1 {
		fmt.Fprintf(w, "Error, empty message\r\n")
		return nil
	}

	page := shortlivedpage.Get(receivedPageHash)
	if reflect.DeepEqual(page, shortlivedpage.ShortLivedPage{}) {
		errMsg := fmt.Sprintf("No such hash %s", receivedPageHash)
		fmt.Fprintf(w, errMsg)
		return errors.New(errMsg)
	}
	fmt.Fprintf(w, "%s\n", *page.HtmlPage)
	return nil
}

func ApiHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Incoming %s [%v] connection from %s with size %d bytes",
		r.Method,
		r.Header,
		r.RemoteAddr,
		r.ContentLength)

	switch r.Method {
	case "GET":
		if err := LoadPage(w, r); err != nil {
			log.Printf("ERROR: %s", err)
			return
		}
	}
}
