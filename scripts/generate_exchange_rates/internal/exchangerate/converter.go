package exchangerate

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

func ConvertUSDToZAR(usdValue decimal.Decimal, usdToZARRate decimal.Decimal) decimal.Decimal {
	return usdValue.Mul(usdToZARRate)
}

func ConvertCryptoToZAR(cryptoValue decimal.Decimal, cryptoUSDRate decimal.Decimal, usdToZARRate decimal.Decimal) decimal.Decimal {
	return cryptoValue.Mul(cryptoUSDRate).Mul(usdToZARRate)
}

func ConvertZARToCrypto(zarValue decimal.Decimal, cryptoZARRate decimal.Decimal) decimal.Decimal {
	if cryptoZARRate.IsZero() {
		return decimal.Zero
	}
	return zarValue.Div(cryptoZARRate)
}

func FetchAllRatesForDate(date time.Time, coins []string, cache *RateCache, maxFallbackDays int) (map[string]CoinRate, ZARExchangeRate, []error) {
	rates := make(map[string]CoinRate)
	var errors []error

	usdToZARRate, usdToZARDate, _, usdToZARErr := FetchWithFallback(date, func(d time.Time) (decimal.Decimal, time.Time, error) {
		rate, rateDate, err := FetchFrankfurterRate(d)
		return rate, rateDate, err
	}, maxFallbackDays)

	if usdToZARErr != nil {
		errors = append(errors, fmt.Errorf("USD→ZAR rate: %w", usdToZARErr))
	}

	zarExchange := ZARExchangeRate{
		USDToZARRate:   usdToZARRate,
		USDToZARDate:   usdToZARDate.Format("2006-01-02"),
		USDToZARSource: "Frankfurter",
	}

	if usdToZARErr != nil && usdToZARDate.IsZero() {
		zarExchange.USDToZARDate = "N/A"
	}

	for _, coin := range coins {
		cached, exists := cache.Get(coin, date)
		if exists {
			zarRate := cached.Value.Mul(usdToZARRate)
			rates[coin] = CoinRate{
				ZARPerUnit:  zarRate,
				UnitsForZAR: ConvertZARToCrypto(zarRate, zarRate),
				RateDate:    cached.Date.Format("2006-01-02"),
				RateSource:  cached.Source,
				Note:        "From cache",
			}
			continue
		}

		usdRate, rateDate, note, err := FetchWithFallback(date, func(d time.Time) (decimal.Decimal, time.Time, error) {
			rate, rateDate, _, err := FetchBinanceRate(coin, d)
			return rate, rateDate, err
		}, maxFallbackDays)

		if err != nil {
			errors = append(errors, fmt.Errorf("%s rate: %w", coin, err))
			rates[coin] = CoinRate{
				ZARPerUnit:  decimal.Zero,
				UnitsForZAR: decimal.Zero,
				RateDate:    "N/A",
				RateSource:  "N/A",
				Note:        "Error fetching rate",
			}
			continue
		}

		cache.Set(coin, rateDate, usdRate, "Binance")

		zarRate := usdRate.Mul(usdToZARRate)
		rates[coin] = CoinRate{
			ZARPerUnit:  zarRate,
			UnitsForZAR: ConvertZARToCrypto(zarRate, zarRate),
			RateDate:    rateDate.Format("2006-01-02"),
			RateSource:  "Binance",
			Note:        note,
		}
	}

	return rates, zarExchange, errors
}
