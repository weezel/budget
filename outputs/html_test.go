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
					From: time.Date(2020, 10, 1, 1, 0, 0, 0, time.UTC),
					To:   time.Date(2020, 12, 1, 1, 0, 0, 0, time.UTC),
					Spendings: map[time.Time][]external.SpendingHistory{
						time.Date(2020, 10, 1, 0, 0, 0, 0, time.UTC): {
							{
								ID:        0,
								Username:  "Dille",
								MonthYear: time.Date(2020, 10, 1, 1, 0, 0, 0, time.UTC),
								Spending:  10,
								EventName: "beer",
							},
							{
								ID:        1,
								Username:  "Dille",
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
						},
						time.Date(2020, 11, 1, 0, 0, 0, 0, time.UTC): {
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
    <h3>Alkaen 01-10-2020 ja 01-12-2020 asti</h3>
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
        <td>0</td>
        <td>01-10-2020</td>
        <td>10</td>
        <td>beer</td>
    </tr>
    <tr>
        <td>1</td>
        <td>01-10-2020</td>
        <td>20.5</td>
        <td>pad thai</td>
    </tr>
    <tr>
        <td>2</td>
        <td>01-10-2020</td>
        <td>850.99</td>
        <td>shoes</td>
    </tr>
    <tr>
        <td>3</td>
        <td>01-11-2020</td>
        <td>444.4</td>
        <td>moar beer</td>
    </tr>
    <tr>
        <td>4</td>
        <td>01-11-2020</td>
        <td>555.5</td>
        <td>cat food</td>
    </tr>
    <tr>
        <td>5</td>
        <td>01-11-2020</td>
        <td>666.6</td>
        <td>dog food</td>
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
			got, err := HTML(tt.args.spending)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: HTML() error = %v, wantErr %v",
					tt.name, err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("%s: HTML() diff = %s", tt.name, diff)
			}
		})
	}
}
