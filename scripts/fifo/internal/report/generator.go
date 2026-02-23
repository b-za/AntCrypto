package report

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/beerguevara/antcrypto/fifo/internal/classifier"
	"github.com/beerguevara/antcrypto/fifo/internal/fifo"
	"github.com/beerguevara/antcrypto/fifo/internal/parser"
	"github.com/beerguevara/antcrypto/fifo/internal/pool"
)

type ReportRow struct {
	FinancialYear string
	TransRef      string
	Date          string
	Description   string
	Type          string
	LotReference  string
	QtyChange     decimal.Decimal
	BalanceUnits  decimal.Decimal
	TotalCost     decimal.Decimal
	BalanceValue  decimal.Decimal
	Fee           decimal.Decimal
	Proceeds      decimal.Decimal
	Profit        decimal.Decimal
	UnitCost      decimal.Decimal
}

type ReportGenerator struct {
	reportRows      []ReportRow
	currentBalance  decimal.Decimal
	totalInflowCost decimal.Decimal
}

func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{
		reportRows:      []ReportRow{},
		currentBalance:  decimal.Zero,
		totalInflowCost: decimal.Zero,
	}
}

func (rg *ReportGenerator) GenerateReport(transactions []*classifier.Transaction, allocator *fifo.FIFOAllocator, errorLog *parser.JSONLErrorLogger) [][]string {
	rg.processTransactions(transactions, allocator, errorLog)

	var records [][]string
	headers := []string{
		"Financial Year", "Trans Ref", "Date", "Description", "Type",
		"Lot Reference", "Qty Change", "Balance Units", "Total Cost (ZAR)",
		"Balance Value (ZAR)", "Fee (ZAR)", "Proceeds (ZAR)", "Profit (ZAR)", "Unit Cost (ZAR)",
	}
	records = append(records, headers)

	for _, row := range rg.reportRows {
		record := []string{
			row.FinancialYear,
			row.TransRef,
			row.Date,
			row.Description,
			row.Type,
			row.LotReference,
			row.QtyChange.String(),
			row.BalanceUnits.String(),
			row.TotalCost.String(),
			row.BalanceValue.String(),
			row.Fee.String(),
			row.Proceeds.String(),
			row.Profit.String(),
			row.UnitCost.String(),
		}
		records = append(records, record)
	}

	return records
}

func (rg *ReportGenerator) processTransactions(transactions []*classifier.Transaction, allocator *fifo.FIFOAllocator, errorLog *parser.JSONLErrorLogger) {
	for _, tx := range transactions {
		txTime := time.Unix(tx.Timestamp, 0)
		financialYear := rg.getFinancialYear(txTime)

		switch {
		case tx.Type == classifier.InflowBuy || tx.Type == classifier.InflowBuyForOther || tx.Type == classifier.InflowOther:
			rg.processInflow(tx, financialYear)
		case tx.Type == classifier.OutflowSell || tx.Type == classifier.OutflowOther ||
			tx.Type == classifier.OutflowFeeBuy || tx.Type == classifier.OutflowFeeBuyForOther ||
			tx.Type == classifier.OutflowFeeInOther || tx.Type == classifier.OutflowFeeSell || tx.Type == classifier.OutflowFeeOutOther:

			// Get lots consumed by THIS SPECIFIC transaction (by Row number)
			consumedLots := allocator.GetLotsConsumedByTransaction(tx.Row)
			rg.processOutflow(tx, financialYear, consumedLots)
		}
	}
}

func (rg *ReportGenerator) processInflow(tx *classifier.Transaction, financialYear string) {
	unitCost := tx.ValueAmount.Div(tx.BalanceDelta.Abs())
	lotReference := fmt.Sprintf("%s_XXXXXXX", tx.Type)

	if tx.LotReference != nil {
		lotReference = *tx.LotReference
	}

	rg.currentBalance = rg.currentBalance.Add(tx.BalanceDelta)
	rg.totalInflowCost = rg.totalInflowCost.Add(tx.ValueAmount)

	row := ReportRow{
		FinancialYear: financialYear,
		TransRef:      tx.Reference,
		Date:          time.Unix(tx.Timestamp, 0).Format("2006-01-02 15:04:05"),
		Description:   tx.Description,
		Type:          string(tx.Type),
		LotReference:  lotReference,
		QtyChange:     tx.BalanceDelta,
		BalanceUnits:  rg.currentBalance,
		TotalCost:     tx.ValueAmount,
		BalanceValue:  rg.currentBalance.Mul(unitCost),
		Fee:           decimal.Zero,
		Proceeds:      decimal.Zero,
		Profit:        decimal.Zero,
		UnitCost:      unitCost,
	}

	rg.reportRows = append(rg.reportRows, row)
}

func (rg *ReportGenerator) processOutflow(tx *classifier.Transaction, financialYear string, consumedLots []pool.Lot) {
	if len(consumedLots) == 0 {
		return
	}

	totalProceeds := tx.ValueAmount.Abs()

	for _, lot := range consumedLots {
		totalCost := lot.Quantity.Mul(lot.UnitCost)
		profit := totalProceeds.Mul(lot.Quantity.Div(tx.BalanceDelta.Abs())).Sub(totalCost)

		rg.currentBalance = rg.currentBalance.Sub(lot.Quantity)

		row := ReportRow{
			FinancialYear: financialYear,
			TransRef:      tx.Reference,
			Date:          time.Unix(tx.Timestamp, 0).Format("2006-01-02 15:04:05"),
			Description:   tx.Description,
			Type:          string(tx.Type),
			LotReference:  lot.Reference,
			QtyChange:     lot.Quantity.Neg(),
			BalanceUnits:  rg.currentBalance,
			TotalCost:     decimal.Zero,
			BalanceValue:  rg.currentBalance.Mul(lot.UnitCost),
			Fee:           decimal.Zero,
			Proceeds:      totalProceeds.Mul(lot.Quantity.Div(tx.BalanceDelta.Abs())),
			Profit:        profit,
			UnitCost:      lot.UnitCost,
		}

		rg.reportRows = append(rg.reportRows, row)
	}
}

func (rg *ReportGenerator) getFinancialYear(date time.Time) string {
	year := date.Year()
	month := date.Month()

	if month >= 3 {
		return fmt.Sprintf("FY%d", year)
	}
	return fmt.Sprintf("FY%d", year-1)
}
