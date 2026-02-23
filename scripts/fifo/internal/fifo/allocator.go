package fifo

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/beerguevara/antcrypto/fifo/internal/classifier"
	"github.com/beerguevara/antcrypto/fifo/internal/pool"
)

type Allocation struct {
	Transaction  *classifier.Transaction
	ConsumedLots []pool.Lot
	RemainingQty decimal.Decimal
}

type FIFOAllocator struct {
	poolManager *pool.PoolManager
	allocations []*Allocation

	// Track lots consumed by each specific transaction
	// Use tx.Row as unique key (since Reference may be shared)
	txToLots map[int][]pool.Lot
}

func NewFIFOAllocator(pm *pool.PoolManager) *FIFOAllocator {
	return &FIFOAllocator{
		poolManager: pm,
		allocations: []*Allocation{},
		txToLots:    make(map[int][]pool.Lot),
	}
}

func (fa *FIFOAllocator) AllocateTransaction(tx *classifier.Transaction, linkedLotRef *string) error {
	quantityNeeded := tx.BalanceDelta.Abs()

	var consumedLots []pool.Lot
	var remainingQty decimal.Decimal
	var err error

	if tx.Type == classifier.OutflowOther && linkedLotRef != nil {
		consumedLot, remaining, err := fa.poolManager.ConsumeFromSpecificLot(*linkedLotRef, quantityNeeded)
		if err != nil {
			return fmt.Errorf("failed to consume from specific lot %s: %w", *linkedLotRef, err)
		}

		consumedLots = append(consumedLots, *consumedLot)
		remainingQty = remaining

		if remainingQty.IsPositive() {
			priority := pool.GetPriorityForTransactionType(tx.Type)
			additionalLots, remaining, err := fa.poolManager.Consume(remainingQty, priority)
			if err != nil {
				return fmt.Errorf("failed to consume additional lots for %s: %w", tx.Type, err)
			}
			consumedLots = append(consumedLots, additionalLots...)
			remainingQty = remaining
		}
	} else {
		priority := pool.GetPriorityForTransactionType(tx.Type)
		consumedLots, remainingQty, err = fa.poolManager.Consume(quantityNeeded, priority)
		if err != nil {
			return fmt.Errorf("failed to consume lots for %s: %w", tx.Type, err)
		}
	}

	if remainingQty.IsPositive() {
		return fmt.Errorf("insufficient lots, still need %s", remainingQty.String())
	}

	// Store which specific lots this transaction consumed
	// Use tx.Row as unique identifier (since Reference may be shared)
	if len(consumedLots) > 0 {
		// Copy lots slice to avoid shared references
		lotsCopy := make([]pool.Lot, len(consumedLots))
		copy(lotsCopy, consumedLots)
		fa.txToLots[tx.Row] = lotsCopy
	}

	allocation := &Allocation{
		Transaction:  tx,
		ConsumedLots: consumedLots,
		RemainingQty: decimal.Zero,
	}

	fa.allocations = append(fa.allocations, allocation)

	return nil
}

func (fa *FIFOAllocator) GetAllocations() []*Allocation {
	return fa.allocations
}

// GetLotsConsumedByTransaction returns lots consumed by specific transaction (by Row number)
func (fa *FIFOAllocator) GetLotsConsumedByTransaction(row int) []pool.Lot {
	if lots, exists := fa.txToLots[row]; exists {
		return lots
	}
	return []pool.Lot{}
}
