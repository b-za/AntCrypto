package pool

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/beerguevara/antcrypto/fifo/internal/classifier"
)

type Lot struct {
	Reference string
	Quantity  decimal.Decimal
	UnitCost  decimal.Decimal
	Timestamp int64
	Pool      string
}

type Pool struct {
	Name string
	Lots []*Lot
}

type PoolManager struct {
	pools map[string]*Pool
}

func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools: make(map[string]*Pool),
	}
}

func (pm *PoolManager) InitializePools() {
	pm.pools["in_buy"] = &Pool{Name: "in_buy", Lots: []*Lot{}}
	pm.pools["in_buy_for_other"] = &Pool{Name: "in_buy_for_other", Lots: []*Lot{}}
	pm.pools["in_other"] = &Pool{Name: "in_other", Lots: []*Lot{}}
}

func (pm *PoolManager) AddLot(poolName string, quantity, unitCost decimal.Decimal, timestamp int64, rawLine string) error {
	pool, exists := pm.pools[poolName]
	if !exists {
		return fmt.Errorf("pool %s does not exist", poolName)
	}

	lot := &Lot{
		Reference: GenerateLotReference(poolName, rawLine),
		Quantity:  quantity,
		UnitCost:  unitCost,
		Timestamp: timestamp,
		Pool:      poolName,
	}

	pool.Lots = append(pool.Lots, lot)
	return nil
}

func (pm *PoolManager) AddLotWithReference(poolName string, quantity, unitCost decimal.Decimal, timestamp int64, lotRef string) error {
	pool, exists := pm.pools[poolName]
	if !exists {
		return fmt.Errorf("pool %s does not exist", poolName)
	}

	lot := &Lot{
		Reference: lotRef,
		Quantity:  quantity,
		UnitCost:  unitCost,
		Timestamp: timestamp,
		Pool:      poolName,
	}

	pool.Lots = append(pool.Lots, lot)
	return nil
}

func GenerateLotReference(poolName string, rawLine string) string {
	hash := sha1.Sum([]byte(rawLine))
	hashStr := hex.EncodeToString(hash[:7])
	return fmt.Sprintf("%s_%s", poolName, hashStr)
}

func (pm *PoolManager) Consume(quantityNeeded decimal.Decimal, priority []string) ([]Lot, decimal.Decimal, error) {
	var consumedLots []Lot
	var remainingQty = quantityNeeded

	for _, poolName := range priority {
		pool, exists := pm.pools[poolName]
		if !exists || len(pool.Lots) == 0 {
			continue
		}

		for remainingQty.IsPositive() && len(pool.Lots) > 0 {
			lot := pool.Lots[0]

			if lot.Quantity.GreaterThanOrEqual(remainingQty) {
				consumedLot := &Lot{
					Reference: lot.Reference,
					Quantity:  remainingQty,
					UnitCost:  lot.UnitCost,
					Timestamp: lot.Timestamp,
					Pool:      lot.Pool,
				}
				consumedLots = append(consumedLots, *consumedLot)

				lot.Quantity = lot.Quantity.Sub(remainingQty)
				remainingQty = decimal.Zero

				if lot.Quantity.IsZero() {
					pool.Lots = pool.Lots[1:]
				}
				break
			} else {
				consumedLots = append(consumedLots, *lot)
				remainingQty = remainingQty.Sub(lot.Quantity)
				pool.Lots = pool.Lots[1:]
			}
		}

		if remainingQty.IsZero() {
			break
		}
	}

	if remainingQty.IsPositive() {
		return consumedLots, remainingQty, fmt.Errorf("insufficient lots across all pools, still need %s", remainingQty.String())
	}

	return consumedLots, decimal.Zero, nil
}

func (pm *PoolManager) ConsumeFromSpecificLot(lotReference string, quantityNeeded decimal.Decimal) (*Lot, decimal.Decimal, error) {
	for _, pool := range pm.pools {
		for i, lot := range pool.Lots {
			if lot.Reference == lotReference {
				if lot.Quantity.GreaterThanOrEqual(quantityNeeded) {
					consumedLot := &Lot{
						Reference: lot.Reference,
						Quantity:  quantityNeeded,
						UnitCost:  lot.UnitCost,
						Timestamp: lot.Timestamp,
						Pool:      lot.Pool,
					}

					lot.Quantity = lot.Quantity.Sub(quantityNeeded)
					if lot.Quantity.IsZero() {
						pool.Lots = append(pool.Lots[:i], pool.Lots[i+1:]...)
					}

					return consumedLot, decimal.Zero, nil
				} else {
					consumedLot := &Lot{
						Reference: lot.Reference,
						Quantity:  lot.Quantity,
						UnitCost:  lot.UnitCost,
						Timestamp: lot.Timestamp,
						Pool:      lot.Pool,
					}

					remaining := quantityNeeded.Sub(lot.Quantity)
					pool.Lots = append(pool.Lots[:i], pool.Lots[i+1:]...)

					return consumedLot, remaining, nil
				}
			}
		}
	}

	return nil, quantityNeeded, fmt.Errorf("lot with reference %s not found", lotReference)
}

func (pm *PoolManager) GetPoolBalance(poolName string) decimal.Decimal {
	pool, exists := pm.pools[poolName]
	if !exists {
		return decimal.Zero
	}

	var balance decimal.Decimal
	for _, lot := range pool.Lots {
		balance = balance.Add(lot.Quantity)
	}
	return balance
}

func (pm *PoolManager) GetTotalBalance() decimal.Decimal {
	var total decimal.Decimal
	for _, poolName := range []string{"in_buy", "in_buy_for_other", "in_other"} {
		total = total.Add(pm.GetPoolBalance(poolName))
	}
	return total
}

func GetPriorityForTransactionType(txType classifier.TransactionType) []string {
	switch txType {
	case classifier.OutflowSell:
		return []string{"in_buy", "in_buy_for_other", "in_other"}
	case classifier.OutflowOther:
		return []string{"in_buy_for_other", "in_other", "in_buy"}
	case classifier.OutflowFeeBuy:
		return []string{"in_buy", "in_buy_for_other", "in_other"}
	case classifier.OutflowFeeBuyForOther:
		return []string{"in_buy_for_other", "in_other", "in_buy"}
	case classifier.OutflowFeeInOther:
		return []string{"in_other", "in_buy_for_other", "in_buy"}
	case classifier.OutflowFeeSell:
		return []string{"in_buy", "in_buy_for_other", "in_other"}
	case classifier.OutflowFeeOutOther:
		return []string{"in_buy_for_other", "in_other", "in_buy"}
	default:
		return []string{"in_buy", "in_buy_for_other", "in_other"}
	}
}
