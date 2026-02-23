package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/beerguevara/antcrypto/fifo/internal/classifier"
	"github.com/beerguevara/antcrypto/fifo/internal/config"
	"github.com/beerguevara/antcrypto/fifo/internal/fifo"
	"github.com/beerguevara/antcrypto/fifo/internal/pool"
)

type ReportCoordinator struct {
	coinPoolsMap map[string]*pool.PoolManager
	transactions []*classifier.Transaction
	allocations  []*fifo.Allocation
	config       *config.Config
}

func NewReportCoordinator(coinPoolsMap map[string]*pool.PoolManager, transactions []*classifier.Transaction, allocations []*fifo.Allocation, cfg *config.Config) *ReportCoordinator {
	return &ReportCoordinator{
		coinPoolsMap: coinPoolsMap,
		transactions: transactions,
		allocations:  allocations,
		config:       cfg,
	}
}

func (rc *ReportCoordinator) GenerateAllReports(reportsDir string) error {
	var errorCollection []error

	// Generate FIFO reports for each coin
	if err := rc.generateFIFOReports(reportsDir); err != nil {
		errorCollection = append(errorCollection, err)
	}

	// Generate inventory report
	if err := rc.generateInventoryReport(reportsDir); err != nil {
		errorCollection = append(errorCollection, err)
	}

	// Generate profit/loss report
	if err := rc.generateProfitLossReport(reportsDir); err != nil {
		errorCollection = append(errorCollection, err)
	}

	// Generate transfers report
	if err := rc.generateTransfersReport(reportsDir); err != nil {
		errorCollection = append(errorCollection, err)
	}

	if len(errorCollection) > 0 {
		return fmt.Errorf("encountered %d errors during report generation: %v", len(errorCollection), errorCollection)
	}

	return nil
}

func (rc *ReportCoordinator) generateFIFOReports(reportsDir string) error {
	// Generate FIFO report for each coin
	for coin, pm := range rc.coinPoolsMap {
		// Filter transactions for this coin (case-insensitive)
		var coinTransactions []*classifier.Transaction
		for _, tx := range rc.transactions {
			if strings.EqualFold(tx.Currency, coin) {
				coinTransactions = append(coinTransactions, tx)
			}
		}

		if len(coinTransactions) == 0 {
			continue
		}

		// Create allocator for this coin
		allocator := fifo.NewFIFOAllocator(pm)

		// Reconstruct txToLots from allocations for this coin (case-insensitive)
		txToLots := make(map[int][]pool.Lot)
		for _, allocation := range rc.allocations {
			if strings.EqualFold(allocation.Transaction.Currency, coin) && len(allocation.ConsumedLots) > 0 {
				// Copy lots to avoid shared references
				lotsCopy := make([]pool.Lot, len(allocation.ConsumedLots))
				copy(lotsCopy, allocation.ConsumedLots)
				txToLots[allocation.Transaction.Row] = lotsCopy
			}
		}
		allocator.SetTxToLots(txToLots)

		generator := NewFIFOReportGenerator(rc.config)
		records := generator.GenerateReport(coinTransactions, pm, allocator)

		outputPath := fmt.Sprintf("%s/%s_fifo.csv", reportsDir, coin)
		if err := WriteCSV(outputPath, records); err != nil {
			return fmt.Errorf("failed to write FIFO report for %s: %w", coin, err)
		}
	}

	return nil
}

func (rc *ReportCoordinator) generateInventoryReport(reportsDir string) error {
	var allRecords [][]string

	for coin, pm := range rc.coinPoolsMap {
		generator := NewInventoryReportGenerator(rc.config)
		coinRecords := generator.GenerateReportForCoin(pm, coin)

		// Skip headers for subsequent coins
		if len(allRecords) > 0 {
			coinRecords = coinRecords[1:]
		}

		allRecords = append(allRecords, coinRecords...)
	}

	outputPath := fmt.Sprintf("%s/inventory.csv", reportsDir)
	if err := WriteCSV(outputPath, allRecords); err != nil {
		return fmt.Errorf("failed to write inventory report: %w", err)
	}

	return nil
}

func (rc *ReportCoordinator) generateProfitLossReport(reportsDir string) error {
	generator := NewProfitLossReportGenerator(rc.config)
	records := generator.GenerateReport(rc.allocations)

	outputPath := fmt.Sprintf("%s/financial_year_profit_loss.csv", reportsDir)
	if err := WriteCSV(outputPath, records); err != nil {
		return fmt.Errorf("failed to write profit/loss report: %w", err)
	}

	return nil
}

func (rc *ReportCoordinator) generateTransfersReport(reportsDir string) error {
	generator := NewTransfersReportGenerator(rc.config)
	records := generator.GenerateReport(rc.transactions)

	outputPath := fmt.Sprintf("%s/transfers.csv", reportsDir)
	if err := WriteCSV(outputPath, records); err != nil {
		return fmt.Errorf("failed to write transfers report: %w", err)
	}

	return nil
}

func (rc *ReportCoordinator) getFinancialYear(timestamp int64) string {
	date := time.Unix(timestamp, 0)
	year := date.Year()
	month := int(date.Month())

	if month >= rc.config.Settings.FinancialYearStart {
		return fmt.Sprintf("FY%d", year+1)
	}
	return fmt.Sprintf("FY%d", year)
}
