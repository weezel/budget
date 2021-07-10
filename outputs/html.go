package outputs

import (
	"bytes"
	"embed"
	"html/template"
	"log"
	"weezel/budget/external"
)

//go:embed spendings.gohtml
var htmlTemplateFS embed.FS

func HTML(spending external.SpendingHTMLOutput) ([]byte, error) {
	var tpl *template.Template
	var buf bytes.Buffer = bytes.Buffer{}

	tpl, err := template.ParseFS(htmlTemplateFS, "spendings.gohtml")
	if err != nil {
		log.Printf("Couldn't parse spendings.gohtml %s\n", err)
	}

	if err = tpl.ExecuteTemplate(&buf, "spendings.gohtml", spending); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}
