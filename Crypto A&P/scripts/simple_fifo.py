#!/usr/bin/env python3
import csv
from decimal import Decimal, getcontext, ROUND_HALF_UP
from datetime import datetime
from collections import deque

getcontext().prec = 28

class Lot:
    __slots__ = ('qty', 'unit_cost', 'ref', 'date')
    def __init__(self, qty: Decimal, unit_cost: Decimal, ref: str, date: str):
        self.qty = qty
        self.unit_cost = unit_cost
        self.ref = ref
        self.date = date

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

def financial_year(dt):
    return dt.year + 1 if dt.month >= 3 else dt.year

def main():
    input_file = '../data/xbt.csv'
    output_file = 'simple_fifo_xbt.csv'
    
    rows = []
    with open(input_file, 'r') as f:
        reader = csv.DictReader(f)
        for row in reader:
            row['Balance delta'] = dec(row['Balance delta'])
            row['Value amount'] = dec(row['Value amount'])
            row['_dt'] = datetime.strptime(row['Timestamp (UTC)'], '%Y-%m-%d %H:%M:%S')
            rows.append(row)
    
    rows.sort(key=lambda r: r['_dt'])
    
    lots = deque()
    
    output_rows = []
    trans_count = {'buy': 0, 'sell': 0, 'other': 0, 'fee': 0}
    
    for row in rows:
        dt = row['_dt']
        fy = financial_year(dt)
        qty_delta = row['Balance delta']
        desc = row['Description']
        ref = row['Reference']
        value_amount = row['Value amount']
        
        if qty_delta == 0:
            continue
        
        if 'fee' in desc.lower():
            trans_count['fee'] += 1
            fee_id = f'F_{trans_count["fee"]:03d}'
            output_rows.append({
                'FY': fy,
                'Trans Ref': fee_id,
                'Date': row['Timestamp (UTC)'],
                'Description': desc,
                'Type': 'Fee',
                'Lot Ref': '',
                'Qty Change': q8(qty_delta),
                'Unit Cost': '',
                'Total Cost': '',
                'Proceeds': '',
                'Profit': '',
            })
            continue
        
        if qty_delta > 0:
            trans_count['buy'] += 1
            trans_id = f'B_{trans_count["buy"]:03d}'
            qty = qty_delta
            unit_cost = value_amount / qty if qty != 0 else Decimal('0')
            total_cost = qty * unit_cost
            
            lot = Lot(qty=qty, unit_cost=unit_cost, ref=ref, date=row['Timestamp (UTC)'])
            lots.append(lot)
            
            output_rows.append({
                'FY': fy,
                'Trans Ref': trans_id,
                'Date': row['Timestamp (UTC)'],
                'Description': desc,
                'Type': 'Buy',
                'Lot Ref': ref,
                'Qty Change': q8(qty),
                'Unit Cost': s2(unit_cost),
                'Total Cost': s2(total_cost),
                'Proceeds': s2(Decimal('0')),
                'Profit': s2(Decimal('0')),
            })
        else:
            sell_qty = -qty_delta
            proceeds_total = value_amount
            
            if desc.startswith('Sold'):
                trans_count['sell'] += 1
                trans_id = f'S_{trans_count["sell"]:03d}'
                trans_type = 'Sell'
            else:
                trans_count['other'] += 1
                trans_id = f'O_{trans_count["other"]:03d}'
                trans_type = 'Other'
            
            remaining = sell_qty
            total_qty_for_sale = sell_qty
            
            while remaining > Decimal('0.0000000001') and lots:
                lot = lots[0]
                consume = lot.qty if lot.qty <= remaining else remaining
                
                unit_cost = lot.unit_cost
                total_cost = consume * unit_cost
                split_proceeds = proceeds_total * (consume / total_qty_for_sale) if total_qty_for_sale > 0 else Decimal('0')
                
                if trans_type == 'Sell':
                    profit = split_proceeds - total_cost
                else:
                    split_proceeds = Decimal('0')
                    profit = Decimal('0')
                
                lot.qty -= consume
                if lot.qty <= Decimal('0.0000000001'):
                    lots.popleft()
                
                output_rows.append({
                    'FY': fy,
                    'Trans Ref': trans_id,
                    'Date': row['Timestamp (UTC)'],
                    'Description': desc,
                    'Type': trans_type,
                    'Lot Ref': lot.ref,
                    'Qty Change': q8(-consume),
                    'Unit Cost': s2(unit_cost),
                    'Total Cost': s2(total_cost.copy_abs()),
                    'Proceeds': s2(split_proceeds),
                    'Profit': s2(profit),
                })
                
                remaining -= consume
            
            if remaining > Decimal('0.0000000001'):
                unit_cost = Decimal('0')
                total_cost = Decimal('0')
                split_proceeds = proceeds_total * (remaining / total_qty_for_sale) if total_qty_for_sale > 0 else Decimal('0')
                
                if trans_type == 'Sell':
                    profit = split_proceeds - total_cost
                else:
                    split_proceeds = Decimal('0')
                    profit = Decimal('0')
                
                output_rows.append({
                    'FY': fy,
                    'Trans Ref': trans_id,
                    'Date': row['Timestamp (UTC)'],
                    'Description': desc,
                    'Type': trans_type,
                    'Lot Ref': 'N/A',
                    'Qty Change': q8(-remaining),
                    'Unit Cost': s2(unit_cost),
                    'Total Cost': s2(total_cost.copy_abs()),
                    'Proceeds': s2(split_proceeds),
                    'Profit': s2(profit),
                })
    
    fieldnames = ['FY', 'Trans Ref', 'Date', 'Description', 'Type', 'Lot Ref', 'Qty Change', 'Unit Cost', 'Total Cost', 'Proceeds', 'Profit']
    with open(output_file, 'w', newline='') as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        for orow in output_rows:
            writer.writerow(orow)
    
    print(f"Wrote {output_file} with {len(output_rows)} rows")
    print(f"Remaining lots: {len(lots)}")

if __name__ == '__main__':
    main()
