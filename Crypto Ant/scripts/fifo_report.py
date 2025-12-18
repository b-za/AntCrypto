#!/usr/bin/env python3
import csv
from decimal import Decimal, getcontext, ROUND_HALF_UP
from datetime import datetime
from collections import deque, defaultdict
import sys
import os
import glob

getcontext().prec = 28



# Utility functions
ALPHABET = [chr(c) for c in range(ord('A'), ord('Z')+1)]

def gen_txn_ids(prefix):
    # Yields prefix + 000, 001, ...
    count = 0
    while True:
        yield f"{prefix}{count:03d}"
        count += 1


def parse_dt(s):
    # Example format: 2021-01-06 12:25:17
    return datetime.strptime(s, '%Y-%m-%d %H:%M:%S')


def financial_year(dt):
    # FY runs Mar (3) .. Feb (2). If month >= 3, FY = year + 1; else year
    return dt.year + 1 if dt.month >= 3 else dt.year


def dec(x):
    if isinstance(x, Decimal):
        return x
    s = str(x).strip()
    if s == '':
        return Decimal('0')
    return Decimal(s)


def q8(x: Decimal) -> str:
    return f"{x.quantize(Decimal('0.00000001'), rounding=ROUND_HALF_UP)}"


def r2(x: Decimal) -> Decimal:
    return x.quantize(Decimal('0.01'), rounding=ROUND_HALF_UP)


def s2(x: Decimal) -> str:
    return f"{r2(x)}"

class Lot:
    __slots__ = ('qty', 'unit_cost', 'ref')
    def __init__(self, qty: Decimal, unit_cost: Decimal, ref: str):
        self.qty = qty
        self.unit_cost = unit_cost
        self.ref = ref


def generate_fy_report(fy, sales, lots_by_ccy, balance_units, balance_value, output_dir, timestamp):
    output_csv = os.path.join(output_dir, f"fy{fy}_report_{timestamp}.csv")
    with open(output_csv, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow(['Transactions for FY', fy])
        writer.writerow(['Date', 'Currency', 'Trans Ref', 'Lot Ref', 'Qty Sold', 'Unit Cost (ZAR)', 'Total Cost (ZAR)', 'Proceeds (ZAR)', 'Profit (ZAR)', 'Fee (ZAR)'])
        for sale in sales:
            writer.writerow([sale['Date'], sale['Currency'], sale['Trans Ref'], sale['Lot Ref'], sale['Qty Sold'], sale['Unit Cost'], sale['Total Cost'], sale['Proceeds'], sale['Profit'], sale.get('Fee (ZAR)', '0.00')])
        writer.writerow([])
        writer.writerow(['Balances at end of FY', fy])
        writer.writerow(['Currency', 'Balance Units', 'Balance Value (ZAR)', 'Remaining Lots'])
        for ccy in sorted(lots_by_ccy.keys()):
            units = balance_units[ccy]
            value = balance_value[ccy]
            lots_str = '; '.join(f"{lot.ref}:{q8(lot.qty)}:{s2(lot.unit_cost)}" for lot in lots_by_ccy[ccy])
            writer.writerow([ccy, q8(units), s2(value), lots_str])

    print(f"Wrote {output_csv}")


def main(input_csv, output_csv):
    rows = []
    with open(input_csv, newline='') as f:
        reader = csv.DictReader(f)
        for row in reader:
            row['Balance delta'] = dec(row['Balance delta'])
            row['Value amount'] = dec(row['Value amount'])
            row['_dt'] = parse_dt(row['Timestamp (UTC)'])
            rows.append(row)

    rows.sort(key=lambda r: r['_dt'])

    lots_by_ccy = defaultdict(deque)
    balance_units = defaultdict(lambda: Decimal('0'))
    balance_value = defaultdict(lambda: Decimal('0'))

    # Transaction ID generators per Type per FY to keep IDs smaller (optional)
    # But user only requires unique 3-letter per transaction overall; simpler: global counters
    buy_id_gen = gen_txn_ids('B')
    sell_id_gen = gen_txn_ids('S')

    output_rows = []

    for row in rows:
        ccy = row['Currency']
        dt = row['_dt']
        fy = financial_year(dt)
        qty_delta = row['Balance delta']
        desc = row['Description']
        ref = row['Reference']
        value_amount = row['Value amount']

        if qty_delta == 0:
            continue

        if 'fee' in desc.lower():
            # Treat as fee, include in output but no inventory change
            output_rows.append({
                'Financial Year': fy,
                'Trans Ref': '',
                'Date': row['Timestamp (UTC)'],
                'Description': desc,
                'Type': 'Fee',
                'Lot Reference': '',
                'Qty Change': '',
                'Unit Cost (ZAR)': '',
                'Total Cost (ZAR)': '',
                'Proceeds (ZAR)': '',
                'Profit (ZAR)': '',
                'Fee (ZAR)': s2(value_amount),
                'Balance Units': q8(balance_units[ccy]),
                'Balance Value (ZAR)': s2(balance_value[ccy]),
            })
            continue

        if qty_delta > 0:
            # Buy lot
            qty = qty_delta
            unit_cost = value_amount / qty if qty != 0 else Decimal('0')
            lots_by_ccy[ccy].append(Lot(qty=qty, unit_cost=unit_cost, ref=ref))

            balance_units[ccy] += qty
            total_cost = qty * unit_cost
            balance_value[ccy] += total_cost

            trans_id = next(buy_id_gen)

            output_rows.append({
                'Financial Year': fy,
                'Trans Ref': trans_id,
                'Date': row['Timestamp (UTC)'],
                'Description': desc,
                'Type': 'Buy',
                'Lot Reference': ref,
                'Qty Change': q8(qty),
                'Unit Cost (ZAR)': s2(unit_cost),
                'Total Cost (ZAR)': s2(total_cost),
                'Proceeds (ZAR)': s2(Decimal('0')),
                'Profit (ZAR)': s2(Decimal('0')),
                'Fee (ZAR)': s2(Decimal('0')),
                'Balance Units': q8(balance_units[ccy]),
                'Balance Value (ZAR)': s2(balance_value[ccy]),
            })
        else:
            # Sell (or any outflow)
            sell_qty = -qty_delta  # positive
            proceeds_total = value_amount
            # If there are no lots, we will create a negative inventory entry (allowed here)
            # But better: consume from empty -> create lot with zero cost to allow proceeds
            if not lots_by_ccy[ccy]:
                lots_by_ccy[ccy].append(Lot(qty=Decimal('0'), unit_cost=Decimal('0'), ref='N/A'))

            trans_id = next(sell_id_gen)
            remaining = sell_qty

            # Pre-calc to allocate proceeds proportionally by quantity
            total_qty_for_sale = sell_qty

            while remaining > Decimal('0.0000000001') and lots_by_ccy[ccy]:
                lot = lots_by_ccy[ccy][0]
                consume = lot.qty if lot.qty <= remaining else remaining
                if consume <= 0:
                    break

                unit_cost = lot.unit_cost
                total_cost = consume * unit_cost
                # Allocate proceeds proportionally by fraction of quantity sold in this split
                split_proceeds = proceeds_total * (consume / total_qty_for_sale)
                profit = split_proceeds - total_cost

                # Update lot and balances
                lot.qty -= consume
                if lot.qty <= Decimal('0.0000000001'):
                    lots_by_ccy[ccy].popleft()

                balance_units[ccy] -= consume
                balance_value[ccy] -= total_cost

                output_rows.append({
                    'Financial Year': fy,
                    'Trans Ref': trans_id,
                    'Date': row['Timestamp (UTC)'],
                    'Description': desc,
                    'Type': 'Sell',
                    'Lot Reference': lot.ref,
                    'Qty Change': q8(-consume),
                    'Unit Cost (ZAR)': s2(unit_cost),
                    'Total Cost (ZAR)': s2(total_cost.copy_abs()),
                    'Proceeds (ZAR)': s2(split_proceeds),
                    'Profit (ZAR)': s2(profit),
                    'Fee (ZAR)': s2(Decimal('0')),
                    'Balance Units': q8(balance_units[ccy]),
                    'Balance Value (ZAR)': s2(balance_value[ccy]),
                })

                remaining -= consume

            # If after consuming all available lots we still have remaining (shouldn't usually happen),
            # treat remainder as sold from zero-cost lot
            if remaining > Decimal('0.0000000001'):
                unit_cost = Decimal('0')
                total_cost = Decimal('0')
                split_proceeds = proceeds_total * (remaining / total_qty_for_sale)
                profit = split_proceeds - total_cost
                balance_units[ccy] -= remaining
                # balance_value unchanged as zero cost
                output_rows.append({
                    'Financial Year': fy,
                    'Trans Ref': trans_id,
                    'Date': row['Timestamp (UTC)'],
                    'Description': desc,
                    'Type': 'Sell',
                    'Lot Reference': 'N/A',
                    'Qty Change': q8(-remaining),
                    'Unit Cost (ZAR)': s2(unit_cost),
                    'Total Cost (ZAR)': s2(total_cost.copy_abs()),
                    'Proceeds (ZAR)': s2(split_proceeds),
                    'Profit (ZAR)': s2(profit),
                    'Fee (ZAR)': s2(Decimal('0')),
                    'Balance Units': q8(balance_units[ccy]),
                    'Balance Value (ZAR)': s2(balance_value[ccy]),
                })
                remaining = Decimal('0')

    # Write output CSV
    fieldnames = [
        'Financial Year','Trans Ref','Date','Description','Type','Lot Reference',
        'Qty Change','Unit Cost (ZAR)','Total Cost (ZAR)','Proceeds (ZAR)','Profit (ZAR)',
        'Fee (ZAR)','Balance Units','Balance Value (ZAR)'
    ]
    with open(output_csv, 'w', newline='') as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        for orow in output_rows:
            writer.writerow(orow)

    print(f"Wrote {output_csv} with {len(output_rows)} rows.")


def process_fy(csv_files, output_dir, timestamp):
    rows = []
    for csv_file in csv_files:
        with open(csv_file, newline='') as f:
            reader = csv.DictReader(f)
            for row in reader:
                row['Balance delta'] = dec(row['Balance delta'])
                row['Value amount'] = dec(row['Value amount'])
                row['_dt'] = parse_dt(row['Timestamp (UTC)'])
                rows.append(row)

    rows.sort(key=lambda r: r['_dt'])

    lots_by_ccy = defaultdict(deque)
    balance_units = defaultdict(lambda: Decimal('0'))
    balance_value = defaultdict(lambda: Decimal('0'))

    sales_per_fy = defaultdict(list)

    current_fy = None

    buy_id_gen = gen_txn_ids('B')
    sell_id_gen = gen_txn_ids('S')

    for row in rows:
        ccy = row['Currency']
        dt = row['_dt']
        fy = financial_year(dt)
        qty_delta = row['Balance delta']
        desc = row['Description']
        ref = row['Reference']
        value_amount = row['Value amount']

        if qty_delta == 0:
            continue

        if 'fee' in desc.lower():
            sales_per_fy[fy].append({
                'Date': row['Timestamp (UTC)'],
                'Currency': ccy,
                'Trans Ref': '',
                'Lot Ref': '',
                'Qty Sold': '',
                'Unit Cost': '',
                'Total Cost': '',
                'Proceeds': '',
                'Profit': '',
                'Fee (ZAR)': s2(value_amount),
            })
            continue

        if current_fy is not None and fy != current_fy:
            generate_fy_report(current_fy, sales_per_fy[current_fy], lots_by_ccy, balance_units, balance_value, output_dir, timestamp)

        current_fy = fy

        if qty_delta > 0:
            qty = qty_delta
            unit_cost = value_amount / qty if qty != 0 else Decimal('0')
            lots_by_ccy[ccy].append(Lot(qty=qty, unit_cost=unit_cost, ref=ref))
            balance_units[ccy] += qty
            total_cost = qty * unit_cost
            balance_value[ccy] += total_cost
        else:
            sell_qty = -qty_delta
            proceeds_total = value_amount
            if not lots_by_ccy[ccy]:
                lots_by_ccy[ccy].append(Lot(qty=Decimal('0'), unit_cost=Decimal('0'), ref='N/A'))

            trans_id = next(sell_id_gen)
            remaining = sell_qty
            total_qty_for_sale = sell_qty

            while remaining > Decimal('0.0000000001') and lots_by_ccy[ccy]:
                lot = lots_by_ccy[ccy][0]
                consume = lot.qty if lot.qty <= remaining else remaining
                if consume <= 0:
                    break

                unit_cost = lot.unit_cost
                total_cost = consume * unit_cost
                split_proceeds = proceeds_total * (consume / total_qty_for_sale)
                profit = split_proceeds - total_cost

                lot.qty -= consume
                if lot.qty <= Decimal('0.0000000001'):
                    lots_by_ccy[ccy].popleft()

                balance_units[ccy] -= consume
                balance_value[ccy] -= total_cost

                sales_per_fy[fy].append({
                    'Date': row['Timestamp (UTC)'],
                    'Currency': ccy,
                    'Trans Ref': trans_id,
                    'Lot Ref': lot.ref,
                    'Qty Sold': q8(-consume),
                    'Unit Cost': s2(unit_cost),
                    'Total Cost': s2(total_cost.copy_abs()),
                    'Proceeds': s2(split_proceeds),
                    'Profit': s2(profit),
                    'Fee (ZAR)': s2(Decimal('0')),
                })

                remaining -= consume

            if remaining > Decimal('0.0000000001'):
                unit_cost = Decimal('0')
                total_cost = Decimal('0')
                split_proceeds = proceeds_total * (remaining / total_qty_for_sale)
                profit = split_proceeds - total_cost
                balance_units[ccy] -= remaining
                sales_per_fy[fy].append({
                    'Date': row['Timestamp (UTC)'],
                    'Currency': ccy,
                    'Trans Ref': trans_id,
                    'Lot Ref': 'N/A',
                    'Qty Sold': q8(-remaining),
                    'Unit Cost': s2(unit_cost),
                    'Total Cost': s2(total_cost.copy_abs()),
                    'Proceeds': s2(split_proceeds),
                    'Profit': s2(profit),
                    'Fee (ZAR)': s2(Decimal('0')),
                })
                remaining = Decimal('0')

    if current_fy is not None:
        generate_fy_report(current_fy, sales_per_fy[current_fy], lots_by_ccy, balance_units, balance_value, output_dir, timestamp)


if __name__ == '__main__':
    data_dir = '../data'
    csv_files = glob.glob(os.path.join(data_dir, '*.csv'))
    timestamp = datetime.now().strftime('%Y_%m_%d_%H%M')
    for csv_file in csv_files:
        base = os.path.basename(csv_file).rsplit('.', 1)[0]
        output_csv = os.path.join('../reports', f"{base}_fifo_{timestamp}.csv")
        main(csv_file, output_csv)
    process_fy(csv_files, '../reports', timestamp)
