package classifier

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/beerguevara/antcrypto/fifo/pkg/utils"
)

type TransactionType string

const (
	InflowBuyForOther     TransactionType = "in_buy_for_other"
	InflowBuy             TransactionType = "in_buy"
	InflowOther           TransactionType = "in_other"
	OutflowSell           TransactionType = "out_sell"
	OutflowOther          TransactionType = "out_other"
	OutflowFeeBuy         TransactionType = "out_fee_buy"
	OutflowFeeBuyForOther TransactionType = "out_fee_buy_for_other"
	OutflowFeeInOther     TransactionType = "out_fee_in_other"
	OutflowFeeSell        TransactionType = "out_fee_sell"
	OutflowFeeOutOther    TransactionType = "out_fee_out_other"
)

type Transaction struct {
	WalletID              string
	Row                   int
	Timestamp             int64  // Unix timestamp for easier comparison
	TimestampStr          string // String representation for error logging
	Description           string
	Currency              string
	BalanceDelta          decimal.Decimal
	ValueAmount           decimal.Decimal
	Reference             string
	CryptocurrencyAddress string
	CryptocurrencyTxID    string
	Type                  TransactionType
	LinkedRef             *string
	LinkedTransaction     *Transaction
	LotReference          *string
}

type Classifier struct {
}

func NewClassifier() *Classifier {
	return &Classifier{}
}

func (c *Classifier) Classify(tx Transaction) (TransactionType, error) {
	if tx.BalanceDelta.IsPositive() {
		return c.classifyInflow(tx)
	} else if tx.BalanceDelta.IsNegative() {
		return c.classifyOutflow(tx)
	}
	return "", fmt.Errorf("transaction has zero balance delta, cannot classify")
}

func (c *Classifier) classifyInflow(tx Transaction) (TransactionType, error) {
	description := strings.TrimSpace(tx.Description)

	if utils.HasPrefixCaseInsensitive(description, "Bought") {
		return InflowBuy, nil
	}

	return InflowOther, nil
}

func (c *Classifier) classifyOutflow(tx Transaction) (TransactionType, error) {
	description := strings.TrimSpace(tx.Description)

	if utils.HasPrefixCaseInsensitive(description, "Sold") {
		return OutflowSell, nil
	}

	if utils.HasPrefixCaseInsensitive(description, "Trading Fee") || utils.HasPrefixCaseInsensitive(description, "Fee") {
		return OutflowFeeBuy, nil
	}

	return OutflowOther, nil
}

func (c *Classifier) ReclassifyAsBuyForOther(tx *Transaction) {
	tx.Type = InflowBuyForOther
}
