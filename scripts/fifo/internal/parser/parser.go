package parser

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Transaction struct {
	WalletID              string
	Row                   int
	Timestamp             time.Time
	Description           string
	Currency              string
	BalanceDelta          decimal.Decimal
	ValueAmount           decimal.Decimal
	Reference             string
	CryptocurrencyAddress string
	CryptocurrencyTxID    string
}

type CSVReader struct {
	filePath string
}

func NewCSVReader(filePath string) *CSVReader {
	return &CSVReader{filePath: filePath}
}

func (r *CSVReader) ReadAll() ([]Transaction, error) {
	file, err := os.Open(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file has no data rows")
	}

	var transactions []Transaction

	for i, record := range records {
		if i == 0 {
			continue // Skip header row
		}

		tx, err := r.parseRecord(record)
		if err != nil {
			return nil, fmt.Errorf("failed to parse row %d: %w", i+1, err)
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func (r *CSVReader) parseRecord(record []string) (Transaction, error) {
	if len(record) < 12 {
		return Transaction{}, fmt.Errorf("record has %d columns, expected at least 12", len(record))
	}

	row, err := strconv.Atoi(record[1])
	if err != nil {
		return Transaction{}, fmt.Errorf("invalid row number: %w", err)
	}

	timestamp, err := time.Parse("2006-01-02 15:04:05", record[2])
	if err != nil {
		return Transaction{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	balanceDelta, err := decimal.NewFromString(record[5])
	if err != nil {
		return Transaction{}, fmt.Errorf("invalid balance delta: %w", err)
	}

	valueAmount, err := decimal.NewFromString(record[12])
	if err != nil {
		return Transaction{}, fmt.Errorf("invalid value amount: %w", err)
	}

	tx := Transaction{
		WalletID:     record[0],
		Row:          row,
		Timestamp:    timestamp,
		Description:  strings.TrimSpace(record[3]),
		Currency:     record[4],
		BalanceDelta: balanceDelta,
		ValueAmount:  valueAmount,
		Reference:    record[13],
	}

	if len(record) > 8 && record[8] != "" {
		tx.CryptocurrencyTxID = record[8]
	}

	if len(record) > 9 && record[9] != "" {
		tx.CryptocurrencyAddress = record[9]
	}

	return tx, nil
}

func WriteCSV(filePath string, records [][]string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	return nil
}

type ErrorLogger interface {
	LogError(err ErrorEntry)
}

type ErrorEntry struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Coin      string `json:"coin"`
	Row       int    `json:"row"`
	Timestamp string `json:"timestamp"`
	Details   string `json:"details,omitempty"`
}

type JSONLErrorLogger struct {
	file *os.File
}

func NewJSONLErrorLogger(filePath string) (*JSONLErrorLogger, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open error log file: %w", err)
	}

	return &JSONLErrorLogger{file: file}, nil
}

func (l *JSONLErrorLogger) LogError(entry ErrorEntry) {
	jsonEntry := fmt.Sprintf(`{"type":"%s","message":"%s","coin":"%s","row":%d,"timestamp":"%s"`,
		entry.Type, entry.Message, entry.Coin, entry.Row, entry.Timestamp)
	if entry.Details != "" {
		jsonEntry += fmt.Sprintf(`,"details":"%s"`, entry.Details)
	}
	jsonEntry += "}\n"
	l.file.WriteString(jsonEntry)
}

func (l *JSONLErrorLogger) Close() error {
	return l.file.Close()
}
