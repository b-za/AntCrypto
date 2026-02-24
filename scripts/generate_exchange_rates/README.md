# Exchange Rate Template Generator (Step 1)

A standalone program to generate a YAML template with all dates that need exchange rate data for cross-currency cryptocurrency transactions.

## Purpose

This program scans all `in_other` and `out_other` cryptocurrency transactions and generates a YAML template with placeholder values (0.00). You will manually fill in the exchange rates, then Part 2 will use this data for cross-currency conversions.

## Prerequisites

- Go 1.21 or higher
- Existing config.yaml file in parent directory

## Installation & Build

```bash
cd scripts/generate_exchange_rates
go mod tidy
GO111MODULE=on GOMOD="$(pwd)/go.mod" go build -mod=mod -o generate_exchange_rates
```

## Usage

```bash
# Process all roots defined in config
./generate_exchange_rates

# Process specific root(s) only
./generate_exchange_rates --roots cap,cal

# Custom config location
./generate_exchange_rates --config path/to/config.yaml
```

## Output

The program generates three files in a timestamped directory under the first root's reports folder:

```
<root>/reports/<timestamp>/
  ├── exchange_rates_template.yaml           # Main template file (YOU FILL THIS IN)
  ├── exchange_rates_template_validation.log # Validation summary
  └── exchange_rates_error_log.jsonl # Error log (if any)
```

## YAML Template Structure

```yaml
dates_required:
  - date: "2017-08-28"
      transaction_references:
        - 037b4604
        - another_ref_same_day
      usd_prices:
        btc: "0.00"      # FILL IN: Bitcoin price in USD on this date
        eth: "0.00"      # FILL IN: Ethereum price in USD on this date
        xrp: "0.00"      # FILL IN: Ripple price in USD on this date
        ltc: "0.00"      # FILL IN: Litecoin price in USD on this date
      usd_to_zar_rate: "0.00"  # FILL IN: USD to ZAR rate on this date

  - date: "2017-08-30"
      transaction_references:
        - 7481d775
      usd_prices:
        btc: "0.00"
        eth: "0.00"
        xrp: "0.00"
        ltc: "0.00"
      usd_to_zar_rate: "0.00"

usage: For each date, fill in the USD prices (BTC, ETH, XRP, LTC) and USD→ZAR exchange rate. These will be used in Part 2 to calculate cross-currency conversions.
```

## How to Fill In the Template

1. **Look up historical prices** for each date using your preferred source:
   - CoinGecko, Binance, Yahoo Finance, etc.
   - Find the closing price or average price for each coin on that date

2. **Fill in the usd_prices fields**:
   - Find the USD price for BTC on 2017-08-28
   - Find the USD price for ETH on 2017-08-28
   - Find the USD price for XRP on 2017-08-28
   - Find the USD price for LTC on 2017-08-28

3. **Fill in the usd_to_zar_rate field**:
   - Find the USD → ZAR exchange rate on that date
   - Use 1 ZAR = X USD rate (e.g., 13.50 = 1 USD)

4. **Save the updated YAML file**

## Example Filled Data

For date "2017-08-30" with transactions 7481d775:

```yaml
  - date: "2017-08-30"
      transaction_references:
        - 7481d775
      usd_prices:
        btc: "4521.50"    # Example: BTC was $4,521.50 on this date
        eth: "320.00"      # Example: ETH was $320.00 on this date
        xrp: "0.0075"      # Example: XRP was $0.0075 on this date
        ltc: "55.00"       # Example: LTC was $55.00 on this date
      usd_to_zar_rate: "13.50"    # Example: 1 USD = 13.50 ZAR
```

## Transaction Classification

The program filters transactions based on description and balance delta:

| Description Pattern | Balance Delta | Type |
|--------------------|----------------|------|
| "bought..." | Positive | `buy` (excluded) |
| "sold..." | Negative | `sell` (excluded) |
| Contains "fee" | Negative | `fee` (excluded) |
| Other inflow | Positive | `in_other` (**included**) |
| Other outflow | Negative | `out_other` (**included**) |

## Next Steps

After completing this template:

1. ✅ **Step 1 COMPLETE**: Template generated with placeholder values
2. ⏳ **Step 2**: Fill in actual exchange rates in the YAML
3. ⏳ **Step 3**: Run Part 2 script (to be provided) that will:
   - Read the filled YAML
   - Calculate equivalent units for each transaction
   - Generate exchange.csv report

## Notes

- The template groups transactions by date to minimize the number of rates you need to look up
- If multiple transactions occurred on the same date, they share one set of rates
- All dates from in_other and out_other transactions are included
- The usd_to_zar_rate is the same for all coins on a given date
