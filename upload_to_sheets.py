import gspread
from google.oauth2.service_account import Credentials
import json
import time
import logging

# Set up logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logging.info("Starting script...")

# Define the scope
scope = ["https://spreadsheets.google.com/feeds", "https://www.googleapis.com/auth/drive"]

# Add your service account credentials
creds = Credentials.from_service_account_file('concise-complex-344219-062a255ca56f.json', scopes=scope)

# Authorize the client
client = gspread.authorize(creds)

# Open the Google Sheet by name or ID
spreadsheet = client.open("Jenkins PR Tracker")  # or use client.open_by_key("YOUR_SHEET_ID")

# Load the grouped PRs JSON file
with open('grouped_prs_prs_gounthar_and_others_2024-12-01_to_2025-01-15.json') as f:
    grouped_prs = json.load(f)

# Iterate through each PR group and create a new sheet for each title
for pr in grouped_prs:
    title = pr["title"]
    prs = pr["prs"]

    # Prepare the data for the sheet
    data = [["Repository", "PR Number", "State", "Created At", "Updated At"]]
    for p in prs:
        # Add hyperlinks to the Repository and PR Number columns
        repo_link = f'=HYPERLINK("https://github.com/{p["repository"]}"; "{p["repository"]}")'
        pr_link = f'=HYPERLINK("https://github.com/{p["repository"]}/pull/{p["number"]}"; "{p["number"]}")'
        data.append([repo_link, pr_link, p["state"], p["createdAt"], p["updatedAt"]])

    try:
        # Check if a sheet with the same title already exists
        try:
            sheet = spreadsheet.worksheet(title)
            logging.info(f"Sheet '{title}' already exists. Updating it...")
        except gspread.exceptions.WorksheetNotFound:
            # Create a new sheet if it doesn't exist
            logging.info(f"Creating new sheet for '{title}'...")
            sheet = spreadsheet.add_worksheet(title=title, rows=100, cols=10)

        # Clear the sheet and update it with the new data
        sheet.clear()

        # Update the sheet with the data, treating hyperlinks as formulas
        sheet.update(range_name="A1", values=data, value_input_option="USER_ENTERED")

        # Apply conditional formatting based on PR state
        # Green for merged, orange for open, red for closed
        format_requests = []
        for row_idx, p in enumerate(prs, start=2):  # Start from row 2 (skip header)
            color = {
                "MERGED": {"red": 0.0, "green": 1.0, "blue": 0.0, "alpha": 1.0},  # Green
                "OPEN": {"red": 1.0, "green": 0.5, "blue": 0.0, "alpha": 1.0},    # Orange
                "CLOSED": {"red": 1.0, "green": 0.0, "blue": 0.0, "alpha": 1.0},  # Red
            }.get(p["state"], {"red": 1.0, "green": 1.0, "blue": 1.0, "alpha": 1.0})  # Default to white

            format_requests.append({
                "range": f"A{row_idx}:E{row_idx}",
                "format": {
                    "backgroundColor": color
                }
            })

        # Apply all formatting requests in a single batch
        if format_requests:
            sheet.batch_format(format_requests)

        # Add a delay to avoid hitting the quota limit
        time.sleep(5)  # Pause for 5 seconds between requests

    except gspread.exceptions.APIError as e:
        logging.error(f"Failed to update sheet '{title}': {e}")
        continue

logging.info("Data has been uploaded to Google Sheets.")
