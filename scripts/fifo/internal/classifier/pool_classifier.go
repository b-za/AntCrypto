package classifier

import (
	"github.com/shopspring/decimal"
)

const (
	daysThreshold     = 7
	quantityThreshold = 90 // 90%
)

var (
	quantityThresholdDecimal = decimal.NewFromInt(100)
)

type PoolClassifier struct {
}

func NewPoolClassifier() *PoolClassifier {
	return &PoolClassifier{}
}

func (c *PoolClassifier) ClassifyPools(transactions []*Transaction) {
	inBuyTransactions := c.getInBuyTransactions(transactions)
	outOtherTransactions := c.getOutOtherTransactions(transactions)

	linkedOutOthers := make(map[string]bool)

	for _, outOther := range outOtherTransactions {
		if linkedOutOthers[outOther.Reference] {
			continue
		}

		candidate := c.findBestMatch(outOther, inBuyTransactions, linkedOutOthers)
		if candidate != nil {
			candidate.Type = InflowBuyForOther
			if candidate.LotReference != nil {
				outOther.LinkedRef = candidate.LotReference
			}
			linkedOutOthers[outOther.Reference] = true
		}
	}
}

func (c *PoolClassifier) getInBuyTransactions(transactions []*Transaction) []*Transaction {
	var inBuys []*Transaction
	for _, tx := range transactions {
		if tx.Type == InflowBuy {
			inBuys = append(inBuys, tx)
		}
	}
	return inBuys
}

func (c *PoolClassifier) getOutOtherTransactions(transactions []*Transaction) []*Transaction {
	var outOthers []*Transaction
	for _, tx := range transactions {
		if tx.Type == OutflowOther {
			outOthers = append(outOthers, tx)
		}
	}
	return outOthers
}

func (c *PoolClassifier) findBestMatch(outOther *Transaction, inBuys []*Transaction, linkedOutOthers map[string]bool) *Transaction {
	var bestMatch *Transaction
	minTimeDiff := int64(daysThreshold * 24 * 60 * 60)

	for _, inBuy := range inBuys {
		if inBuy.Type != InflowBuy {
			continue
		}

		if inBuy.Timestamp >= outOther.Timestamp {
			continue
		}

		timeDiff := outOther.Timestamp - inBuy.Timestamp
		if timeDiff > int64(daysThreshold*24*60*60) {
			continue
		}

		quantityRatio := outOther.BalanceDelta.Abs().Div(inBuy.BalanceDelta.Abs()).Mul(quantityThresholdDecimal)
		if quantityRatio.LessThan(decimal.NewFromInt(quantityThreshold)) {
			continue
		}

		if timeDiff < minTimeDiff {
			minTimeDiff = timeDiff
			bestMatch = inBuy
		}
	}

	return bestMatch
}
