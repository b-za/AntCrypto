package unitcost

import (
	"github.com/beerguevara/antcrypto/fifo/internal/classifier"
	"github.com/shopspring/decimal"
)

type UnitCostResult struct {
	Value       decimal.Decimal
	IsCustom    bool
	CustomNotes string
}

func CalculateCustomUnitCost(tx *classifier.Transaction) UnitCostResult {
	return UnitCostResult{
		Value:       tx.ValueAmount.Div(tx.BalanceDelta.Abs()),
		IsCustom:    false,
		CustomNotes: "",
	}
}
