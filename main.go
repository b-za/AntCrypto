package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

type ExchangeTransaction struct {
	Date         time.Time
	Reference    string
	Coin         string
	ZARValue     decimal.Decimal
	Direction    string
	IsOutflow    bool
}

type DateRates struct {
	Date                  string            `yaml:"date"`
	TransactionReferences []string          `yaml:"transaction_references"`
	USDPrices             map[string]string `yaml:"usd_prices"`      // BTC, ETH, XRP, LTC prices in USD
	USDToZARRate          string            `yaml:"usd_to_zar_rate"` // USD to ZAR rate
}

type RatesTemplate struct {
	DatesRequired []DateRates `yaml:"dates_required"`
	Usage         string      `yaml:"usage"`
}

type coinFile struct {
	coin string
	path string
}

func main() {
	configPath := flag.String("config", "../../config/config.yaml", "Path to config file")
	rootsFlag := flag.String("roots", "", "Comma-separated list of root aliases to process (default: all)")
	flag.Parse()

	roots, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\\n", err)
		os.Exit(1)
	}

	if *rootsFlag != "" {
		roots = filterRoots(roots, *rootsFlag)
	}

	if len(roots) == 0 {
		fmt.Println("No roots to process")
		return
	}

	allTransactions := []ExchangeTransaction{}
	coinSet := make(map[string]bool)

	for _, root := range roots {
		txs, coins, err := processRoot(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process root %s: %v\\n", root.Alias, err)
			continue
		}
		allTransactions = append(allTransactions, txs...)
		for _, coin := range coins {
			coinSet[coin] = true
		}
	}

	coins := getSortedCoins(coinSet)
	fmt.Printf("Discovered %d coins: %s\\n", len(coins), strings.Join(coins, ", "))

	sort.Slice(allTransactions, func(i, j int) bool {
		return allTransactions[i].Date.Before(allTransactions[j].Date)
	})

	templatesDir := filepath.Join(roots[0].Path, "exchange_templates", time.Now().Format("2006_01_02_1504"))
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create templates directory: %v\\n", err)
		os.Exit(1)
	}

	errorLogPath := filepath.Join(templatesDir, "exchange_rates_template_error_log.jsonl")
	errorLog, err := os.Create(errorLogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create error log: %v\\n", err)
		os.Exit(1)
	}
	defer errorLog.Close()

	validationLogPath := filepath.Join(templatesDir, "exchange_rates_template_validation.log")
	validationLog, err := os.Create(validationLogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create validation log: %v\\n", err)
		os.Exit(1)
	}
	defer validationLog.Close()

	dateRatesMap := make(map[string]*DateRates)
	for _, tx := range allTransactions {
		dateStr := tx.Date.Format("2006-01-02")

		if dateRatesMap[dateStr] == nil {
			dateRatesMap[dateStr] = &DateRates{
				Date:                  dateStr,
				TransactionReferences: []string{},
				USDPrices:             make(map[string]string),
				USDToZARRate:          "0.00",
			}
		}

		dateRatesMap[dateStr].TransactionReferences = append(dateRatesMap[dateStr].TransactionReferences, tx.Reference)
	}

	dateRates := make([]DateRates, 0, len(dateRatesMap))
	for _, dr := range dateRatesMap {
		sort.Strings(dr.TransactionReferences)

		zeroPrice := "0.00"
		dr.USDPrices["btc"] = zeroPrice
		dr.USDPrices["eth"] = zeroPrice
		dr.USDPrices["xrp"] = zeroPrice
		dr.USDPrices["ltc"] = zeroPrice

		dateRates = append(dateRates, *dr)
	}

	sort.Slice(dateRates, func(i, j int) bool {
		return dateRates[i].Date < dateRates[j].Date
	})

	template := RatesTemplate{
		DatesRequired: dateRates,
		Usage:         "For each date, fill in the USD prices (BTC, ETH, XRP, LTC) and USD→ZAR exchange rate. These will be used in Part 2 to calculate cross-currency conversions.",
	}

	yamlPath := filepath.Join(templatesDir, "exchange_rates_template.yaml")
	yamlBytes, err := yaml.Marshal(template)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal YAML: %v\\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(yamlPath, yamlBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write YAML: %v\\n", err)
		os.Exit(1)
	}

	writeValidationSummary(validationLogPath, len(allTransactions), len(dateRates), coins)

	fmt.Printf("Successfully generated exchange rates template:\\n")
	fmt.Printf("  YAML file: %s\\n", yamlPath)
	fmt.Printf("  Validation log: %s\\n", validationLogPath)
	fmt.Printf("\\n")
	fmt.Printf("NEXT STEPS:\\n")
	fmt.Printf("  1. Fill in USD prices and USD→ZAR rates in the YAML file\\n")
	fmt.Printf("  2. Run Part 2 code to use these rates for cross-currency conversions\\n")
}

type RootConfig struct {
	Alias string `yaml:"alias"`
	Path  string `yaml:"path"`
}

type Config struct {
	Roots []RootConfig `yaml:"roots"`
}

func loadConfig(path string) ([]RootConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg.Roots, nil
}

func filterRoots(roots []RootConfig, aliases string) []RootConfig {
	aliasMap := make(map[string]bool)
	for _, alias := range strings.Split(aliases, ",") {
		aliasMap[strings.TrimSpace(alias)] = true
	}

	var filtered []RootConfig
	for _, root := range roots {
		if aliasMap[root.Alias] {
			filtered = append(filtered, root)
		}
	}
	return filtered
}

func processRoot(root RootConfig) ([]ExchangeTransaction, []string, error) {
	dataDir := filepath.Join(root.Path, "data")
	coinFiles, err := findCoinFiles(dataDir)
	if err != nil {
		return nil, nil, err
	}

	var transactions []ExchangeTransaction
	coinSet := make(map[string]bool)

	for _, cf := range coinFiles {
		coinTransactions, err := processCoinFile(cf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to process coin file %s: %v\\n", cf.coin, err)
			continue
		}
		transactions = append(transactions, coinTransactions...)
		if len(coinTransactions) > 0 {
			coinSet[cf.coin] = true
		}
	}

	return transactions, getSortedCoins(coinSet), nil
}

func findCoinFiles(dataDir string) ([]coinFile, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}

	var coinFiles []coinFile
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".csv" {
			continue
		}

		filename := entry.Name()
		coin := strings.ToLower(filename[:len(filename)-4])

		if coin == "zar" {
			continue
		}

		coinFiles = append(coinFiles, coinFile{
			coin: coin,
			path:  filepath.Join(dataDir, filename),
		})
	}

	return coinFiles, nil
}

func processCoinFile(cf coinFile) ([]ExchangeTransaction, error) {
	file, err := os.Open(cf.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("no data rows in file")
	}

	var exchangeTxs []ExchangeTransaction

	for i, record := range records {
		if i == 0 {
			continue
		}

		if len(record) < 13 {
			continue
		}

		timestamp, err := time.Parse("2006-01-02 15:04:05", record[2])
		if err != nil {
			continue
		}

		balanceDelta, err := decimal.NewFromString(record[5])
		if err != nil {
			continue
		}

		valueAmount, err := decimal.NewFromString(record[12])
		if err != nil {
			continue
		}

		reference := record[13]
		description := strings.ToLower(strings.TrimSpace(record[3]))

		txType := classifyTransaction(description, balanceDelta)
		if txType == "in_other" || txType == "out_other" {
			direction := "in"
			isOutflow := false
			if txType == "out_other" {
				direction = "out"
				isOutflow = true
			}

			exchangeTxs = append(exchangeTxs, ExchangeTransaction{
				Date:      timestamp,
				Reference: reference,
				Coin:      strings.ToLower(record[4]),
				ZARValue:  valueAmount.Abs(),
				Direction: direction,
				IsOutflow: isOutflow,
			})
		}
	}

	return exchangeTxs, nil
}

func classifyTransaction(description string, balanceDelta decimal.Decimal) string {
	if balanceDelta.IsPositive() {
		if strings.HasPrefix(description, "bought") {
			return "buy"
		} else {
			return "in_other"
		}
	} else if balanceDelta.IsNegative() {
		if strings.HasPrefix(description, "sold") {
			return "sell"
		} else if strings.Contains(description, "fee") {
			return "fee"
		} else {
			return "out_other"
		}
	}
	return "unknown"
}

func getSortedCoins(coinSet map[string]bool) []string {
	coins := make([]string, 0, len(coinSet))
	for coin := range coinSet {
		coins = append(coins, coin)
	}
	sort.Strings(coins)
	return coins
}

func writeValidationSummary(logPath string, totalTxs int, totalDates int, coins []string) {
	logFile, err := os.Create(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create validation log: %v\\n", err)
		return
	}
	defer logFile.Close()

	fmt.Fprintf(logFile, "=== Exchange Rates Template Validation Summary ===\\n")
	fmt.Fprintf(logFile, "Generated at: %s\\n\\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(logFile, "Summary Statistics:\\n")
	fmt.Fprintf(logFile, "  Total in_other/out_other transactions: %d\\n", totalTxs)
	fmt.Fprintf(logFile, "  Unique dates requiring rates: %d\\n", totalDates)
	fmt.Fprintf(logFile, "  Coins discovered: %d (%s)\\n", len(coins), strings.Join(coins, ", "))
	fmt.Fprintf(logFile, "Template File Structure:\\n")
	fmt.Fprintf(logFile, "  For each date, you need to fill in:\\n")
	fmt.Fprintf(logFile, "    - usd_prices.btc: Bitcoin price in USD\\n")
	fmt.Fprintf(logFile, "    - usd_prices.eth: Ethereum price in USD\\n")
	fmt.Fprintf(logFile, "    - usd_prices.xrp: Ripple price in USD\\n")
	fmt.Fprintf(logFile, "    - usd_prices.ltc: Litecoin price in USD\\n")
	fmt.Fprintf(logFile, "    - usd_to_zar_rate: USD to ZAR exchange rate\\n")
	fmt.Fprintf(logFile, "Usage Instructions:\\n")
	fmt.Fprintf(logFile, "  1. Look up historical prices for each date\\n")
	fmt.Fprintf(logFile, "  2. Fill in the usd_prices and usd_to_zar_rate fields\\n")
	fmt.Fprintf(logFile, "  3. Save the updated YAML file\\n")
	fmt.Fprintf(logFile, "  4. Run Part 2 script to generate exchange.csv\\n")
}
