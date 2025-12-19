#!/usr/bin/env python3
import csv
import os
import glob
import json
from decimal import Decimal
from datetime import datetime, timedelta
from collections import defaultdict

def parse_dt(s):
    return datetime.strptime(s, '%Y-%m-%d %H:%M:%S')

def dec(x):
    if isinstance(x, Decimal):
        return x
    s = str(x).strip()
    if s == '':
        return Decimal('0')
    return Decimal(s)

def main():
    data_dir = '../data'
    csv_files = glob.glob(os.path.join(data_dir, '*.csv'))
    
    rows_by_ccy = defaultdict(list)
    for csv_file in csv_files:
        with open(csv_file, newline='') as f:
            reader = csv.DictReader(f)
            for row in reader:
                row['Balance delta'] = dec(row['Balance delta'])
                row['Value amount'] = dec(row['Value amount'])
                row['_dt'] = parse_dt(row['Timestamp (UTC)'])
                ccy = row['Currency']
                rows_by_ccy[ccy].append(row)
    
    for ccy in rows_by_ccy:
        rows_by_ccy[ccy].sort(key=lambda r: r['_dt'])
    
    mapping = {}
    
    for ccy, rows in rows_by_ccy.items():
        buys = []
        others = []
        
        for row in rows:
            qty_delta = row['Balance delta']
            desc = row['Description']
            ref = row['Reference']
            
            if qty_delta == 0 or 'fee' in desc.lower():
                continue
            
            if qty_delta > 0 and desc.startswith('Bought'):
                buys.append({
                    'ref': ref,
                    'qty': qty_delta,
                    'dt': row['_dt'],
                    'timestamp': row['Timestamp (UTC)'],
                    'matched': False
                })
            elif qty_delta < 0 and not desc.startswith('Sold'):
                others.append({
                    'qty': abs(qty_delta),
                    'dt': row['_dt'],
                    'timestamp': row['Timestamp (UTC)'],
                    'desc': desc
                })
        
        ccy_mapping = {}
        
        for other in others:
            best_match = None
            best_time_diff = None
            
            for buy in buys:
                if buy['matched']:
                    continue
                
                time_diff = other['dt'] - buy['dt']
                if time_diff < timedelta(0):
                    continue
                if time_diff > timedelta(days=7):
                    continue
                
                qty_ratio = other['qty'] / buy['qty'] if buy['qty'] > 0 else Decimal('0')
                if qty_ratio < Decimal('0.90'):
                    continue
                
                if best_match is None or time_diff < best_time_diff:
                    best_match = buy
                    best_time_diff = time_diff
            
            if best_match:
                best_match['matched'] = True
                ccy_mapping[best_match['ref']] = {
                    'other_timestamp': other['timestamp'],
                    'other_desc': other['desc'],
                    'buy_qty': str(best_match['qty']),
                    'other_qty': str(other['qty'])
                }
        
        if ccy_mapping:
            mapping[ccy] = ccy_mapping
    
    output_file = os.path.join(data_dir, 'buys_for_others.json')
    with open(output_file, 'w') as f:
        json.dump(mapping, f, indent=2)
    
    print(f"Wrote {output_file}")
    for ccy in mapping:
        print(f"  {ccy}: {len(mapping[ccy])} buys matched to others")

if __name__ == '__main__':
    main()
