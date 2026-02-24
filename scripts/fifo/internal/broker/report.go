package broker

import (
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/beerguevara/antcrypto/fifo/internal/classifier"
)

type BrokerReportGenerator struct {
	rows []BrokerRow
}

func NewBrokerReportGenerator() *BrokerReportGenerator {
	return &BrokerReportGenerator{
		rows: []BrokerRow{},
	}
}

func (brg *BrokerReportGenerator) AddEntryRow(brokerLot *BrokerLot, tx *classifier.Transaction) {
	coinOut := brokerLot.CoinOut
	if coinOut == "" {
		coinOut = brokerLot.CoinIn
	}

	brg.rows = append(brg.rows, BrokerRow{
		Date:               time.Unix(brokerLot.TimestampEntry, 0).Format("2006-01-02 15:04:05"),
		Type:               "ENTRY",
		OriginalTxRef:      tx.Reference,
		BrokerLotRef:       brokerLot.Reference,
		OriginalLotRef:     brokerLot.OriginalLotRef,
		CoinIn:             brokerLot.CoinIn,
		UnitsIn:            brokerLot.UnitsIn,
		CoinOut:            coinOut,
		UnitsOut:           decimal.Zero,
		ZARValueRaw:        brokerLot.ZARValueRaw,
		ZARValueCalculated: brokerLot.ZARValueCalculated,
		ZARDifference:      brokerLot.ZARDifference,
		UnitCost:           decimal.Zero,
		ExchangeRate:       brokerLot.ExchangeRateUsed,
		CrossRate:          decimal.Zero,
		Notes:              "",
	})
}

func (brg *BrokerReportGenerator) AddExitRows(brokerLot *BrokerLot, tx *classifier.Transaction, unitCost decimal.Decimal) {
	coinOut := brokerLot.CoinOut
	if coinOut == "" {
		coinOut = brokerLot.CoinIn
	}

	brg.rows = append(brg.rows, BrokerRow{
		Date:               time.Unix(brokerLot.TimestampExit, 0).Format("2006-01-02 15:04:05"),
		Type:               "EXIT",
		OriginalTxRef:      tx.Reference,
		BrokerLotRef:       brokerLot.Reference,
		OriginalLotRef:     brokerLot.OriginalLotRef,
		CoinIn:             brokerLot.CoinIn,
		UnitsIn:            brokerLot.UnitsIn,
		CoinOut:            coinOut,
		UnitsOut:           brokerLot.UnitsOut,
		ZARValueRaw:        brokerLot.ZARValueRaw,
		ZARValueCalculated: brokerLot.ZARValueCalculated,
		ZARDifference:      brokerLot.ZARDifference,
		UnitCost:           unitCost,
		ExchangeRate:       decimal.Zero,
		CrossRate:          brokerLot.CrossRate,
		Notes: fmt.Sprintf("Returned from broker using %s",
			time.Unix(brokerLot.TimestampEntry, 0).Format("2006-01-02")),
	})
}

func (brg *BrokerReportGenerator) GenerateCSV() [][]string {
	sort.Slice(brg.rows, func(i, j int) bool {
		return brg.rows[i].Date < brg.rows[j].Date
	})

	var records [][]string

	headers := []string{
		"Date", "Type", "Original Tx Ref", "Broker Lot Ref",
		"Original Lot Ref", "Coin In", "Units In",
		"ZAR Value Raw", "ZAR Value Calc", "ZAR Difference",
		"Coin Out", "Units Out", "Unit Cost",
		"Exchange Rate", "Cross Rate", "Notes",
	}
	records = append(records, headers)

	for _, row := range brg.rows {
		record := []string{
			row.Date,
			row.Type,
			row.OriginalTxRef,
			row.BrokerLotRef,
			row.OriginalLotRef,
			row.CoinIn,
			row.UnitsIn.StringFixed(8),
			row.ZARValueRaw.StringFixed(2),
			row.ZARValueCalculated.StringFixed(2),
			row.ZARDifference.StringFixed(2),
			row.CoinOut,
			row.UnitsOut.StringFixed(8),
			row.UnitCost.StringFixed(2),
			row.ExchangeRate.StringFixed(8),
			row.CrossRate.StringFixed(8),
			row.Notes,
		}
		records = append(records, record)
	}

	return records
}
