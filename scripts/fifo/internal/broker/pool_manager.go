package broker

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/beerguevara/antcrypto/fifo/internal/classifier"
	"github.com/beerguevara/antcrypto/fifo/internal/pool"
)

func (bpm *BrokerPoolManager) AddToBrokerPool(
	tx *classifier.Transaction,
	consumedLots []pool.Lot,
) ([]*BrokerLot, error) {

	dateStr := time.Unix(tx.Timestamp, 0).Format("2006-01-02")

	// Get exchange rates for this date
	usdPrice, err := bpm.templateLoader.GetCoinPrice(dateStr, tx.Currency)
	if err != nil {
		return nil, fmt.Errorf("USD price not found for coin %s on %s: %w", tx.Currency, dateStr, err)
	}

	usdToZARRate, err := bpm.templateLoader.GetZARRate(dateStr)
	if err != nil {
		return nil, fmt.Errorf("USD to ZAR rate not found on %s: %w", dateStr, err)
	}

	// Calculate ZAR per unit using exchange rates
	zarPerUnit := usdPrice.Mul(usdToZARRate)
	zarCalculated := tx.BalanceDelta.Abs().Mul(zarPerUnit)

	// Calculate difference
	zarDifference := tx.ValueAmount.Abs().Sub(zarCalculated)

	var brokerLots []*BrokerLot

	// Create ONE broker lot per original lot consumed
	for _, lot := range consumedLots {
		brokerRef := fmt.Sprintf("broker_%s_%s", tx.Currency, generateHash())

		// If multiple lots, split ZAR values proportionally
		var zarValueRaw, zarValueCalc, zarDiff decimal.Decimal
		if len(consumedLots) > 1 {
			proportion := lot.Quantity.Div(tx.BalanceDelta.Abs())
			zarValueRaw = tx.ValueAmount.Abs().Mul(proportion)
			zarValueCalc = zarCalculated.Mul(proportion)
			zarDiff = zarDifference.Mul(proportion)
		} else {
			zarValueRaw = tx.ValueAmount.Abs()
			zarValueCalc = zarCalculated
			zarDiff = zarDifference
		}

		brokerLot := &BrokerLot{
			Reference:      brokerRef,
			OriginalTxRef:  tx.Reference, // All lots from same out_other share this
			OriginalLotRef: lot.Reference,

			CoinIn:         tx.Currency,
			UnitsIn:        lot.Quantity,
			TimestampEntry: tx.Timestamp,

			ZARValueRaw:        zarValueRaw,
			ZARValueCalculated: zarValueCalc,
			ZARDifference:      zarDiff,
			ExchangeRateUsed:   zarPerUnit,
		}

		// Add to appropriate broker pool
		bpm.GetCoinPool(tx.Currency).Lots = append(
			bpm.GetCoinPool(tx.Currency).Lots,
			brokerLot,
		)

		brokerLots = append(brokerLots, brokerLot)
	}

	return brokerLots, nil
}

func (bpm *BrokerPoolManager) ConsumeFromBroker(
	timestamp int64,
	coinNeeded string, // Coin being returned (e.g., "eth")
	unitsNeeded decimal.Decimal,
) ([]*BrokerLot, error) {

	var usedLots []*BrokerLot
	var remainingUnits = unitsNeeded

	// Priority 1: Try to consume from same coin pool (FIFO)
	sameCoinPool := bpm.GetCoinPool(coinNeeded)
	sameCoinLots, _ := consumeFromPool(sameCoinPool, remainingUnits, timestamp)
	if len(sameCoinLots) > 0 {
		usedLots = append(usedLots, sameCoinLots...)
		remainingUnits = remainingUnits.Sub(sumQuantities(sameCoinLots))
	}

	// Priority 2: If still need units, consume from other pools with conversion
	if remainingUnits.IsPositive() {
		dateEntry := time.Unix(timestamp, 0).Format("2006-01-02")

		// Try any other coin pool that has lots (any order)
		for coinName, pool := range bpm.Pools {
			if coinName == coinNeeded || len(pool.Lots) == 0 {
				continue // Skip target coin pool (already tried) or empty
			}

			crossRate, err := bpm.templateLoader.GetCrossRate(dateEntry, coinName, coinNeeded)
			if err != nil {
				continue // Try next coin pool
			}

			// Calculate target units needed from this source pool
			targetUnitsNeeded := remainingUnits.Div(crossRate)

			// Consume from this pool
			convertedLots, _ := consumeFromPool(pool, targetUnitsNeeded, timestamp)

			if len(convertedLots) > 0 {
				// Set conversion info on used lots
				for _, lot := range convertedLots {
					lot.CoinOut = coinNeeded
					lot.CrossRate = crossRate

					// Track partial exit references for audit
					if len(lot.PartialExitLots) == 0 {
						lot.PartialExitLots = []string{}
					}
					exitRef := fmt.Sprintf("exit_%d", time.Now().UnixNano())
					lot.PartialExitLots = append(lot.PartialExitLots, exitRef)
				}

				usedLots = append(usedLots, convertedLots...)
				remainingUnits = remainingUnits.Sub(sumQuantities(convertedLots))

				if remainingUnits.IsZero() {
					break
				}
			}
		}
	}

	if remainingUnits.IsPositive() {
		return usedLots, fmt.Errorf("insufficient broker lots: still need %s of %s", remainingUnits.String(), coinNeeded)
	}

	// Mark lots as fully returned if exhausted
	for _, lot := range usedLots {
		if lot.UnitsOut.GreaterThanOrEqual(lot.UnitsIn) {
			lot.IsFullyReturned = true
		}
	}

	return usedLots, nil
}

func consumeFromPool(
	pool *BrokerPool,
	unitsNeeded decimal.Decimal,
	timestamp int64,
) ([]*BrokerLot, decimal.Decimal) {

	var partialLots []*BrokerLot
	var remainingQty = unitsNeeded

	for remainingQty.IsPositive() && len(pool.Lots) > 0 {
		lot := pool.Lots[0] // FIFO: oldest first

		if lot.UnitsIn.GreaterThanOrEqual(remainingQty) {
			// Partial consumption - create independent EXIT record
			// But keep original lot in pool with reduced units
			partialInfo := &BrokerLot{
				Reference:      lot.Reference, // Same broker lot reference
				OriginalTxRef:  lot.OriginalTxRef,
				OriginalLotRef: lot.OriginalLotRef,

				CoinIn:         lot.CoinIn,
				UnitsIn:        decimal.Zero, // Empty for EXIT row
				TimestampEntry: lot.TimestampEntry,

				ZARValueRaw:        decimal.Zero, // Empty for EXIT row
				ZARValueCalculated: decimal.Zero, // Empty for EXIT row
				ZARDifference:      decimal.Zero, // Empty for EXIT row

				CoinOut:  lot.CoinOut,
				UnitsOut: remainingQty, // Amount being returned now

				ExchangeRateUsed: lot.ExchangeRateUsed,
				CrossRate:        lot.CrossRate,
			}

			partialLots = append(partialLots, partialInfo)

			// Reduce original lot's units (keep in pool)
			lot.UnitsIn = lot.UnitsIn.Sub(remainingQty)
			lot.UnitsOut = lot.UnitsOut.Add(remainingQty)

			// Check if now fully returned
			if lot.UnitsOut.GreaterThanOrEqual(lot.UnitsIn) {
				lot.IsFullyReturned = true
			}

			remainingQty = decimal.Zero

			if lot.UnitsIn.IsZero() {
				pool.Lots = pool.Lots[1:] // Remove from pool
			}
		} else {
			// Full consumption - use existing lot directly for EXIT row
			partialLots = append(partialLots, lot)

			// Update original lot
			lot.UnitsOut = lot.UnitsOut.Add(lot.UnitsIn)
			lot.IsFullyReturned = true

			remainingQty = remainingQty.Sub(lot.UnitsIn)
			pool.Lots = pool.Lots[1:]
		}
	}

	return partialLots, unitsNeeded.Sub(remainingQty)
}

func (bpm *BrokerPoolManager) CalculateWeightedUnitCost(
	lots []*BrokerLot,
	totalUnitsNeeded decimal.Decimal,
) decimal.Decimal {

	var totalWeightedCost decimal.Decimal

	for _, lot := range lots {
		// Calculate cost for this lot's contribution
		// Cost = (ZAR Value When Entered) / Units Returned

		var zarValueAtEntry decimal.Decimal

		if lot.CoinOut != "" && lot.CoinOut != lot.CoinIn {
			// Conversion: Use cross-rate to get ZAR value at entry
			zarValueAtEntry = lot.UnitsIn.Mul(lot.ExchangeRateUsed).Mul(lot.CrossRate)
		} else {
			// No conversion: Use calculated ZAR value directly
			zarValueAtEntry = lot.ZARValueCalculated
		}

		totalWeightedCost = totalWeightedCost.Add(zarValueAtEntry)
	}

	// Weighted unit cost = Total ZAR Value / Total Units Returned
	return totalWeightedCost.Div(totalUnitsNeeded)
}

func (bpm *BrokerPoolManager) RemoveFullyReturnedLots() {
	for _, pool := range bpm.Pools {
		var remaining []*BrokerLot
		for _, lot := range pool.Lots {
			if !lot.IsFullyReturned {
				remaining = append(remaining, lot)
			}
		}
		pool.Lots = remaining
	}
}
