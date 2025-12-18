#!/usr/bin/env python3
import csv
import os
import glob
from decimal import Decimal
from collections import defaultdict

def parse_fy_report(filepath):
    fy = int(os.path.basename(filepath).split('_')[0][2:])
    transactions = []
    balances = defaultdict(lambda: {'units': Decimal('0'), 'value': Decimal('0')})

    with open(filepath, 'r') as f:
        reader = csv.reader(f)
        section = None
        for row in reader:
            if not row:
                continue
            if row[0] == 'Solds for FY':
                section = 'transactions'
            elif row[0] == 'Balances at end of FY':
                section = 'balances'
            elif section == 'transactions' and len(row) >= 10 and row[0] and row[0] != 'Date':
                try:
                    proceeds = Decimal(row[8].replace(',', ''))
                    cost = Decimal(row[7].replace(',', ''))
                    profit = Decimal(row[9].replace(',', ''))
                    transactions.append((proceeds, cost, profit))
                except:
                    pass
            elif section == 'balances' and len(row) >= 3 and row[0] and row[0] != 'Currency' and row[1] and row[2]:
                ccy = row[0]
                units_str = row[1].replace(',', '')
                value_str = row[2].replace(',', '')
                try:
                    units = Decimal(units_str)
                    value = Decimal(value_str)
                    balances[ccy]['units'] = units
                    balances[ccy]['value'] = value
                except:
                    pass

    # For losses (profit < 0)
    loss_trans = [t for t in transactions if t[2] < 0]
    proceeds_loss = sum(t[0] for t in loss_trans)
    cost_loss = sum(t[1] for t in loss_trans)
    profit_loss = sum(t[2] for t in loss_trans)

    # For gains (profit > 0)
    gain_trans = [t for t in transactions if t[2] > 0]
    proceeds_gain = sum(t[0] for t in gain_trans)
    cost_gain = sum(t[1] for t in gain_trans)
    profit_gain = sum(t[2] for t in gain_trans)

    return fy, proceeds_loss, cost_loss, profit_loss, proceeds_gain, cost_gain, profit_gain, balances

def main():
    reports_dir = '../reports'
    # Find latest folder
    subdirs = [d for d in os.listdir(reports_dir) if os.path.isdir(os.path.join(reports_dir, d))]
    subdirs.sort(reverse=True)
    latest_dir = os.path.join(reports_dir, subdirs[0])

    fy_files = glob.glob(os.path.join(latest_dir, 'fy*_report.csv'))

    overview = []
    for f in fy_files:
        fy, proceeds_loss, cost_loss, profit_loss, proceeds_gain, cost_gain, profit_gain, balances = parse_fy_report(f)
        net = profit_loss + profit_gain
        total_value = sum(balances[ccy]['value'] for ccy in balances)
        overview.append((fy, proceeds_loss, cost_loss, profit_loss, proceeds_gain, cost_gain, profit_gain, net, total_value, balances))

    overview.sort(key=lambda x: x[0])

    # Output CSV
    output_file = os.path.join(latest_dir, 'overview_report.csv')
    with open(output_file, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow(['FY', 'Losses Proceeds (ZAR)', 'Losses Base Cost (ZAR)', 'Losses Gain/Loss (ZAR)', 'Gains Proceeds (ZAR)', 'Gains Base Cost (ZAR)', 'Gains Gain/Loss (ZAR)', 'Net Gain/Loss (ZAR)', 'Total Coin Value (ZAR)', 'BCH Units', 'BCH Value (ZAR)', 'ETH Units', 'ETH Value (ZAR)', 'XBT Units', 'XBT Value (ZAR)', 'XRP Units', 'XRP Value (ZAR)', 'LTC Units', 'LTC Value (ZAR)'])
        for fy, proceeds_loss, cost_loss, profit_loss, proceeds_gain, cost_gain, profit_gain, net, total_value, bal in overview:
            writer.writerow([
                fy,
                f"{proceeds_loss:.2f}",
                f"{cost_loss:.2f}",
                f"{profit_loss:.2f}",
                f"{proceeds_gain:.2f}",
                f"{cost_gain:.2f}",
                f"{profit_gain:.2f}",
                f"{net:.2f}",
                f"{total_value:.2f}",
                f"{bal['BCH']['units']:.8f}",
                f"{bal['BCH']['value']:.2f}",
                f"{bal['ETH']['units']:.8f}",
                f"{bal['ETH']['value']:.2f}",
                f"{bal['XBT']['units']:.8f}",
                f"{bal['XBT']['value']:.2f}",
                f"{bal['XRP']['units']:.8f}",
                f"{bal['XRP']['value']:.2f}",
                f"{bal['LTC']['units']:.8f}",
                f"{bal['LTC']['value']:.2f}",
            ])

    print(f"Wrote {output_file}")

if __name__ == '__main__':
    main()