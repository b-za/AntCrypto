#!/usr/bin/env python3
import subprocess
import sys
import os

def run_script(script_name):
    script_path = os.path.join(os.path.dirname(__file__), script_name)
    print(f"\n{'='*50}")
    print(f"Running {script_name}...")
    print('='*50)
    result = subprocess.run([sys.executable, script_path], cwd=os.path.dirname(__file__))
    if result.returncode != 0:
        print(f"Error: {script_name} failed with return code {result.returncode}")
        sys.exit(result.returncode)

def main():
    print("Crypto FIFO Tax Report Generator")
    print("================================")
    
    run_script('identify_buys_for_others.py')
    run_script('fifo_report.py')
    run_script('overview_report.py')
    
    print(f"\n{'='*50}")
    print("All reports generated successfully!")
    print('='*50)

if __name__ == '__main__':
    main()
