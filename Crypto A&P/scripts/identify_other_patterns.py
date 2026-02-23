#!/usr/bin/env python3
import csv
import os
import glob
import json
from collections import defaultdict
from decimal import Decimal

def parse_dt(s):
    from datetime import datetime
    return datetime.strptime(s, '%Y-%m-%d %H:%M:%S')

def main():
    data_dir = '../data'
    csv_files = glob.glob(os.path.join(data_dir, '*.csv'))
    
    # Collect all "Other" transaction descriptions
    # Excluding: Bought, Sold, Fee transactions
    other_descriptions = defaultdict(list)
    
    for csv_file in csv_files:
        with open(csv_file, 'r') as f:
            reader = csv.DictReader(f)
            for row in reader:
                desc = row['Description'].strip()
                qty_delta = Decimal(row['Balance delta'])
                
                # Check if this is an "Other" transaction
                # Positive but not "Bought" = incoming Other
                # Negative but not "Sold" = outgoing Other
                is_other = False
                
                if qty_delta > 0:
                    # Incoming
                    if not desc.startswith('Bought') and 'Received' not in desc:
                        is_other = True
                elif qty_delta < 0:
                    # Outgoing
                    if not desc.startswith('Sold') and 'fee' not in desc.lower():
                        is_other = True
                
                if is_other:
                    ccy = row['Currency']
                    other_descriptions[ccy].append(desc)
    
    # Deduplicate descriptions per currency
    for ccy in other_descriptions:
        other_descriptions[ccy] = sorted(set(other_descriptions[ccy]))
    
    # Save to JSON
    output_file = os.path.join(data_dir, 'other_patterns.json')
    with open(output_file, 'w') as f:
        json.dump(other_descriptions, f, indent=2, sort_keys=True)
    
    print(f"Wrote {output_file}")
    for ccy, descs in sorted(other_descriptions.items()):
        print(f"\n{ccy}: {len(descs)} other patterns")
        for desc in descs:
            print(f"  - {desc}")

if __name__ == '__main__':
    main()
