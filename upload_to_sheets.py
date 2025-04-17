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
import random
import os

# Set up logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logging.info("Starting script...")

def get_backoff_duration(attempt, base_delay=5, max_delay=300):
    """
    Calculate exponential backoff duration with jitter.
    
    Args:
        attempt: The current retry attempt number (0-based)
        base_delay: The base delay in seconds (default: 5)
        max_delay: Maximum delay in seconds (default: 300)
    
    Returns:
        Delay duration in seconds
    """
    delay = min(base_delay * (2 ** attempt), max_delay)
    jitter = random.uniform(0, 0.1 * delay)  # Add up to 10% jitter
    return delay + jitter

def handle_google_api_error(e, attempt, max_retries):
    """
    Handle different types of Google Sheets API errors.
    
    Returns:
        tuple: (should_retry, wait_time)
    """
    error_str = str(e).lower()
    
    # Rate limit errors
    if "429" in error_str or "quota" in error_str:
        wait_time = get_backoff_duration(attempt)
        logging.warning(f"Rate limit exceeded. Attempt {attempt + 1}/{max_retries}. Waiting {wait_time:.1f} seconds...")
        return True, wait_time
    
    # Backend errors (500s)
    if any(code in error_str for code in ["500", "502", "503", "504"]):
        wait_time = get_backoff_duration(attempt, base_delay=10)
        logging.warning(f"Backend error encountered. Attempt {attempt + 1}/{max_retries}. Waiting {wait_time:.1f} seconds...")
        return True, wait_time
    
    # Authorization errors
    if "401" in error_str or "403" in error_str:
        logging.error("Authorization error. Please check your credentials.")
        return False, 0
    
    # Invalid request errors
    if "400" in error_str:
        logging.error("Invalid request error. Please check your input data.")
        return False, 0
    
    # Default case - retry with standard backoff
    wait_time = get_backoff_duration(attempt)
    logging.warning(f"Unexpected error. Attempt {attempt + 1}/{max_retries}. Waiting {wait_time:.1f} seconds...")
    return True, wait_time

def retry_with_backoff(func, max_retries=5, initial_delay=5):
    """
    Retry a function with exponential backoff and improved error handling.
    """
    last_error = None
    
    for attempt in range(max_retries):
        try:
            return func()
        except gspread.exceptions.APIError as e:
            should_retry, wait_time = handle_google_api_error(e, attempt, max_retries)
            if not should_retry or attempt == max_retries - 1:
                raise
            time.sleep(wait_time)
            last_error = e
        except gspread.exceptions.SpreadsheetNotFound:
            logging.error("Spreadsheet not found. Please check the spreadsheet ID or name.")
            raise
        except gspread.exceptions.WorksheetNotFound:
            logging.error("Worksheet not found. Please check the worksheet name.")
            raise
        except Exception as e:
            logging.error(f"Unexpected error: {str(e)}")
            raise
    
    if last_error:
        raise last_error
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
    Update a sheet with enhanced retry logic and rate limiting.
    """
    def update():
        try:
            # First clear the entire sheet
            sheet.clear()
            time.sleep(1)  # Add a small delay after clearing
            
            # Validate data before updating
            if not data or not isinstance(data, (list, tuple)):
                raise ValueError("Invalid data format. Expected non-empty list or tuple.")
            
            # Then update with new data
            sheet.update(range_name=range_name, values=data, value_input_option="USER_ENTERED")
            logging.info(f"Successfully updated sheet with {len(data)} rows of data")
            time.sleep(2)  # Add delay between operations
        except gspread.exceptions.APIError as e:
            logging.error(f"API error during sheet update: {str(e)}")
            raise
        except Exception as e:
            logging.error(f"Error during sheet update: {str(e)}")
            raise
    
    return retry_with_backoff(update)

def format_sheet_with_retry(sheet, range_name, format_dict):
    """
    Format a sheet with enhanced retry logic and rate limiting.
    """
    def format():
        try:
            sheet.format(range_name, format_dict)
            logging.info(f"Successfully formatted range {range_name}")
            time.sleep(2)  # Add delay between operations
        except Exception as e:
            logging.error(f"Error during sheet formatting: {str(e)}")
            raise
    
    return retry_with_backoff(format)

def batch_format_with_retry(sheet, format_requests):
    """
    Batch format a sheet with enhanced retry logic and rate limiting.
    """
    def batch_format():
        try:
            sheet.batch_format(format_requests)
            logging.info(f"Successfully applied {len(format_requests)} format requests")
            time.sleep(2)  # Add delay between operations
        except Exception as e:
            logging.error(f"Error during batch formatting: {str(e)}")
            raise
    
    return retry_with_backoff(batch_format)

def validate_pr_data(pr):
    """
    Validate PR data structure and required fields.
    Returns (is_valid, error_message)
    """
    required_fields = {
        "title": str,
        "repository": str,
        "number": (int, str),  # Allow both int and str
        "state": str,
        "createdAt": str,
        "updatedAt": str,
        "checkStatus": (str, type(None))  # Allow string or None
    }
    
    for field, field_type in required_fields.items():
        if field not in pr:
            return False, f"Missing required field: {field}"
        if not isinstance(pr[field], field_type):
            if isinstance(field_type, tuple):
                if not any(isinstance(pr[field], t) for t in field_type):
                    return False, f"Invalid type for {field}: expected {field_type}, got {type(pr[field])}"
            else:
                return False, f"Invalid type for {field}: expected {field_type}, got {type(pr[field])}"
    
    # Validate state values
    if pr["state"] not in ["OPEN", "CLOSED", "MERGED"]:
        return False, f"Invalid state value: {pr['state']}"
    
    # Validate date formats
    try:
        datetime.fromisoformat(pr["createdAt"].replace("Z", "+00:00"))
        datetime.fromisoformat(pr["updatedAt"].replace("Z", "+00:00"))
    except ValueError as e:
        return False, f"Invalid date format: {e}"
    
    return True, None

def group_prs_by_title(prs):
    """
    Group PRs by title and calculate statistics.
    """
    title_groups = {}
    for pr in prs:
        title = pr["title"]
        if title not in title_groups:
            title_groups[title] = {
                "title": title,
                "prs": [],
                "open": 0,
                "closed": 0,
                "merged": 0
            }
        
        group = title_groups[title]
        group["prs"].append(pr)
        if pr["state"] == "OPEN":
            group["open"] += 1
        elif pr["state"] == "CLOSED":
            group["closed"] += 1
        elif pr["state"] == "MERGED":
            group["merged"] += 1
    
    return list(title_groups.values())

def process_consolidated_data(consolidated_file):
    """
    Process consolidated PR data and validate it.
    Returns (grouped_prs, failing_prs, errors)
    """
    errors = []
    valid_prs = []
    
    try:
        with open(consolidated_file) as f:
            prs = json.load(f)
    except FileNotFoundError:
        return None, None, [f"File {consolidated_file} not found."]
    except json.JSONDecodeError:
        return None, None, [f"Error decoding {consolidated_file}."]
    
    if not isinstance(prs, list):
        return None, None, ["Consolidated data must be a list of PRs."]
    
    # Validate each PR
    for i, pr in enumerate(prs):
        is_valid, error = validate_pr_data(pr)
        if is_valid:
            valid_prs.append(pr)
        else:
            errors.append(f"PR at index {i}: {error}")
    
    if not valid_prs:
        return None, None, ["No valid PRs found in consolidated data."]
    
    # Group valid PRs by title
    grouped_prs = group_prs_by_title(valid_prs)
    
    # Extract failing PRs
    failing_prs = [
        {
            "title": pr["title"],
            "url": f"https://github.com/{pr['repository']}/pull/{pr['number']}",
            "status": pr["checkStatus"]
        }
        for pr in valid_prs
        if pr["state"] == "OPEN" and pr["checkStatus"] in ["ERROR", "FAILURE"]
    ]
    
    return grouped_prs, failing_prs, errors

# Function to process successful builds data
def process_successful_builds_data(csv_file):
    """
    Process successful builds CSV data.
    Returns a list of records or None if file not found.
    """
    if not os.path.exists(csv_file):
        logging.warning(f"Successful builds file {csv_file} not found.")
        return None

    try:
        successful_builds = []
        with open(csv_file, 'r') as f:
            # Skip header line
            header = f.readline().strip()
            if not header.startswith("PR_URL,JDK_VERSION"):
                logging.error(f"Invalid header in {csv_file}. Expected 'PR_URL,JDK_VERSION'")
                return None

            # Process each line
            for line in f:
                line = line.strip()
                if not line:
                    continue

                parts = line.split(',')
                if len(parts) >= 2:
                    pr_url = parts[0]
                    jdk_version = parts[1]

                    # Extract PR number and repository from URL
                    match = re.search(r'https://github.com/([^/]+/[^/]+)/pull/(\d+)', pr_url)
                    if match:
                        repo = match.group(1)
                        pr_number = match.group(2)

                        successful_builds.append({
                            "pr_url": pr_url,
                            "jdk_version": jdk_version,
                            "repository": repo,
                            "pr_number": pr_number
                        })
                    else:
                        logging.warning(f"Could not parse PR URL: {pr_url}")

        return successful_builds
    except Exception as e:
        logging.error(f"Error processing successful builds file: {str(e)}")
        return None

# Function to process test results data
def process_test_results_data(csv_file):
    """
    Process the test_results.csv file to extract PR URLs, JDK versions, and test results.
    Returns a dictionary with two lists: 'passed' and 'failed', each containing dictionaries with PR URL, JDK version, repository, and PR number.
    """
    if not os.path.exists(csv_file):
        logging.info(f"Test results file {csv_file} not found.")
        return None

    test_results = {
        'passed': [],
        'failed': []
    }

    try:
        with open(csv_file, 'r') as f:
            # Check header
            header = f.readline().strip()
            if not header.startswith("PR_URL,JDK_VERSION,TEST_RESULT"):
                logging.error(f"Invalid header in {csv_file}. Expected 'PR_URL,JDK_VERSION,TEST_RESULT'")
                return None

            # Process each line
            for line in f:
                line = line.strip()
                if not line:
                    continue

                parts = line.split(',')
                if len(parts) >= 3:
                    pr_url = parts[0]
                    jdk_version = parts[1]
                    test_result = parts[2]

                    # Extract PR number and repository from URL
                    match = re.search(r'https://github.com/([^/]+/[^/]+)/pull/(\d+)', pr_url)
                    if match:
                        repo = match.group(1)
                        pr_number = match.group(2)

                        pr_data = {
                            "pr_url": pr_url,
                            "jdk_version": jdk_version,
                            "repository": repo,
                            "pr_number": pr_number
                        }

                        if test_result == "TESTS_PASSED":
                            test_results['passed'].append(pr_data)
                        elif test_result == "TESTS_FAILED":
                            test_results['failed'].append(pr_data)
                        else:
                            logging.warning(f"Unknown test result: {test_result} for PR: {pr_url}")
                    else:
                        logging.warning(f"Could not parse PR URL: {pr_url}")

        return test_results
    except Exception as e:
        logging.error(f"Error processing test results file: {str(e)}")
        return None

# Main execution
if len(sys.argv) < 3 or len(sys.argv) > 4:
    print("Usage: python3 upload_to_sheets.py <consolidated-prs-json-file> <failing-prs-error-state> [force-update]")
    print("  force-update: Optional parameter. Set to 'true' to force update even if no changes detected.")
    sys.exit(1)

CONSOLIDATED_FILE = sys.argv[1]
FAILING_PRS_ERROR = sys.argv[2].lower() == 'true'
FORCE_UPDATE = len(sys.argv) > 3 and sys.argv[3].lower() == 'true'
SUCCESSFUL_BUILDS_FILE = os.path.join(os.path.dirname(CONSOLIDATED_FILE), "successful_builds.csv")
TEST_RESULTS_FILE = os.path.join(os.path.dirname(CONSOLIDATED_FILE), "test_results.csv")

# Process consolidated data
grouped_prs, failing_prs, errors = process_consolidated_data(CONSOLIDATED_FILE)

if errors:
    for error in errors:
        logging.error(error)
    if not grouped_prs:  # Fatal errors
        sys.exit(1)
    else:  # Non-fatal errors
        logging.warning("Some PRs were invalid but processing will continue.")

# Define the scope
scope = ["https://spreadsheets.google.com/feeds", "https://www.googleapis.com/auth/drive"]

# Add your service account credentials
# Check for environment variable first, then fall back to local file
credentials_file = os.environ.get('GOOGLE_APPLICATION_CREDENTIALS', 'concise-complex-344219-062a255ca56f.json')
try:
    creds = Credentials.from_service_account_file(credentials_file, scopes=scope)
    logging.info(f"Using credentials from: {credentials_file}")
except FileNotFoundError:
    # For GitHub Actions, try the file we created in the workflow
    fallback_file = 'google-credentials.json'
    logging.info(f"Credentials file {credentials_file} not found, trying {fallback_file}")
    creds = Credentials.from_service_account_file(fallback_file, scopes=scope)

# Authorize the client
client = gspread.authorize(creds)

# Open the Google Sheet by name or ID
spreadsheet = client.open("Jenkins PR Tracker")  # or use client.open_by_key("YOUR_SHEET_ID")

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
has_successful_local_builds = False  # Flag to track if we have successful local builds
has_tests_passed = False  # Flag to track if we have PRs with passing tests
has_tests_failed = False  # Flag to track if we have PRs with failing tests

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
    ["Merged PRs", merged_prs, f"{merged_percentage:.2f}%", "", "", ""]
]

# Function to safely get worksheet ID with retry
def get_worksheet_id_with_retry(spreadsheet, sheet_name, max_retries=3):
    """
    Safely get a worksheet ID with retry logic for rate limiting.
    """
    for attempt in range(max_retries):
        try:
            worksheet = spreadsheet.worksheet(sheet_name)
            return worksheet.id
        except gspread.exceptions.APIError as e:
            if "429" in str(e) and attempt < max_retries - 1:
                wait_time = get_backoff_duration(attempt, base_delay=20)
                logging.warning(f"Rate limit hit when getting worksheet ID. Waiting {wait_time:.1f} seconds...")
                time.sleep(wait_time)
            elif "404" in str(e) or "RESOURCE_NOT_FOUND" in str(e):
                logging.warning(f"Worksheet '{sheet_name}' not found")
                return None
            else:
                logging.error(f"Error getting worksheet ID: {str(e)}")
                return None
        except Exception as e:
            logging.error(f"Unexpected error getting worksheet ID: {str(e)}")
            return None

    return None

# Add successful local builds section if we have any
if has_successful_local_builds:
    logging.info("Attempting to add Local Build Success section to summary data")
    successful_builds_sheet_id = get_worksheet_id_with_retry(spreadsheet, "Local Build Success")

    if successful_builds_sheet_id:
        logging.info("Adding Local Build Success section to summary data")
        summary_data.extend([
            ["", "", "", "", "", ""],
            ["Local Build Success", "", "", "", "", ""],
            ["PRs that build locally but fail on Jenkins", successful_builds_count, "", "", "", ""],
            ["View Details", f'=HYPERLINK("#gid={successful_builds_sheet_id}"; "Go to Local Build Success Sheet")', "", "", "", ""]
        ])
    else:
        logging.warning("Could not add Local Build Success section because sheet ID is not available")
else:
    logging.info("Not adding Local Build Success section because has_successful_local_builds is False")

# Add tests passed section if we have any
if has_tests_passed:
    logging.info("Attempting to add Tests Passed section to summary data")
    # Wait a bit to avoid rate limiting
    time.sleep(2)
    tests_passed_sheet_id = get_worksheet_id_with_retry(spreadsheet, "Local Build Tests Pass")

    if tests_passed_sheet_id:
        logging.info("Adding Tests Passed section to summary data")
        summary_data.extend([
            ["", "", "", "", "", ""],
            ["Local Build Tests Pass", "", "", "", "", ""],
            ["PRs that build locally with passing tests", tests_passed_count, "", "", "", ""],
            ["View Details", f'=HYPERLINK("#gid={tests_passed_sheet_id}"; "Go to Tests Pass Sheet")', "", "", "", ""]
        ])
    else:
        logging.warning("Could not add Tests Passed section because sheet ID is not available")
else:
    logging.info("Not adding Tests Passed section because has_tests_passed is False")

# Add tests failed section if we have any
if has_tests_failed:
    logging.info("Attempting to add Tests Failed section to summary data")
    # Wait a bit to avoid rate limiting
    time.sleep(2)
    tests_failed_sheet_id = get_worksheet_id_with_retry(spreadsheet, "Local Build Tests Fail")

    if tests_failed_sheet_id:
        logging.info("Adding Tests Failed section to summary data")
        summary_data.extend([
            ["", "", "", "", "", ""],
            ["Local Build Tests Fail", "", "", "", "", ""],
            ["PRs that build locally but tests fail", tests_failed_count, "", "", "", ""],
            ["View Details", f'=HYPERLINK("#gid={tests_failed_sheet_id}"; "Go to Tests Fail Sheet")', "", "", "", ""]
        ])
    else:
        logging.warning("Could not add Tests Failed section because sheet ID is not available")
else:
    logging.info("Not adding Tests Failed section because has_tests_failed is False")

# Add plugin-specific statistics section
summary_data.extend([
    ["", "", "", "", "", ""],
    ["Plugin-Specific Statistics", "", "", "", "", ""],
    ["Plugin", "Total PRs", "Open PRs", "Closed PRs", "Merged PRs", "Link to Sheet"]
])

# Add plugin-specific stats and links to individual sheets
for plugin, stats in plugin_stats.items():
    sheet_name = sanitize_sheet_name(plugin)
    if not sheet_name:
        logging.error(f"Invalid sheet name generated for plugin '{plugin}'. Skipping sheet creation.")
        continue

    try:
        plugin_sheet = spreadsheet.worksheet(sheet_name)
    except gspread.exceptions.WorksheetNotFound:
        logging.info(f"Creating new sheet for '{sheet_name}'...")
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

# Get the Summary sheet ID for the "Back to Summary" link
summary_sheet_id = summary_sheet.id

# Process successful builds data
successful_builds = process_successful_builds_data(SUCCESSFUL_BUILDS_FILE)

# Update the flag if we have successful builds
if successful_builds and isinstance(successful_builds, list) and len(successful_builds) > 0:
    has_successful_local_builds = True
    successful_builds_count = len(successful_builds)
    logging.info(f"Found {successful_builds_count} successful local builds")
else:
    logging.info(f"No successful builds found or invalid format: {successful_builds}")

# Process test results data
logging.info(f"Looking for test results file at: {TEST_RESULTS_FILE}")
test_results = process_test_results_data(TEST_RESULTS_FILE)

# Set flags for test results
has_tests_passed = False
has_tests_failed = False
tests_passed_count = 0
tests_failed_count = 0

if test_results and isinstance(test_results, dict):
    if 'passed' in test_results and len(test_results['passed']) > 0:
        has_tests_passed = True
        tests_passed_count = len(test_results['passed'])
        logging.info(f"Found {tests_passed_count} PRs with passing tests")
    else:
        logging.info("No PRs with passing tests found in test results")

    if 'failed' in test_results and len(test_results['failed']) > 0:
        has_tests_failed = True
        tests_failed_count = len(test_results['failed'])
        logging.info(f"Found {tests_failed_count} PRs with failing tests")
    else:
        logging.info("No PRs with failing tests found in test results")
else:
    logging.info(f"No test results found or invalid format: {test_results}")

# Force the flags to true for testing purposes
if FORCE_UPDATE:
    logging.info("Force update is enabled, setting test result flags to true for testing")
    has_tests_passed = True
    has_tests_failed = True
    tests_passed_count = 0
    tests_failed_count = 0

# Debug logs for the flags
logging.info(f"has_successful_local_builds flag is set to: {has_successful_local_builds}")
logging.info(f"has_tests_passed flag is set to: {has_tests_passed}")
logging.info(f"has_tests_failed flag is set to: {has_tests_failed}")

# Check if successful builds file has changed since last run
def has_successful_builds_changed():
    """
    Check if the successful_builds.csv file has changed since the last run.
    Returns True if the file has changed or if it's the first run.
    """
    last_run_file = os.path.join(os.path.dirname(SUCCESSFUL_BUILDS_FILE), "last_successful_builds.csv")

    # If the successful builds file doesn't exist, return False (no changes)
    if not os.path.exists(SUCCESSFUL_BUILDS_FILE):
        logging.info("Successful builds file doesn't exist.")
        return False

    # If the last run file doesn't exist, this is the first run
    if not os.path.exists(last_run_file):
        logging.info("First run for successful builds tracking.")
        # Create a copy for future comparison
        try:
            import shutil
            shutil.copy2(SUCCESSFUL_BUILDS_FILE, last_run_file)
            logging.info(f"Created initial copy of successful builds file at {last_run_file}")
        except Exception as e:
            logging.error(f"Error creating copy of successful builds file: {str(e)}")
        return True

    # Compare the current file with the last run file
    try:
        with open(SUCCESSFUL_BUILDS_FILE, 'r') as f1, open(last_run_file, 'r') as f2:
            current_content = f1.read()
            last_content = f2.read()

            if current_content != last_content:
                logging.info("Successful builds file has changed since last run.")
                # Update the last run file
                with open(last_run_file, 'w') as f:
                    f.write(current_content)
                return True
            else:
                logging.info("Successful builds file has not changed since last run.")
                return False
    except Exception as e:
        logging.error(f"Error comparing successful builds files: {str(e)}")
        return True  # Default to updating if there's an error

# Check if successful builds have changed
successful_builds_changed = has_successful_builds_changed()

# Create and update the successful builds sheet
if (successful_builds and isinstance(successful_builds, list) and len(successful_builds) > 0 and
    (successful_builds_changed or FORCE_UPDATE)):
    logging.info(f"Found {len(successful_builds)} successful builds to add to the sheet")
    logging.info(f"Updating sheet because: changed={successful_builds_changed}, force={FORCE_UPDATE}")
    try:
        try:
            successful_builds_sheet = spreadsheet.worksheet("Local Build Success")
            logging.info("Local Build Success sheet already exists. Updating it...")
        except gspread.exceptions.WorksheetNotFound:
            logging.info("Creating new Local Build Success sheet...")
            successful_builds_sheet = spreadsheet.add_worksheet(title="Local Build Success", rows=100, cols=10)

        # Prepare the data for the successful builds sheet
        successful_builds_data = [
            ["Back to Summary", f'=HYPERLINK("#gid={summary_sheet_id}"; "Back to Summary")', "", "", ""],
            ["", "", "", "", ""],  # Empty row for spacing
            ["Repository", "PR Number", "PR URL", "JDK Version", "Status"]
        ]

        # Process each successful build
        for build in successful_builds:
            successful_builds_data.append([
                build["repository"],
                build["pr_number"],
                f'=HYPERLINK("{build["pr_url"]}"; "PR #{build["pr_number"]}")',
                build["jdk_version"],
                "Builds locally"
            ])

        # Update the successful builds sheet
        update_sheet_with_retry(successful_builds_sheet, successful_builds_data)

        # Format the header row
        format_sheet_with_retry(successful_builds_sheet, "A3:E3", {
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

        # Add a note to the sheet explaining its purpose
        successful_builds_sheet.update_cell(1, 3, "PRs that build successfully locally but fail on Jenkins")
        format_sheet_with_retry(successful_builds_sheet, "C1", {
            "textFormat": {
                "italic": True
            }
        })

        # We no longer need to add a link here as we've already added it to the summary data
        logging.info("Successfully created/updated Local Build Success sheet")
    except Exception as e:
        logging.error(f"Error creating/updating Local Build Success sheet: {str(e)}")
elif successful_builds and isinstance(successful_builds, list) and len(successful_builds) > 0:
    logging.info(f"Found {len(successful_builds)} successful builds but skipping update because no changes detected and force update not enabled")
else:
    logging.info("No successful builds found or successful_builds.csv file not available")

# Function to safely get or create worksheet with retry
def get_or_create_worksheet_with_retry(spreadsheet, sheet_name, rows=100, cols=10, max_retries=3):
    """
    Safely get or create a worksheet with retry logic for rate limiting.
    """
    for attempt in range(max_retries):
        try:
            try:
                worksheet = spreadsheet.worksheet(sheet_name)
                logging.info(f"{sheet_name} sheet already exists. Updating it...")
                return worksheet
            except gspread.exceptions.WorksheetNotFound:
                logging.info(f"Creating new {sheet_name} sheet...")
                # Add a delay before creating to avoid rate limits
                time.sleep(2)
                return spreadsheet.add_worksheet(title=sheet_name, rows=rows, cols=cols)
        except gspread.exceptions.APIError as e:
            if "429" in str(e) and attempt < max_retries - 1:
                wait_time = get_backoff_duration(attempt, base_delay=20)
                logging.warning(f"Rate limit hit when creating worksheet. Waiting {wait_time:.1f} seconds...")
                time.sleep(wait_time)
            else:
                logging.error(f"API error creating worksheet: {str(e)}")
                raise
        except Exception as e:
            logging.error(f"Unexpected error creating worksheet: {str(e)}")
            raise

    raise Exception(f"Failed to get or create worksheet {sheet_name} after {max_retries} attempts")

# Create and update the tests passed sheet
if has_tests_passed and FORCE_UPDATE:
    passed_count = len(test_results['passed']) if test_results and isinstance(test_results, dict) and 'passed' in test_results else 0
    logging.info(f"Found {passed_count} PRs with passing tests to add to the sheet")
    logging.info(f"Updating tests passed sheet because: force={FORCE_UPDATE}")
    try:
        # Wait a bit to avoid rate limiting
        time.sleep(5)
        tests_passed_sheet = get_or_create_worksheet_with_retry(spreadsheet, "Local Build Tests Pass")

        # Prepare the data for the tests passed sheet
        tests_passed_data = [
            ["Back to Summary", f'=HYPERLINK("#gid={summary_sheet_id}"; "Back to Summary")', "", "", ""],
            ["", "", "", "", ""],  # Empty row for spacing
            ["Repository", "PR Number", "PR URL", "JDK Version", "Status"]
        ]

        # Process each PR with passing tests
        if test_results and isinstance(test_results, dict) and 'passed' in test_results:
            for pr in test_results['passed']:
                tests_passed_data.append([
                    pr["repository"],
                    pr["pr_number"],
                    f'=HYPERLINK("{pr["pr_url"]}"; "PR #{pr["pr_number"]}")',
                    pr["jdk_version"],
                    "Tests Passed"
                ])
        else:
            # Add a placeholder row if no data
            tests_passed_data.append([
                "No data available",
                "",
                "",
                "",
                ""
            ])

        # Update the tests passed sheet
        update_sheet_with_retry(tests_passed_sheet, tests_passed_data)

        # Format the header row
        format_sheet_with_retry(tests_passed_sheet, "A3:E3", {
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

        # Add a note to the sheet explaining its purpose
        tests_passed_sheet.update_cell(1, 3, "PRs that build successfully locally and pass tests")
        format_sheet_with_retry(tests_passed_sheet, "C1", {
            "textFormat": {
                "italic": True
            }
        })

        logging.info("Successfully created/updated Local Build Tests Pass sheet")
    except Exception as e:
        logging.error(f"Error creating/updating Local Build Tests Pass sheet: {str(e)}")
elif test_results and isinstance(test_results, dict) and 'passed' in test_results and len(test_results['passed']) > 0:
    logging.info(f"Found {len(test_results['passed'])} PRs with passing tests but skipping update because force update not enabled")
else:
    logging.info("No PRs with passing tests found or test_results.csv file not available")

# Create and update the tests failed sheet
if has_tests_failed and FORCE_UPDATE:
    failed_count = len(test_results['failed']) if test_results and isinstance(test_results, dict) and 'failed' in test_results else 0
    logging.info(f"Found {failed_count} PRs with failing tests to add to the sheet")
    logging.info(f"Updating tests failed sheet because: force={FORCE_UPDATE}")
    try:
        # Wait a bit to avoid rate limiting
        time.sleep(5)
        tests_failed_sheet = get_or_create_worksheet_with_retry(spreadsheet, "Local Build Tests Fail")

        # Prepare the data for the tests failed sheet
        tests_failed_data = [
            ["Back to Summary", f'=HYPERLINK("#gid={summary_sheet_id}"; "Back to Summary")', "", "", ""],
            ["", "", "", "", ""],  # Empty row for spacing
            ["Repository", "PR Number", "PR URL", "JDK Version", "Status"]
        ]

        # Process each PR with failing tests
        if test_results and isinstance(test_results, dict) and 'failed' in test_results:
            for pr in test_results['failed']:
                tests_failed_data.append([
                    pr["repository"],
                    pr["pr_number"],
                    f'=HYPERLINK("{pr["pr_url"]}"; "PR #{pr["pr_number"]}")',
                    pr["jdk_version"],
                    "Tests Failed"
                ])
        else:
            # Add a placeholder row if no data
            tests_failed_data.append([
                "No data available",
                "",
                "",
                "",
                ""
            ])

        # Update the tests failed sheet
        update_sheet_with_retry(tests_failed_sheet, tests_failed_data)

        # Format the header row
        format_sheet_with_retry(tests_failed_sheet, "A3:E3", {
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

        # Add a note to the sheet explaining its purpose
        tests_failed_sheet.update_cell(1, 3, "PRs that build successfully locally but fail tests")
        format_sheet_with_retry(tests_failed_sheet, "C1", {
            "textFormat": {
                "italic": True
            }
        })

        logging.info("Successfully created/updated Local Build Tests Fail sheet")
    except Exception as e:
        logging.error(f"Error creating/updating Local Build Tests Fail sheet: {str(e)}")
elif test_results and isinstance(test_results, dict) and 'failed' in test_results and len(test_results['failed']) > 0:
    logging.info(f"Found {len(test_results['failed'])} PRs with failing tests but skipping update because force update not enabled")
else:
    logging.info("No PRs with failing tests found or test_results.csv file not available")

# Log the summary data for debugging
logging.info(f"Summary data has {len(summary_data)} rows")
logging.info(f"First few rows: {summary_data[:5]}")
if len(summary_data) > 10:
    logging.info(f"Rows 10-15: {summary_data[10:15]}")

# Update the summary sheet
update_sheet_with_retry(summary_sheet, summary_data)

# Reorder sheets to make the Summary sheet first
sheets = spreadsheet.worksheets()
if sheets[0].title != "Summary":
    summary_sheet_index = next((i for i, sheet in enumerate(sheets) if sheet.title == "Summary"), None)
    if summary_sheet_index is not None:
        spreadsheet.reorder_worksheets(
            [sheets[summary_sheet_index]] + [sheet for i, sheet in enumerate(sheets) if i != summary_sheet_index])

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

# Process successful builds data
successful_builds = process_successful_builds_data(SUCCESSFUL_BUILDS_FILE)

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
