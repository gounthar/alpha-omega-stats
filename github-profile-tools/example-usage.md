# GitHub Profile Analyzer - Example Usage

This document shows real examples of using the GitHub Profile Analyzer.

## 🎯 Quick Test Run

### 1. Basic Analysis
```bash
# Analyze a well-known GitHub user
./github-user-analyzer -user octocat -template resume -verbose

# Expected output:
# 🎉 Analysis Complete for @octocat
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 👤 Name: The Octocat
# 🏢 Company: GitHub
# 📍 Location: San Francisco
#
# 📊 Profile Statistics:
#    • Career Level: Senior
#    • Experience: 8 years
#    • Repositories: 42 total
#    • Community Impact: 15234 stars received
```

### 2. Different Templates
```bash
# Technical deep-dive
./github-user-analyzer -user octocat -template technical

# Executive summary
./github-user-analyzer -user octocat -template executive

# ATS-optimized
./github-user-analyzer -user octocat -template ats
```

### 3. Output Formats
```bash
# JSON only (for further processing)
./github-user-analyzer -user octocat -format json

# Markdown only (direct use in documents)
./github-user-analyzer -user octocat -format markdown

# Both formats (default)
./github-user-analyzer -user octocat -format both
```

## 📝 Sample Output Files

### Resume Template Output (`octocat_profile_resume.md`)
```markdown
# GitHub Professional Profile - octocat

**Name:** The Octocat
**Location:** San Francisco
**Company:** GitHub

*🐙 🐱 Awesome octopus that lives in your computer*

## 📊 Contribution Overview

- **1,337** total contributions across **8** years of active development
- **42** repositories with **15,234** stars received
- Active contributor in **5** organizations
- Proficient in **12** programming languages
- Career Level: **Senior**

## 🏢 Organization Contributions

### GitHub
*How people build software.*

- **Role:** Member
- **Contributions:** 25 repositories
- **Key Projects:** Hello-World, Spoon-Knife, octocat.github.io

## 💼 Notable Projects

### [Hello-World](https://github.com/octocat/Hello-World) ⭐ 1,420
**Description:** My first repository on GitHub!

- **Language:** JavaScript | **Size:** 2.1 MB
- **Technologies:** node, npm, javascript, git
- **Contributions:** 87 commits (+2,341/-892 lines)

### [Spoon-Knife](https://github.com/octocat/Spoon-Knife) ⭐ 12,345
**Description:** This repo is for demonstration purposes only.

- **Language:** HTML | **Size:** 0.5 MB
- **Technologies:** html, css, git
- **Contributions:** 23 commits (+456/-78 lines)

## 🛠 Technical Skills

### Programming Languages
- **JavaScript:** Advanced (35.2% of codebase, 18 projects)
- **HTML:** Advanced (22.1% of codebase, 15 projects)
- **CSS:** Intermediate (18.7% of codebase, 12 projects)
- **Ruby:** Intermediate (12.4% of codebase, 8 projects)
- **Python:** Beginner (6.8% of codebase, 4 projects)

### Technology Stack
- **Frameworks:** React, Vue, Express, Rails
- **Databases:** PostgreSQL, MongoDB, Redis
- **Cloud Platforms:** AWS, GitHub Actions
- **DevOps & Tools:** Docker, Kubernetes, CI/CD

## 🤝 Professional Insights

- **Open Source Contributions:** 42 repositories
- **Cross-Organization Work:** Contributed to 5 different organizations
- **Leadership Experience:** Demonstrates ability to create and maintain projects; Shows collaborative leadership across multiple teams
- **Overall Impact Score:** 8.7/10

## 📈 Activity Timeline

- **Most Active Period:** 2023
- **Consistency Score:** 7.8/10
- **Recent Activity:** Active in 8 repositories in the last 30 days
- **Recommended Roles:** Senior Full-Stack Engineer, Technical Lead, Frontend Team Lead

---
*Profile generated on October 5, 2025 | GitHub: [@octocat](https://github.com/octocat)*
```

### ATS Template Output (`octocat_profile_ats.md`)
```markdown
GITHUB PROFESSIONAL PROFILE - OCTOCAT

Name: The Octocat
Location: San Francisco
Current Company: GitHub

TECHNICAL SKILLS

Programming Languages: JavaScript, HTML, CSS, Ruby, Python, Go, Java, TypeScript

Frameworks and Libraries: React, Vue, Express, Rails, Django, Spring Boot

Databases: PostgreSQL, MongoDB, Redis, MySQL

Cloud Platforms: AWS, GitHub Actions, Docker, Kubernetes

PROFESSIONAL EXPERIENCE

Software Developer | 8 Years Active Development
Career Level: Senior

Key Achievements:
- Developed and maintained 42 software repositories
- Contributed 1337 commits across 12 programming languages
- Received 15234 community stars for open source contributions
- Collaborated across 5 professional organizations
- Demonstrated high-impact technical leadership and project ownership

ORGANIZATIONAL EXPERIENCE

GitHub - Member
Contributed to 25 projects

NOTABLE PROJECTS

Hello-World
Description: My first repository on GitHub!
Technology: JavaScript
Community Recognition: 1420 stars

Spoon-Knife
Description: This repo is for demonstration purposes only.
Technology: HTML
Community Recognition: 12345 stars

TECHNICAL CERTIFICATIONS AND EXPERTISE

JavaScript Development - Advanced Level
Experience: 18 projects, 8.0 years

HTML Development - Advanced Level
Experience: 15 projects, 7.5 years

CSS Development - Intermediate Level
Experience: 12 projects, 6.2 years
```

## 🔧 Testing the Build

### 1. Build and Test
```bash
cd github-profile-tools

# Install dependencies
go mod download

# Build the application
go build -o github-user-analyzer ./cmd/github-user-analyzer

# Test with a public user
export GITHUB_TOKEN="your_token_here"
./github-user-analyzer -user octocat -verbose
```

### 2. Expected Directory Structure After Run
```
github-profile-tools/
├── data/
│   └── profiles/
│       ├── octocat_profile.json
│       ├── octocat_profile_resume.md
│       ├── octocat_profile_technical.md
│       ├── octocat_profile_executive.md
│       └── octocat_profile_ats.md
├── github-user-analyzer              # Built binary
└── ... (source files)
```

### 3. Verify JSON Output
```bash
# Check the generated JSON has all expected fields
cat data/profiles/octocat_profile.json | jq '.insights.careerLevel'
# Expected: "senior" or similar

cat data/profiles/octocat_profile.json | jq '.languages | length'
# Expected: Number > 0

cat data/profiles/octocat_profile.json | jq '.repositories | length'
# Expected: Number > 0
```

## 🎯 Use Cases

### 1. Resume Enhancement
```bash
# Generate resume section
./github-user-analyzer -user YOUR_USERNAME -template resume -format markdown

# Copy relevant sections to your resume:
# - Technical Skills section
# - Notable Projects section
# - Professional Insights section
```

### 2. Job Application Optimization
```bash
# For technical interviews
./github-user-analyzer -user YOUR_USERNAME -template technical

# For ATS systems at large companies
./github-user-analyzer -user YOUR_USERNAME -template ats

# For leadership/management roles
./github-user-analyzer -user YOUR_USERNAME -template executive
```

### 3. Profile Comparison
```bash
# Compare yourself to industry leaders
./github-user-analyzer -user industry_leader -template technical
./github-user-analyzer -user YOUR_USERNAME -template technical

# Compare impact scores, languages, project counts
```

### 4. Team Analysis
```bash
# Analyze team members for project assignments
for member in dev1 dev2 dev3; do
    ./github-user-analyzer -user "$member" -template technical -output "./team-analysis/"
done
```

## 🐛 Troubleshooting Test Issues

### Common Test Problems

1. **"User not found"**
   ```bash
   # Try with a different public user
   ./github-user-analyzer -user torvalds
   ./github-user-analyzer -user gaearon
   ./github-user-analyzer -user sindresorhus
   ```

2. **"Rate limit exceeded"**
   ```bash
   # Wait and retry, or check your token
   echo $GITHUB_TOKEN | cut -c1-10  # Should show token prefix
   ```

3. **"Build errors"**
   ```bash
   # Ensure Go version is correct
   go version  # Should be 1.24.0+

   # Clean and rebuild
   go clean -cache
   go mod tidy
   go build -o github-user-analyzer ./cmd/github-user-analyzer
   ```

### Performance Expectations

- **Small profiles** (< 50 repos): 10-30 seconds
- **Medium profiles** (50-200 repos): 30-90 seconds
- **Large profiles** (200+ repos): 1-3 minutes

The tool includes rate limiting and retry logic to handle GitHub API limits gracefully.

## ✅ Success Indicators

After a successful run, you should see:

1. **Console output** with profile summary
2. **JSON file** with complete data structure
3. **Markdown file** with formatted profile
4. **No error messages** in verbose mode
5. **Reasonable execution time** (< 3 minutes for most profiles)

## 🚀 Next Steps

Once you've tested successfully:

1. **Analyze your own profile**
2. **Try different templates** for various use cases
3. **Customize the output** for your specific needs
4. **Integrate into your job application workflow**

The tool is designed to give you data-driven insights into your GitHub activity that you can use to enhance your professional profile and stand out in job applications!