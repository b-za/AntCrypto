package report

import (
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/beerguevara/antcrypto/fifo/internal/config"
	"github.com/beerguevara/antcrypto/fifo/internal/pool"
)

type InventoryRow struct {
	FinancialYear   string
	Coin            string
	LotReference    string
	Quantity        decimal.Decimal
	DateOfPurchase  string
	UnitCost        decimal.Decimal
	CustomCost      bool
	CustomCostNotes string
	TotalCost       decimal.Decimal
	Pool            string
	IsSummary       bool
}

type InventoryReportGenerator struct {
	config     *config.Config
	coin       string
	reportRows []InventoryRow
}

func NewInventoryReportGenerator(cfg *config.Config) *InventoryReportGenerator {
	return &InventoryReportGenerator{
		config:     cfg,
		reportRows: []InventoryRow{},
	}
}

func (ig *InventoryReportGenerator) GenerateReportForCoin(poolManager *pool.PoolManager, coin string) [][]string {
	ig.coin = coin
	ig.collectInventoryRows(poolManager)
	ig.addSummaryLines()
	return ig.buildCSV()
}

func (ig *InventoryReportGenerator) collectInventoryRows(pm *pool.PoolManager) {
	poolNames := []string{"in_buy", "in_buy_for_other", "in_other"}

	for _, poolName := range poolNames {
		pool, exists := pm.Pools[poolName]
		if !exists {
			continue
		}

		for _, lot := range pool.Lots {
			if lot.Quantity.IsPositive() || lot.Quantity.IsZero() {
				financialYear := ig.getFinancialYearFromTimestamp(lot.Timestamp)

				row := InventoryRow{
					FinancialYear:   financialYear,
					Coin:            ig.coin,
					LotReference:    lot.Reference,
					Quantity:        lot.Quantity,
					DateOfPurchase:  time.Unix(lot.Timestamp, 0).Format("2006-01-02 15:04:05"),
					UnitCost:        lot.UnitCost,
					CustomCost:      lot.CustomCost,
					CustomCostNotes: lot.CustomCostNotes,
					TotalCost:       lot.Quantity.Mul(lot.UnitCost),
					Pool:            poolName,
					IsSummary:       false,
				}

				ig.reportRows = append(ig.reportRows, row)
			}
		}
	}

	// Sort: Financial Year → Coin → Date of Purchase (oldest first)
	sort.Slice(ig.reportRows, func(i, j int) bool {
		if ig.reportRows[i].FinancialYear != ig.reportRows[j].FinancialYear {
			return ig.reportRows[i].FinancialYear < ig.reportRows[j].FinancialYear
		}
		if ig.reportRows[i].Coin != ig.reportRows[j].Coin {
			return ig.reportRows[i].Coin < ig.reportRows[j].Coin
		}
		return ig.reportRows[i].DateOfPurchase < ig.reportRows[j].DateOfPurchase
	})
}

func (ig *InventoryReportGenerator) addSummaryLines() {
	// Group by financial year and coin
	summaries := make(map[string]*inventorySummary)

	for _, row := range ig.reportRows {
		if row.IsSummary {
			continue
		}

		key := row.FinancialYear + "_" + row.Coin
		if _, exists := summaries[key]; !exists {
			summaries[key] = &inventorySummary{}
		}

		summaries[key].Quantity = summaries[key].Quantity.Add(row.Quantity)
		summaries[key].TotalValue = summaries[key].TotalValue.Add(row.TotalCost)
	}

	// Add summary rows
	for key, summary := range summaries {
		// Parse key like "FY2023_XBT" into FY and coin
		parts := ig.parseKey(key)
		financialYear := parts[0]
		coin := parts[1]

		summaryRow := InventoryRow{
			FinancialYear:   financialYear,
			Coin:            coin,
			LotReference:    "Total",
			Quantity:        summary.Quantity,
			DateOfPurchase:  "",
			UnitCost:        decimal.Zero,
			CustomCost:      false,
			CustomCostNotes: "",
			TotalCost:       summary.TotalValue,
			Pool:            "",
			IsSummary:       true,
		}

		ig.reportRows = append(ig.reportRows, summaryRow)
	}
}

type inventorySummary struct {
	Quantity   decimal.Decimal
	TotalValue decimal.Decimal
}

func (ig *InventoryReportGenerator) parseKey(key string) []string {
	// Parse key like "FY2023_xbt" into ["FY2023", "xbt"]
	// Find the underscore that separates FY from coin
	for i := 0; i < len(key); i++ {
		if key[i] == '_' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key, ""}
}

func (ig *InventoryReportGenerator) buildCSV() [][]string {
	var records [][]string

	headers := []string{
		"Financial Year", "Coin", "Lot Reference", "Quantity",
		"Date of Original Purchase", "Unit Cost (ZAR)", "Custom Cost", "Custom Cost Notes",
		"Total Cost (ZAR)", "Pool",
	}
	records = append(records, headers)

	for _, row := range ig.reportRows {
		var record []string

		if row.IsSummary {
			record = []string{
				row.FinancialYear,
				row.Coin,
				row.LotReference,
				row.Quantity.String(),
				row.DateOfPurchase,
				row.UnitCost.String(),
				fmt.Sprintf("%t", row.CustomCost),
				row.CustomCostNotes,
				row.TotalCost.String(),
				row.Pool,
			}
		} else {
			record = []string{
				row.FinancialYear,
				row.Coin,
				row.LotReference,
				row.Quantity.String(),
				row.DateOfPurchase,
				row.UnitCost.String(),
				fmt.Sprintf("%t", row.CustomCost),
				row.CustomCostNotes,
				row.TotalCost.String(),
				row.Pool,
			}
		}

		records = append(records, record)
	}

	return records
}

func (ig *InventoryReportGenerator) getFinancialYearFromTimestamp(timestamp int64) string {
	date := time.Unix(timestamp, 0)
	year := date.Year()
	month := int(date.Month())

	if month >= ig.config.Settings.FinancialYearStart {
		return fmt.Sprintf("FY%d", year+1)
	}
	return fmt.Sprintf("FY%d", year)
}
