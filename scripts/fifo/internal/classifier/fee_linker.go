package classifier

type FeeLinker struct {
}

func NewFeeLinker() *FeeLinker {
	return &FeeLinker{}
}

func (f *FeeLinker) LinkFees(transactions []*Transaction) {
	refMap := make(map[string]*Transaction)

	for _, tx := range transactions {
		if tx.Type != OutflowSell && tx.Type != OutflowOther && tx.Type != InflowBuy && tx.Type != InflowOther && tx.Type != InflowBuyForOther {
			continue
		}
		refMap[tx.Reference] = tx
	}

	for _, tx := range transactions {
		if !f.isFee(tx) {
			continue
		}

		parent, exists := refMap[tx.Reference]
		if !exists {
			continue
		}

		tx.LinkedTransaction = parent
		f.subclassifyFee(tx, parent.Type)
	}
}

func (f *FeeLinker) isFee(tx *Transaction) bool {
	return tx.Type == OutflowFeeBuy || tx.Type == OutflowFeeBuyForOther || tx.Type == OutflowFeeInOther || tx.Type == OutflowFeeSell || tx.Type == OutflowFeeOutOther
}

func (f *FeeLinker) subclassifyFee(tx *Transaction, parentType TransactionType) {
	switch parentType {
	case InflowBuy:
		tx.Type = OutflowFeeBuy
	case InflowBuyForOther:
		tx.Type = OutflowFeeBuyForOther
	case InflowOther:
		tx.Type = OutflowFeeInOther
	case OutflowSell:
		tx.Type = OutflowFeeSell
	case OutflowOther:
		tx.Type = OutflowFeeOutOther
	default:
		tx.Type = OutflowFeeBuy // Default to out_fee_buy
	}
}
