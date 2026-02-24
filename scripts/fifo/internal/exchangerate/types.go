package exchangerate

import (
	"time"

	"github.com/shopspring/decimal"
)

type ExchangeRateData struct {
	TransactionDate string              `yaml:"transaction_date"`
	TransactionRef  string              `yaml:"transaction_ref"`
	Direction       string              `yaml:"direction"`
	FromCoin        string              `yaml:"from_coin"`
	ZARValue        decimal.Decimal     `yaml:"zar_value"`
	Rates           map[string]CoinRate `yaml:"rates"`
	ZARExchange     ZARExchangeRate     `yaml:"zar_exchange"`
}

type CoinRate struct {
	ZARPerUnit  decimal.Decimal `yaml:"zar_per_unit"`
	UnitsForZAR decimal.Decimal `yaml:"units_for_zar"`
	RateDate    string          `yaml:"rate_date"`
	RateSource  string          `yaml:"rate_source"`
	Note        string          `yaml:"note"`
}

type ZARExchangeRate struct {
	USDToZARRate   decimal.Decimal `yaml:"usd_to_zar_rate"`
	USDToZARDate   string          `yaml:"usd_to_zar_date"`
	USDToZARSource string          `yaml:"usd_to_zar_source"`
}

type ExchangeRatesYAML struct {
	ExchangeRates []ExchangeRateData `yaml:"exchange_rates"`
}

type CachedRate struct {
	Value  decimal.Decimal
	Date   time.Time
	Source string
}

type RateCache struct {
	cache map[string]CachedRate
}

func NewRateCache() *RateCache {
	return &RateCache{
		cache: make(map[string]CachedRate),
	}
}

func (rc *RateCache) Get(coin string, date time.Time) (CachedRate, bool) {
	key := cacheKey(coin, date)
	rate, exists := rc.cache[key]
	return rate, exists
}

func (rc *RateCache) Set(coin string, date time.Time, value decimal.Decimal, source string) {
	key := cacheKey(coin, date)
	rc.cache[key] = CachedRate{
		Value:  value,
		Date:   date,
		Source: source,
	}
}

func cacheKey(coin string, date time.Time) string {
	return coin + "_" + date.Format("2006-01-02")
}

// Template types for loading exchange_rates_template.yaml
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
