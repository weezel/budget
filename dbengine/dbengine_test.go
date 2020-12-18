package dbengine

import (
	"database/sql"
	"reflect"
	"testing"
	"time"
	"weezel/budget/external"

	_ "github.com/mattn/go-sqlite3"
)

func Test_getSalaryDataByUser(t *testing.T) {
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()

	UpdateDBReference(memDb)
	CreateSchema(memDb)

	_, err := memDb.Exec(`
INSERT INTO salary (username, salary, recordtime) VALUES
	('alice', 2400.0, '06-2020'),
	('tom', 666.6, '06-2020'),
	('alice', 3400.0, '07-2020');`)
	if err != nil {
		t.Fatalf("Unexpected error in SQL INSERT: %v", err)
	}

	type args struct {
		username string
		month    time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{
			"Simple result",
			args{"alice", time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC)},
			3400.0,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSalaryDataByUser(tt.args.username, tt.args.month)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: getSalaryDataByUser() error = %v, wantErr %v",
					tt.name,
					err,
					tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("%s: getSalaryDataByUser() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}
		})
	}
}

func TestGetSalaryCompensatedDebts(t *testing.T) {
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()

	UpdateDBReference(memDb)
	CreateSchema(memDb)

	_, err := memDb.Exec(`
INSERT INTO salary (username, salary, recordtime) VALUES
	('alice',  128.0, '06-2020'),
	('tom',    512.0, '06-2020'),
	('alice',  512.0, '07-2020'),
	('tom',    256.0, '07-2020'),
	('alice', 4096.0, '08-2020'),
	('tom',   3840.0, '08-2020');`)
	if err != nil {
		t.Fatalf("Unexpected error in SQL INSERT: %v", err)
	}

	_, err = memDb.Exec(`
INSERT INTO budget (username, shopname, category, purchasedate, price) VALUES
	('alice', 'lidl',   '', '06-2020',   0.0),
	('tom',   'ikea',   '', '06-2020',   8.0),
	('alice', 'lidl',   '', '07-2020',   1.0),
	('alice', 'lidl',   '', '07-2020',   2.0),
	('alice', 'lidl',   '', '07-2020',   4.0),
	('tom',   'ikea',   '', '07-2020',  16.0),
	('alice', 'amazon', '', '08-2020', 128.0),
	('alice', 'amazon', '', '08-2020',  32.0),
	('tom',   'siwa',   '', '08-2020', 256.0),
	('tom',   'siwa',   '', '08-2020', 512.0);`)
	if err != nil {
		t.Fatalf("Unexpected error in SQL INSERT: %v", err)
	}

	type args struct {
		month time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    []DebtData
		wantErr bool
	}{
		{
			"Lesser salary and purchases owes",
			args{time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "alice",
					Expanses: 0.0,
					Owes:     2.0,
					Salary:   128.0,
					Date:     "06-2020",
				},
				{
					Username: "tom",
					Expanses: 8.0,
					Owes:     0.0,
					Salary:   512.0,
					Date:     "06-2020",
				},
			},
			false,
		},
		{
			"Lesser salary but more purchases",
			args{time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "tom",
					Expanses: 16.0,
					Owes:     0.0,
					Salary:   256.0,
					Date:     "07-2020",
				},
				{
					Username: "alice",
					Expanses: 7.0,
					Owes:     4.5,
					Salary:   512.0,
					Date:     "07-2020",
				},
			},
			false,
		},
		{
			"More salary and more purchases",
			args{time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "tom",
					Expanses: 768.0,
					Owes:     0.0,
					Salary:   3840.0,
					Date:     "08-2020",
				},
				{
					Username: "alice",
					Expanses: 160.0,
					Owes:     570.0,
					Salary:   4096.0,
					Date:     "08-2020",
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSalaryCompensatedDebts(tt.args.month)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: GetSalaryCompensatedDebts() error = %+v, wantErr %+v",
					tt.name,
					err,
					tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: GetSalaryCompensatedDebts() = %+v, want %+v",
					tt.name,
					got,
					tt.want)
			}
		})
	}
}

func Test_GetMonthlySpending(t *testing.T) {
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()

	UpdateDBReference(memDb)
	CreateSchema(memDb)

	_, err := memDb.Exec(`
INSERT INTO budget (username, shopname, category, purchasedate, price) VALUES
	('alice', 'lidl',   '', '01-2020',  12.0),
	('alice', 'lidl',   '', '01-2020',   2.0),
	('tom',   'lidl',   '', '01-2020',  10.0),
	('alice', 'lidl',   '', '02-2020',   9.0),
	('tom',   'lidl',   '', '02-2020',  15.4),
	('alice', 'lidl',   '', '03-2020', 17.66),
	('alice', 'lidl',   '', '03-2020',  15.8),
	('tom',   'lidl',   '', '03-2020',   4.4),
	('alice', 'lidl',   '', '04-2020', 318.9),
	('tom',   'lidl',   '', '04-2020', 559.9),
	('alice', 'lidl',   '', '04-2020',   4.3),
	('tom',   'ikea',   '', '06-2020',   8.0),
	('alice', 'lidl',   '', '07-2020',   1.0),
	('alice', 'lidl',   '', '07-2020',   2.0),
	('alice', 'lidl',   '', '07-2020',   4.0),
	('tom',   'ikea',   '', '07-2020',  16.0),
	('alice', 'amazon', '', '08-2020', 128.0),
	('alice', 'amazon', '', '08-2020',  32.0),
	('tom',   'siwa',   '', '08-2021', 256.0),
	('tom',   'siwa',   '', '08-2021', 512.0) ;`)
	if err != nil {
		t.Fatalf("Unexpected error in SQL INSERT: %v", err)
	}

	tests := []struct {
		name    string
		want    []external.SpendingHistory
		wantErr bool
	}{
		{
			"Get annual spending",
			[]external.SpendingHistory{
				{
					Username:  "alice",
					MonthYear: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					Spending:  14.0,
				},
				{
					Username:  "tom",
					MonthYear: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					Spending:  10.0,
				},
				{
					Username:  "alice",
					MonthYear: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
					Spending:  9.0,
				},
				{
					Username:  "tom",
					MonthYear: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
					Spending:  15.4,
				},
				{
					Username:  "alice",
					MonthYear: time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC),
					Spending:  33.46,
				},
				{
					Username:  "tom",
					MonthYear: time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC),
					Spending:  4.4,
				},
				{
					Username:  "alice",
					MonthYear: time.Date(2020, 4, 1, 0, 0, 0, 0, time.UTC),
					Spending:  323.2,
				},
				{
					Username:  "tom",
					MonthYear: time.Date(2020, 4, 1, 0, 0, 0, 0, time.UTC),
					Spending:  559.9,
				},
				{
					Username:  "tom",
					MonthYear: time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
					Spending:  8.0,
				},
				{
					Username:  "alice",
					MonthYear: time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
					Spending:  7.0,
				},
				{
					Username:  "tom",
					MonthYear: time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
					Spending:  16.0,
				},
				{
					Username:  "alice",
					MonthYear: time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
					Spending:  160.0,
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetMonthlySpending()
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: getSpendingNumbers() error = %+v, wantErr %+v",
					tt.name,
					err,
					tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: getSpendingNumbers() = %+v, want %+v",
					tt.name,
					got,
					tt.want)
			}

			// var b []byte = []byte{}
			// if b, err = plotters.LineHistogramOfAnnualSpending(got); err != nil {
			// 	t.Errorf("AIJAI: %s", err)
			// }
			// ioutil.WriteFile("testikuva.png", b, 0600)
		})
	}
}

func TestGetSalariesByMonth(t *testing.T) {
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()

	UpdateDBReference(memDb)
	CreateSchema(memDb)

	_, err := memDb.Exec(`
INSERT INTO salary (username, salary, recordtime) VALUES
	('alice',    8.0,  '01-2020'),
	('tom',      4.0,  '01-2020'),
	('alice',  128.0,  '06-2020'),
	('alice',  512.0,  '07-2020'),
	('tom',    256.0,  '07-2020'),
	('alice',    0.0,  '08-2020'),
	('tom',    3840.0, '08-2020');`)
	if err != nil {
		t.Fatalf("Unexpected error in SQL INSERT: %v", err)
	}
	type args struct {
		startMonth time.Time
		endMonth   time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    []DebtData
		wantErr bool
	}{
		{
			"Get salaries on February",
			args{time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{},
			false,
		},
		{
			"Get salaries on June",
			args{time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "alice",
					Salary:   1.0,
					Date:     "06-2020",
				},
			},
			false,
		},
		{
			"Get salaries on July",
			args{time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "alice",
					Salary:   1.0,
					Date:     "07-2020",
				},
				{
					Username: "tom",
					Salary:   1.0,
					Date:     "07-2020",
				},
			},
			false,
		},
		{
			"Get salaries on August",
			args{time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "alice",
					Salary:   0.0,
					Date:     "08-2020",
				},
				{
					Username: "tom",
					Salary:   1.0,
					Date:     "08-2020",
				},
			},
			false,
		},
		{
			"Get half year salaries from Jan to Jun",
			args{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "alice",
					Salary:   1.0,
					Date:     "01-2020",
				},
				{
					Username: "alice",
					Salary:   1.0,
					Date:     "06-2020",
				},
				{
					Username: "tom",
					Salary:   1.0,
					Date:     "01-2020",
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSalariesByMonthRange(tt.args.startMonth, tt.args.endMonth)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: GetSalariesByMonth() error = %v, wantErr %v",
					tt.name,
					err,
					tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: GetSalariesByMonth() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}
		})
	}
}

func TestGetMonthlyPurchasesByUser(t *testing.T) {
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()

	UpdateDBReference(memDb)
	CreateSchema(memDb)

	_, err := memDb.Exec(`
	INSERT INTO budget (username, shopname, category, purchasedate, price) VALUES
	('alice', 'lidl',   '', '01-2020',  12.0),
	('alice', 'lidl',   '', '01-2020',   2.0),
	('tom',   'lidl',   '', '01-2020',  10.0),
	('alice', 'lidl',   '', '02-2020',   9.0),
	('tom',   'lidl',   '', '02-2020',  15.4),
	('alice', 'lidl',   '', '03-2020', 17.66),
	('alice', 'lidl',   '', '03-2020',  15.8),
	('tom',   'lidl',   '', '03-2020',   4.4),
	('alice', 'lidl',   '', '04-2020', 318.9),
	('tom',   'lidl',   '', '04-2020', 559.9),
	('alice', 'lidl',   '', '04-2020',   4.3),
	('tom',   'ikea',   '', '06-2020',   8.0),
	('alice', 'lidl',   '', '07-2020',   1.0),
	('alice', 'lidl',   '', '07-2020',   2.0),
	('alice', 'lidl',   '', '07-2020',   4.0),
	('tom',   'ikea',   '', '07-2020',  16.0),
	('alice', 'amazon', '', '08-2020', 128.0),
	('alice', 'amazon', '', '08-2020',  32.0),
	('tom',   'siwa',   '', '08-2021', 256.0),
	('tom',   'siwa',   '', '08-2021', 512.0) ;`)
	if err != nil {
		t.Fatalf("Unexpected error in SQL INSERT: %v", err)
	}

	type args struct {
		username string
		month    time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    []external.SpendingHistory
		wantErr bool
	}{
		{
			"Ding dong",
			args{"alice", time.Date(2020, 7, 1, 1, 0, 0, 0, time.UTC)},
			[]external.SpendingHistory{
				{
					MonthYear: time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
					EventName: "lidl",
					Spending:  1.0,
				},
				{
					MonthYear: time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
					EventName: "lidl",
					Spending:  2.0,
				},
				{
					MonthYear: time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
					EventName: "lidl",
					Spending:  4.0,
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetMonthlyPurchasesByUser(tt.args.username, tt.args.month)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: GetMonthlyPurchasesByUser() error = %v, wantErr %v",
					tt.name,
					err,
					tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: GetMonthlyPurchasesByUser() = %v, want %v",
					tt.name,
					got,
					tt.want)
			}
		})
	}
}
