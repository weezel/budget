package debtcontrol

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
	"weezel/budget/db"
	"weezel/budget/logger"
)

func CalculateCompensatedDebts(
	ctx context.Context,
	user1 *db.StatisticsAggrByTimespanRow,
	user2 *db.StatisticsAggrByTimespanRow,
) error {
	debts := []*db.StatisticsAggrByTimespanRow{
		user1,
		user2,
	}

	if len(debts) != 2 {
		return fmt.Errorf("Not enough, or too much users to calculate debts (was %d)",
			len(debts))
	}

	// Descending order regarding to salary
	sort.Slice(debts, func(i, j int) bool {
		return debts[i].Salary < debts[j].Salary
	})

	sumSalaries := float64(debts[1].Salary + debts[0].Salary)
	lowerIncomeRatio := debts[0].Salary / sumSalaries
	greaterIncomeRatio := debts[1].Salary / sumSalaries

	debts[1].Owes = debts[1].ExpensesSum * lowerIncomeRatio
	debts[0].Owes = debts[0].ExpensesSum * greaterIncomeRatio

	debt := math.Abs(debts[0].Owes - debts[1].Owes)

	logger.Debugf("Debts fetched: %#v", debts)
	logger.Debugf("Sum of salaries: %.2f", sumSalaries)
	logger.Debugf("Lower income ration: %.2f", lowerIncomeRatio)
	logger.Debugf("Lower income owes: %.2f", debts[0].Owes)
	logger.Debugf("Greater income ration: %.2f", greaterIncomeRatio)
	logger.Debugf("Greater income owes: %.2f", debts[1].Owes)
	logger.Debugf("Debt in the end: %.2f", debt)

	if debts[0].Owes < debts[1].Owes {
		debts[0].Owes = debt
		debts[1].Owes = 0.0
	} else {
		debts[0].Owes = 0.0
		debts[1].Owes = debt
	}

	return nil
}

// FillDebts fills debt related data to stats parameter. Map is being used to combine different user's data together
// and calculating compensated debts.
func FillDebts(ctx context.Context, stats []*db.StatisticsAggrByTimespanRow) {
	byMonth := map[time.Time][]*db.StatisticsAggrByTimespanRow{}
	for i := range stats {
		if _, ok := byMonth[stats[i].EventDate]; !ok {
			byMonth[stats[i].EventDate] = []*db.StatisticsAggrByTimespanRow{}
		}
		byMonth[stats[i].EventDate] = append(byMonth[stats[i].EventDate], stats[i])

		if len(byMonth[stats[i].EventDate]) == 2 {
			values := byMonth[stats[i].EventDate]
			if err := CalculateCompensatedDebts(ctx, values[0], values[1]); err != nil {
				logger.Errorf("compensated debt update failed: %s", err)
				continue
			}
		}
	}
}
