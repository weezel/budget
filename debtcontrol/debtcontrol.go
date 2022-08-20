package debtcontrol

import (
	"context"
	"fmt"
	"math"
	"sort"
	"weezel/budget/db"
	"weezel/budget/logger"
)

func GetSalaryCompensatedDebts(
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

	lowerIncomeOwes := debts[1].ExpensesSum * lowerIncomeRatio
	greaterIncomeOwes := debts[0].ExpensesSum * greaterIncomeRatio

	totalExpenses := float64(debts[0].ExpensesSum + debts[1].ExpensesSum)
	expRatioByLowerInc := debts[0].ExpensesSum / totalExpenses
	expRatioByGreaterInc := debts[1].ExpensesSum / totalExpenses

	debt := math.Abs(greaterIncomeOwes - lowerIncomeOwes)

	logger.Debugf("Sum of salaries: %.2f", sumSalaries)
	logger.Debugf("Lower income ration: %.2f", lowerIncomeRatio)
	logger.Debugf("Lower income owes: %.2f", lowerIncomeOwes)
	logger.Debugf("Greater income ration: %.2f", greaterIncomeRatio)
	logger.Debugf("Greater income owes: %.2f", greaterIncomeOwes)
	logger.Debugf("Expenses ratio by lower income: %.2f", expRatioByLowerInc)
	logger.Debugf("Expenses ratio by greater income: %.2f", expRatioByGreaterInc)
	logger.Debugf("Total expenses: %.2f", totalExpenses)
	logger.Debugf("Debt in the end: %.2f", debt)

	if expRatioByLowerInc < expRatioByGreaterInc {
		debts[0].Owes = debt
		debts[1].Owes = 0.0
	} else {
		debts[0].Owes = 0.0
		debts[1].Owes = debt
	}
	logger.Debugf("Debts fetched: %#v", debts)

	return nil
}
