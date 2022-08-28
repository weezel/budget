package debtcontrol

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"testing"
	"weezel/budget/db"

	"github.com/google/go-cmp/cmp"
)

const (
	floatDelta = float64(1e-4)
)

func TestGetSalaryCompensatedDebts(t *testing.T) {
	type args struct {
		ctx   context.Context
		user1 *db.StatisticsAggrByTimespanRow
		user2 *db.StatisticsAggrByTimespanRow
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		updatedArgs map[string]float64 // Args get updated after the fn call
	}{
		{
			name: "Person with smaller salary has no purchases",
			args: args{
				ctx: nil,
				user1: &db.StatisticsAggrByTimespanRow{
					Username:    "alice",
					ExpensesSum: 0,
					Salary:      900.0,
					Owes:        0,
				},
				user2: &db.StatisticsAggrByTimespanRow{
					Username:    "tom",
					ExpensesSum: 80.0,
					Salary:      1000.0,
					Owes:        0,
				},
			},
			wantErr: false,
			updatedArgs: map[string]float64{
				"alice": 37.8947,
				"tom":   0.0,
			},
		},
		{
			name: "Person with greater salary has no purchases",
			args: args{
				ctx: nil,
				user1: &db.StatisticsAggrByTimespanRow{
					Username:    "alice",
					ExpensesSum: 80.0,
					Salary:      900.0,
					Owes:        0,
				},
				user2: &db.StatisticsAggrByTimespanRow{
					Username:    "tom",
					ExpensesSum: 0,
					Salary:      1000.0,
					Owes:        0,
				},
			},
			wantErr: false,
			updatedArgs: map[string]float64{
				"alice": 0.0,
				"tom":   42.1052,
			},
		},
		{
			name: "Person with smaller salary has more purchases",
			args: args{
				ctx: nil,
				user1: &db.StatisticsAggrByTimespanRow{
					Username:    "alice",
					ExpensesSum: 40.0,
					Salary:      1000.0,
					Owes:        0,
				},
				user2: &db.StatisticsAggrByTimespanRow{
					Username:    "tom",
					ExpensesSum: 60.0,
					Salary:      900,
					Owes:        0,
				},
			},
			wantErr: false,
			updatedArgs: map[string]float64{
				"alice": 12.6315,
				"tom":   0.0,
			},
		},
		{
			name: "Person with greater salary has more purchases",
			args: args{
				ctx: nil,
				user1: &db.StatisticsAggrByTimespanRow{
					Username:    "alice",
					ExpensesSum: 100.0,
					Salary:      1000.0,
					Owes:        0,
				},
				user2: &db.StatisticsAggrByTimespanRow{
					Username:    "tom",
					ExpensesSum: 20.0,
					Salary:      700,
					Owes:        0,
				},
			},
			wantErr: false,
			updatedArgs: map[string]float64{
				"alice": 0.0,
				"tom":   29.4117,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GetSalaryCompensatedDebts(tt.args.ctx, tt.args.user1, tt.args.user2)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSalaryCompensatedDebts() error = %v, wantErr %v", err, tt.wantErr)
			}

			if math.Abs(tt.updatedArgs[tt.args.user1.Username]-tt.args.user1.Owes) > floatDelta {
				t.Errorf("%s: %s: %.4f != %.4f",
					tt.name,
					tt.args.user1.Username,
					tt.updatedArgs[tt.args.user1.Username],
					tt.args.user1.Owes)
			}

			if math.Abs(tt.updatedArgs[tt.args.user2.Username]-tt.args.user2.Owes) > floatDelta {
				t.Errorf("%s: %s: %.4f != %.4f",
					tt.name,
					tt.args.user2.Username,
					tt.updatedArgs[tt.args.user2.Username],
					tt.args.user2.Owes)
			}
		})
	}
}

func TestFillDebts(t *testing.T) {
	// Stats data contains spending aggregations per month but no debts.
	data, err := os.ReadFile("stats_data.json")
	if err != nil {
		t.Fatal(err)
	}

	expectedData, err := os.ReadFile("stats_data_expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected []*db.StatisticsAggrByTimespanRow
	if err = json.Unmarshal(expectedData, &expected); err != nil {
		t.Fatal(err)
	}

	var exampleStats []*db.StatisticsAggrByTimespanRow
	if err := json.Unmarshal(data, &exampleStats); err != nil {
		t.Fatal(err)
	}

	FillDebts(context.Background(), exampleStats)
	if diff := cmp.Diff(expected, exampleStats); diff != "" {
		t.Errorf("%s: differs:\n%s\n", t.Name(), diff)
	}
}
