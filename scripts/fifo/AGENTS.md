# AGENTS.md - Technical Documentation for AI Agents

This document provides comprehensive technical details for AI agents working with the FIFO cryptocurrency profit/loss calculator system and exchange rate template generator.

## System Architecture

### Component Overview

**FIFO Calculator** (`scripts/fifo/`):
```
main.go (Entry Point)
  ├── parser (CSVReader)
  │   ├── Read CSV files
  │   ├── Parse transactions
  │   └── Error logging
  ├── classifier (TransactionType detection)
  │   ├── Classify inflows/outflows
  │   ├── PoolClassifier (transaction linking)
  │   └── FeeLinker (fee association)
  ├── pool (PoolManager)
  │   ├── Initialize pools (in_buy, in_buy_for_other, in_other)
  │   ├── Add lots to pools
  │   └── Consume lots for allocations
  ├── fifo (FIFOAllocator)
  │   ├── Allocate transactions to lots
  │   ├── Track allocations
  │   └── Maintain transaction-to-lots mapping
  └── report (ReportCoordinator)
      ├── Generate FIFO reports (per-coin)
      ├── Generate profit/loss report (aggregated)
      ├── Generate inventory report (aggregated)
      └── Generate transfers report (aggregated)
```

**Exchange Rate Template Generator** (`scripts/generate_exchange_rates/`):
```
main.go (Entry Point)
  └── CSV Reader & Date Collector
      ├── Read all coin CSV files
      ├── Filter for in_other/out_other transactions
      ├── Group by date
      └── Generate YAML template with placeholder rates
```

## Data Flow

**FIFO Calculator**:
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

**Exchange Rate Template Generator** (Step 1):
```
1. Load config.yaml
   ↓
2. Find all coin CSV files in data directory
   ↓
3. For each coin:
   a. Parse CSV → []Transaction
   b. Filter for in_other and out_other only
   c. Collect: date, reference, ZAR value, coin, direction
   ↓
4. Group transactions by date (same date = one rate entry)
   ↓
5. Generate YAML template with:
   - date
   - list of transaction references for that date
   - usd_prices fields (btc, eth, xrp, ltc) - all "0.00"
   - usd_to_zar_rate field - "0.00"
   ↓
6. Save to timestamped exchange_templates/<timestamp>/ directory
   ↓
7. Generate validation log
```

## Component Breakdown

### 1. Parser (`scripts/fifo/internal/parser/parser.go`)

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
-   Index 4: Currency
- Index 5: Balance delta (decimal)
- Index 8: Cryptocurrency transaction ID (optional)
- Index 9: Cryptocurrency address (optional)
- Index 12: Value amount (decimal)
- Index 13: Reference

### 2. Classifier (`scripts/fifo/internal/classifier/`)

**Purpose**: Determine transaction type from description and relationships.

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

### 3. Pool Manager (`scripts/fifo/internal/pool/pool_manager.go`)

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
- `AddLotWithReference(poolName, quantity, unitCost, timestamp, lotRef)`: Adds lot to specified pool
- `Consume(quantityNeeded, priority []string)`: Consumes from pools in priority order
- `ConsumeFromSpecificLot(lotRef, quantityNeeded)`: Consumes from specific lot (used for transfers)

### 4. FIFO Allocator (`scripts/fifo/internal/fifo/allocator.go`)

**Purpose**: Allocate outflow transactions to lots using FIFO method.

**Allocation Algorithm**:
```
AllocateTransaction(tx, linkedLotRef):
    1. Get quantity needed = abs(tx.BalanceDelta)

    2. If tx is out_other AND has linkedLotRef:
        a. Consume from specific lot first
        b. If remaining > 0, consume from pools by priority

    3. Else:
        a. Consume from pools by priority

    4. If remaining quantity > 0:
        ERROR: Insufficient lots
```

### 5. Report Coordinator (`scripts/fifo/internal/report/coordinator.go`)

**Purpose**: Coordinate generation of all report types.

**Report Generation**:
```go
GenerateAllReports(reportsDir):
    1. generateFIFOReports() - Per-coin FIFO ledgers
    2. generateInventoryReport() - Current holdings across all coins
    3. generateProfitLossReport() - Aggregated P&L by financial year
    4. generateTransfersReport() - All non-buy/sell transactions
```

### 6. Exchange Rate Template Generator (`scripts/generate_exchange_rates/main.go`)

**Purpose**: Generate YAML template with placeholder exchange rates for cross-currency tracking (Step 1 of 2-step process).

**Key Features**:
- Scans all CSV files for in_other and out_other transactions
- Groups transactions by date (same date = one rate entry)
- Creates placeholder fields for manual price filling
- Generates timestamped templates in separate directory

**Data Structures**:
```go
type ExchangeTransaction struct {
    Date         time.Time
    Reference    string
    Coin         string
    ZARValue     decimal.Decimal
    Direction    string  // "in" or "out"
    IsOutflow    bool
}

type DateRates struct {
    Date                  string            `yaml:"date"`
    TransactionReferences []string          `yaml:"transaction_references"`
    USDPrices             map[string]string `yaml:"usd_prices"`      // BTC, ETH, XRP, LTC prices in USD
    USDToZARRate          string            `yaml:"usd_to_zar_rate"` // USD to ZAR rate
}
```

**Output Structure**:
```yaml
dates_required:
  - date: "2017-08-28"
      transaction_references:
        - 037b4604
        - 7481d775
      usd_prices:
        btc: "0.00"    # ← FILL IN: Bitcoin price in USD
        eth: "0.00"    # ← FILL IN: Ethereum price in USD
        xrp: "0.00"    # ← FILL IN: Ripple price in USD
        ltc: "0.00"    # ← FILL IN: Litecoin price in USD
      usd_to_zar_rate: "0.00"  # ← FILL IN: USD → ZAR rate
```

**Output Location**:
```
<path/to/root>/exchange_templates/<timestamp>/
  ├── exchange_rates_template.yaml           # YOUR TEMPLATE FILE
  └── exchange_rates_template_validation.log # Validation summary
```

## Exchange Rate Template Generator Usage

### Installation
```bash
cd scripts/generate_exchange_rates
go mod tidy
GO111MODULE=on go build -o generate_exchange_rates
```

### Running
```bash
# Process all roots
./generate_exchange_rates

# Process specific root(s) only
./generate_exchange_rates --roots cap

# Custom config location
./generate_exchange_rates --config path/to/config.yaml
```

### Filling in Exchange Rates (Manual Step)

After generating the template, you need to fill in:

1. **usd_prices**: Historical USD prices for each coin on each date
   - Look up BTC/ETH/XRP/LTC closing prices
   - Sources: CoinGecko, Binance, Yahoo Finance, etc.

2. **usd_to_zar_rate**: USD → ZAR exchange rate on each date
   - Sources: Frankfurter API, historical rates database

3. Save the updated YAML file

### Example Filled Data
```yaml
  - date: "2017-08-30"
      usd_prices:
        btc: "4521.50"    # BTC was $4,521.50 on this date
        eth: "320.00"      # ETH was $320.00 on this date
        xrp: "0.0075"      # XRP was $0.0075 on this date
        ltc: "55.00"       # LTC was $55.00 on this date
      usd_to_zar_rate: "13.50"  # 1 USD = 13.50 ZAR
```

### Part 2 Workflow (To Be Implemented)

After filling in the template, Part 2 script will:
1. Find latest exchange_rates_template.yaml in exchange_templates/
2. Load the filled rates
3. Generate exchange.csv with cross-currency conversions
4. Calculate equivalent units for each transaction in all coins
5. Determine net inflow/outflow by currency

### Validation Log Format

```
=== Exchange Rates Template Validation Summary ===
Generated at: 2026-02-24 05:34:58

Summary Statistics:
  Total in_other/out_other transactions: 16
  Unique dates requiring rates: 11
  Coins discovered: 4 (eth, ltc, xbt, xrp)
  Coins discovered: 4 (eth, ltc, xbt, xrp)

Template File Structure:
  For each date, you need to fill in:
    - usd_prices.btc: Bitcoin price in USD
    - usd_prices.eth: Ethereum price in USD
    - usd_prices.xrp: Ripple price in USD
    - usd_prices.ltc: Litecoin price in USD
    - usd_to_zar_rate: USD to ZAR exchange rate

Usage Instructions:
  1. Look up historical prices for each date
  2. Fill in the usd_prices and usd_to_zar_rate fields
  3. Save the updated YAML file
  4. Run Part 2 script to generate exchange.csv
```

## Transaction Type Reference

### Complete Transaction Type List

| Type | Category | Description | Pool Priority |
|------|----------|-------------|----------------|
| `in_buy` | Inflow | Direct purchase | First for sales |
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
        continue
    }
    tx.Type = txType
}
```

### Pool Management Pattern
```go
pm := pool.NewPoolManager()
pm.InitializePools()

pm.AddLotWithReference(poolName, quantity, unitCost, timestamp, lotRef)

pm.Consume(quantityNeeded, priority)
```

### Exchange Rate Generation Pattern

```go
// Collect transactions
transactions := []ExchangeTransaction{}
for _, root := range roots {
    txs, coins, err := processRoot(root)
    allTransactions = append(allTransactions, txs...)
    for _, coin := range coins {
        coinSet[coin] = true
    }
}

// Group by date
dateRatesMap := make(map[string]*DateRates)
for _, tx := range allTransactions {
    dateStr := tx.Date.Format("2006-01-02")

    if dateRatesMap[dateStr] == nil {
        dateRatesMap[dateStr] = &DateRates{
            Date:                  dateStr,
            TransactionReferences: []string{},
            USDPrices:             make(map[string]string),
            USDToZARRate:          "0.00",
        }
    }

    dateRatesMap[dateStr].TransactionReferences = append(dateRatesMap[dateStr].TransactionReferences, tx.Reference)
}
```

## Key Dependencies

### Go Modules
```
github.com/shopspring/decimal v1.4.0
gopkg.in/yaml.v3 v3.0.1
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
- **FIFO Allocation**: O(a × p) where a = allocations, p = pool count
- **Report Generation**: O(r) where r = report row count
- **Date Grouping**: O(n) for exchange rate template

### Space Complexity
- **Transaction Storage**: O(n × t) where t = transaction size
- **Pool Storage**: O(l) where l = total lots
- **Allocation Storage**: O(a) where a = total allocations
- **Report Storage**: O(r) where r = report row count
- **Template Storage**: O(d) where d = unique dates

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
1. Verify `financial_year_start` in config.yaml
2. Check month comparison logic (>= 3)
3. Validate year addition/subtraction
4. Test boundary dates (Feb 28/29, Mar 1)

### Exchange Rate Template Issues

**1. Missing Exchange Rates**

**Symptoms**: YAML template has "0.00" values only

**Debug Steps**:
1. Check if exchange_rates_template.yaml has been filled in
2. Verify usd_prices and usd_to_zar_rate fields populated
3. Validation log shows "Unique dates requiring rates" > 0

**2. Wrong Conversion Calculation**

**Symptoms**: exchange.csv has wrong equivalent units

**Debug Steps**:
1. Verify ZAR → USD conversion uses correct rate
2. Check USD → BTC/ETH/XRP/LTC conversion uses correct rates
3. Validate formula: units = ZAR / (USD × USD→CoinRate)
4. Example: 9,967.40 ZAR ÷ 13.50 = 738.33 USD ÷ 4,521.50 = 0.163 BTC

**3. Template Not Found**

**Symptoms**: Part 2 script cannot find exchange_rates_template.yaml

**Debug Steps**:
1. Verify exchange_templates/<timestamp>/ directory exists
2. Check for exchange_rates_template.yaml file
3. Verify Part 2 script finds latest version correctly

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

## Conclusion

This system provides accurate FIFO-based cost basis calculation for cryptocurrency transactions and exchange rate template generation for cross-currency tracking. The architecture supports multiple coins, proper financial year handling, and detailed transaction tracking.

For questions or improvements, refer to the component breakdown and code conventions sections above.
