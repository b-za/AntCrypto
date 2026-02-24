package report

import (
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/beerguevara/antcrypto/fifo/internal/classifier"
	"github.com/beerguevara/antcrypto/fifo/internal/config"
	"github.com/beerguevara/antcrypto/fifo/internal/unitcost"
)

type TransferRow struct {
	FinancialYear   string
	Coin            string
	Date            string
	Description     string
	Type            string
	QtyChange       decimal.Decimal
	Value           decimal.Decimal
	UnitCost        decimal.Decimal
	CustomCost      bool
	CustomCostNotes string
	LotReference    string
	Pool            string
}

type TransfersReportGenerator struct {
	config     *config.Config
	reportRows []TransferRow
}

func NewTransfersReportGenerator(cfg *config.Config) *TransfersReportGenerator {
	return &TransfersReportGenerator{
		config:     cfg,
		reportRows: []TransferRow{},
	}
}

func (tr *TransfersReportGenerator) GenerateReport(transactions []*classifier.Transaction) [][]string {
	tr.filterAndProcessTransactions(transactions)
	return tr.buildCSV()
}

func (tr *TransfersReportGenerator) filterAndProcessTransactions(transactions []*classifier.Transaction) {
	for _, tx := range transactions {
		// Skip in_buy and out_sell transactions
		if tx.Type == classifier.InflowBuy || tx.Type == classifier.OutflowSell {
			continue
		}

		txTime := time.Unix(tx.Timestamp, 0)
		financialYear := tr.getFinancialYear(tx.Timestamp)

		// Calculate Unit Cost or Unit Cost Value
		var unitCost decimal.Decimal
		var isCustomCost bool
		var customCostNotes string

		if tx.Type == classifier.InflowBuy || tx.Type == classifier.InflowBuyForOther || tx.Type == classifier.InflowOther {
			// Incoming - Unit Cost from purchase
			if tx.Type == classifier.InflowOther {
				result := unitcost.CalculateCustomUnitCost(tx)
				unitCost = result.Value
				isCustomCost = result.IsCustom
				customCostNotes = result.CustomNotes
			} else {
				unitCost = tx.ValueAmount.Div(tx.BalanceDelta.Abs())
				isCustomCost = false
				customCostNotes = ""
			}
		} else {
			// Outgoing or fees - Unit Cost Value (what's being sent out)
			// For fees, show value; for other outflows, could be zero
			unitCost = tx.ValueAmount
			isCustomCost = false
			customCostNotes = ""
		}

		row := TransferRow{
			FinancialYear:   financialYear,
			Coin:            tx.Currency,
			Date:            txTime.Format("2006-01-02 15:04:05"),
			Description:     tx.Description,
			Type:            string(tx.Type),
			QtyChange:       tx.BalanceDelta,
			Value:           tx.ValueAmount,
			UnitCost:        unitCost,
			CustomCost:      isCustomCost,
			CustomCostNotes: customCostNotes,
			LotReference:    tr.getLotReference(tx),
			Pool:            tr.getPoolFromType(tx.Type),
		}

		tr.reportRows = append(tr.reportRows, row)
	}

	// Sort chronologically by Date
	sort.Slice(tr.reportRows, func(i, j int) bool {
		return tr.reportRows[i].Date < tr.reportRows[j].Date
	})
}

func (tr *TransfersReportGenerator) getLotReference(tx *classifier.Transaction) string {
	if tx.LotReference != nil {
		return *tx.LotReference
	}
	return ""
}

func (tr *TransfersReportGenerator) getPoolFromType(txType classifier.TransactionType) string {
	return string(txType)
}

func (tr *TransfersReportGenerator) buildCSV() [][]string {
	var records [][]string

	headers := []string{
		"Financial Year", "Coin", "Date", "Description", "Type", "Qty Change",
		"Value (ZAR)", "Unit Cost (ZAR) / Unit Cost Value", "Custom Cost", "Custom Cost Notes",
		"Lot Reference", "Pool",
	}
	records = append(records, headers)

	for _, row := range tr.reportRows {
		record := []string{
			row.FinancialYear,
			row.Coin,
			row.Date,
			row.Description,
			row.Type,
			row.QtyChange.String(),
			row.Value.String(),
			row.UnitCost.String(),
			fmt.Sprintf("%t", row.CustomCost),
			row.CustomCostNotes,
			row.LotReference,
			row.Pool,
		}
		records = append(records, record)
	}

	return records
}

func (tr *TransfersReportGenerator) getFinancialYear(timestamp int64) string {
	date := time.Unix(timestamp, 0)
	year := date.Year()
	month := int(date.Month())

	if month >= tr.config.Settings.FinancialYearStart {
		return fmt.Sprintf("FY%d", year+1)
	}
	return fmt.Sprintf("FY%d", year)
}
