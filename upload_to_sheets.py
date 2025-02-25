import gspread
import gspread.exceptions  # Ensure exceptions are properly referenced if not already imported
from google.oauth2.service_account import Credentials
import json
import time
import logging
from datetime import datetime

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
with open('grouped_prs_prs_gounthar_and_others_2024-12-01_to_2025-01-22.json') as f:
    grouped_prs = json.load(f)

# Load the failing PRs JSON file
try:
    with open('failing-prs.json') as f:
        failing_prs = json.load(f)
except FileNotFoundError:
    logging.error("failing-prs.json file not found.")
    failing_prs = None
except json.JSONDecodeError:
    logging.error("Error decoding failing-prs.json.")
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
    try:
        plugin_sheet = spreadsheet.worksheet(plugin)
    except gspread.exceptions.WorksheetNotFound:
        plugin_sheet = spreadsheet.add_worksheet(title=plugin, rows=100, cols=10)

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
summary_sheet.clear()

# Reorder sheets to make the Summary sheet first
sheets = spreadsheet.worksheets()
if sheets[0].title != "Summary":
    summary_sheet_index = next((i for i, sheet in enumerate(sheets) if sheet.title == "Summary"), None)
    if summary_sheet_index is not None:
        spreadsheet.reorder_worksheets(
            [sheets[summary_sheet_index]] + [sheet for i, sheet in enumerate(sheets) if i != summary_sheet_index])

# Get the Summary sheet ID for the "Back to Summary" link
summary_sheet_id = summary_sheet.id

# Create a new sheet for failing PRs
if failing_prs:
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
    if (
            isinstance(failing_prs, dict) and
            "data" in failing_prs and
            isinstance(failing_prs["data"], dict) and
            "search" in failing_prs["data"] and
            isinstance(failing_prs["data"]["search"], dict) and
            "nodes" in failing_prs["data"]["search"]
    ):

        # process each PR
        for pr in failing_prs["data"]["search"]["nodes"]:
            failing_prs_data.append([pr["title"], f'=HYPERLINK("{pr["url"]}"; "{pr["url"]}")',
                                 pr["commits"]["nodes"][0]["commit"]["statusCheckRollup"]["state"]])
    else:
        logging.error("Unexpected structure in failing_prs JSON data.")

    # Clear the sheet and update it with the new data
    failing_prs_sheet.clear()
    failing_prs_sheet.update(range_name="A1", values=failing_prs_data, value_input_option="USER_ENTERED")
    # Add a delay to avoid hitting the quota limit
    time.sleep(5)  # Pause for 5 seconds between requests

    # Format the column titles (bold font and background color)
    failing_prs_sheet.format("A3:C3", {  # Format only the column titles (row 3)
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

    failing_prs_count = 0
    if failing_prs and isinstance(failing_prs, dict) and "data" in failing_prs and \
        isinstance(failing_prs["data"], dict) and "search" in failing_prs["data"] and \
        isinstance(failing_prs["data"]["search"], dict) and "nodes" in failing_prs["data"]["search"]:
        failing_prs_count = len(failing_prs["data"]["search"]["nodes"])

# Add a link to the "Failing PRs" sheet in the "Summary" sheet and  include the count
summary_data.append(["Failing PRs", failing_prs_count, "", "", "", f'=HYPERLINK("#gid={failing_prs_sheet.id}"; "Failing PRs")'])
if failing_prs and 'failing_prs_sheet' in locals():
    summary_data.append(["Failing PRs", failing_prs_count, "", "", "", f'=HYPERLINK("#gid={failing_prs_sheet.id}"; "Failing PRs")'])
else:
    summary_data.append(["Failing PRs", failing_prs_count, "", "", "", "No failing PRs data available"])

summary_sheet.update(range_name="A1", values=summary_data, value_input_option="USER_ENTERED")

# Format the summary sheet
summary_sheet.format("A1:F1", {
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

# Iterate through each PR group and create a new sheet for each title
for pr in grouped_prs:
    title = pr["title"]
    prs = pr["prs"]

    # Prepare the data for the sheet
    data = [
        ["Back to Summary", f'=HYPERLINK("#gid={summary_sheet_id}"; "Back to Summary")', "", "", ""],
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
            sheet = spreadsheet.worksheet(title)
            logging.info(f"Sheet '{title}' already exists. Updating it...")
        except gspread.exceptions.WorksheetNotFound:
            # Create a new sheet if it doesn't exist
            logging.info(f"Creating new sheet for '{title}'...")
            sheet = spreadsheet.add_worksheet(title=title, rows=100, cols=10)

        # Clear the sheet and update it with the new data
        sheet.clear()
        sheet.update(range_name="A1", values=data, value_input_option="USER_ENTERED")

        # Format the column titles (bold font and background color)
        sheet.format("A3:E3", {  # Format only the column titles (row 3)
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

        # Apply conditional formatting based on PR state
        # Green for merged, orange for open, red for closed
        format_requests = []
        for row_idx, p in enumerate(prs, start=4):  # Start from row 4 (skip header and "Back to Summary" row)
            color = {
                "MERGED": {"red": 0.0, "green": 1.0, "blue": 0.0, "alpha": 1.0},  # Green
                "OPEN": {"red": 1.0, "green": 0.5, "blue": 0.0, "alpha": 1.0},  # Orange
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
