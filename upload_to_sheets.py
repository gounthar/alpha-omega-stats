import gspread
from oauth2client.service_account import ServiceAccountCredentials
import json
import time
import logging
from tqdm import tqdm

# Set up logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logging.info("Starting script...")

# Define the scope
scope = ["https://spreadsheets.google.com/feeds", "https://www.googleapis.com/auth/drive"]

# Add your service account credentials
creds = ServiceAccountCredentials.from_json_keyfile_name('concise-complex-344219-062a255ca56f.json', scope)

# Authorize the client
client = gspread.authorize(creds)

# Open the Google Sheet by name or ID
# Replace "Your Google Sheet Name" with the actual name or ID of your Google Sheet
spreadsheet = client.open("Jenkins PR Tracker")  # or use client.open_by_key("YOUR_SHEET_ID")

# Load the grouped PRs JSON file
with open('grouped_prs_prs_gounthar_and_others_2024-12-01_to_2025-01-15.json') as f:
    grouped_prs = json.load(f)

# Iterate through each PR group and create a new sheet for each title
for pr in tqdm(grouped_prs, desc="Processing PRs"):
    title = pr["title"]
    prs = pr["prs"]

    # Prepare the data for the sheet
    data = [["Repository", "PR Number", "State", "Created At", "Updated At"]]
    for p in prs:
        data.append([p["repository"], p["number"], p["state"], p["createdAt"], p["updatedAt"]])

    try:
        # Check if a sheet with the same title already exists
        try:
            sheet = spreadsheet.worksheet(title)
            logging.info(f"Sheet '{title}' already exists. Updating it...")
        except gspread.exceptions.WorksheetNotFound:
            # Create a new sheet if it doesn't exist
            logging.info(f"Creating new sheet for '{title}'...")
            sheet = spreadsheet.add_worksheet(title=title, rows=100, cols=10)

        # Update the sheet with the new data
        sheet.clear()
        sheet.update(range_name='A1', values=data)  # Fix the deprecation warning

        # Add a delay to avoid hitting the quota limit
        time.sleep(2)  # Pause for 2 seconds between requests

    except gspread.exceptions.APIError as e:
        logging.error(f"Failed to update sheet '{title}': {e}")
        continue

logging.info("Data has been uploaded to Google Sheets.")
