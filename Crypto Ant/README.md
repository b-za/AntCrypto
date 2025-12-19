# Crypto FIFO Tax Reports

This tool processes your crypto trading history to generate detailed First-In-First-Out (FIFO) inventory reports for tax purposes. It calculates cost basis, profits/losses, and end-of-year balances using FIFO accounting.

## What It Does

- **Categorizes Transactions:** Automatically sorts buys, sells, fees, and other transactions from your CSV data.
- **FIFO Tracking:** Tracks inventory lots, matching sells to oldest buys for accurate cost basis.
- **Financial Year Reports:** Groups data by tax year (March 1 - Feb 28/29) as per South African tax rules.
- **Multiple Outputs:** Generates per-currency details, yearly summaries, and an overall overview.

## How to Use

1. **Prepare Your Data:**
   - Place your CSV files in the `data/` folder.
   - Each CSV should have columns: Timestamp (UTC), Description, Reference, Value amount, Balance delta, Currency.
   - Files are typically named like `ltc.csv`, `eth.csv`, etc.

2. **Run the Reports:**
   - Open a terminal in the `scripts/` folder.
   - Run: `python fifo_report.py`
   - Then run: `python overview_report.py`

3. **Find Your Reports:**
   - Outputs are saved in a new timestamped folder under `reports/` (e.g., `reports/2025_12_19_0725/`).
   - Includes individual currency reports (e.g., `ltc_fifo.csv`), yearly reports (e.g., `fy2021_report.csv`), and `overview_report.csv`.

## Outputs Explained

- **Currency Reports:** Detailed transaction history with balances and profits.
- **Yearly Reports:** Summaries of buys, sells, fees, and balances per tax year.
- **Overview:** High-level gains/losses and holdings across all years.

Use these for your SARS crypto tax return. Consult a tax professional for advice.

## Requirements

- Python 3
- Standard libraries (no extras needed)

For technical details, see `scripts/prompt.md`.