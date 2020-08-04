package dbengine

import (
	"database/sql"
	"log"
	"reflect"
	"testing"
	"time"

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
		log.Fatalf("Unexpected error in SQL INSERT: %v", err)
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
		log.Fatalf("Unexpected error in SQL INSERT: %v", err)
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
	('tom',   'siwa',   '', '08-2020', 512.0)
	;`)
	if err != nil {
		log.Fatalf("Unexpected error in SQL INSERT: %v", err)
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
