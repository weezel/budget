package outputs

import (
	"bytes"
	"embed"
	"html/template"
	"weezel/budget/dbengine"
)

//go:embed stats.gohtml
var dataTemplateFS embed.FS

//go:embed expenses.gohtml
var expensesTemplateFS embed.FS

func RenderStatsHTML(templateVars dbengine.StatisticsVars) ([]byte, error) {
	filename := "stats.gohtml"

	tpl, err := template.ParseFS(dataTemplateFS, filename)
	if err != nil {
		return []byte{}, err
	}

	buf := bytes.Buffer{}
	if err = tpl.ExecuteTemplate(&buf, filename, templateVars); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

func RenderExpensesHTML(templateVars dbengine.ExpensesVars) ([]byte, error) {
	filename := "expenses.html"

	tpl, err := template.ParseFS(expensesTemplateFS, filename)
	if err != nil {
		return []byte{}, err
	}

	buf := bytes.Buffer{}
	if err = tpl.ExecuteTemplate(&buf, filename, templateVars); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}
