# FIFO Crypto Profit/Loss Calculator

A sophisticated cryptocurrency profit/loss calculation system using FIFO (First-In, First-Out) cost basis method. Designed for accurate tax reporting and financial analysis of cryptocurrency transactions across multiple coins.

## Problem Statement

Cryptocurrency taxation requires accurate tracking of cost basis for each transaction. When you sell cryptocurrency, you need to know:
- **Which specific lots were sold** (not just average cost)
- **What you paid for those lots** (cost basis)
- **Profit or loss realized** (sale proceeds - cost basis)
- **Tax implications** (held short-term vs long-term)

This program solves these challenges by:
- Using FIFO method to match sales with oldest acquisitions
- Tracking individual lots with unique references
- Calculating profit/loss for each sale transaction
- Aggregating results by financial year for tax reporting
- Generating exchange rate templates for cross-currency tracking

## Features

### Multi-Coin Support
- Process multiple cryptocurrencies (BTC, ETH, LTC, XRP, etc.)
- - **FIFO Cost Basis**: Accurate lot matching with oldest-first allocation
- **Financial Year Reporting**: Automatic aggregation by fiscal year (configurable)
- **Comprehensive Reports**: Four detailed report types covering all aspects
- **Transaction Classification**: Automatic detection of transaction types
- **Pool Management**: Intelligent grouping of transactions into acquisition pools
- **Exchange Rate Templates**: YAML templates for cross-currency conversions (Step 1)

### Report Types
1. **FIFO Reports** (per-coin): Complete transaction history with lot consumption
2. **Profit/Loss Report**: Aggregated profit/loss by financial year across all coins
3. **Inventory Report**: Current holdings with cost basis
4. **Transfers Report**: All non-buy/sell transactions across all coins

### Key Capabilities
- Accurate FIFO cost basis tracking for tax purposes
- Detailed transaction classification (8 transaction types)
- Automatic pool detection for transfer tracking
- Comprehensive error logging for problematic transactions
- Configurable financial year (March start by default)
- Timestamped report directories for audit trail
- Exchange rate template generation for multi-currency analysis

## Quick Start

### Prerequisites
- Go 1.21 or higher
- CSV transaction files in expected format
- Config file (see Configuration section)

### Installation

```bash
cd scripts/fifo
go mod tidy
go build -o fifo-calc
```

### Basic Usage

```bash
# Process all roots defined in config
./fifo-calc

# Process specific root(s) only
./fifo-calc --roots cap

# From project root
cd scripts/fifo
go run main.go --config ../../config/config.yaml
```

### Example Command

```bash
# From project root
cd scripts/fifo
go run main.go --config ../../config/config.yaml --roots cap,cal
```

Output will be generated in timestamped directory:
```
/path/to/data/reports/2026_02_24_0001/
```

## Configuration

Create a `config.yaml` file:

```yaml
roots:
  - alias: "cap"
    path: "/path/to/Crypto A&P"
  - alias: "cal"
    path: "/path/to/Crypto Ant"

settings:
  financial_year_start: 3  # Financial year starts in March (1-12)
  hash_length: 7           # Length of SHA-1 hash for lot references

financial_year_start: 3  # Financial year starts in March (1-12)
```

### Configuration Options

| Option | Type | Default | Description |
|--------|-------|----------|-------------|
| `roots` | array | required | List of data directories to process |
| `roots[].alias` | string | required | Short identifier for root |
| `roots[].path` | string | required | Full path to data directory |
| `financial_year_start` | int | 3 | Month when financial year begins (1-12) |
| `hash_length` | int | 7 | Length of generated lot reference hashes |

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
|--------|-------------|----------|
| Wallet ID | Unique wallet identifier | `3226120146793182111` |
| Row | Transaction row number | `1` |
| Timestamp (UTC) | Transaction date/time | `2024-02-15 00:23:54` |
| Description | Transaction description | `"Sold 0.05 BTC/ZAR @ 1,000,000"` |
| Currency | Cryptocurrency code | `XBT` |
| Balance delta | Quantity change (+/-) | `-0.05375100` |
| Available balance delta | Available quantity change | `-0.05375100` |
| Balance | Total balance after transaction | `0.14934000` |
| Available balance | Available balance after transaction | `0.14934000` |
| Cryptocurrency transaction ID | Blockchain transaction hash | `abc123...` |
| Cryptocurrency address | Wallet address | `1Htve...` |
| Value currency | Fiat currency code | `ZAR` |
| Value amount | Fiat value amount | `10000.00` |
| Reference | Unique transaction reference | `b98eb80d` |

### Example CSV Row

```csv
3226120146793182111,1,2017-08-28 19:43:03,"Bought BTC 0.15934 for ZAR 10,000.00",XBT,0.15934000,0.15934000,0.15934000,0.15934000,,,ZAR,10000.00,b98eb80d
3226120146793182111,3,2017-08-28 20:09:07,Test Transaction,XBT,-0.00162595,0.00000000,0.15622401,0.15622401,0.15622401,81f444caf48f...,1Htve3kDRY1m5uG2nWUyni86kptUycNoMoMo,ZAR,99.99,037b4604
3226120146793182111,4,2017-08-28 20:09:07,Bitcoin send fee,XBT,-0.00149004,0.00000000,0.15622401,0.15622401,,,ZAR,91.64,037b4604
```

## Output Reports

### 1. FIFO Reports ({coin}_fifo.csv)

**Purpose**: Per-coin transaction history with cost basis tracking.

**Columns (14)**:
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

**Use Case**: Complete transaction history with cost basis tracking.

**Example**:
```csv
Financial Year,Transaction Reference,Date,Description,Type,Lot Reference,Quantity Change,Balance Units,Total Cost (ZAR),Balance Value (ZAR),Fee (ZAR),Proceeds (ZAR),Profit (ZAR),Unit Cost (ZAR)
FY2018,b98eb80d,2017-08-28 19:43:03,"Bought BTC 0.15934 for ZAR 10,000.00",in_buy,,,,,,0.15934,,10000.00,10000.00,,,0,,,10000.00,,0.15934
FY2018,037b4604,2017-08-28 20:09:07,Test Transaction,in_other,,,,,,,-0.00163,,,,0,,99.99,,-0.00163,,,,99.99,,,,99.99,,-0.00163,,,,0,,100.00
FY2018,7481d775,2017-08-30 18:32:21,all bitcoin,in_buy_for_other_23126a25de1d7,-0.15622401,,,-0.15622401,0.003183841,0.15622401,,,9,967.40,,0,,9,9287.96,722.05,62758.88,,967.40,,967.40,,,,967.40,0.00163
```

### 2. Profit/Loss Report (financial_year_profit_loss.csv)

**Purpose**: Aggregated profit/loss by financial year across all coins for tax reporting.

**Columns (12)**:
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

**Use Case**: Tax reporting and profit/loss analysis.

**Example**:
```csv
Financial Year,Coin,Sale Reference,Lot Reference,Quantity Sold,Unit Cost (ZAR),Total Cost (ZAR),Selling Price (ZAR),Proceeds (ZAR),Profit/Loss (ZAR),Fee Reference,Fee Amount (ZAR)
FY2024,XBT,40253547,in_buy_e0da843a4d1c8d,0.004796150000,71940.40,10000,10000.00,4076.60,,0.004796150,,4076.60
FY2024,XRP,3a6478da,in_buy_31e76929561974,0.66420.00,13.28,43.40,28.82,,0,15.54,,0.15.54
FY2024,Combined,,,7,060717.88,242384.74,181666.86,,967.40,722.05,62758.88,967.40
```

**Summary Rows**:
- `Losses Total`: Aggregates all loss sales for FY
- `Profits Total`: Aggregates all profit sales for FY
- `Combined`: Total aggregation including both losses and profits

### 3. Inventory Report (inventory.csv)

**Purpose**: Current holdings across all coins with cost basis.

**Columns (8)**:
- Financial Year
- Coin
- Lot Reference
- Quantity
- Date of Original Purchase
- Unit Cost (ZAR)
- Total Cost (ZAR)
- Pool

**Use Case**: Current portfolio valuation and unrealized gains/losses.

**Example**:
```csv
Financial Year,Coin,Lot Reference,Quantity,Date of Original Purchase,Unit Cost (ZAR),Total Cost (ZAR),Pool
FY2025,eth,in_buy_9f1aa62adbd007,1.992,2025-04-10 12:41:40,31420.62588.64,in_buy
FY2025,eth,in_buy_9f1aa62adbd007,0.00162595,0.5072.39,in_buy
FY2025,ltc,in_other_de5502fc177ad0,3.499,2023-12-08 14:23.05,1449.67,50.72.39,in_other
```

**Summary Rows**:
- `Total`: Aggregated quantity and value per FY and coin

### 4. Transfers Report (transfers.csv)

**Purpose**: All non-buy/sell transactions across all coins.

**Columns (10)**:
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

**Use Case**: Tracking transfers, gifts, fees, and other movements.

**Example**:
```csv
Financial Year,Coin,Date,Description,Type,Quantity Change,Value (ZAR),Unit Cost (ZAR) / Unit Cost Value,Lot Reference,Pool
FY2018,XBT,2017-08-30 18:32:21,all bitcoin,out_other,-0.15622401,9,967.40,,Lot A,,,in_buy
FY2018,XBT,2024-02-15 00:23:54,Trading fee,out_fee_sell,-0.000215,214.26,,,,in_buy
FY2023,XRP,2023-12-08 13:45:18,Received Ripple,in_other,749.80,9339.61,12.46,in_other
```

## Exchange Rate Templates (New)

### Exchange Rate Template Generator (`scripts/generate_exchange_rates/`)

**Purpose**: Generate YAML templates with placeholder exchange rates for cross-currency tracking (Step 1 of 2-step process).

**Output Location**:
```
<path/to/data>/exchange_templates/<timestamp>/
  ├── exchange_rates_template.yaml           # YOUR TEMPLATE FILE
  └── exchange_rates_template_validation.log # Validation summary
```

### Template Structure
```yaml
dates_required:
  - date: "2017-08-28"
      transaction_references:
        - 037b4604
        - 7481d775
      usd_prices:
        btc: "0.00"    # ← FILL IN: Bitcoin price in USD
        eth: "0.00"    # ← FILL IN: Example: "4521.50"
        xrp: "0.00"    # ← FILL IN: Example: "0.0075"
        ltc: "0.00"    # ← FILL IN: Example: "55.00"
      usd_to_zar_rate: "0.00"  # ← FILL IN: Example: "13.50"
```

### Usage

```bash
cd scripts/generate_exchange_rates
go mod tidy
GO111MODULE=on go build -o generate_exchange_rates
./generate_exchange_rates --config ../../config.yaml --roots cap
```

### Filling in Exchange Rates

After generating the template, manually fill in:
1. **usd_prices**: Historical USD prices for each coin on each date
2. **usd_to_zar_rate**: USD → ZAR exchange rate on each date

### Part 2 Workflow (To Be Implemented)

After filling in the template, Part 2 script will:
1. Find latest exchange_rates_template.yaml in exchange_templates/
2. Load the filled rates
3. Generate exchange.csv with cross-currency conversions
4. Calculate equivalent units for each transaction in all coins
5. Determine net inflow/outflow by currency

### Example Filled Template
```yaml
  - date: "2017-08-30"
      usd_prices:
        btc: "4521.50"    # BTC was $4,521.50 on this date
        eth: "320.00"      # ETH was $320.00 on this date
        xrp: "0.0075"      # XRP was $0.0075 on this date
        ltc: "55.00"       # LTC was $55.00 on this date
      usd_to_zar_rate: "13.50"  # 1 USD = 13.50 ZAR
```

## Financial Years

The program calculates financial years based on the `financial_year_start` configuration (default: March start).

### Calculation Logic
```go
if month >= 3 {
    return fmt.Sprintf("FY%d", year + 1)  // March onwards → next year's FY
}
return fmt.Sprintf("FY%d", year)           // Jan/Feb → current year's FY
```

### Examples
| Date | Month | Financial Year |
|------|-------|---------------|
| 2024-02-15 | Feb | FY2024 |
| 2024-03-01 | Mar | FY2025 |
| 2024-06-15 | Jun | FY2025 |
| 2025-02-28 | Feb | FY2025 |

## Troubleshooting

### Issue: Empty Reports

**Cause**: No transactions processed or all transactions failed.

**Solution**:
1. Check error_log.jsonl for parsing errors
2. Verify CSV files have correct format
3. Ensure currency codes match (case-insensitive)
4. Review configuration file

### Issue: Incorrect Profit/Loss

**Cause**: Wrong financial year configuration or transaction misclassification.

**Solution**:
1. Verify `financial_year_start` in config.yaml
2. Check transaction descriptions match expected patterns
3. Review classification in error_log.jsonl
4. Calculate expected profit manually and compare

### Issue: Missing Lots

**Cause**: Insufficient purchases to fulfill sales.

**Solution**:
1. Check pool manager initialization
2. Verify in_buy transactions have positive balance delta
3. Review lot consumption order in error_log.jsonl
4. Check if transfers are being linked correctly

## Technical Debt

### Development

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

## License

This is a private project for cryptocurrency tax calculation and financial analysis.
