package report

import (
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/beerguevara/antcrypto/fifo/internal/classifier"
	"github.com/beerguevara/antcrypto/fifo/internal/config"
	"github.com/beerguevara/antcrypto/fifo/internal/fifo"
)

type ProfitLossRow struct {
	FinancialYear   string
	Coin            string
	SaleReference   string
	LotReference    string
	QuantitySold    decimal.Decimal
	UnitCost        decimal.Decimal
	CustomCost      bool
	CustomCostNotes string
	TotalCost       decimal.Decimal
	SellingPrice    decimal.Decimal
	Proceeds        decimal.Decimal
	ProfitLoss      decimal.Decimal
	FeeReference    string
	FeeAmount       decimal.Decimal
	IsSummary       bool
}

type ProfitLossReportGenerator struct {
	config     *config.Config
	reportRows []ProfitLossRow
}

type FinancialYearSummary struct {
	FY            string
	TotalSales    decimal.Decimal
	TotalCost     decimal.Decimal
	LossSales     decimal.Decimal
	ProfitSales   decimal.Decimal
	NetProfit     decimal.Decimal
	TotalProceeds decimal.Decimal
}

func NewProfitLossReportGenerator(cfg *config.Config) *ProfitLossReportGenerator {
	return &ProfitLossReportGenerator{
		config:     cfg,
		reportRows: []ProfitLossRow{},
	}
}

func (pl *ProfitLossReportGenerator) GenerateReport(allocations []*fifo.Allocation) [][]string {
	pl.processAllocations(allocations)
	return pl.buildCSV()
}

func (pl *ProfitLossReportGenerator) processAllocations(allocations []*fifo.Allocation) {
	summaries := make(map[string]*FinancialYearSummary)

	for _, allocation := range allocations {
		if allocation.Transaction.Type != classifier.OutflowSell {
			// Only process sales, not other outflows
			continue
		}

		financialYear := pl.getFinancialYear(allocation.Transaction.Timestamp)

		// Update summary
		if _, exists := summaries[financialYear]; !exists {
			summaries[financialYear] = &FinancialYearSummary{
				FY:            financialYear,
				TotalSales:    decimal.Zero,
				TotalCost:     decimal.Zero,
				TotalProceeds: decimal.Zero,
				LossSales:     decimal.Zero,
				ProfitSales:   decimal.Zero,
				NetProfit:     decimal.Zero,
			}
		}

		// Calculate total cost, proceeds, and profit for this transaction
		transactionTotalCost := decimal.Zero
		transactionTotalProceeds := decimal.Zero
		transactionTotalProfit := decimal.Zero

		// Calculate per-lot selling price
		sellingPricePerUnit := allocation.Transaction.ValueAmount.Abs().Div(allocation.Transaction.BalanceDelta.Abs())

		for _, lot := range allocation.ConsumedLots {
			totalCost := lot.Quantity.Mul(lot.UnitCost)
			proceedsPerLot := sellingPricePerUnit.Mul(lot.Quantity)
			profitPerLot := proceedsPerLot.Sub(totalCost)

			transactionTotalCost = transactionTotalCost.Add(totalCost)
			transactionTotalProceeds = transactionTotalProceeds.Add(proceedsPerLot)
			transactionTotalProfit = transactionTotalProfit.Add(profitPerLot)

			// Create row for this lot
			row := ProfitLossRow{
				FinancialYear:   financialYear,
				Coin:            allocation.Transaction.Currency,
				SaleReference:   allocation.Transaction.Reference,
				LotReference:    lot.Reference,
				QuantitySold:    lot.Quantity,
				UnitCost:        lot.UnitCost,
				CustomCost:      lot.CustomCost,
				CustomCostNotes: lot.CustomCostNotes,
				TotalCost:       totalCost,
				SellingPrice:    sellingPricePerUnit,
				Proceeds:        proceedsPerLot,
				ProfitLoss:      profitPerLot,
				FeeReference:    "",
				FeeAmount:       decimal.Zero,
				IsSummary:       false,
			}

			pl.reportRows = append(pl.reportRows, row)
		}

		// Update summary with totals from this transaction
		summaries[financialYear].TotalSales = summaries[financialYear].TotalSales.Add(decimal.NewFromInt(1))
		summaries[financialYear].TotalCost = summaries[financialYear].TotalCost.Add(transactionTotalCost)
		summaries[financialYear].TotalProceeds = summaries[financialYear].TotalProceeds.Add(transactionTotalProceeds)

		if transactionTotalProfit.IsNegative() {
			summaries[financialYear].LossSales = summaries[financialYear].LossSales.Add(decimal.NewFromInt(1))
		} else {
			summaries[financialYear].ProfitSales = summaries[financialYear].ProfitSales.Add(decimal.NewFromInt(1))
		}
		summaries[financialYear].NetProfit = summaries[financialYear].NetProfit.Add(transactionTotalProfit)
	}

	// Add fee lines
	pl.addFeeLines(allocations)

	// Add summary lines
	pl.addSummaryLines(summaries)

	// Sort by Financial Year → Sale Reference
	sort.Slice(pl.reportRows, func(i, j int) bool {
		if pl.reportRows[i].IsSummary != pl.reportRows[j].IsSummary {
			return !pl.reportRows[i].IsSummary
		}
		if pl.reportRows[i].FinancialYear != pl.reportRows[j].FinancialYear {
			return pl.reportRows[i].FinancialYear < pl.reportRows[j].FinancialYear
		}
		return pl.reportRows[i].SaleReference < pl.reportRows[j].SaleReference
	})
}

func (pl *ProfitLossReportGenerator) addFeeLines(allocations []*fifo.Allocation) {
	for _, allocation := range allocations {
		if allocation.Transaction.Type == classifier.OutflowFeeBuy ||
			allocation.Transaction.Type == classifier.OutflowFeeBuyForOther ||
			allocation.Transaction.Type == classifier.OutflowFeeInOther ||
			allocation.Transaction.Type == classifier.OutflowFeeSell ||
			allocation.Transaction.Type == classifier.OutflowFeeOutOther {

			financialYear := pl.getFinancialYear(allocation.Transaction.Timestamp)

			row := ProfitLossRow{
				FinancialYear:   financialYear,
				Coin:            allocation.Transaction.Currency,
				SaleReference:   "",
				LotReference:    "",
				QuantitySold:    allocation.Transaction.BalanceDelta.Abs(),
				UnitCost:        decimal.Zero,
				CustomCost:      false,
				CustomCostNotes: "",
				TotalCost:       decimal.Zero,
				SellingPrice:    decimal.Zero,
				Proceeds:        decimal.Zero,
				ProfitLoss:      decimal.Zero,
				FeeReference:    allocation.Transaction.Reference,
				FeeAmount:       allocation.Transaction.ValueAmount.Abs(),
				IsSummary:       false,
			}

			pl.reportRows = append(pl.reportRows, row)
		}
	}
}

func (pl *ProfitLossReportGenerator) addSummaryLines(summaries map[string]*FinancialYearSummary) {
	for _, summary := range summaries {
		// Loss summary line
		if summary.LossSales.IsPositive() {
			lossSummary := ProfitLossRow{
				FinancialYear:   summary.FY,
				Coin:            "",
				SaleReference:   "Losses Total",
				LotReference:    "",
				QuantitySold:    summary.LossSales,
				UnitCost:        decimal.Zero,
				CustomCost:      false,
				CustomCostNotes: "",
				TotalCost:       summary.TotalCost,
				SellingPrice:    decimal.Zero,
				Proceeds:        summary.TotalProceeds,
				ProfitLoss:      summary.TotalProceeds.Sub(summary.TotalCost),
				FeeReference:    "",
				FeeAmount:       decimal.Zero,
				IsSummary:       true,
			}
			pl.reportRows = append(pl.reportRows, lossSummary)
		}

		// Profit summary line
		if summary.ProfitSales.IsPositive() {
			profitSummary := ProfitLossRow{
				FinancialYear:   summary.FY,
				Coin:            "",
				SaleReference:   "Profits Total",
				LotReference:    "",
				QuantitySold:    summary.ProfitSales,
				UnitCost:        decimal.Zero,
				CustomCost:      false,
				CustomCostNotes: "",
				TotalCost:       summary.TotalCost,
				SellingPrice:    decimal.Zero,
				Proceeds:        summary.TotalProceeds,
				ProfitLoss:      summary.TotalProceeds.Sub(summary.TotalCost),
				FeeReference:    "",
				FeeAmount:       decimal.Zero,
				IsSummary:       true,
			}
			pl.reportRows = append(pl.reportRows, profitSummary)
		}

		// Combined line
		combinedSummary := ProfitLossRow{
			FinancialYear:   summary.FY,
			Coin:            "",
			SaleReference:   "Combined",
			LotReference:    "",
			QuantitySold:    summary.TotalSales,
			UnitCost:        decimal.Zero,
			CustomCost:      false,
			CustomCostNotes: "",
			TotalCost:       summary.TotalCost,
			SellingPrice:    decimal.Zero,
			Proceeds:        summary.TotalProceeds,
			ProfitLoss:      summary.NetProfit,
			FeeReference:    "",
			FeeAmount:       decimal.Zero,
			IsSummary:       true,
		}
		pl.reportRows = append(pl.reportRows, combinedSummary)
	}
}

func (pl *ProfitLossReportGenerator) buildCSV() [][]string {
	var records [][]string

	headers := []string{
		"Financial Year", "Coin", "Sale Reference", "Lot Reference", "Quantity Sold",
		"Unit Cost (ZAR)", "Custom Cost", "Custom Cost Notes", "Total Cost (ZAR)", "Selling Price (ZAR)",
		"Proceeds (ZAR)", "Profit/Loss (ZAR)", "Fee Reference", "Fee Amount (ZAR)",
	}
	records = append(records, headers)

	for _, row := range pl.reportRows {
		record := []string{
			row.FinancialYear,
			row.Coin,
			row.SaleReference,
			row.LotReference,
			row.QuantitySold.String(),
			row.UnitCost.String(),
			fmt.Sprintf("%t", row.CustomCost),
			row.CustomCostNotes,
			row.TotalCost.String(),
			row.SellingPrice.String(),
			row.Proceeds.String(),
			row.ProfitLoss.String(),
			row.FeeReference,
			row.FeeAmount.String(),
		}
		records = append(records, record)
	}

	return records
}

func (pl *ProfitLossReportGenerator) getFinancialYear(timestamp int64) string {
	date := time.Unix(timestamp, 0)
	year := date.Year()
	month := int(date.Month())

	if month >= pl.config.Settings.FinancialYearStart {
		return fmt.Sprintf("FY%d", year+1)
	}
	return fmt.Sprintf("FY%d", year)
}
