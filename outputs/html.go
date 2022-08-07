package outputs

import (
	"bytes"
	"database/sql"
	"embed"
	"html/template"
	"time"
	"weezel/budget/db"
)

//go:embed stats.gohtml
var dataTemplateFS embed.FS

//go:embed expenses.gohtml
var expensesTemplateFS embed.FS

type ExpensesVars struct {
	From       time.Time
	To         time.Time
	Aggregated []*db.GetAggrExpensesByTimespanRow
	Detailed   []*db.GetExpensesByTimespanRow
}

type StatisticsVars struct {
	From       time.Time
	To         time.Time
	Statistics []*db.StatisticsAggrByTimespanRow
}

func FormatNullFloat(f sql.NullFloat64) float64 {
	if f.Valid {
		return f.Float64
	}
	return 0.0
}

func RenderStatsHTML(templateVars StatisticsVars) ([]byte, error) {
	filename := "stats.gohtml"

	tpl, err := template.New(filename).Funcs(template.FuncMap{
		"FormatNullFloat": FormatNullFloat,
	}).ParseFS(dataTemplateFS, filename)
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	if err = tpl.ExecuteTemplate(&buf, filename, templateVars); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func RenderExpensesHTML(templateVars ExpensesVars) ([]byte, error) {
	filename := "expenses.gohtml"

	tpl, err := template.New(filename).Funcs(template.FuncMap{
		"FormatNullFloat": FormatNullFloat,
	}).ParseFS(expensesTemplateFS, filename)
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	if err = tpl.ExecuteTemplate(&buf, filename, templateVars); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
