# FIFO Crypto Profit/Loss Calculator

A sophisticated cryptocurrency profit/loss calculation system using FIFO (First-In, First-Out) cost basis method. Designed for accurate tax reporting and financial analysis of cryptocurrency transactions across multiple coins.

## Problem Statement

Cryptocurrency taxation requires accurate tracking of cost basis for each transaction. When you sell cryptocurrency, you need to know:

1. **Which specific lots were sold** (not just average cost)
2. **What you paid for those lots** (cost basis)
3. **Profit or loss realized** (sale proceeds - cost basis)
4. **Tax implications** (held short-term vs long-term)

This program solves these challenges by:
- Using FIFO method to match sales with oldest acquisitions
- Tracking individual lots with unique references
- Calculating profit/loss for each sale transaction
- Aggregating results by financial year for tax reporting

## Features

- **Multi-Coin Support**: Process multiple cryptocurrencies (BTC, ETH, LTC, XRP, etc.)
- **FIFO Cost Basis**: Accurate lot matching with oldest-first allocation
- **Financial Year Reporting**: Automatic aggregation by fiscal year (configurable)
- **Comprehensive Reports**: Four detailed report types covering all aspects
- **Transaction Classification**: Automatic detection of transaction types from descriptions
- **Pool Management**: Intelligent grouping of transactions into acquisition pools
- **Error Logging**: Detailed error tracking for problematic transactions
- **Configurable**: Financial year start date, hash length, and multiple data roots

## Quick Start

### Prerequisites

- Go 1.21 or higher
- CSV transaction files in the expected format
- Config file (see Configuration section)

### Installation

```bash
cd scripts/fifo
go build -o fifo-calc
```

### Basic Usage

```bash
# Process all roots defined in config
./fifo-calc

# Process specific root(s) only
./fifo-calc --roots cap,cal
```

### Example Command

```bash
# From project root
cd scripts/fifo
go run main.go --config ../../config/config.yaml
```

Output will be generated in timestamped directories:
```
/Users/bh/code/beerguevara/AntCrypto/Crypto A&P/reports/2026_02_24_0001/
```

## Configuration

Create a `config.yaml` file:

```yaml
roots:
  - alias: "cap"    # Friendly name for data root
    path: "/path/to/Crypto A&P/data"
  - alias: "cal"
    path: "/path/to/Crypto Ant/data"

settings:
  financial_year_start: 3  # Financial year starts in March (1-12)
  hash_length: 7            # Length of generated lot reference hashes
```

### Configuration Options

| Option | Type | Default | Description |
|--------|-------|----------|-------------|
| `roots` | array | required | List of data directories to process |
| `roots[].alias` | string | required | Short identifier for root |
| `roots[].path` | string | required | Full path to data directory |
| `financial_year_start` | int | 3 | Month when financial year begins (1-12) |
| `hash_length` | int | 7 | Length of SHA-1 hash for lot references |

## Input Format

### Directory Structure

```
/path/to/data/
├── xbt.csv      # Bitcoin transactions
├── xrp.csv      # Ripple transactions
├── eth.csv      # Ethereum transactions
├── ltc.csv      # Litecoin transactions
└── zar.csv      # Fiat currency (ignored)
```

### CSV Format

Each coin's transactions are stored in a separate CSV file with the following columns:

| Column | Description | Example |
|---------|-------------|----------|
| Wallet ID | Unique wallet identifier | `3226120146793182111` |
| Row | Transaction row number | `1` |
| Timestamp (UTC) | Transaction date/time | `2024-02-15 00:23:54` |
| Description | Transaction description | `"Sold 0.05 BTC/ZAR @ 1,000,000"` |
| Currency | Cryptocurrency code | `XBT` |
| Balance delta | Quantity change (+/-) | `0.15934000` |
| Available balance delta | Available quantity change | `0.15934000` |
| Balance | Total balance after transaction | `0.15934000` |
| Available balance | Available balance after transaction | `0.15934000` |
| Cryptocurrency transaction ID | Blockchain transaction hash | `abc123...` |
| Cryptocurrency address | Wallet address | `1BvBMSEYstWetq...` |
| Value currency | Fiat currency code | `ZAR` |
| Value amount | Fiat value | `10000.00` |
| Reference | Unique transaction reference | `b98eb80d` |

### Example CSV Row

```csv
3226120146793182111,1,2024-02-15 00:23:54,"Sold 0.05 BTC/ZAR @ 1,000,000",XBT,-0.05375100,0.00000000,0.19892184,0.16190184,,,ZAR,53751.00,40253547
```

## Output Reports

The program generates four types of reports in a timestamped directory:

### 1. FIFO Reports (`{coin}_fifo.csv`)

**Per-coin FIFO ledger** showing all transactions with lot consumption.

**Columns:**
- Financial Year
- Transaction Reference
- Date
- Description
- Type
- Lot Reference
- Quantity Change
- Balance Units (running total)
- Total Cost (ZAR)
- Balance Value (ZAR)
- Fee (ZAR)
- Proceeds (ZAR)
- Profit (ZAR)
- Unit Cost (ZAR)

**Use Case:** Complete transaction history with cost basis tracking.

**Example:**
```csv
FY2018,b98eb80d,2017-08-28 21:43:03,"Bought BTC 0.15934",in_buy_for_other,in_buy_for_other_231126a25de1d7,0.15934,0.15934,10000,10000,0,0,0,62758.88
FY2018,037b4604,2017-08-28 22:09:07,Test Transaction,out_other,in_buy_for_other_231126a25de1d7,-0.00162595,0.15771405,0,9897.96,0,99.99,-2.05,62758.88
```

### 2. Profit/Loss Report (`financial_year_profit_loss.csv`)

**Aggregated profit/loss across all coins** by financial year.

**Columns:**
- Financial Year
- **Coin** ← NEW
- Sale Reference
- Lot Reference
- Quantity Sold
- Unit Cost (ZAR)
- Total Cost (ZAR)
- Selling Price (ZAR)
- Proceeds (ZAR)
- Profit/Loss (ZAR)
- Fee Reference
- Fee Amount (ZAR)

**Use Case:** Tax reporting and profit/loss analysis.

**Example:**
```csv
FY2024,XBT,40253547,in_buy_e0da843a4d1c8d,0.004796,150000,719.40,1000000,4796.00,4076.60,,0
FY2024,XRP,3a6478da,in_buy_31e76929561974,0.664,20.00,13.28,43.40,28.82,15.54,,0
FY2024,Combined,,7,0,60717.88,0,242384.74,181666.86,,0
```

**Summary Rows:**
- `Losses Total`: Total loss transactions
- `Profits Total`: Total profit transactions
- `Combined`: Aggregated totals

### 3. Inventory Report (`inventory.csv`)

**Current holdings across all coins** with cost basis.

**Columns:**
- Financial Year
- Coin
- Lot Reference
- Quantity
- Date of Original Purchase
- Unit Cost (ZAR)
- Total Cost (ZAR)
- Pool

**Use Case:** Current portfolio valuation and unrealized gains/losses.

**Example:**
```csv
FY2025,eth,in_buy_9f1aa62adbd007,1.992,2025-04-10 12:41:40,31420,62588.64,in_buy
FY2023,ltc,in_other_de5502fc177ad0,3.499,2023-12-08 14:23:05,1449.67,5072.39,in_other
FY2023,xbt,Total,0.06277584,,0,52553.42,
```

**Summary Rows:**
- `Total`: Aggregated quantity and value by FY and coin

### 4. Transfers Report (`transfers.csv`)

**All non-buy/sell transactions** across all coins.

**Columns:**
- Financial Year
- Coin
- Date
- Description
- Type
- Quantity Change
- Value (ZAR)
- Unit Cost (ZAR) / Unit Cost Value
- Lot Reference
- Pool

**Use Case:** Tracking transfers, gifts, fees, and other movements.

**Example:**
```csv
FY2024,XBT,2024-02-15 02:23:54,Trading fee,out_fee_sell,-0.000215,214.26,214.26,,out_fee_sell
FY2023,XRP,2023-12-08 13:45:18,Received Ripple,in_other,749.80,9339.61,12.46,in_other_440023b08a9aa9,in_other
```

## Financial Years

The program calculates financial years based on the `financial_year_start` configuration (default: March).

### Calculation Logic

```go
if month >= 3 {
    return fmt.Sprintf("FY%d", year+1)  // March onwards → next year's FY
}
return fmt.Sprintf("FY%d", year)        // Jan/Feb → current year's FY
```

### Examples

| Date | Month | Financial Year |
|------|-------|---------------|
| 2024-01-15 | Jan | FY2024 |
| 2024-02-28 | Feb | FY2024 |
| 2024-03-01 | Mar | FY2025 |
| 2024-06-15 | Jun | FY2025 |
| 2025-02-28 | Feb | FY2025 |
| 2025-03-01 | Mar | FY2026 |

### Default Financial Year Definition

**FY2025**: 1 March 2024 → 28 February 2025
**FY2026**: 1 March 2025 → 28 February 2026

This matches South African financial year requirements (March start).

## Architecture

### High-Level Flow

```
┌─────────────┐
│  CSV Files  │  (xbt.csv, xrp.csv, etc.)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Parser    │  → Read CSV, parse transactions
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Classifier  │  → Identify transaction types
└──────┬──────┘
       │
       ├──────────────┐
       │              │
       ▼              ▼
┌─────────────┐  ┌─────────────┐
│Pool Manager │  │ Fee Linker  │  (Classify pools, link fees)
└──────┬──────┘  └─────────────┘
       │
       ▼
┌─────────────┐
│   FIFO      │  → Allocate sales to lots
│ Allocator   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Report    │  → Generate all reports
│Coordinator │
└─────────────┘
       │
       ├──────────┬──────────┬──────────┐
       ▼          ▼          ▼          ▼
  FIFO.csv   ProfitLoss.csv Inventory.csv Transfers.csv
```

### Key Components

1. **Parser**: Reads CSV files and converts to internal transaction format
2. **Classifier**: Determines transaction type from description (buy, sell, fee, etc.)
3. **Pool Classifier**: Groups transactions into acquisition pools for cost basis tracking
4. **Fee Linker**: Associates fees with their parent transactions
5. **Pool Manager**: Maintains lots in different pools with FIFO order
6. **FIFO Allocator**: Matches sales to oldest lots first
7. **Report Coordinator**: Generates all report types from processed data

## Transaction Types

### Inflows (Acquisitions)

| Type | Description | Pool |
|------|-------------|-------|
| `in_buy` | Direct purchase | `in_buy` |
| `in_buy_for_other` | Purchase for transfer/gift | `in_buy_for_other` |
| `in_other` | Other inflow (e.g., received) | `in_other` |

### Outflows (Dispositions)

| Type | Description | Pool Priority |
|------|-------------|---------------|
| `out_sell` | Sale | `in_buy` → `in_buy_for_other` → `in_other` |
| `out_other` | Other outflow (transfer/gift) | `in_buy_for_other` → `in_other` → `in_buy` |
| `out_fee_buy` | Fee on purchase | Same as parent |
| `out_fee_buy_for_other` | Fee on purchase for other | Same as parent |
| `out_fee_in_other` | Fee on other inflow | Same as parent |
| `out_fee_sell` | Fee on sale | Same as parent |
| `out_fee_out_other` | Fee on other outflow | Same as parent |

## FIFO Allocation Algorithm

The FIFO allocator matches sales to acquisitions using the following rules:

1. **Pool Priority**: Consume lots in specific pool order
   - Sales: `in_buy` → `in_buy_for_other` → `in_other`
   - Other outflows: `in_buy_for_other` → `in_other` → `in_buy`

2. **Oldest First**: Within each pool, consume lots by acquisition date (oldest first)

3. **Specific Lot Allocation**: If transaction is linked to a specific lot (e.g., transfer), consume that lot first

4. **Partial Consumption**: Lots can be partially consumed, remaining quantity stays in pool

5. **Insufficient Lots**: Error if not enough lots available to fulfill transaction

### Example

```
Pool Status:
  in_buy:
    - Lot A: 0.5 BTC (purchased 2024-01-01)
    - Lot B: 0.3 BTC (purchased 2024-01-15)

Sale: 0.4 BTC on 2024-02-01
Allocation:
  - Consume 0.4 BTC from Lot A (oldest first)
  - Lot A remaining: 0.1 BTC

Sale: 0.2 BTC on 2024-02-15
Allocation:
  - Consume 0.1 BTC from Lot A (remainder)
  - Consume 0.1 BTC from Lot B
  - Lot B remaining: 0.2 BTC
```

## Pool Classification

Transactions are automatically classified into pools based on:

1. **Description Keywords**: "Bought" → `in_buy`, "Sold" → `out_sell`
2. **Time-Based Matching**: Outflows within 7 days of an inflow with ≥90% quantity match
3. **Fee Linking**: Fees automatically linked to parent transactions by reference

### Pool Classifier Rules

Matches outflows to inflows when:
- Time difference ≤ 7 days
- Quantity ratio ≥ 90% (outflow quantity / inflow quantity)
- Inflow occurs before outflow
- Best match is selected by minimum time difference

## Error Logging

Errors are logged to `error_log.jsonl` with details:

```json
{"type":"CLASSIFICATION_ERROR","message":"Failed to classify transaction","coin":"XBT","row":42,"timestamp":"2024-02-15 00:23:54"}
{"type":"ALLOCATION_ERROR","message":"Insufficient lots, still need 0.1","coin":"XBT","row":57,"timestamp":"2024-02-20 12:00:00"}
```

Error types:
- `CLASSIFICATION_ERROR`: Failed to determine transaction type
- `POOL_ERROR`: Failed to add lot to pool
- `ALLOCATION_ERROR`: Failed to allocate sale to lots
- `PROCESS_ERROR`: General processing failure

## Usage Examples

### Example 1: Process All Coins

```bash
cd scripts/fifo
go run main.go --config ../../config/config.yaml
```

Output:
```
Successfully processed root cap
Successfully processed root cal
```

### Example 2: Generate Reports for Specific Year

```bash
# Run program
go run main.go --config ../../config/config.yaml

# Reports generated in timestamped directory:
# /path/to/reports/2026_02_24_0001/
```

### Example 3: Analyze Profit/Loss by Financial Year

```bash
# View profit/loss report
cat /path/to/reports/2026_02_24_0001/financial_year_profit_loss.csv

# Filter for specific financial year
awk '/^FY2025,/ { print }' financial_year_profit_loss.csv
```

### Example 4: Check Current Inventory

```bash
# View inventory report
cat /path/to/reports/2026_02_24_0001/inventory.csv

# Calculate total value
awk 'NR>1 { sum += $7 } END { print "Total Value: " sum " ZAR" }' inventory.csv
```

## Development

### Building

```bash
cd scripts/fifo
go build -o fifo-calc
```

### Testing

```bash
cd scripts/fifo
go test ./...
```

### Dependencies

```bash
go mod download
go mod tidy
```

### Code Structure

```
scripts/fifo/
├── main.go                      # Entry point
├── go.mod                       # Go module definition
├── internal/
│   ├── config/                   # Configuration management
│   ├── parser/                   # CSV parsing
│   ├── classifier/               # Transaction classification
│   ├── pool/                     # Pool/lot management
│   ├── fifo/                     # FIFO allocation logic
│   └── report/                   # Report generation
└── pkg/
    └── utils/                    # Utility functions
```

## Troubleshooting

### Issue: Empty reports

**Cause:** No transactions processed or all transactions failed.

**Solution:**
1. Check error_log.jsonl for errors
2. Verify CSV files have correct format
3. Ensure currency codes match (case-insensitive)

### Issue: Incorrect profit/loss calculations

**Cause:** Wrong financial year configuration or transaction misclassification.

**Solution:**
1. Verify `financial_year_start` in config.yaml
2. Check transaction descriptions match expected patterns
3. Review classification in error_log.jsonl

### Issue: Missing transactions

**Cause:** Transactions not classified or filtered out.

**Solution:**
1. Check if ZAR transactions are being filtered (expected)
2. Verify reference IDs are unique
3. Review pool classification logic

### Issue: Lots not matching

**Cause:** Insufficient acquisitions or wrong pool classification.

**Solution:**
1. Verify purchase transactions have positive balance delta
2. Check pool classifier rules (7 days, 90% quantity)
3. Review allocation errors in error_log.jsonl

## Best Practices

1. **Regular Processing**: Run after each trading session to keep reports current
2. **Backup Reports**: Keep historical reports for year-over-year comparison
3. **Validate Inputs**: Ensure CSV files have correct headers and formatting
4. **Review Errors**: Check error_log.jsonl after each run for anomalies
5. **Archive Old Data**: Move processed CSV files to archive directory
6. **Track Configuration**: Version config.yaml with financial year changes

## Technical Notes

- **Decimal Precision**: Uses `shopspring/decimal` for exact monetary calculations
- **Hash Generation**: SHA-1 hash for unique lot references (configurable length)
- **Case Insensitivity**: Coin codes and descriptions are case-insensitive
- **Transaction Ordering**: Maintains original order for FIFO processing
- **Memory Efficiency**: Processes one root at a time, aggregates before reporting

## License

This is a private project for cryptocurrency tax calculation and financial analysis.

## Support

For issues or questions:
1. Check error_log.jsonl for specific errors
2. Review this documentation for configuration details
3. Verify input data format matches expected CSV structure
4. Validate financial year settings match your jurisdiction's requirements

## Changelog

### Recent Improvements

- ✅ **Fixed Data Aggregation Bug**: Reports now include all coins, not just the last processed coin
- ✅ **Fixed Inventory Coin Column**: Coin column now correctly populated with actual coin names
- ✅ **Fixed Financial Year Bug**: Correct FY calculation for March start year (FY2025 = 1 Mar 2024 to 28 Feb 2025)
- ✅ **Added Coin Column to Profit/Loss**: Each sale row now shows the coin that was sold
- ✅ **Removed Per-Coin Flag**: All coins now processed together for consistent reporting
- ✅ **Aggregated Reporting**: Single reports span all coins with correct financial year assignments

### Future Enhancements

- [ ] Support for additional cost basis methods (LIFO, HIFO, specific lot)
- [ ] Tax rate integration for automatic tax calculation
- [ ] Multiple currency support for international transactions
- [ ] Web-based dashboard for report visualization
- [ ] Automatic report email notifications
- [ ] Integration with tax preparation software exports
