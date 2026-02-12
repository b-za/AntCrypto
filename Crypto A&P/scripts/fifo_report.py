#!/usr/bin/env python3
import csv
import json
from decimal import Decimal, getcontext, ROUND_HALF_UP
from datetime import datetime
from collections import deque, defaultdict
import sys
import os
import glob

getcontext().prec = 28

def load_buys_for_others_mapping():
    mapping_file = os.path.join(os.path.dirname(__file__), '..', 'data', 'buys_for_others.json')
    if os.path.exists(mapping_file):
        with open(mapping_file, 'r') as f:
            return json.load(f)
    return {}



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


def generate_fy_report(fy, buys, buys_for_others, sales, fees, others, lots_by_ccy, balance_units, balance_value, output_dir, timestamp):
    output_csv = os.path.join(output_dir, f"fy{fy}_report.csv")
    with open(output_csv, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow(['Boughts for FY', fy])
        writer.writerow(['Date', 'Currency', 'Description', 'Trans Ref', 'Lot Ref', 'Qty Bought', 'Unit Cost (ZAR)', 'Total Cost (ZAR)', 'Proceeds (ZAR)', 'Profit (ZAR)', 'Fee (ZAR)'])
        for buy in buys:
            writer.writerow([buy['Date'], buy['Currency'], buy.get('Description', ''), buy['Trans Ref'], buy['Lot Ref'], buy['Qty Bought'], buy['Unit Cost'], buy['Total Cost'], buy['Proceeds'], buy['Profit'], buy.get('Fee (ZAR)', '0.00')])
        writer.writerow([])

        writer.writerow(['Solds for FY', fy])
        writer.writerow(['Date', 'Currency', 'Description', 'Trans Ref', 'Lot Ref', 'Qty Sold', 'Unit Cost (ZAR)', 'Total Cost (ZAR)', 'Proceeds (ZAR)', 'Profit (ZAR)', 'Fee (ZAR)'])
        for sale in sales:
            writer.writerow([sale['Date'], sale['Currency'], sale.get('Description', ''), sale['Trans Ref'], sale['Lot Ref'], sale['Qty Sold'], sale['Unit Cost'], sale['Total Cost'], sale['Proceeds'], sale['Profit'], sale.get('Fee (ZAR)', '0.00')])
        writer.writerow([])

        writer.writerow(['Buys for Others FY', fy])
        writer.writerow(['Date', 'Currency', 'Description', 'Trans Ref', 'Lot Ref', 'Qty Bought', 'Unit Cost (ZAR)', 'Total Cost (ZAR)', 'Proceeds (ZAR)', 'Profit (ZAR)', 'Fee (ZAR)'])
        for buy in buys_for_others:
            writer.writerow([buy['Date'], buy['Currency'], buy.get('Description', ''), buy['Trans Ref'], buy['Lot Ref'], buy['Qty Bought'], buy['Unit Cost'], buy['Total Cost'], buy['Proceeds'], buy['Profit'], buy.get('Fee (ZAR)', '0.00')])
        writer.writerow([])

        writer.writerow(['Others for FY', fy])
        writer.writerow(['Date', 'Currency', 'Description', 'Trans Ref', 'Lot Ref', 'Qty Other', 'Unit Cost (ZAR)', 'Total Cost (ZAR)', 'Proceeds (ZAR)', 'Profit (ZAR)', 'Fee (ZAR)'])
        for other in others:
            writer.writerow([other['Date'], other['Currency'], other.get('Description', ''), other['Trans Ref'], other['Lot Ref'], other['Qty Sold'], other['Unit Cost'], other['Total Cost'], other['Proceeds'], other['Profit'], other.get('Fee (ZAR)', '0.00')])
        writer.writerow([])
        writer.writerow(['Balances at end of FY', fy])
        writer.writerow(['Currency', 'Total Units', 'Total Value (ZAR)', 'Lot Ref', 'Lot Qty', 'Lot Unit Cost (ZAR)', 'Lot Total Value (ZAR)'])
        for ccy in sorted(lots_by_ccy.keys()):
            units = balance_units[ccy]
            total_value = sum((lot.qty * r2(lot.unit_cost) for lot in lots_by_ccy[ccy]), Decimal('0'))
            # Total row
            writer.writerow([ccy, q8(units), s2(total_value), '', '', '', ''])
            # Lot rows
            for lot in lots_by_ccy[ccy]:
                lot_value = lot.qty * r2(lot.unit_cost)
                writer.writerow([ccy, '', '', lot.ref, q8(lot.qty), s2(lot.unit_cost), s2(lot_value)])


        categories = {}
        for fee in fees:
            cat = fee['Category']
            if cat not in categories:
                categories[cat] = []
            categories[cat].append(fee)
        for cat in sorted(categories.keys()):
            if cat == 'Buying':
                writer.writerow(['Buying Fees'])
                writer.writerow(['Date', 'Description', 'Trans Ref', 'Lot Ref', 'Fee (ZAR)'])
                for fee in categories[cat]:
                    writer.writerow([fee['Date'], fee['Description'], fee['Trans Ref'], fee['Lot Ref'], fee['Fee (ZAR)']])
                total_fee = sum(Decimal(fee['Fee (ZAR)']) for fee in categories[cat])
                writer.writerow(['Total Buying Fees', '', '', '', s2(total_fee)])
            elif cat == 'Selling':
                writer.writerow(['Selling Fees'])
                writer.writerow(['Date', 'Description', 'Trans Ref', 'Lot Ref', 'Fee (ZAR)'])
                for fee in categories[cat]:
                    writer.writerow([fee['Date'], fee['Description'], fee['Trans Ref'], fee['Lot Ref'], fee['Fee (ZAR)']])
                total_fee = sum(Decimal(fee['Fee (ZAR)']) for fee in categories[cat])
                writer.writerow(['Total Selling Fees', '', '', '', s2(total_fee)])
            else:
                writer.writerow(['Other Fees'])
                writer.writerow(['Date', 'Description', 'Trans Ref', 'Lot Ref', 'Fee (ZAR)'])
                for fee in categories[cat]:
                    writer.writerow([fee['Date'], fee['Description'], fee['Trans Ref'], fee['Lot Ref'], fee['Fee (ZAR)']])
                total_fee = sum(Decimal(fee['Fee (ZAR)']) for fee in categories[cat])
                writer.writerow(['Total Other Fees', '', '', '', s2(total_fee)])
            writer.writerow([])

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

    if rows:
        ccy = rows[0]['Currency']
    else:
        ccy = 'UNK'

    lots_by_ccy = defaultdict(deque)
    balance_units = defaultdict(lambda: Decimal('0'))
    balance_value = defaultdict(lambda: Decimal('0'))

    # Transaction ID generators per currency
    buy_id_gen = gen_txn_ids(f'B_{ccy.upper()}_')
    sell_id_gen = gen_txn_ids(f'S_{ccy.upper()}_')

    output_rows = []
    last_trans_desc = ''
    last_trans_ref = ''

    for row in rows:
        ccy = row['Currency']
        dt = row['_dt']
        fy = financial_year(dt)
        qty_delta = row['Balance delta']
        desc = row['Description']
        is_send = 'send' in desc.lower() or 'sent' in desc.lower() or 'transfer' in desc.lower() or 'ant' in desc.lower()
        ref = row['Reference']
        value_amount = row['Value amount']

        if qty_delta == 0:
            continue

        if 'fee' in desc.lower():
            # Treat as fee, include in inventory change
            balance_units[ccy] += qty_delta
            balance_value[ccy] -= value_amount
            output_rows.append({
                'Financial Year': fy,
                'Trans Ref': last_trans_ref,
                'Date': row['Timestamp (UTC)'],
                'Description': f"Fee for {last_trans_desc}",
                'Type': 'Fee',
                'Lot Reference': '',
                'Qty Change': q8(qty_delta),
                'Unit Cost (ZAR)': '',
                'Total Cost (ZAR)': '',
                'Proceeds (ZAR)': '',
                'Profit (ZAR)': '',
                'Fee (ZAR)': s2(value_amount),
                'Balance Units': q8(balance_units[ccy]),
                'Balance Value (ZAR)': s2(balance_value[ccy]),
            })
            continue

        if qty_delta > 0 and desc.startswith('Bought'):
             # Buy lot
             qty = qty_delta
             unit_cost = value_amount / qty if qty != 0 else Decimal('0')
             lots_by_ccy[ccy].append(Lot(qty=qty, unit_cost=unit_cost, ref=ref))

             balance_units[ccy] += qty
             total_cost = qty * unit_cost
             balance_value[ccy] += total_cost

             trans_id = next(buy_id_gen)
             last_trans_ref = trans_id


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
        elif qty_delta > 0:
             # Other positive delta (not a buy)
             balance_units[ccy] += qty_delta
             balance_value[ccy] += value_amount

             output_rows.append({
                 'Financial Year': fy,
                 'Trans Ref': '',
                 'Date': row['Timestamp (UTC)'],
                 'Description': desc,
                 'Type': 'Other',
                 'Lot Reference': '',
                 'Qty Change': q8(qty_delta),
                 'Unit Cost (ZAR)': '',
                 'Total Cost (ZAR)': '',
                 'Proceeds (ZAR)': '',
                 'Profit (ZAR)': '',
                 'Fee (ZAR)': '',
                 'Balance Units': q8(balance_units[ccy]),
                 'Balance Value (ZAR)': s2(balance_value[ccy]),
             })
        else:
            # Sell or Send (any outflow)
            sell_qty = -qty_delta  # positive
            proceeds_total = value_amount
            trans_type = 'Sell' if desc.startswith('Sold') else 'Other'
            # If there are no lots, we will create a negative inventory entry (allowed here)
            # But better: consume from empty -> create lot with zero cost to allow proceeds
            if not lots_by_ccy[ccy]:
                lots_by_ccy[ccy].append(Lot(qty=Decimal('0'), unit_cost=Decimal('0'), ref='N/A'))

            trans_id = next(sell_id_gen)
            last_trans_ref = trans_id
            remaining = sell_qty

            # Pre-calc to allocate proceeds proportionally by quantity
            total_qty_for_sale = sell_qty

            while True:
                if remaining <= Decimal('0.0000000001') or not lots_by_ccy[ccy]:
                    break
                lot = lots_by_ccy[ccy][0]
                consume = lot.qty if lot.qty <= remaining else remaining
                if consume <= 0:
                    break

                unit_cost = lot.unit_cost
                total_cost = consume * unit_cost
                # Allocate proceeds proportionally by fraction of quantity sold in this split
                split_proceeds = proceeds_total * (consume / total_qty_for_sale)
                profit = split_proceeds - total_cost
                if trans_type == 'Other':
                    split_proceeds = Decimal('0')
                    profit = Decimal('0')

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
                'Type': trans_type,
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
            

            if remaining > 0:
                unit_cost = Decimal('0')
                total_cost = Decimal('0')
                split_proceeds = proceeds_total * (remaining / total_qty_for_sale)
                profit = split_proceeds - total_cost
                if trans_type == 'Other':
                    split_proceeds = Decimal('0')
                    profit = Decimal('0')
                balance_units[ccy] -= remaining
                # balance_value unchanged as zero cost
                output_rows.append({
                    'Financial Year': fy,
                    'Trans Ref': trans_id,
                    'Date': row['Timestamp (UTC)'],
                    'Description': desc,
                    'Type': trans_type,
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

            last_trans_desc = desc

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


def find_lot_by_ref(lots_deque, ref):
    for i, lot in enumerate(lots_deque):
        if lot.ref == ref:
            return i, lot
    return None, None

def process_fy(csv_files, output_dir, timestamp):
    buys_for_others_mapping = load_buys_for_others_mapping()
    
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

    buys_per_fy = defaultdict(list)
    buys_for_others_per_fy = defaultdict(list)
    sales_per_fy = defaultdict(list)
    fees_per_fy = defaultdict(list)
    others_per_fy = defaultdict(list)
    last_trans_per_ccy = defaultdict(str)
    last_trans_ref_per_ccy = defaultdict(str)

    buy_refs_for_others = set()
    for ccy, refs in buys_for_others_mapping.items():
        for ref in refs.keys():
            buy_refs_for_others.add((ccy, ref))

    current_fy = None

    buy_id_gens = {}
    sell_id_gens = {}

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
            balance_units[ccy] += qty_delta
            balance_value[ccy] -= value_amount
            category = 'Other'
            if 'Bought' in last_trans_per_ccy[ccy]:
                category = 'Buying'
            elif 'Sold' in last_trans_per_ccy[ccy]:
                category = 'Selling'
            fees_per_fy[fy].append({
                'Category': category,
                'Date': row['Timestamp (UTC)'],
                'Description': f"Fee for {last_trans_per_ccy[ccy]}",
                'Trans Ref': last_trans_ref_per_ccy[ccy],
                'Lot Ref': '',
                'Fee (ZAR)': s2(value_amount),
            })
            continue

        if current_fy is not None and fy != current_fy:
            generate_fy_report(current_fy, buys_per_fy[current_fy], buys_for_others_per_fy[current_fy], sales_per_fy[current_fy], fees_per_fy[current_fy], others_per_fy[current_fy], lots_by_ccy, balance_units, balance_value, output_dir, timestamp)

        current_fy = fy

        if qty_delta > 0 and desc.startswith('Bought'):
            qty = qty_delta
            unit_cost = value_amount / qty if qty != 0 else Decimal('0')
            lots_by_ccy[ccy].append(Lot(qty=qty, unit_cost=unit_cost, ref=ref))
            balance_units[ccy] += qty
            total_cost = qty * unit_cost
            balance_value[ccy] += total_cost
            if ccy not in buy_id_gens:
                buy_id_gens[ccy] = gen_txn_ids(f'B_{ccy.upper()}_')
            trans_id = next(buy_id_gens[ccy])
            last_trans_per_ccy[ccy] = desc
            last_trans_ref_per_ccy[ccy] = trans_id
            
            is_for_other = (ccy, ref) in buy_refs_for_others
            buy_record = {
                'Date': row['Timestamp (UTC)'],
                'Currency': ccy,
                'Description': desc,
                'Trans Ref': trans_id,
                'Lot Ref': ref,
                'Qty Bought': q8(qty),
                'Unit Cost': s2(unit_cost),
                'Total Cost': s2(total_cost),
                'Proceeds': s2(Decimal('0')),
                'Profit': s2(Decimal('0')),
                'Fee (ZAR)': s2(Decimal('0')),
            }
            if is_for_other:
                buys_for_others_per_fy[fy].append(buy_record)
            else:
                buys_per_fy[fy].append(buy_record)
                
        elif qty_delta > 0:
            balance_units[ccy] += qty_delta
            balance_value[ccy] += value_amount
            others_per_fy[fy].append({
                'Date': row['Timestamp (UTC)'],
                'Currency': ccy,
                'Description': desc,
                'Trans Ref': '',
                'Lot Ref': '',
                'Qty Sold': q8(qty_delta),
                'Unit Cost': '',
                'Total Cost': '',
                'Proceeds': '',
                'Profit': '',
                'Fee (ZAR)': '',
            })

        else:
            sell_qty = -qty_delta
            proceeds_total = value_amount
            trans_type = 'Sell' if desc.startswith('Sold') else 'Other'
            if not lots_by_ccy[ccy]:
                lots_by_ccy[ccy].append(Lot(qty=Decimal('0'), unit_cost=Decimal('0'), ref='N/A'))

            if ccy not in sell_id_gens:
                sell_id_gens[ccy] = gen_txn_ids(f'S_{ccy.upper()}_')
            trans_id = next(sell_id_gens[ccy])
            last_trans_ref_per_ccy[ccy] = trans_id
            remaining = sell_qty
            total_qty_for_sale = sell_qty
            
            matched_lot_ref = None
            if trans_type == 'Other' and ccy in buys_for_others_mapping:
                for buy_ref, match_info in buys_for_others_mapping[ccy].items():
                    if match_info['other_timestamp'] == row['Timestamp (UTC)']:
                        matched_lot_ref = buy_ref
                        break
            
            if matched_lot_ref:
                idx, lot = find_lot_by_ref(lots_by_ccy[ccy], matched_lot_ref)
                if lot is not None:
                    consume = lot.qty if lot.qty <= remaining else remaining
                    unit_cost = lot.unit_cost
                    total_cost = consume * unit_cost
                    split_proceeds = Decimal('0')
                    profit = Decimal('0')
                    
                    lot.qty -= consume
                    if lot.qty <= Decimal('0.0000000001'):
                        del lots_by_ccy[ccy][idx]
                    
                    balance_units[ccy] -= consume
                    balance_value[ccy] -= total_cost
                    
                    others_per_fy[fy].append({
                        'Date': row['Timestamp (UTC)'],
                        'Currency': ccy,
                        'Description': desc,
                        'Trans Ref': trans_id,
                        'Lot Ref': matched_lot_ref,
                        'Qty Sold': q8(-consume),
                        'Unit Cost': s2(unit_cost),
                        'Total Cost': s2(total_cost.copy_abs()),
                        'Proceeds': s2(split_proceeds),
                        'Profit': s2(profit),
                        'Fee (ZAR)': s2(Decimal('0')),
                    })
                    remaining -= consume
            
            while remaining > Decimal('0.0000000001') and lots_by_ccy[ccy]:
                lot = lots_by_ccy[ccy][0]
                consume = lot.qty if lot.qty <= remaining else remaining
                if consume <= 0:
                    break

                unit_cost = lot.unit_cost
                total_cost = consume * unit_cost
                split_proceeds = proceeds_total * (consume / total_qty_for_sale) if total_qty_for_sale > 0 else Decimal('0')
                profit = split_proceeds - total_cost
                if trans_type == 'Other':
                    split_proceeds = Decimal('0')
                    profit = Decimal('0')

                lot.qty -= consume
                if lot.qty <= Decimal('0.0000000001'):
                    lots_by_ccy[ccy].popleft()

                balance_units[ccy] -= consume
                balance_value[ccy] -= total_cost

                if trans_type == 'Sell':
                    sales_per_fy[fy].append({
                        'Date': row['Timestamp (UTC)'],
                        'Currency': ccy,
                        'Description': desc,
                        'Trans Ref': trans_id,
                        'Lot Ref': lot.ref,
                        'Qty Sold': q8(-consume),
                        'Unit Cost': s2(unit_cost),
                        'Total Cost': s2(total_cost.copy_abs()),
                        'Proceeds': s2(split_proceeds),
                        'Profit': s2(profit),
                        'Fee (ZAR)': s2(Decimal('0')),
                    })
                else:
                    others_per_fy[fy].append({
                        'Date': row['Timestamp (UTC)'],
                        'Currency': ccy,
                        'Description': desc,
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
                split_proceeds = proceeds_total * (remaining / total_qty_for_sale) if total_qty_for_sale > 0 else Decimal('0')
                profit = split_proceeds - total_cost
                if trans_type == 'Other':
                    split_proceeds = Decimal('0')
                    profit = Decimal('0')
                balance_units[ccy] -= remaining
                if trans_type == 'Sell':
                    sales_per_fy[fy].append({
                        'Date': row['Timestamp (UTC)'],
                        'Currency': ccy,
                        'Description': desc,
                        'Trans Ref': trans_id,
                        'Lot Ref': 'N/A',
                        'Qty Sold': q8(-remaining),
                        'Unit Cost': s2(unit_cost),
                        'Total Cost': s2(total_cost.copy_abs()),
                        'Proceeds': s2(split_proceeds),
                        'Profit': s2(profit),
                        'Fee (ZAR)': s2(Decimal('0')),
                    })
                else:
                    others_per_fy[fy].append({
                        'Date': row['Timestamp (UTC)'],
                        'Currency': ccy,
                        'Description': desc,
                        'Trans Ref': trans_id,
                        'Lot Ref': 'N/A',
                        'Qty Sold': q8(-remaining),
                        'Unit Cost': s2(unit_cost),
                        'Total Cost': s2(total_cost.copy_abs()),
                        'Proceeds': s2(split_proceeds),
                        'Profit': s2(profit),
                        'Fee (ZAR)': s2(Decimal('0')),
                    })

            last_trans_per_ccy[ccy] = desc

    if current_fy is not None:
        generate_fy_report(current_fy, buys_per_fy[current_fy], buys_for_others_per_fy[current_fy], sales_per_fy[current_fy], fees_per_fy[current_fy], others_per_fy[current_fy], lots_by_ccy, balance_units, balance_value, output_dir, timestamp)


if __name__ == '__main__':
    data_dir = '../data'
    csv_files = glob.glob(os.path.join(data_dir, '*.csv'))
    timestamp = datetime.now().strftime('%Y_%m_%d_%H%M')
    output_dir = os.path.join('../reports', timestamp)
    os.makedirs(output_dir, exist_ok=True)
    for csv_file in csv_files:
        base = os.path.basename(csv_file).rsplit('.', 1)[0]
        output_csv = os.path.join(output_dir, f"{base}_fifo.csv")
        main(csv_file, output_csv)
    process_fy(csv_files, output_dir, timestamp)
