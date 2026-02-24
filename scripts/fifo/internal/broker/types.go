package broker

import (
	"fmt"
	"os"
	"strings"

	"github.com/shopspring/decimal"
)

type BrokerRow struct {
	Date string
	Type string // "ENTRY" or "EXIT"

	// References
	OriginalTxRef string // Reference from raw CSV (out_other or in_other)
	BrokerLotRef  string // Unique broker lot reference

	// Entry data
	OriginalLotRef string          // Original FIFO lot(s) reference
	CoinIn         string          // Coin entering broker
	UnitsIn        decimal.Decimal // Units sent to broker

	// Exit data
	CoinOut  string          // Coin leaving broker (empty if same coin)
	UnitsOut decimal.Decimal // Units returned from broker (0 for ENTRY rows)

	// Value data
	ZARValueRaw        decimal.Decimal // ZAR from raw CSV
	ZARValueCalculated decimal.Decimal // ZAR from exchange rate calculation
	ZARDifference      decimal.Decimal // Difference (Raw - Calculated)

	// Cost data (for EXIT rows only)
	UnitCost decimal.Decimal // Custom unit cost calculated (blank for ENTRY)

	// Rate data
	ExchangeRate decimal.Decimal // Exchange rate used (at entry time, blank for EXIT)
	CrossRate    decimal.Decimal // Cross-rate if coin converted (at entry time)

	Notes string // Additional information
}

type BrokerError struct {
	Type        string // "MISSING_RATE", "NO_BROKER_LOTS", "TEMPLATE_ERROR"
	Message     string
	Date        string
	Coin        string
	Transaction string
	Severity    string // "WARN", "ERROR", "FATAL"
}

func LogAndAskUser(errors []BrokerError) bool {
	for _, err := range errors {
		logMessage := ""
		switch err.Severity {
		case "FATAL":
			logMessage = fmt.Sprintf("[FATAL] %s", err.Message)
		case "ERROR":
			logMessage = fmt.Sprintf("[ERROR] %s", err.Message)
		case "WARN":
			logMessage = fmt.Sprintf("[WARN] %s", err.Message)
		default:
			logMessage = fmt.Sprintf("[INFO] %s", err.Message)
		}
		if err.Date != "" {
			logMessage = fmt.Sprintf("%s (Date: %s)", logMessage, err.Date)
		}
		if err.Coin != "" {
			logMessage = fmt.Sprintf("%s (Coin: %s)", logMessage, err.Coin)
		}
		if err.Transaction != "" {
			logMessage = fmt.Sprintf("%s (Tx: %s)", logMessage, err.Transaction)
		}
		fmt.Fprintf(os.Stderr, "%s\n", logMessage)
	}

	if len(errors) == 0 || errors[0].Severity != "FATAL" {
		fmt.Fprint(os.Stderr, "Continue with defaults? (y/n): ")
		var response string
		fmt.Scanln(&response)
		return strings.ToLower(strings.TrimSpace(response)) == "y"
	}
	return false
}
