I have uploaded a CSV file containing my crypto trading history. Please run the fifo_report.py script to generate the detailed FIFO (First-In-First-Out) inventory tracking report.

The script will process the data according to the following logic:

*1. Data Preparation:*
•⁠  ⁠*Item:* Treat the 'Currency' (e.g., ETH) as the inventory item.
•⁠  ⁠*Sort:* Ensure data is sorted by ⁠ Timestamp (UTC) ⁠ oldest to newest.
•⁠  ⁠*Financial Year Logic:* Calculate a ⁠ Financial Year ⁠ column. The financial year runs from *1 March to End Feb*.
    * If Month >= 3 (March - Dec): Financial Year = Year + 1.
    * If Month < 3 (Jan - Feb): Financial Year = Year.
    * Example: 2021-01-06 is FY2021. 2021-03-01 is FY2022.

*2. Inventory Logic (FIFO):*
•⁠  ⁠*Buys (Inflows):* Rows where ⁠ Balance delta ⁠ is *positive*.
    * Cost Basis: ⁠ Value amount ⁠.
    * Unit Cost: ⁠ Value amount ⁠ / ⁠ Balance delta ⁠.
    * Lot ID: ⁠ Reference ⁠.
•⁠  ⁠*Sells (Outflows):* Rows where ⁠ Balance delta ⁠ is *negative*.
    * Quantity Sold: Absolute value of ⁠ Balance delta ⁠.
    * Proceeds: ⁠ Value amount ⁠.
    * FIFO Logic: Deduct the sold quantity from the oldest available Buy Lot(s).
    * Splits: If a sale consumes multiple lots, generate *separate output rows* for each lot consumed.

*3. Output Columns:*
The script generates a CSV with the following columns:
•⁠  ⁠⁠ Financial Year ⁠: The calculated financial year (e.g., 2021, 2022).
•⁠  ⁠⁠ Trans Ref ⁠: Unique 3-letter ID for each transaction (e.g., BAA, SAA). Splits share the same ID.
•⁠  ⁠⁠ Date ⁠: ⁠ Timestamp (UTC) ⁠.
•⁠  ⁠⁠ Description ⁠: Original description.
•⁠  ⁠⁠ Type ⁠: 'Buy' or 'Sell'.
•⁠  ⁠⁠ Lot Reference ⁠: The ⁠ Reference ⁠ of the specific Buy lot being touched.
•⁠  ⁠⁠ Qty Change ⁠: The quantity moved in this specific row (Positive for Buy, Negative for Sell).
•⁠  ⁠⁠ Unit Cost (ZAR) ⁠: The specific unit cost of the Lot.
•⁠  ⁠⁠ Total Cost (ZAR) ⁠: ⁠ abs(Qty Change) * Unit Cost ⁠.
•⁠  ⁠⁠ Proceeds (ZAR) ⁠: For sells, the portion of ⁠ Value amount ⁠ allocated to this row.
•⁠  ⁠⁠ Profit (ZAR) ⁠: ⁠ Proceeds - Total Cost ⁠.
•⁠  ⁠⁠ Balance Units ⁠: Running total of units held.
•⁠  ⁠⁠ Balance Value (ZAR) ⁠: Running total of cost basis.