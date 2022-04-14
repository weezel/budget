package dbengine

import (
	"database/sql"
	"math"
	"reflect"
	"testing"
	"time"
	"weezel/budget/external"

	"github.com/google/go-cmp/cmp"
	_ "github.com/mattn/go-sqlite3"
)

const (
	floatDelta float64 = 1e-2
)

func Test_getSalaryDataByUser(t *testing.T) {
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()

	UpdateDBReference(memDb)
	CreateSchema(memDb)

	_, err := memDb.Exec(`
INSERT INTO salary (username, salary, recordtime) VALUES
	('alice', 2400.0, '2020-06-01'),
	('tom',    666.6, '2020-06-01'),
	('alice', 3400.0, '2020-07-01');`)
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
	('alice',  128.0, '2020-06-01'),
	('tom',    512.0, '2020-06-01'),
	('alice', 1000.0, '2020-07-01'),
	('tom',    900.0, '2020-07-01'),
	('alice', 1000.0, '2020-08-01'),
	('tom',    700.0, '2020-08-01'),
	('alice',  900.0, '2021-01-01'),
	('tom',   1000.0, '2021-01-01'),
	('alice',  900.0, '2021-02-01'),
	('tom',   1000.0, '2021-02-01');`)
	if err != nil {
		t.Fatalf("Unexpected error in SQL INSERT: %v", err)
	}

	_, err = memDb.Exec(`
INSERT INTO budget (username, shopname, category, purchasedate, price) VALUES
	('alice', 'empty',     '', '2020-06-01',    0.0),
	('tom',   'ikea',      '', '2020-06-01',    8.0),
	('alice', 'stuff1',    '', '2020-07-01',   20.0),
	('alice', 'stuff2',    '', '2020-07-01',   20.0),
	('tom',   'tar',       '', '2020-07-01',   20.0),
	('tom',   'jar',       '', '2020-07-01',   20.0),
	('tom',   'feathers',  '', '2020-07-01',   20.0),
	('alice', 'a',         '', '2020-08-01',   50.0),
	('alice', 'b',         '', '2020-08-01',   50.0),
	('tom',   'muchos',    '', '2020-08-01',   10.0),
	('tom',   'grander',   '', '2020-08-01',   10.0),
	('alice', 'empty',     '', '2021-01-01',    0.0),
	('tom',   'stuff',     '', '2021-01-01',   80.0),
	('alice', 'stuff',     '', '2021-02-01',   80.0),
	('tom',   'empty',     '', '2021-02-01',    0.0);`)
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
			"Person with smaller salary has no purchases",
			args{time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "alice",
					Expanses: 0.0,
					Owes:     37.894737,
					Salary:   900.0,
					Date:     time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Username: "tom",
					Expanses: 80.0,
					Owes:     0.0,
					Salary:   1000.0,
					Date:     time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			false,
		},
		{
			"Person with greater salary has no purchases",
			args{time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "alice",
					Expanses: 80.0,
					Owes:     0.0,
					Salary:   900.0,
					Date:     time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Username: "tom",
					Expanses: 0.0,
					Owes:     42.105263,
					Salary:   1000.0,
					Date:     time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			false,
		},
		{
			"Person with smaller salary has more purchases",
			args{time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "tom",
					Expanses: 60.0,
					Owes:     0.0,
					Salary:   900.0,
					Date:     time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Username: "alice",
					Expanses: 40.0,
					Owes:     12.631579,
					Salary:   1000.0,
					Date:     time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			false,
		},
		{
			"Person with greater salary has more purchases",
			args{time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "tom",
					Expanses: 20.0,
					Owes:     29.411765,
					Salary:   700.0,
					Date:     time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Username: "alice",
					Expanses: 100.0,
					Owes:     0.0,
					Salary:   1000.0,
					Date:     time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
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

			for idx := range got {
				if got[idx].Username != tt.want[idx].Username {
					t.Errorf("%s: Username[%d]: got=%s, expected=%s",
						tt.name,
						idx,
						got[idx].Username,
						tt.want[idx].Username)
				}

				if !got[idx].Date.Equal(tt.want[idx].Date) {
					t.Errorf("%s: Date[%d]: got=%s, expected=%s",
						tt.name,
						idx,
						got[idx].Date,
						tt.want[idx].Date)
				}

				if math.Abs(got[idx].Expanses-tt.want[idx].Expanses) > floatDelta {
					t.Errorf("%s: Expanses[%d]: got=%f, expected=%f",
						tt.name,
						idx,
						got[idx].Expanses,
						tt.want[idx].Expanses)
				}

				if math.Abs(got[idx].Salary-tt.want[idx].Salary) > floatDelta {
					t.Errorf("%s: Salary[%d]: got=%f, expected=%f",
						tt.name,
						idx,
						got[idx].Salary,
						tt.want[idx].Salary)
				}

				if math.Abs(got[idx].Owes-tt.want[idx].Owes) > floatDelta {
					t.Errorf("%s: Owes[%d]: got=%f, expected=%f",
						tt.name,
						idx,
						got[idx].Owes,
						tt.want[idx].Owes)
				}
			}
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
	('alice',    8.0,  '2020-01-01'),
	('tom',      4.0,  '2020-01-01'),
	('alice',  128.0,  '2020-06-01'),
	('alice',  512.0,  '2020-07-01'),
	('tom',    256.0,  '2020-07-01'),
	('alice',    0.0,  '2020-08-01'),
	('tom',    3840.0, '2020-08-01');`)
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
			"Get February salaries",
			args{time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{},
			false,
		},
		{
			"Get June salaries",
			args{time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "alice",
					Salary:   1.0,
					Date:     time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			false,
		},
		{
			"Get July salaries",
			args{time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "alice",
					Salary:   1.0,
					Date:     time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Username: "tom",
					Salary:   1.0,
					Date:     time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			false,
		},
		{
			"Get August salaries",
			args{time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "alice",
					Salary:   0.0,
					Date:     time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Username: "tom",
					Salary:   1.0,
					Date:     time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			false,
		},
		{
			"Get half year salaries from Jan to Jun",
			args{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC)},
			[]DebtData{
				{
					Username: "alice",
					Salary:   1.0,
					Date:     time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Username: "alice",
					Salary:   1.0,
					Date:     time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Username: "tom",
					Salary:   1.0,
					Date:     time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
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

func TestGetMonthlyPurchases(t *testing.T) {
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()

	UpdateDBReference(memDb)
	CreateSchema(memDb)

	_, err := memDb.Exec(`
	INSERT INTO budget (username, shopname, category, purchasedate, price) VALUES
	('alice', 'lidl',   '', '2020-01-01',  12.0),
	('alice', 'lidl',   '', '2020-01-01',   2.0),
	('tom',   'lidl',   '', '2020-01-01',  10.0),
	('alice', 'lidl',   '', '2020-02-01',   9.0),
	('tom',   'lidl',   '', '2020-02-01',  15.4),
	('alice', 'lidl',   '', '2020-03-01', 17.66),
	('alice', 'lidl',   '', '2020-03-01',  15.8),
	('tom',   'lidl',   '', '2020-03-01',   4.4),
	('alice', 'lidl',   '', '2020-04-01', 318.9),
	('tom',   'lidl',   '', '2020-04-01', 559.9),
	('alice', 'lidl',   '', '2020-04-01',   4.3),
	('tom',   'ikea',   '', '2020-06-01',   8.0),
	('alice', 'lidl',   '', '2020-07-01',   1.0),
	('alice', 'lidl',   '', '2020-07-01',   2.0),
	('alice', 'lidl',   '', '2020-07-01',   4.0),
	('tom',   'ikea',   '', '2020-07-01',  16.0),
	('alice', 'amazon', '', '2020-08-01', 128.0),
	('alice', 'amazon', '', '2020-08-01',  32.0),
	('tom',   'siwa',   '', '2021-08-01', 256.0),
	('tom',   'siwa',   '', '2021-08-01', 512.0);`)
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
		want    []external.SpendingHistory
		wantErr bool
	}{
		{
			"One month",
			args{
				startMonth: time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
				endMonth:   time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
			},
			[]external.SpendingHistory{
				{
					ID:        13,
					Username:  "alice",
					MonthYear: time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
					EventName: "lidl",
					Spending:  1.0,
				},
				{
					ID:        14,
					Username:  "alice",
					MonthYear: time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
					EventName: "lidl",
					Spending:  2.0,
				},
				{
					ID:        15,
					Username:  "alice",
					MonthYear: time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
					EventName: "lidl",
					Spending:  4.0,
				},
				{
					ID:        16,
					Username:  "tom",
					MonthYear: time.Date(2020, 7, 1, 0, 0, 0, 0, time.UTC),
					EventName: "ikea",
					Spending:  16.0,
				},
			},
			false,
		},
		{
			"Two months",
			args{
				startMonth: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				endMonth:   time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
			},
			[]external.SpendingHistory{
				{
					ID:        2,
					Username:  "alice",
					MonthYear: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					EventName: "lidl",
					Spending:  2.0,
				},
				{
					ID:        1,
					Username:  "alice",
					MonthYear: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					EventName: "lidl",
					Spending:  12.0,
				},
				{
					MonthYear: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					Username:  "tom",
					EventName: "lidl",
					Spending:  10,
					ID:        3,
				},
				{
					ID:        4,
					Username:  "alice",
					MonthYear: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
					EventName: "lidl",
					Spending:  9.0,
				},
				{
					MonthYear: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
					Username:  "tom",
					EventName: "lidl",
					Spending:  15.4,
					ID:        5,
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetMonthlyPurchases(tt.args.startMonth, tt.args.endMonth)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: GetMonthlyPurchasesByUser() error = %v, wantErr %v",
					tt.name,
					err,
					tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("%s: GetMonthlyPurchasesByUser() mismatch:\n%s",
					tt.name,
					diff)
			}
		})
	}
}

func TestGetMonthlyData(t *testing.T) {
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()

	UpdateDBReference(memDb)
	CreateSchema(memDb)

	_, err := memDb.Exec(`
	INSERT INTO budget (username, shopname, category, purchasedate, price) VALUES
	('alice', 'lidl',   '', '2020-01-01',  12.0),
	('alice', 'lidl',   '', '2020-01-01',   2.0),
	('tom',   'lidl',   '', '2020-01-01',  10.0),
	('alice', 'lidl',   '', '2020-02-01',   9.0),
	('tom',   'lidl',   '', '2020-02-01',  15.4),
	('alice', 'lidl',   '', '2020-03-01', 17.66),
	('alice', 'lidl',   '', '2020-03-01',  15.8),
	('tom',   'lidl',   '', '2020-03-01',   4.4),
	('alice', 'lidl',   '', '2020-04-01', 318.9),
	('tom',   'lidl',   '', '2020-04-01', 559.9),
	('alice', 'lidl',   '', '2020-04-01',   4.3),
	('tom',   'ikea',   '', '2020-06-01',   8.0),
	('alice', 'lidl',   '', '2020-07-01',   1.0),
	('alice', 'lidl',   '', '2020-07-01',   2.0),
	('alice', 'lidl',   '', '2020-07-01',   4.0),
	('tom',   'ikea',   '', '2020-07-01',  16.0),
	('tom',   'siwa',   '', '2020-08-01', 256.0),
	('tom',   'siwa',   '', '2020-08-01', 512.0);`)
	if err != nil {
		t.Fatalf("Unexpected error in SQL INSERT: %v", err)
	}

	_, err = memDb.Exec(`
	INSERT INTO salary (username, salary, recordtime) VALUES
	('alice', 2001.0, '2020-01-01'),
	('alice', 2002.0, '2020-02-01'),
	('alice', 2003.0, '2020-03-01'),
	('tom',   1601.0, '2020-01-01'),
	('tom',   1602.0, '2020-02-01'),
	('tom',   1603.0, '2020-03-01'),
	('tom',   1606.0, '2020-06-01'),
	('tom',   1608.0, '2020-08-01');`)
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
		want    []external.SpendingHistory
		wantErr bool
	}{
		{
			name: "One month data",
			args: args{
				startMonth: time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
				endMonth:   time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
			},
			want: []external.SpendingHistory{
				{
					ID:        0,
					Username:  "tom",
					MonthYear: time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
					Spending:  8.0,
					Salary:    1606.0,
				},
			},
			wantErr: false,
		},
		{
			name: "Three months data",
			args: args{
				startMonth: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				endMonth:   time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC),
			},
			want: []external.SpendingHistory{
				{
					ID:        0,
					Username:  "alice",
					MonthYear: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					Spending:  14.0,
					Salary:    2001,
				},
				{
					Username:  "alice",
					MonthYear: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
					Spending:  9,
					Salary:    2002.0,
				},
				{
					Username:  "alice",
					MonthYear: time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC),
					Spending:  33.46,
					Salary:    2003.0,
				},
				{
					ID:        0,
					Username:  "tom",
					MonthYear: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
					Spending:  10,
					Salary:    1601.0,
				},
				{
					Username:  "tom",
					MonthYear: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
					Spending:  15.4,
					Salary:    1602.0,
				},
				{
					Username:  "tom",
					MonthYear: time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC),
					Spending:  4.4,
					Salary:    1603.0,
				},
			},
			wantErr: false,
		},
		{
			name: "Only other has purchases",
			args: args{
				startMonth: time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
				endMonth:   time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC),
			},
			want: []external.SpendingHistory{
				{
					ID:        0,
					Username:  "tom",
					MonthYear: time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
					Spending:  768.0,
					Salary:    1608.0,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetMonthlyData(tt.args.startMonth, tt.args.endMonth)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMonthlyData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("%s: GetMonthlyData() differs:\n%s",
					tt.name, diff)
			}
		})
	}
}

func TestDeleteSpendingByID(t *testing.T) {
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
		bid      int64
		username string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "First row, mismatcing user",
			args: args{
				bid:      1,
				username: "tom",
			},
			wantErr: true,
		},
		{
			name: "First row, correct user",
			args: args{
				bid:      1,
				username: "alice",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DeleteSpendingByID(tt.args.bid, tt.args.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: DeleteSpendingByID() error=%v, wantErr %v",
					tt.name, err, tt.wantErr)
			}

			rowAfterDeletion, err := GetSpendingRowByID(
				tt.args.bid, tt.args.username)
			if err == nil {
				t.Errorf("%s: deletion of ID (%d) in budget table had failed, content was (%#v): %s",
					tt.name, tt.args.bid, rowAfterDeletion, err)
			}

		})
	}
}

func TestInsertPurchase(t *testing.T) {
	memDb, _ := sql.Open("sqlite3", ":memory:")
	defer memDb.Close()

	UpdateDBReference(memDb)
	CreateSchema(memDb)

	type args struct {
		username     string
		shopName     string
		category     string
		purchaseDate time.Time
		price        float64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Insert testing 1",
			args: args{
				username:     "Jaakko",
				shopName:     "Stockmann",
				category:     "Food",
				purchaseDate: time.Date(2022, 12, 5, 0, 0, 0, 0, time.UTC),
				price:        23.5,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InsertPurchase(tt.args.username, tt.args.shopName, tt.args.category, tt.args.purchaseDate, tt.args.price)
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertPurchase() error = %v, wantErr %v", err, tt.wantErr)
			}
			rows, err := memDb.Query("SELECT * FROM budget;")
			if err != nil {
				t.Error(err)
			}
			for rows.Next() {
				var id int64
				var username string
				var shopname string
				var category string
				var purcDate time.Time
				var price float32

				if err = rows.Scan(&id, &username, &shopname, &category, &purcDate, &price); err != nil {
					t.Error(err)
				}

				if id != 1 {
					t.Errorf("ID=%d, want=1", id)
				}
				if username != tt.args.username {
					t.Errorf("Username=%s, want=%s", username, tt.args.username)
				}
				if shopname != tt.args.shopName {
					t.Errorf("Shop name=%s, want=%s", shopname, tt.args.shopName)
				}
				if category != tt.args.category {
					t.Errorf("Category=%s, want=%s", category, tt.args.category)
				}
				if !purcDate.Equal(tt.args.purchaseDate) {
					t.Errorf("Purchase date=%#v, want=%#v", purcDate, tt.args.purchaseDate)
				}
			}
		})
	}
}
