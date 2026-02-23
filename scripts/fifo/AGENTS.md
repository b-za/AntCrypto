# AGENTS.md - Technical Documentation for AI Agents

This document provides comprehensive technical details for AI agents working with the FIFO cryptocurrency profit/loss calculator system.

## System Architecture

### Component Overview

```
main.go (Entry Point)
  ├── parser (CSVReader)
  │   ├── Read CSV files
  │   ├── Parse transactions
  │   └── Error logging
  │
  ├── classifier (TransactionType detection)
  │   ├── Classify inflows/outflows
  │   ├── PoolClassifier (transaction linking)
  │   └── FeeLinker (fee association)
  │
  ├── pool (PoolManager)
  │   ├── Initialize pools (in_buy, in_buy_for_other, in_other)
  │   ├── Add lots to pools
  │   └── Consume lots for allocations
  │
  ├── fifo (FIFOAllocator)
  │   ├── Allocate transactions to lots
  │   ├── Track allocations
  │   └── Maintain transaction-to-lots mapping
  │
  └── report (ReportCoordinator)
      ├── Generate FIFO reports (per-coin)
      ├── Generate profit/loss report (aggregated)
      ├── Generate inventory report (aggregated)
      └── Generate transfers report (aggregated)
```

### Data Flow

```
1. Load config.yaml
   ↓
2. Find all coin CSV files in data directory
   ↓
3. For each coin:
   a. Parse CSV → []Transaction
   b. Convert to classifier.Transaction
   c. Classify transaction types
   d. Classify pools (link outflows to inflows)
   e. Link fees to parent transactions
   f. Add lots to PoolManager (inflows only)
   g. Allocate outflows to lots (FIFO)
   h. Collect: transactions, allocations, poolManager
   ↓
4. Aggregate all coins' data
   ↓
5. Generate reports:
   - FIFO (per-coin)
   - Profit/Loss (all coins)
   - Inventory (all coins)
   - Transfers (all coins)
   ↓
6. Write CSV files to timestamped directory
```

## Component Breakdown

### 1. Parser (`internal/parser/parser.go`)

**Purpose**: Read CSV files and convert to internal transaction format.

**Key Functions**:
- `ReadAll()`: Reads entire CSV file, returns []Transaction
- `parseRecord()`: Converts CSV row to Transaction struct
- `WriteCSV()`: Writes [][]string to CSV file

**Data Structures**:
```go
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
```

**CSV Column Mapping**:
- Index 0: Wallet ID
- Index 1: Row (int)
- Index 2: Timestamp (format: "2006-01-02 15:04:05")
- Index 3: Description
- Index 4: Currency
- Index 5: Balance delta (decimal)
- Index 8: Cryptocurrency transaction ID (optional)
- Index 9: Cryptocurrency address (optional)
- Index 12: Value amount (decimal)
- Index 13: Reference

**Error Handling**:
- JSONL error logger for parsing failures
- Records type, message, coin, row, timestamp

### 2. Classifier (`internal/classifier/`)

**Purpose**: Determine transaction type from description and relationships.

#### Classifier (`classifier.go`)

**Transaction Types** (8 types total):

**Inflows (3 types)**:
- `in_buy`: Direct purchase
- `in_buy_for_other`: Purchase intended for transfer/gift
- `in_other`: Other inflow (received)

**Outflows (5 types)**:
- `out_sell`: Sale
- `out_other`: Other outflow (transfer/gift)
- `out_fee_buy`: Fee on purchase
- `out_fee_buy_for_other`: Fee on purchase for other
- `out_fee_in_other`: Fee on other inflow
- `out_fee_sell`: Fee on sale
- `out_fee_out_other`: Fee on other outflow

**Classification Rules**:
```go
// Inflow
if description starts with "Bought" → in_buy
else → in_other

// Outflow
if description starts with "Sold" → out_sell
if description contains "Fee" or "Trading Fee" → out_fee_buy (default)
else → out_other
```

**Data Structures**:
```go
type Transaction struct {
    // ... parser fields ...
    Timestamp      int64  // Unix timestamp
    TimestampStr   string // String representation
    Type           TransactionType
    LinkedRef      *string
    LinkedTransaction *Transaction
    LotReference   *string
}
```

#### Pool Classifier (`pool_classifier.go`)

**Purpose**: Link outflows to recent inflows for pool classification.

**Algorithm**:
1. Collect all `in_buy` transactions
2. For each `out_other` transaction:
   - Find best matching `in_buy` within 7 days
   - Match criteria:
     - Time difference ≤ 7 days (in_buy must be before out_other)
     - Quantity ratio ≥ 90% (out_other quantity / in_buy quantity)
   - Reclassify matched `in_buy` as `in_buy_for_other`
   - Store lot reference in `out_other.LinkedRef`

**Constants**:
- `daysThreshold = 7`
- `quantityThreshold = 90` (percentage)

**Example**:
```
in_buy: 2024-02-01, +1.0 BTC, "Bought 1.0 BTC"
out_other: 2024-02-05, -0.95 BTC, "Transfer to wallet"
→ Match! (4 days apart, 95% quantity)
→ in_buy reclassified to in_buy_for_other
```

#### Fee Linker (`fee_linker.go`)

**Purpose**: Link fee transactions to parent transactions by reference.

**Algorithm**:
1. Build reference map of all non-fee transactions
2. For each fee transaction:
   - Look up parent by reference ID
   - Set `tx.LinkedTransaction = parent`
   - Subclassify fee based on parent's transaction type

**Fee Subclassification**:
- Parent `in_buy` → `out_fee_buy`
- Parent `in_buy_for_other` → `out_fee_buy_for_other`
- Parent `in_other` → `out_fee_in_other`
- Parent `out_sell` → `out_fee_sell`
- Parent `out_other` → `out_fee_out_other`

**Example**:
```
Parent: ref="abc123", type=in_buy, "Bought BTC"
Fee: ref="abc123", type=out_fee_buy, "Trading fee"
→ Fee linked to parent via shared reference
```

### 3. Pool Manager (`internal/pool/pool_manager.go`)

**Purpose**: Manage lots in FIFO order for cost basis tracking.

**Data Structures**:
```go
type Pool struct {
    Name string
    Lots []*Lot
}

type Lot struct {
    Reference string
    Quantity  decimal.Decimal
    UnitCost  decimal.Decimal
    Timestamp int64
    Pool      string
}

type PoolManager struct {
    Pools map[string]*Pool
}
```

**Pool Initialization**:
```go
pm.InitializePools()
// Creates 3 pools:
// - in_buy
// - in_buy_for_other
// - in_other
```

**Key Operations**:

**Add Lot**:
```go
pm.AddLotWithReference(poolName, quantity, unitCost, timestamp, lotRef)
// Adds lot to specified pool
// Maintains FIFO order by append
```

**Consume Lots**:
```go
pm.Consume(quantityNeeded, priority []string)
// Consumes from pools in priority order
// Returns: []Lot, remainingQuantity, error
// FIFO: Consumes from pool[0] first, removes empty lots
```

**Priority Rules**:
```go
// Sales (out_sell): in_buy → in_buy_for_other → in_other
GetPriorityForTransactionType(OutflowSell)
  → ["in_buy", "in_buy_for_other", "in_other"]

// Other outflows (out_other): in_buy_for_other → in_other → in_buy
GetPriorityForTransactionType(OutflowOther)
  → ["in_buy_for_other", "in_other", "in_buy"]
```

**Consume Specific Lot**:
```go
pm.ConsumeFromSpecificLot(lotRef, quantityNeeded)
// Consumes from specific lot by reference
// Used for linked transactions (transfers)
// Returns: *Lot, remainingQuantity, error
```

### 4. FIFO Allocator (`internal/fifo/allocator.go`)

**Purpose**: Allocate outflow transactions to lots using FIFO method.

**Data Structures**:
```go
type Allocation struct {
    Transaction  *classifier.Transaction
    ConsumedLots []pool.Lot
    RemainingQty decimal.Decimal
}

type FIFOAllocator struct {
    poolManager *pool.PoolManager
    allocations []*Allocation
    txToLots    map[int][]pool.Lot  // Row → lots consumed
}
```

**Allocation Algorithm**:

```go
AllocateTransaction(tx, linkedLotRef):
    1. Get quantity needed = abs(tx.BalanceDelta)

    2. If tx is out_other AND has linkedLotRef:
         a. Consume from specific lot first
         b. If remaining > 0, consume from pools by priority

    3. Else:
         a. Consume from pools by priority

    4. If remaining quantity > 0:
         ERROR: Insufficient lots

    5. Store allocation:
         a. Add to allocations list
         b. Map tx.Row → consumedLots (for FIFO report)
```

**Example**:
```
Pool: in_buy
  - Lot A: 0.5 BTC (2024-01-01)
  - Lot B: 0.3 BTC (2024-01-15)

Transaction: Sell 0.6 BTC (2024-02-01)
Allocation:
  1. Priority: [in_buy, in_buy_for_other, in_other]
  2. Consume from in_buy:
     - Lot A: 0.5 BTC (fully consumed)
     - Lot B: 0.1 BTC (partially consumed)
  3. Remaining: 0.0 BTC
  4. Success!

Result:
  - Allocation.Transaction = Sell transaction
  - Allocation.ConsumedLots = [Lot A (0.5), Lot B (0.1)]
  - txToLots[sellTx.Row] = [Lot A, Lot B]
```

**Key Methods**:
- `AllocateTransaction()`: Main allocation logic
- `GetAllocations()`: Return all allocations
- `GetLotsConsumedByTransaction(row int)`: Get lots for specific transaction
- `SetTxToLots(map[int][]pool.Lot)`: Set transaction-to-lots mapping

### 5. Report Coordinator (`internal/report/coordinator.go`)

**Purpose**: Coordinate generation of all report types.

**Data Structures**:
```go
type ReportCoordinator struct {
    coinPoolsMap  map[string]*pool.PoolManager  // coin → pool manager
    transactions  []*classifier.Transaction    // All transactions from all coins
    allocations   []*fifo.Allocation            // All allocations from all coins
    config        *config.Config
}
```

**Report Generation**:
```go
GenerateAllReports(reportsDir):
    1. generateFIFOReports()
       - For each coin:
         - Filter transactions by coin (case-insensitive)
         - Create FIFO report generator
         - Generate report with coin's pool manager and allocator
         - Write to {coin}_fifo.csv

    2. generateInventoryReport()
       - For each coin:
         - Generate inventory report for coin
       - Aggregate all inventory rows
       - Write to inventory.csv

    3. generateProfitLossReport()
       - Generate from all allocations (all coins)
       - Write to financial_year_profit_loss.csv

    4. generateTransfersReport()
       - Generate from all transactions (all coins)
       - Write to transfers.csv
```

**Financial Year Calculation**:
```go
getFinancialYear(timestamp):
    date = UnixTime(timestamp)
    year = date.Year()
    month = int(date.Month())

    if month >= 3:  // March onwards
        return fmt.Sprintf("FY%d", year + 1)
    else:
        return fmt.Sprintf("FY%d", year)
```

**Example**:
- 2024-02-15 → month=2 → FY2024
- 2024-03-01 → month=3 → FY2025
- 2024-06-15 → month=6 → FY2025

### 6. Report Generators (`internal/report/`)

#### FIFO Report (`fifo.go`)

**Purpose**: Generate per-coin FIFO ledger showing all transactions.

**Data Flow**:
```go
GenerateReport(transactions, poolManager, allocator):
    For each transaction:
        if inflow (buy/other):
            - Calculate unit cost = ValueAmount / BalanceDelta
            - Add to current balance
            - Create inflow row

        if outflow (sell/fee/other):
            - Get lots consumed by this transaction
            - For each lot:
                - Calculate proceeds per lot
                - Calculate profit = proceeds - (lot.quantity × lot.unitCost)
                - Subtract from current balance
                - Create outflow row per lot

    Return CSV with headers
```

**Output Columns** (14):
1. Financial Year
2. Transaction Reference
3. Date
4. Description
5. Type
6. Lot Reference
7. Quantity Change
8. Balance Units (running total)
9. Total Cost (ZAR)
10. Balance Value (ZAR)
11. Fee (ZAR)
12. Proceeds (ZAR)
13. Profit (ZAR)
14. Unit Cost (ZAR)

#### Profit/Loss Report (`profit_loss.go`)

**Purpose**: Generate aggregated profit/loss by financial year across all coins.

**Data Structures**:
```go
type ProfitLossRow struct {
    FinancialYear string
    Coin         string  // NEW: coin that was sold
    SaleReference string
    LotReference  string
    QuantitySold  decimal.Decimal
    UnitCost      decimal.Decimal
    TotalCost     decimal.Decimal
    SellingPrice  decimal.Decimal
    Proceeds      decimal.Decimal
    ProfitLoss    decimal.Decimal
    FeeReference  string
    FeeAmount     decimal.Decimal
    IsSummary     bool
}

type FinancialYearSummary struct {
    FY            string
    TotalSales    decimal.Decimal
    TotalCost     decimal.Decimal
    LossSales     decimal.Decimal
    ProfitSales   decimal.Decimal
    NetProfit     decimal.Decimal
    TotalProceeds decimal.Decimal
}
```

**Generation Algorithm**:
```go
GenerateReport(allocations):
    summaries = map[FinancialYear]*FinancialYearSummary

    For each allocation:
        if transaction.type == out_sell:
            1. Calculate financial year
            2. Create/update summary for this FY
            3. For each consumed lot:
                - Calculate profit for this lot
                - Create detail row
                - Add to summary totals

        if transaction.type == out_fee_*:
            1. Calculate financial year
            2. Create fee row (no lot reference)

    Add summary rows:
        - Losses Total (if any)
        - Profits Total (if any)
        - Combined (total)

    Sort by: Financial Year, Sale Reference
```

**Output Columns** (12):
1. Financial Year
2. **Coin** ← NEW
3. Sale Reference
4. Lot Reference
5. Quantity Sold
6. Unit Cost (ZAR)
7. Total Cost (ZAR)
8. Selling Price (ZAR)
9. Proceeds (ZAR)
10. Profit/Loss (ZAR)
11. Fee Reference
12. Fee Amount (ZAR)

**Summary Rows**:
- `Losses Total`: Aggregates all loss sales for FY
- `Profits Total`: Aggregates all profit sales for FY
- `Combined`: Total aggregation including both losses and profits

#### Inventory Report (`inventory.go`)

**Purpose**: Generate current holdings across all coins.

**Generation Algorithm**:
```go
GenerateReportForCoin(poolManager, coin):
    For each pool (in_buy, in_buy_for_other, in_other):
        For each lot with positive quantity:
            1. Calculate financial year from lot timestamp
            2. Create inventory row with coin name
            3. Calculate total cost = quantity × unitCost

    Group by FY and coin for summaries
    Add summary rows with totals

    Sort by: Financial Year, Coin, Date of Purchase
```

**Output Columns** (8):
1. Financial Year
2. Coin
3. Lot Reference
4. Quantity
5. Date of Original Purchase
6. Unit Cost (ZAR)
7. Total Cost (ZAR)
8. Pool

**Summary Rows**:
- `Total`: Aggregated quantity and value per FY and coin

#### Transfers Report (`transfers.go`)

**Purpose**: Generate all non-buy/sell transactions across all coins.

**Generation Algorithm**:
```go
GenerateReport(transactions):
    For each transaction:
        if type != in_buy AND type != out_sell:
            1. Calculate financial year
            2. Calculate unit cost or unit cost value
            3. Create transfer row
            4. Get lot reference if available

    Sort chronologically by Date
```

**Output Columns** (10):
1. Financial Year
2. Coin
3. Date
4. Description
5. Type
6. Quantity Change
7. Value (ZAR)
8. Unit Cost (ZAR) / Unit Cost Value
9. Lot Reference
10. Pool

## Transaction Type Reference

### Complete Transaction Type List

| Type | Category | Description | Pool Priority |
|------|----------|-------------|----------------|
| `in_buy` | Inflow | Direct purchase of cryptocurrency | First for sales |
| `in_buy_for_other` | Inflow | Purchase for transfer/gift | Second for sales |
| `in_other` | Inflow | Received cryptocurrency | Third for sales |
| `out_sell` | Outflow | Sale of cryptocurrency | Consumes from pools |
| `out_other` | Outflow | Transfer/gift of cryptocurrency | Consumes from pools |
| `out_fee_buy` | Fee | Fee on purchase | Linked to `in_buy` |
| `out_fee_buy_for_other` | Fee | Fee on purchase for other | Linked to `in_buy_for_other` |
| `out_fee_in_other` | Fee | Fee on other inflow | Linked to `in_other` |
| `out_fee_sell` | Fee | Fee on sale | Linked to `out_sell` |
| `out_fee_out_other` | Fee | Fee on other outflow | Linked to `out_other` |

### Pool Priority Matrix

| Transaction Type | Pool Priority Order |
|----------------|-------------------|
| `out_sell` | `in_buy` → `in_buy_for_other` → `in_other` |
| `out_other` | `in_buy_for_other` → `in_other` → `in_buy` |
| `out_fee_buy` | `in_buy` → `in_buy_for_other` → `in_other` |
| `out_fee_buy_for_other` | `in_buy_for_other` → `in_other` → `in_buy` |
| `out_fee_in_other` | `in_other` → `in_buy_for_other` → `in_buy` |
| `out_fee_sell` | `in_buy` → `in_buy_for_other` → `in_other` |
| `out_fee_out_other` | `in_buy_for_other` → `in_other` → `in_buy` |

## Code Conventions

### Naming Patterns

**Files**:
- `main.go` - Entry point
- `{package}/{feature}.go` - Feature implementation
- `{package}_test.go` - Unit tests

**Functions**:
- `New{Struct}()` - Constructor
- `Get{Property}()` - Getter
- `Set{Property}()` - Setter
- `Generate{Report}()` - Report generation
- `parse{Format}()` - Parsing
- `process{Data}()` - Processing logic

**Variables**:
- `cfg` - Config
- `tx` - Transaction
- `pm` - Pool Manager
- `err` - Error
- `cf` - Coin File
- `rc` - Report Coordinator
- `pl` - Profit/Loss generator
- `ig` - Inventory generator
- `tr` - Transfers generator
- `fg` - FIFO generator

**Constants**:
- `FY{Year}` - Financial year string
- `in_buy`, `out_sell`, etc. - Transaction types
- `daysThreshold`, `quantityThreshold` - Classification thresholds

### Error Handling

**Pattern**:
```go
if err != nil {
    return fmt.Errorf("failed to {operation}: %w", err)
}
```

**Error Logging**:
```go
errorLog.LogError(parser.ErrorEntry{
    Type:    "ERROR_TYPE",
    Message: err.Error(),
    Coin:    coin,
    Row:      tx.Row,
    Timestamp: tx.TimestampStr,
})
```

**Error Types**:
- `CLASSIFICATION_ERROR`: Transaction type detection failed
- `POOL_ERROR`: Lot management failed
- `ALLOCATION_ERROR`: FIFO allocation failed
- `PROCESS_ERROR`: General processing failure

### Logging

**Error Log Format** (JSONL):
```json
{"type":"ERROR_TYPE","message":"error message","coin":"XBT","row":42,"timestamp":"2024-02-15 00:23:54"}
```

**No Console Logging**: Errors logged to JSONL file only

### Decimal Precision

**Library**: `github.com/shopspring/decimal`

**Usage**:
- All monetary calculations use `decimal.Decimal`
- Avoids floating-point precision issues
- String representation for CSV output

**Operations**:
```go
decimal.NewFromString("100.00")
decimal.NewFromInt(10)
decimal.Zero
amount.Add(other)
amount.Sub(other)
amount.Mul(factor)
amount.Div(divisor)
amount.Abs()
amount.IsPositive()
amount.IsNegative()
amount.IsZero()
```

## Common Patterns

### CSV Reading Pattern

```go
reader := csv.NewReader(file)
records, err := reader.ReadAll()
if err != nil {
    return fmt.Errorf("failed to read CSV: %w", err)
}
```

### Transaction Processing Pattern

```go
for _, tx := range transactions {
    txType, err := c.Classify(*tx)
    if err != nil {
        errorLog.LogError(...)
        continue  // Skip problematic transaction
    }
    tx.Type = txType
}
```

### Pool Management Pattern

```go
pm := pool.NewPoolManager()
pm.InitializePools()

// Add lots for inflows
pm.AddLotWithReference(poolName, quantity, unitCost, timestamp, lotRef)

// Consume lots for outflows
lots, remaining, err := pm.Consume(quantityNeeded, priority)
```

### FIFO Allocation Pattern

```go
allocator := fifo.NewFIFOAllocator(pm)

for _, tx := range transactions {
    if isOutflow(tx) {
        var linkedLotRef *string
        if tx.LinkedRef != nil {
            linkedLotRef = tx.LinkedRef
        }
        err := allocator.AllocateTransaction(tx, linkedLotRef)
        if err != nil {
            errorLog.LogError(...)
        }
    }
}
```

### Report Generation Pattern

```go
generator := NewReportGenerator(cfg)
records := generator.GenerateReport(data)

outputPath := fmt.Sprintf("%s/report.csv", reportsDir)
err := WriteCSV(outputPath, records)
if err != nil {
    return fmt.Errorf("failed to write report: %w", err)
}
```

## Bug History

### Bug 1: Data Aggregation Issue (Fixed)

**Problem**: Reports only contained data from the last processed coin alphabetically (XRP).

**Root Cause**: Created new `ReportCoordinator` per coin, each generating reports to the same filenames. Each coin's data overwrote the previous coin's data.

**Location**: `main.go:245` - Created new `ReportCoordinator` per coin

**Fix**: Refactored to aggregate data from all coins before generating reports:
- Created `coinProcessingResult` struct to return data per coin
- Collected all transactions, allocations, and pool managers
- Created single `ReportCoordinator` with aggregated data
- Generated reports once at the end

**Impact**: Profit/loss and transfers reports now include all coins' data (39 sales vs. 4 sales).

### Bug 2: Empty Coin Column in Inventory (Fixed)

**Problem**: Inventory report had empty Coin column.

**Root Cause**: `getCoinFromPool()` in `inventory.go:145-150` returned empty string. Pool managers are shared across all coins and don't track which coin owns each pool.

**Location**: `inventory.go:59` - Called `getCoinFromPool()` which returned ""

**Fix**:
- Added `coin` field to `InventoryReportGenerator` struct
- Created `GenerateReportForCoin()` method that accepts coin parameter
- Removed `getCoinFromPool()` function
- Use passed coin parameter directly

**Impact**: Inventory report now shows correct coin names for each lot.

### Bug 3: Incorrect Financial Year Calculation (Fixed)

**Problem**: Financial years were off by one year.

**Root Cause**: Incorrect logic in `getFinancialYear()`:
```go
// WRONG
if month >= 3 {
    return fmt.Sprintf("FY%d", year)      // Feb → year-1, Mar → year
}
return fmt.Sprintf("FY%d", year-1)        // Wrong offset
```

**Expected**: FY2025 = 1 Mar 2024 to 28 Feb 2025

**Location**: 5 files - `coordinator.go`, `profit_loss.go`, `inventory.go`, `transfers.go`, `fifo.go`

**Fix**: Corrected logic:
```go
// CORRECT
if month >= 3 {
    return fmt.Sprintf("FY%d", year + 1)    // Mar onwards → next year's FY
}
return fmt.Sprintf("FY%d", year)              // Jan/Feb → current year's FY
```

**Impact**: All transactions now have correct financial year assignments:
- 2024-02-15 → FY2024 (was FY2023)
- 2024-03-01 → FY2025 (was FY2024)
- 2025-04-10 → FY2026 (was FY2025)

### Bug 4: Missing Coin Column in Profit/Loss (Fixed)

**Problem**: Profit/loss report didn't show which coin was sold.

**Root Cause**: `ProfitLossRow` struct didn't include Coin field.

**Location**: `profit_loss.go:15-28` - Missing Coin field

**Fix**:
- Added `Coin` field to `ProfitLossRow` struct
- Populated for sale rows: `Coin: allocation.Transaction.Currency`
- Populated for fee rows: `Coin: allocation.Transaction.Currency`
- Set to empty for summary rows
- Added to CSV headers and output

**Impact**: Users can now identify which cryptocurrency was sold in each transaction.

## Testing Guidelines

### Unit Testing

**Test Structure**:
```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected ExpectedType
    }{
        {"test case 1", input1, expected1},
        {"test case 2", input2, expected2},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FunctionUnderTest(tt.input)
            if result != tt.expected {
                t.Errorf("FunctionUnderTest(%v) = %v, want %v",
                    tt.input, result, tt.expected)
            }
        })
    }
}
```

### Integration Testing

**Test Strategy**:
1. Create test CSV files with known transactions
2. Run full processing pipeline
3. Verify output CSV files match expected results
4. Check financial year assignments
5. Validate profit/loss calculations
6. Confirm lot consumption is FIFO

### Validation Approaches

**FIFO Validation**:
- Verify lots consumed in chronological order
- Check oldest lots consumed first
- Confirm partial lot consumption tracked correctly

**Financial Year Validation**:
- Test transactions on March 1 (should be next FY)
- Test transactions on February 28 (should be current FY)
- Verify year boundaries correct

**Profit/Loss Validation**:
- Calculate expected profit manually for sample transactions
- Compare with generated report
- Verify fee handling correct
- Check summary rows aggregate correctly

**Report Validation**:
- Verify all columns present
- Check row counts match expected
- Validate sorting order
- Confirm no missing or duplicate rows

## Future Enhancements

### Potential Improvements

1. **Additional Cost Basis Methods**:
   - LIFO (Last-In, First-Out)
   - HIFO (Highest-In, First-Out)
   - Specific Lot (user-selected)

2. **Tax Integration**:
   - Configurable tax rates
   - Short-term vs long-term capital gains
   - Automatic tax calculation in reports

3. **Multi-Currency Support**:
   - Multiple fiat currencies (USD, EUR, etc.)
   - Currency conversion at time of transaction
   - Consolidated reporting in base currency

4. **Report Export Formats**:
   - PDF generation
   - Excel workbook with multiple sheets
   - JSON export for API integration

5. **Performance Optimizations**:
   - Parallel processing of coins
   - Incremental report generation
   - Database backend for large datasets

6. **User Interface**:
   - Web dashboard for report visualization
   - Interactive lot selection
   - What-if analysis scenarios

7. **Data Import**:
   - Exchange API integration (Binance, Coinbase, etc.)
   - Automatic transaction sync
   - Real-time portfolio tracking

### Technical Debt

1. **Transaction ID Management**:
   - Current: Uses Reference field for linking
   - Improvement: Use cryptocurrency transaction IDs

2. **Error Recovery**:
   - Current: Skips problematic transactions
   - Improvement: Retry logic, partial processing

3. **Configuration Validation**:
   - Current: Minimal validation
   - Improvement: Schema validation, default values

4. **Logging**:
   - Current: JSONL error log only
   - Improvement: Structured logging levels, metrics

## Maintenance Guidelines

### Adding New Transaction Types

1. Add to `classifier.go` constants:
```go
const (
    NewTransactionType TransactionType = "new_type"
)
```

2. Update classification logic:
```go
func (c *Classifier) classifyOutflow(tx Transaction) (TransactionType, error) {
    if pattern matches {
        return NewTransactionType, nil
    }
    // ...
}
```

3. Add pool priority rule:
```go
func GetPriorityForTransactionType(txType TransactionType) []string {
    switch txType {
    case NewTransactionType:
        return []string{"pool1", "pool2", "pool3"}
    // ...
    }
}
```

4. Update fee linking if needed:
```go
func (f *FeeLinker) subclassifyFee(tx *Transaction, parentType TransactionType) {
    switch parentType {
    case NewTransactionType:
        tx.Type = OutFeeNewType
    // ...
    }
}
```

### Modifying Financial Year Logic

1. Update `config.yaml`:
```yaml
settings:
  financial_year_start: 4  # April start
```

2. Test boundary cases:
- Last day of old FY
- First day of new FY
- Leap year considerations

### Adding New Report Types

1. Create generator file:
```go
// internal/report/new_report.go
type NewReportGenerator struct {
    config     *config.Config
    reportRows []NewReportRow
}

func NewNewReportGenerator(cfg *config.Config) *NewReportGenerator {
    return &NewReportGenerator{config: cfg, reportRows: []NewReportRow{}}
}

func (g *NewReportGenerator) GenerateReport(data DataType) [][]string {
    // Generate report logic
    return g.buildCSV()
}

func (g *NewReportGenerator) buildCSV() [][]string {
    // Build CSV with headers
}
```

2. Add to coordinator:
```go
func (rc *ReportCoordinator) GenerateAllReports(reportsDir string) error {
    // ... existing reports ...
    if err := rc.generateNewReport(reportsDir); err != nil {
        errorCollection = append(errorCollection, err)
    }
    // ...
}
```

## Key Dependencies

### Go Modules

```
github.com/shopspring/decimal v1.4.0
  - Exact decimal arithmetic for monetary values

gopkg.in/yaml.v3 v3.0.1
  - YAML configuration file parsing
```

### Standard Library

- `encoding/csv` - CSV reading/writing
- `fmt` - String formatting
- `os` - File operations
- `time` - Date/time handling
- `sort` - Sorting algorithms
- `crypto/sha1` - Hash generation for lot references
- `encoding/hex` - Hex encoding
- `encoding/json` - JSON error logging

## Performance Characteristics

### Time Complexity

- **CSV Parsing**: O(n) where n = number of transactions
- **Classification**: O(n) for each classifier step
- **Pool Classification**: O(m × k) where m = out_other count, k = in_buy count
- **Fee Linking**: O(n) with map lookups
- **FIFO Allocation**: O(a × p) where a = allocations, p = pool count
- **Report Generation**: O(r) where r = report row count

### Space Complexity

- **Transaction Storage**: O(n × t) where t = transaction size
- **Pool Storage**: O(l) where l = total lots
- **Allocation Storage**: O(a) where a = total allocations
- **Report Generation**: O(r) where r = report row count

### Memory Usage

- **Typical Dataset** (~10,000 transactions):
  - Transaction memory: ~10 MB
  - Pool memory: ~5 MB
  - Allocation memory: ~5 MB
  - Report memory: ~10 MB
  - **Total**: ~30 MB

## Debugging Guidelines

### Common Issues

**1. Missing Transactions**

**Symptoms**: Expected transaction not in output

**Debug Steps**:
1. Check error_log.jsonl for parsing errors
2. Verify CSV format matches expected columns
3. Confirm currency code not filtered (case-insensitive)
4. Check reference ID uniqueness

**2. Incorrect Classification**

**Symptoms**: Transaction type wrong (e.g., sale classified as transfer)

**Debug Steps**:
1. Check transaction description matches patterns
2. Verify description trimming and case handling
3. Review pool classifier matching rules
4. Check fee linking logic

**3. FIFO Allocation Errors**

**Symptoms**: "Insufficient lots" error, unexpected consumption

**Debug Steps**:
1. Verify inflow transactions have positive balance delta
2. Check pool classification (7 days, 90% quantity)
3. Confirm pool priority order
4. Review lot consumption order

**4. Financial Year Misassignment**

**Symptoms**: Transaction in wrong FY

**Debug Steps**:
1. Verify `financial_year_start` configuration
2. Check month comparison logic (>= 3)
3. Validate year addition/subtraction
4. Test boundary dates (Feb 28/29, Mar 1)

### Debug Tools

**Add Logging**:
```go
fmt.Printf("DEBUG: Processing transaction %s\n", tx.Reference)
fmt.Printf("DEBUG: Balance delta: %s\n", tx.BalanceDelta.String())
fmt.Printf("DEBUG: Transaction type: %s\n", tx.Type)
```

**Print Intermediate State**:
```go
for coin, pm := range coinPoolsMap {
    fmt.Printf("DEBUG: Coin %s pools:\n", coin)
    for poolName, pool := range pm.Pools {
        fmt.Printf("  Pool %s: %d lots\n", poolName, len(pool.Lots))
    }
}
```

**Validate Calculations**:
```go
// Verify total cost
totalCost := decimal.Zero
for _, lot := range allocation.ConsumedLots {
    lotCost := lot.Quantity.Mul(lot.UnitCost)
    totalCost = totalCost.Add(lotCost)
    fmt.Printf("DEBUG: Lot %s: %s × %s = %s\n",
        lot.Reference, lot.Quantity, lot.UnitCost, lotCost)
}
fmt.Printf("DEBUG: Total cost: %s\n", totalCost)
```

## Conclusion

This system provides accurate FIFO-based cost basis calculation for cryptocurrency transactions with comprehensive reporting for tax purposes. The architecture supports multiple coins, proper financial year handling, and detailed transaction tracking.

For questions or improvements, refer to the component breakdown and code conventions sections above.
