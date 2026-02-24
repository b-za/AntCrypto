package exchangerate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

type TemplateLoader struct {
	templatePath string
	templateData *RatesTemplate
}

func NewTemplateLoader() *TemplateLoader {
	return &TemplateLoader{}
}

func (tl *TemplateLoader) FindLatestTemplatePath(rootPath string) (string, error) {
	templatesDir := filepath.Join(rootPath, "exchange_templates")

	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return "", fmt.Errorf("no exchange_templates directory found at %s: %w", templatesDir, err)
	}

	var timestampDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			timestampDirs = append(timestampDirs, entry.Name())
		}
	}

	if len(timestampDirs) == 0 {
		return "", fmt.Errorf("no timestamp directories found in exchange_templates")
	}

	sort.Slice(timestampDirs, func(i, j int) bool {
		return timestampDirs[i] > timestampDirs[j] // Newest first
	})

	latestPath := filepath.Join(templatesDir, timestampDirs[0], "exchange_rates_template.yaml")
	return latestPath, nil
}

func (tl *TemplateLoader) LoadLatestTemplate(rootPath string) error {
	templatePath, err := tl.FindLatestTemplatePath(rootPath)
	if err != nil {
		return err
	}

	tl.templatePath = templatePath

	data, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	if err := yaml.Unmarshal(data, &tl.templateData); err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	return nil
}

func (tl *TemplateLoader) GetRatesForDate(date string) (*DateRates, bool) {
	if tl.templateData == nil {
		return nil, false
	}

	for i := range tl.templateData.DatesRequired {
		if tl.templateData.DatesRequired[i].Date == date {
			return &tl.templateData.DatesRequired[i], true
		}
	}

	return nil, false
}

func (tl *TemplateLoader) GetCrossRate(date string, fromCoin, toCoin string) (decimal.Decimal, error) {
	dateRates, found := tl.GetRatesForDate(date)
	if !found {
		return decimal.Zero, fmt.Errorf("exchange rates not found for date: %s", date)
	}

	fromPriceStr, fromExists := dateRates.USDPrices[fromCoin]
	toPriceStr, toExists := dateRates.USDPrices[toCoin]

	if !fromExists || !toExists {
		return decimal.Zero, fmt.Errorf("price not found for coins: %s→%s on %s", fromCoin, toCoin, date)
	}

	fromPrice, err := decimal.NewFromString(fromPriceStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid price for %s: %w", fromCoin, err)
	}

	toPrice, err := decimal.NewFromString(toPriceStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid price for %s: %w", toCoin, err)
	}

	if toPrice.IsZero() {
		return decimal.Zero, fmt.Errorf("zero price for %s", toCoin)
	}

	// Cross-rate = FromPrice / ToPrice
	// Example: BTC $4565.30 / ETH $378.49 = 12.06 ETH per BTC
	return fromPrice.Div(toPrice), nil
}

func (tl *TemplateLoader) GetZARRate(date string) (decimal.Decimal, error) {
	dateRates, found := tl.GetRatesForDate(date)
	if !found {
		return decimal.Zero, fmt.Errorf("exchange rates not found for date: %s", date)
	}

	rate, err := decimal.NewFromString(dateRates.USDToZARRate)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid USD to ZAR rate: %w", err)
	}

	return rate, nil
}

func (tl *TemplateLoader) GetCoinPrice(date string, coin string) (decimal.Decimal, error) {
	dateRates, found := tl.GetRatesForDate(date)
	if !found {
		return decimal.Zero, fmt.Errorf("exchange rates not found for date: %s", date)
	}

	priceStr, exists := dateRates.USDPrices[coin]
	if !exists {
		return decimal.Zero, fmt.Errorf("price not found for coin %s on %s", coin, date)
	}

	price, err := decimal.NewFromString(priceStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid price for %s: %w", coin, err)
	}

	return price, nil
}
