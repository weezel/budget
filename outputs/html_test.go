package outputs

import (
	"testing"
	"time"
	"weezel/budget/external"

	"github.com/google/go-cmp/cmp"
)

func TestHTML(t *testing.T) {
	type args struct {
		spending external.SpendingHTMLOutput
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"Test output",
			args{
				external.SpendingHTMLOutput{
					From: time.Date(2020, 11, 1, 1, 0, 0, 0, time.UTC),
					To:   time.Date(2020, 12, 1, 1, 0, 0, 0, time.UTC),
					Spendings: []external.SpendingHistory{
						{
							Username:  "Dille",
							MonthYear: time.Date(2020, 11, 1, 1, 0, 0, 0, time.UTC),
							Spending:  10,
							EventName: "beer",
						},
						{
							Username:  "Dille",
							MonthYear: time.Date(2020, 11, 1, 1, 0, 0, 0, time.UTC),
							Spending:  20.5,
							EventName: "pad thai",
						},
						{
							Username:  "Dille",
							MonthYear: time.Date(2020, 11, 1, 1, 0, 0, 0, time.UTC),
							Spending:  850.99,
							EventName: "shoes",
						},
					},
				},
			},
			[]byte(
				`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8" />
<title>Kulutukset</title>
</head>
<body>
    <h1>Käyttäjän Dille kulutukset</h1>
    <h3>Alkaen 11-01-2020 ja 12-01-2020 asti</h3>
    <table width=600px>
    <col style="width:150px">
	<col style="width:100px">
	<col style="width:350px">
    <thead>
    <tr>
        <th style="text-align:left">Aika</th>
        <th style="text-align:left">Määrä</th>
        <th style="text-align:left">Kuvaus</th>
    </tr>
    </thead>

    <tbody>
    
    <tr>
        <td>11-01-2020</td>
        <td>10</td>
        <td>beer</td>
    </tr>
    
    <tr>
        <td>11-01-2020</td>
        <td>20.5</td>
        <td>pad thai</td>
    </tr>
    
    <tr>
        <td>11-01-2020</td>
        <td>850.99</td>
        <td>shoes</td>
    </tr>
    
    </tbody>

    </table>
</body>
</html>`),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HTML(tt.args.spending)
			if (err != nil) != tt.wantErr {
				t.Errorf("HTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("HTML() diff = %s", diff)
			}
		})
	}
}
