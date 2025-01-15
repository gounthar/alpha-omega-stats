import gspread
from oauth2client.service_account import ServiceAccountCredentials
import json

# Define the scope
scope = ["https://spreadsheets.google.com/feeds", "https://www.googleapis.com/auth/drive"]

# Add your service account credentials
creds = ServiceAccountCredentials.from_json_keyfile_name('concise-complex-344219-062a255ca56f.json', scope)

# Authorize the client
client = gspread.authorize(creds)

# Open the Google Sheet
sheet = client.open("Your Google Sheet Name").sheet1

# Load the grouped PRs JSON file
with open('grouped_prs_prs_gounthar_and_others_2025-01-01_to_2025-01-15.json') as f:
    grouped_prs = json.load(f)

# Prepare the data for the sheet
data = [["Title", "Merged", "Open", "Closed", "PRs"]]
for pr in grouped_prs:
    data.append([pr["title"], pr["merged"], pr["open"], pr["closed"], "\n".join([p["repository"] + " #" + str(p["number"]) for p in pr["prs"]])])

# Update the sheet with the new data
sheet.clear()
sheet.update('A1', data)

print("Data has been uploaded to Google Sheets.")
