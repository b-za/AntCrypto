# Crypto FIFO Tax Reports

This tool processes your crypto trading history to generate detailed First-In-First-Out (FIFO) inventory reports for tax purposes. It calculates cost basis, profits/losses, and end-of-year balances using FIFO accounting.

## What It Does

- **Categorizes Transactions:** Automatically sorts buys, sells, fees, and other transactions from your CSV data.
- **FIFO Tracking:** Tracks inventory lots, matching sells to oldest buys for accurate cost basis.
- **Buys for Others Detection:** Identifies buys made specifically for transfers/sends (based on time proximity and quantity match) and assigns them directly to those transactions.
- **Financial Year Reports:** Groups data by tax year (March 1 - Feb 28/29) as per South African tax rules.
- **Multiple Outputs:** Generates per-currency details, yearly summaries, and an overall overview.

## How to Use

1. **Prepare Your Data:**
   - Place your CSV files in the `data/` folder.
   - Each CSV should have columns: Timestamp (UTC), Description, Reference, Value amount, Balance delta, Currency.
   - Files are typically named like `ltc.csv`, `eth.csv`, `xbt.csv`, etc.

2. **Run the Reports:**
   - Open a terminal in the `scripts/` folder.
   - Run: `python main.py`
   - This executes all scripts in the correct order:
     1. `identify_buys_for_others.py` - finds buys made for transfers/sends
     2. `fifo_report.py` - generates FIFO and FY reports
     3. `overview_report.py` - generates overview summary

3. **Find Your Reports:**
   - Outputs are saved in a new timestamped folder under `reports/` (e.g., `reports/2025_12_19_1056/`).
   - Includes:
     - Individual currency reports (e.g., `ltc_fifo.csv`)
     - Yearly reports (e.g., `fy2021_report.csv`)
     - `overview_report.csv`

## Outputs Explained

- **Currency Reports:** Detailed transaction history with running balances.
- **Yearly Reports:** Sections for:
  - **Boughts:** Regular buys
  - **Solds:** Sales with FIFO lot assignment and profit/loss
  - **Buys for Others:** Buys identified as being for transfers/sends
  - **Others:** Transfers, sends, and other outflows with cost basis
  - **Balances:** End-of-year holdings per currency with lot details
  - **Fees:** Categorized by Buying, Selling, Other
- **Overview:** High-level gains/losses and holdings across all years.

Use these for your SARS crypto tax return. Consult a tax professional for advice.

## Requirements

- Python 3
- Standard libraries only (no extras needed)

For technical details, see `scripts/prompt.md`.