package outputs

import (
	"testing"
	"time"
	"weezel/budget/external"

	"github.com/google/go-cmp/cmp"
)

func TestHTML(t *testing.T) {
	type args struct {
		spending     external.SpendingHTMLOutput
		templateType TemplateType
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"Test monthly spendings output",
			args{
				spending: external.SpendingHTMLOutput{
					From: time.Date(2020, 10, 1, 1, 0, 0, 0, time.UTC),
					To:   time.Date(2020, 12, 1, 1, 0, 0, 0, time.UTC),
					Spendings: []external.SpendingHistory{
						{
							ID:        0,
							Username:  "Dille",
							MonthYear: time.Date(2020, 10, 1, 1, 0, 0, 0, time.UTC),
							Spending:  10,
							EventName: "beer",
						},
						{
							ID:        1,
							Username:  "Rolle",
							MonthYear: time.Date(2020, 10, 1, 1, 0, 0, 0, time.UTC),
							Spending:  20.5,
							EventName: "pad thai",
						},
						{
							ID:        2,
							Username:  "Dille",
							MonthYear: time.Date(2020, 10, 1, 1, 0, 0, 0, time.UTC),
							Spending:  850.99,
							EventName: "shoes",
						},
						{
							ID:        3,
							Username:  "Dille",
							MonthYear: time.Date(2020, 11, 1, 1, 0, 0, 0, time.UTC),
							Spending:  444.4,
							EventName: "moar beer",
						},
						{
							ID:        4,
							Username:  "Dille",
							MonthYear: time.Date(2020, 11, 1, 1, 0, 0, 0, time.UTC),
							Spending:  555.5,
							EventName: "cat food",
						},
						{
							ID:        5,
							Username:  "Dille",
							MonthYear: time.Date(2020, 11, 1, 1, 0, 0, 0, time.UTC),
							Spending:  666.6,
							EventName: "dog food",
						},
					},
				},
				templateType: MontlySpendingsTemplate,
			},
			[]byte(
				`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8" />
<title>Kuukausittaiset kulutukset</title>
</head>
<body>
    <h3>Kulutukset alkaen 01-10-2020 ja 01-12-2020 asti</h3>
    <table width=650px>
    <col style="width:30px">
    <col style="width:120px">
    <col style="width:120px">
    <col style="width:60px">
    <col style="width:150px">
    <thead>
    <tr>
        <th style="text-align:right">ID</th>
        <th style="text-align:center">Käyttäjä</th>
        <th style="text-align:center">Aika</th>
        <th style="text-align:left">Määrä</th>
        <th style="text-align:left">Kuvaus</th>
    </tr>
    </thead>

    <tbody>
    <tr>
        <td style="text-align:right">0</td>
        <td style="text-align:center">Dille</td>
        <td style="text-align:center">01-10-2020</td>
        <td style="text-align:left">10</td>
        <td style="text-align:left">beer</td>
    </tr>
    <tr>
        <td style="text-align:right">1</td>
        <td style="text-align:center">Rolle</td>
        <td style="text-align:center">01-10-2020</td>
        <td style="text-align:left">20.5</td>
        <td style="text-align:left">pad thai</td>
    </tr>
    <tr>
        <td style="text-align:right">2</td>
        <td style="text-align:center">Dille</td>
        <td style="text-align:center">01-10-2020</td>
        <td style="text-align:left">850.99</td>
        <td style="text-align:left">shoes</td>
    </tr>
    <tr>
        <td style="text-align:right">3</td>
        <td style="text-align:center">Dille</td>
        <td style="text-align:center">01-11-2020</td>
        <td style="text-align:left">444.4</td>
        <td style="text-align:left">moar beer</td>
    </tr>
    <tr>
        <td style="text-align:right">4</td>
        <td style="text-align:center">Dille</td>
        <td style="text-align:center">01-11-2020</td>
        <td style="text-align:left">555.5</td>
        <td style="text-align:left">cat food</td>
    </tr>
    <tr>
        <td style="text-align:right">5</td>
        <td style="text-align:center">Dille</td>
        <td style="text-align:center">01-11-2020</td>
        <td style="text-align:left">666.6</td>
        <td style="text-align:left">dog food</td>
    </tr>
    </tbody>

    </table>
</body>
</html>
`),
			false,
		},
		{
			"Test monthly data output",
			args{
				spending: external.SpendingHTMLOutput{
					From: time.Date(2020, 10, 1, 1, 0, 0, 0, time.UTC),
					To:   time.Date(2020, 12, 1, 1, 0, 0, 0, time.UTC),
					Spendings: []external.SpendingHistory{
						{
							ID:        0,
							Username:  "Dille",
							MonthYear: time.Date(2020, 10, 1, 1, 0, 0, 0, time.UTC),
							Spending:  10,
							Salary:    2000,
						},
						{
							ID:        1,
							Username:  "John",
							MonthYear: time.Date(2020, 10, 1, 1, 0, 0, 0, time.UTC),
							Spending:  5,
							Salary:    4000,
						},
						{
							ID:        2,
							Username:  "Dille",
							MonthYear: time.Date(2020, 11, 1, 1, 0, 0, 0, time.UTC),
							Spending:  2.22,
							Salary:    555.5,
						},
						{
							ID:        3,
							Username:  "John",
							MonthYear: time.Date(2020, 11, 1, 1, 0, 0, 0, time.UTC),
							Spending:  1.11,
							Salary:    7.77,
						},
					},
				},
				templateType: MonthlyDataTemplate,
			},
			[]byte(
				`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8" />
<title>Kuukausittainen datapläjäys</title>
</head>
<body>
    <h1>Tietoja</h1>
    <h3>Alkaen 01-10-2020 ja 01-12-2020 asti</h3>
    <table width=600px>
    <col style="width:150px">
    <col style="width:100px">
    <col style="width:100px">
    <col style="width:100px">
    <thead>
    <tr>
        <th style="text-align:left">Käyttäjä</th>
        <th style="text-align:center">Aika</th>
        <th style="text-align:right">Kulut yhteensä</th>
        <th style="text-align:right">Palkka</th>
    </tr>
    </thead>

    <tbody>
    <tr>
        <td style="text-align:left">Dille</td>
        <td style="text-align:center">01-10-2020</td>
        <td style="text-align:right">10</td>
        <td style="text-align:right">2000</td>
    </tr>
    <tr>
        <td style="text-align:left">John</td>
        <td style="text-align:center">01-10-2020</td>
        <td style="text-align:right">5</td>
        <td style="text-align:right">4000</td>
    </tr>
    <tr>
        <td style="text-align:left">Dille</td>
        <td style="text-align:center">01-11-2020</td>
        <td style="text-align:right">2.22</td>
        <td style="text-align:right">555.5</td>
    </tr>
    <tr>
        <td style="text-align:left">John</td>
        <td style="text-align:center">01-11-2020</td>
        <td style="text-align:right">1.11</td>
        <td style="text-align:right">7.77</td>
    </tr>
    </tbody>

    </table>
</body>
</html>
`),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HTML(tt.args.spending, tt.args.templateType)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: HTML() error = %v, wantErr %v",
					tt.name, err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("%s: HTML() diff:\n%s", tt.name, diff)
			}
		})
	}
}
