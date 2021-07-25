package outputs

import (
	"bytes"
	"embed"
	"html/template"
	"weezel/budget/external"
)

type TemplateType int

const (
	MontlySpendingsTemplate TemplateType = iota
	MonthlyDataTemplate     TemplateType = iota
)

//go:embed monthlydata.gohtml
var dataTemplateFS embed.FS

//go:embed monthlyspendings.gohtml
var spendingsTemplateFS embed.FS

func HTML(spending external.SpendingHTMLOutput, templateType TemplateType) ([]byte, error) {
	var tpl *template.Template
	var err error
	var filename string
	var buf bytes.Buffer = bytes.Buffer{}

	switch templateType {
	case MonthlyDataTemplate:
		filename = "monthlydata.gohtml"
		tpl, err = template.ParseFS(dataTemplateFS, filename)
	case MontlySpendingsTemplate:
		filename = "monthlyspendings.gohtml"
		tpl, err = template.ParseFS(spendingsTemplateFS, filename)
	}
	if err != nil {
		return []byte{}, err
	}

	if err = tpl.ExecuteTemplate(&buf, filename, spending); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}
