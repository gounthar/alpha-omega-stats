import gspread
from google.oauth2.service_account import Credentials
import sys
import csv
import os

# Usage: python3 export_sheet_to_tsv.py <spreadsheet_name_or_id> <worksheet_name> <output_tsv>
if len(sys.argv) != 4:
    print("Usage: python3 export_sheet_to_tsv.py <spreadsheet_name_or_id> <worksheet_name> <output_tsv>")
    sys.exit(1)

SPREADSHEET = sys.argv[1]
WORKSHEET = sys.argv[2]
OUTPUT_TSV = sys.argv[3]

# Credentials
scope = ["https://spreadsheets.google.com/feeds", "https://www.googleapis.com/auth/drive"]
credentials_file = os.environ.get('GOOGLE_APPLICATION_CREDENTIALS', 'concise-complex-344219-062a255ca56f.json')
try:
    creds = Credentials.from_service_account_file(credentials_file, scopes=scope)
except FileNotFoundError:
    # Fallback used in CI or when a different path is set up
    fallback_file = 'google-credentials.json'
    creds = Credentials.from_service_account_file(fallback_file, scopes=scope)
client = gspread.authorize(creds)

# Open spreadsheet and worksheet
try:
    # Prefer opening by key; fall back to opening by title if that fails
    try:
        spreadsheet = client.open_by_key(SPREADSHEET)
    except Exception:
        spreadsheet = client.open(SPREADSHEET)
    sheet = spreadsheet.worksheet(WORKSHEET)
except Exception as e:
    print(f"Error opening sheet: {e}")
    sys.exit(1)

# Get all values
rows = sheet.get_all_values()

# Write to TSV
with open(OUTPUT_TSV, 'w', newline='') as f:
    writer = csv.writer(f, delimiter='\t')
    writer.writerows(rows)

print(f"Exported {len(rows)} rows to {OUTPUT_TSV}")
