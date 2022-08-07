package web

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"text/template"
	"weezel/budget/logger"
	"weezel/budget/shortlivedpage"
)

func LoadPage(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "Error parsing form\r\n")
		return nil
	}
	logger.Infof("Received forms: %v", r.PostForm)
	receivedPageHash := template.HTMLEscapeString(r.FormValue("page_hash"))
	if len(receivedPageHash) < 1 {
		fmt.Fprintf(w, "Error, empty message\r\n")
		return nil
	}

	page := shortlivedpage.Get(receivedPageHash)
	if reflect.DeepEqual(page, shortlivedpage.ShortLivedPage{}) {
		errMsg := fmt.Sprintf("No such hash %s\r\n", receivedPageHash)
		fmt.Fprint(w, errMsg)
		return errors.New(errMsg)
	}
	fmt.Fprintf(w, "%s\n", *page.HTMLPage)
	return nil
}

func APIHandler(w http.ResponseWriter, r *http.Request) {
	logger.Infof("Incoming %s [%v] connection from %s with size %d bytes",
		r.Method,
		r.Header,
		r.RemoteAddr,
		r.ContentLength)

	switch r.Method {
	case "GET":
		if err := LoadPage(w, r); err != nil {
			logger.Errorf("%s", err)
			return
		}
	}
}
