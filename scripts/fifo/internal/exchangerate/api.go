package exchangerate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

const (
	binanceAPIBase     = "https://api.binance.com/api/v3/klines"
	frankfurterAPIBase = "https://api.frankfurter.app"
	requestTimeout     = 10 * time.Second
)

func FetchBinanceRate(coin string, date time.Time) (decimal.Decimal, time.Time, string, error) {
	client := &http.Client{Timeout: requestTimeout}

	startTime := date.Truncate(24 * time.Hour).UnixMilli()
	url := fmt.Sprintf("%s?symbol=%sUSDT&interval=1d&startTime=%d&limit=1",
		binanceAPIBase, normalizeCoinSymbol(coin), startTime)

	resp, err := client.Get(url)
	if err != nil {
		return decimal.Zero, time.Time{}, "", fmt.Errorf("Binance API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return decimal.Zero, time.Time{}, "", fmt.Errorf("Binance API returned status %d: %s", resp.StatusCode, string(body))
	}

	var data [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return decimal.Zero, time.Time{}, "", fmt.Errorf("failed to decode Binance response: %w", err)
	}

	if len(data) == 0 || len(data[0]) < 5 {
		return decimal.Zero, time.Time{}, "", fmt.Errorf("no data available from Binance for %s on %s", coin, date.Format("2006-01-02"))
	}

	closePrice, ok := data[0][4].(string)
	if !ok {
		return decimal.Zero, time.Time{}, "", fmt.Errorf("invalid close price format from Binance")
	}

	price, err := decimal.NewFromString(closePrice)
	if err != nil {
		return decimal.Zero, time.Time{}, "", fmt.Errorf("failed to parse Binance price: %w", err)
	}

	rateDate := time.UnixMilli(int64(data[0][0].(float64))).Truncate(24 * time.Hour)
	return price, rateDate, "Binance", nil
}

func FetchFrankfurterRate(date time.Time) (decimal.Decimal, time.Time, error) {
	client := &http.Client{Timeout: requestTimeout}

	dateStr := date.Format("2006-01-02")
	url := fmt.Sprintf("%s/%s?from=USD&to=ZAR", frankfurterAPIBase, dateStr)

	resp, err := client.Get(url)
	if err != nil {
		return decimal.Zero, time.Time{}, fmt.Errorf("Frankfurter API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return decimal.Zero, time.Time{}, fmt.Errorf("Frankfurter API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Rates struct {
			ZAR string `json:"ZAR"`
		} `json:"rates"`
		Date string `json:"date"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return decimal.Zero, time.Time{}, fmt.Errorf("failed to decode Frankfurter response: %w", err)
	}

	rate, err := decimal.NewFromString(result.Rates.ZAR)
	if err != nil {
		return decimal.Zero, time.Time{}, fmt.Errorf("failed to parse Frankfurter rate: %w", err)
	}

	rateDate, _ := time.Parse("2006-01-02", result.Date)
	return rate, rateDate, nil
}

func FetchWithFallback(originalDate time.Time, fetchFunc func(time.Time) (decimal.Decimal, time.Time, error), maxDays int) (decimal.Decimal, time.Time, string, error) {
	value, actualDate, err := fetchFunc(originalDate)
	if err == nil {
		return value, actualDate, "", nil
	}

	for days := 1; days <= maxDays; days++ {
		dateAfter := originalDate.AddDate(0, 0, days)
		value, actualDate, err := fetchFunc(dateAfter)
		if err == nil {
			note := fmt.Sprintf("Exact date unavailable, used nearest available (+%d days)", days)
			return value, actualDate, note, nil
		}

		dateBefore := originalDate.AddDate(0, 0, -days)
		value, actualDate, err = fetchFunc(dateBefore)
		if err == nil {
			note := fmt.Sprintf("Exact date unavailable, used nearest available (-%d days)", days)
			return value, actualDate, note, nil
		}
	}

	return decimal.Zero, time.Time{}, "", fmt.Errorf("no data available within ±%d days of %s", maxDays, originalDate.Format("2006-01-02"))
}

func normalizeCoinSymbol(coin string) string {
	switch coin {
	case "xbt":
		return "BTC"
	case "eth":
		return "ETH"
	case "xrp":
		return "XRP"
	case "ltc":
		return "LTC"
	default:
		return coin
	}
}
