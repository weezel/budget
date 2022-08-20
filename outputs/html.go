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

type StatisticsVars struct {
	From       time.Time
	To         time.Time
	Statistics []*db.StatisticsAggrByTimespanRow
	Detailed   []*db.GetExpensesByTimespanRow
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
