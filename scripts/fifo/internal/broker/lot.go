package broker

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/beerguevara/antcrypto/fifo/internal/exchangerate"
)

type BrokerLot struct {
	// Identification
	Reference      string // Unique broker lot reference (e.g., "broker_xbt_abc123")
	OriginalTxRef  string // Reference of out_other transaction that created this
	OriginalLotRef string // Original FIFO lot(s) reference

	// Entry data (set when created from out_other)
	CoinIn         string          // Coin entering broker (e.g., "xbt")
	UnitsIn        decimal.Decimal // Units sent to broker
	TimestampEntry int64           // Unix timestamp when entered broker

	// Value calculations (at entry time)
	ZARValueRaw        decimal.Decimal // ZAR ValueAmount from raw CSV (out_other)
	ZARValueCalculated decimal.Decimal // ZAR calculated using exchange rates
	ZARDifference      decimal.Decimal // Difference (Raw - Calculated)
	ExchangeRateUsed   decimal.Decimal // Exchange rate used (ZAR per unit)

	// Exit data (set when consumed by in_other)
	CoinOut       string          // Coin leaving broker (empty if same, or "eth" if converted)
	UnitsOut      decimal.Decimal // Units returned from broker (cumulative)
	TimestampExit int64           // Unix timestamp when exited broker
	ExitRef       string          // Reference of in_other transaction that returned it
	CrossRate     decimal.Decimal // Cross-rate if coin conversion (at entry time)

	// Status tracking
	IsFullyReturned bool     // True when UnitsOut >= UnitsIn
	PartialExitLots []string // References to partial exit lot records (for audit)
}

type BrokerPool struct {
	Coin string
	Lots []*BrokerLot
}

type BrokerPoolManager struct {
	Pools          map[string]*BrokerPool
	templateLoader *exchangerate.TemplateLoader
}

func NewBrokerPoolManager() *BrokerPoolManager {
	return &BrokerPoolManager{
		Pools: make(map[string]*BrokerPool),
	}
}

func (bpm *BrokerPoolManager) SetTemplateLoader(tl *exchangerate.TemplateLoader) {
	bpm.templateLoader = tl
}

func (bpm *BrokerPoolManager) InitializePools() {
	bpm.Pools["xbt"] = &BrokerPool{Coin: "xbt", Lots: []*BrokerLot{}}
	bpm.Pools["eth"] = &BrokerPool{Coin: "eth", Lots: []*BrokerLot{}}
	bpm.Pools["xrp"] = &BrokerPool{Coin: "xrp", Lots: []*BrokerLot{}}
	bpm.Pools["ltc"] = &BrokerPool{Coin: "ltc", Lots: []*BrokerLot{}}
}

func (bpm *BrokerPoolManager) GetCoinPool(coin string) *BrokerPool {
	return bpm.Pools[coin]
}

func generateHash() string {
	hash := sha1.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	hashStr := hex.EncodeToString(hash[:7])
	return hashStr
}

func sumQuantities(lots []*BrokerLot) decimal.Decimal {
	total := decimal.Zero
	for _, lot := range lots {
		total = total.Add(lot.UnitsIn)
	}
	return total
}
