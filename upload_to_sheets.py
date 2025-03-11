import gspread
import gspread.exceptions  # Ensure exceptions are properly referenced if not already imported
from google.oauth2.service_account import Credentials
import json
import time
import logging
from datetime import datetime
import sys
import re
from time import sleep

# Set up logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logging.info("Starting script...")

def retry_with_backoff(func, max_retries=5, initial_delay=5):
    """
    Retry a function with exponential backoff.
    """
    delay = initial_delay
    for attempt in range(max_retries):
        try:
            return func()
        except gspread.exceptions.APIError as e:
            if "429" in str(e):  # Rate limit error
                if attempt == max_retries - 1:
                    raise
                logging.warning(f"Rate limit hit. Waiting {delay} seconds before retry {attempt + 1}/{max_retries}")
                sleep(delay)
                delay *= 2  # Exponential backoff
            else:
                raise
    return None

def sanitize_sheet_name(title, max_length=100):
    """
    Sanitize a title to be used as a Google Sheets worksheet name.
    - Removes invalid characters
    - Truncates to max_length
    - Ensures uniqueness by adding a counter if needed
    """
    # Remove invalid characters and replace spaces with underscores
    sanitized = re.sub(r'[\[\]\\*?/:]', '', title)
    sanitized = sanitized.replace(' ', '_')
    
    # Truncate to max_length
    if len(sanitized) > max_length:
        # Keep the first part of the title and add a hash of the full title
        hash_suffix = str(hash(title))[-8:]
        sanitized = sanitized[:max_length-9] + '_' + hash_suffix
    
    return sanitized

def update_sheet_with_retry(sheet, data, range_name="A1"):
    """
    Update a sheet with retry logic and rate limiting.
    """
    def update():
        # First clear the entire sheet
        sheet.clear()
        time.sleep(1)  # Add a small delay after clearing
        # Then update with new data
        sheet.update(range_name=range_name, values=data, value_input_option="USER_ENTERED")
        time.sleep(2)  # Add delay between operations
    
    retry_with_backoff(update)

def format_sheet_with_retry(sheet, range_name, format_dict):
    """
    Format a sheet with retry logic and rate limiting.
    """
    def format():
        sheet.format(range_name, format_dict)
        time.sleep(2)  # Add delay between operations
    
    retry_with_backoff(format)

def batch_format_with_retry(sheet, format_requests):
    """
    Batch format a sheet with retry logic and rate limiting.
    """
    def batch_format():
        sheet.batch_format(format_requests)
        time.sleep(2)  # Add delay between operations
    
    retry_with_backoff(batch_format)

# Check if input file is provided
if len(sys.argv) != 3:
    print("Usage: python3 upload_to_sheets.py <grouped-prs-json-file> <failing-prs-error-state>")
    sys.exit(1)

GROUPED_PRS_FILE = sys.argv[1]
FAILING_PRS_ERROR = sys.argv[2].lower() == 'true'

# Define the scope
scope = ["https://spreadsheets.google.com/feeds", "https://www.googleapis.com/auth/drive"]

# Add your service account credentials
creds = Credentials.from_service_account_file('concise-complex-344219-062a255ca56f.json', scopes=scope)

# Authorize the client
client = gspread.authorize(creds)

# Open the Google Sheet by name or ID
spreadsheet = client.open("Jenkins PR Tracker")  # or use client.open_by_key("YOUR_SHEET_ID")

# Load the grouped PRs JSON file
try:
    with open(GROUPED_PRS_FILE) as f:
        grouped_prs = json.load(f)
    logging.info(f"Successfully loaded grouped PRs from {GROUPED_PRS_FILE}")
except FileNotFoundError:
    logging.error(f"File {GROUPED_PRS_FILE} not found.")
    sys.exit(1)
except json.JSONDecodeError:
    logging.error(f"Error decoding {GROUPED_PRS_FILE}.")
    sys.exit(1)

# Load the failing PRs JSON file
if not FAILING_PRS_ERROR:
    try:
        with open('all_results.json') as f:
            failing_prs = json.load(f)
        logging.info("Successfully loaded failing PRs data")
    except FileNotFoundError:
        logging.warning("all_results.json file not found. Skipping failing PRs sheet.")
        failing_prs = None
    except json.JSONDecodeError:
        logging.warning("Error decoding all_results.json. Skipping failing PRs sheet.")
        failing_prs = None
else:
    logging.warning("Failing PRs collection failed. Skipping failing PRs sheet.")
    failing_prs = None

# Create a summary sheet
try:
    summary_sheet = spreadsheet.worksheet("Summary")
    logging.info("Summary sheet already exists. Updating it...")
except gspread.exceptions.WorksheetNotFound:
    logging.info("Creating new Summary sheet...")
    summary_sheet = spreadsheet.add_worksheet(title="Summary", rows=100, cols=10)

# Prepare summary data
total_prs = 0
open_prs = 0
closed_prs = 0
merged_prs = 0
plugin_stats = {}
earliest_date = None
latest_date = None

for pr in grouped_prs:
    title = pr["title"]
    prs = pr["prs"]
    total_prs += len(prs)
    open_prs += pr["open"]
    closed_prs += pr["closed"]
    merged_prs += pr["merged"]

    # Plugin-specific stats
    plugin_stats[title] = {
        "total": len(prs),
        "open": pr["open"],
        "closed": pr["closed"],
        "merged": pr["merged"]
    }

    # Find the earliest and latest dates
    for p in prs:
        created_at = datetime.fromisoformat(p["createdAt"].replace("Z", "+00:00"))
        updated_at = datetime.fromisoformat(p["updatedAt"].replace("Z", "+00:00"))

        if earliest_date is None or created_at < earliest_date:
            earliest_date = created_at
        if latest_date is None or updated_at > latest_date:
            latest_date = updated_at

# Calculate percentages
open_percentage = (open_prs / total_prs) * 100 if total_prs > 0 else 0
closed_percentage = (closed_prs / total_prs) * 100 if total_prs > 0 else 0
merged_percentage = (merged_prs / total_prs) * 100 if total_prs > 0 else 0

# Prepare summary data for the sheet
summary_data = [
    ["PR Date Range", f"{earliest_date.strftime('%Y-%m-%d')} to {latest_date.strftime('%Y-%m-%d')}", "", "", "", ""],
    ["Overall PR Statistics", "", "", "", "", ""],
    ["Total PRs", total_prs, "", "", "", ""],
    ["Open PRs", open_prs, f"{open_percentage:.2f}%", "", "", ""],
    ["Closed PRs", closed_prs, f"{closed_percentage:.2f}%", "", "", ""],
    ["Merged PRs", merged_prs, f"{merged_percentage:.2f}%", "", "", ""],
    ["", "", "", "", "", ""],
    ["Plugin-Specific Statistics", "", "", "", "", ""],
    ["Plugin", "Total PRs", "Open PRs", "Closed PRs", "Merged PRs", "Link to Sheet"]
]

# Add plugin-specific stats and links to individual sheets
for plugin, stats in plugin_stats.items():
    sheet_name = sanitize_sheet_name(plugin)
    try:
        plugin_sheet = spreadsheet.worksheet(sheet_name)
    except gspread.exceptions.WorksheetNotFound:
        plugin_sheet = spreadsheet.add_worksheet(title=sheet_name, rows=100, cols=10)

    link = f'=HYPERLINK("#gid={plugin_sheet.id}"; "{plugin}")'
    summary_data.append([
        plugin,
        stats["total"],
        stats["open"],
        stats["closed"],
        stats["merged"],
        link
    ])

# Update the summary sheet
update_sheet_with_retry(summary_sheet, summary_data)

# Reorder sheets to make the Summary sheet first
sheets = spreadsheet.worksheets()
if sheets[0].title != "Summary":
    summary_sheet_index = next((i for i, sheet in enumerate(sheets) if sheet.title == "Summary"), None)
    if summary_sheet_index is not None:
        spreadsheet.reorder_worksheets(
            [sheets[summary_sheet_index]] + [sheet for i, sheet in enumerate(sheets) if i != summary_sheet_index])

# Get the Summary sheet ID for the "Back to Summary" link
summary_sheet_id = summary_sheet.id

# Format the summary sheet
format_sheet_with_retry(summary_sheet, "A1:F1", {
    "textFormat": {
        "bold": True
    },
    "backgroundColor": {
        "red": 0.9,  # Light gray background
        "green": 0.9,
        "blue": 0.9,
        "alpha": 1.0
    },
    "horizontalAlignment": "CENTER"  # Center-align the text
})

# Create a new sheet for failing PRs
if failing_prs and isinstance(failing_prs, list):
    try:
        failing_prs_sheet = spreadsheet.worksheet("Failing PRs")
        logging.info("Failing PRs sheet already exists. Updating it...")
    except gspread.exceptions.WorksheetNotFound:
        logging.info("Creating new Failing PRs sheet...")
        failing_prs_sheet = spreadsheet.add_worksheet(title="Failing PRs", rows=100, cols=10)

    # Prepare the data for the failing PRs sheet
    failing_prs_data = [
        ["Back to Summary", f'=HYPERLINK("#gid={summary_sheet_id}"; "Back to Summary")', "", "", ""],
        ["", "", "", "", ""],  # Empty row for spacing
        ["Title", "URL", "Status"]
    ]

    # Process each failing PR
    for pr in failing_prs:
        failing_prs_data.append([
            pr["title"],
            f'=HYPERLINK("{pr["url"]}"; "{pr["url"]}")',
            pr["status"]
        ])

    # Clear the sheet and update it with the new data
    update_sheet_with_retry(failing_prs_sheet, failing_prs_data)

    # Format the column titles (bold font and background color)
    format_sheet_with_retry(failing_prs_sheet, "A3:C3", {  # Format only the column titles (row 3)
        "textFormat": {
            "bold": True
        },
        "backgroundColor": {
            "red": 0.9,  # Light gray background
            "green": 0.9,
            "blue": 0.9,
            "alpha": 1.0
        },
        "horizontalAlignment": "CENTER"  # Center-align the text
    })

    # Calculate failing PRs count
    failing_prs_count = len(failing_prs)
else:
    failing_prs_count = 0
    logging.warning("No failing PRs found in the data.")

# Add a link to the "Failing PRs" sheet in the "Summary" sheet and include the count
if failing_prs and 'failing_prs_sheet' in locals():
    summary_data.append(["Failing PRs", failing_prs_count, "", "", "", f'=HYPERLINK("#gid={failing_prs_sheet.id}"; "Failing PRs")'])
else:
    summary_data.append(["Failing PRs", failing_prs_count, "", "", "", "No failing PRs found"])
update_sheet_with_retry(summary_sheet, summary_data)

# Iterate through each PR group and create a new sheet for each title
for pr in grouped_prs:
    title = pr["title"]
    prs = pr["prs"]
    sheet_name = sanitize_sheet_name(title)

    # Prepare the data for the sheet
    data = [
        ["Back to Summary", f'=HYPERLINK("#gid={summary_sheet_id}"; "Back to Summary")', "", "", ""],
        [title, "", "", "", ""],  # Add title without label
        ["", "", "", "", ""],  # Empty row for spacing
        ["Repository", "PR Number", "State", "Created At", "Updated At"]
    ]
    for p in prs:
        # Add hyperlinks to the Repository and PR Number columns
        repo_link = f'=HYPERLINK("https://github.com/{p["repository"]}"; "{p["repository"]}")'
        pr_link = f'=HYPERLINK("https://github.com/{p["repository"]}/pull/{p["number"]}"; "{p["number"]}")'
        data.append([repo_link, pr_link, p["state"], p["createdAt"], p["updatedAt"]])

    try:
        # Check if a sheet with the same title already exists
        try:
            sheet = spreadsheet.worksheet(sheet_name)
            logging.info(f"Sheet '{sheet_name}' already exists. Updating it...")
        except gspread.exceptions.WorksheetNotFound:
            # Create a new sheet if it doesn't exist
            logging.info(f"Creating new sheet for '{sheet_name}'...")
            sheet = spreadsheet.add_worksheet(title=sheet_name, rows=100, cols=10)

        # Update sheet with retry logic
        update_sheet_with_retry(sheet, data)

        # Format the "Back to Summary" row - only format the first cell
        format_sheet_with_retry(sheet, "A1", {
            "textFormat": {
                "fontSize": 10
            }
        })

        # Format the title row - only format the first cell
        format_sheet_with_retry(sheet, "A2", {
            "textFormat": {
                "bold": True,
                "fontSize": 10
            },
            "horizontalAlignment": "LEFT"
        })

        # Format the column titles
        format_sheet_with_retry(sheet, "A4:E4", {
            "textFormat": {
                "bold": True
            },
            "backgroundColor": {
                "red": 0.9,
                "green": 0.9,
                "blue": 0.9,
                "alpha": 1.0
            },
            "horizontalAlignment": "CENTER"
        })

        # Apply conditional formatting based on PR state
        format_requests = []
        for row_idx, p in enumerate(prs, start=5):  # Start from row 5 (skip header, title, and "Back to Summary" rows)
            color = {
                "MERGED": {"red": 0.0, "green": 1.0, "blue": 0.0, "alpha": 1.0},
                "OPEN": {"red": 1.0, "green": 0.5, "blue": 0.0, "alpha": 1.0},
                "CLOSED": {"red": 1.0, "green": 0.0, "blue": 0.0, "alpha": 1.0},
            }.get(p["state"], {"red": 1.0, "green": 1.0, "blue": 1.0, "alpha": 1.0})

            format_requests.append({
                "range": f"A{row_idx}:E{row_idx}",
                "format": {
                    "backgroundColor": color
                }
            })

        # Apply all formatting requests in a single batch
        if format_requests:
            batch_format_with_retry(sheet, format_requests)

        # Add a longer delay between sheets to avoid rate limits
        time.sleep(10)  # Increased delay between sheets

    except gspread.exceptions.APIError as e:
        logging.error(f"Failed to update sheet '{sheet_name}': {e}")
        continue

logging.info("Data has been uploaded to Google Sheets.")
