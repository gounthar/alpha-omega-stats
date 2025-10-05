#!/bin/bash
#
# GitHub User Analyzer Wrapper Script
# Provides convenient interface for analyzing GitHub users
#

set -e

# Default configuration
DEFAULT_OUTPUT_DIR="./data/profiles"
DEFAULT_TEMPLATE=""
DEFAULT_FORMAT="both"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Help function
show_help() {
    cat << EOF
GitHub User Analyzer - Generate professional profiles from GitHub data

Usage: $0 [OPTIONS] USERNAME

Arguments:
    USERNAME        GitHub username to analyze (required)

Options:
    -t, --template TYPE    Profile template: resume, technical, executive, ats (default: resume)
    -o, --output DIR       Output directory (default: ./data/profiles)
    -f, --format FORMAT    Output format: markdown, json, both (default: both)
    -v, --verbose          Enable verbose output
    --token TOKEN          GitHub API token (or set GITHUB_TOKEN env var)
    -h, --help            Show this help message

Examples:
    $0 octocat
    $0 -t technical octocat
    $0 --template ats --output ./resumes octocat
    $0 -f markdown -v octocat

Templates:
    resume      - Professional resume enhancement (default)
    technical   - Deep technical analysis and skills breakdown
    executive   - Leadership and high-level impact focus
    ats         - Optimized for Applicant Tracking Systems

Environment Variables:
    GITHUB_TOKEN    GitHub personal access token (required)

EOF
}

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Check if required tools are available
check_dependencies() {
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    if [[ -z "$TOKEN" ]]; then
        log_error "GitHub token is required. Set GITHUB_TOKEN environment variable or use --token flag."
        log_info "Get a token at: https://github.com/settings/tokens"
        exit 1
    fi
}

# Parse command line arguments
parse_args() {
    USERNAME=""
    TEMPLATE="$DEFAULT_TEMPLATE"
    TEMPLATE_SPECIFIED=0
    OUTPUT_DIR="$DEFAULT_OUTPUT_DIR"
    FORMAT="$DEFAULT_FORMAT"
    VERBOSE=""
    TOKEN="${GITHUB_TOKEN:-}"

    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -t|--template)
                TEMPLATE="$2"
                TEMPLATE_SPECIFIED=1
                shift 2
                ;;
            -o|--output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -f|--format)
                FORMAT="$2"
                shift 2
                ;;
            --token)
                TOKEN="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE="-verbose"
                shift
                ;;
            -*)
                log_error "Unknown option: $1"
                echo "Use -h or --help for usage information."
                exit 1
                ;;
            *)
                if [[ -z "$USERNAME" ]]; then
                    USERNAME="$1"
                else
                    log_error "Multiple usernames provided. Only one username is allowed."
                    exit 1
                fi
                shift
                ;;
        esac
    done

    if [[ -z "$USERNAME" ]]; then
        log_error "GitHub username is required"
        echo "Usage: $0 [OPTIONS] USERNAME"
        echo "Use -h or --help for more information."
        exit 1
    fi

    # Validate template (only if specified)
    if [[ $TEMPLATE_SPECIFIED -eq 1 ]]; then
        case "$TEMPLATE" in
            resume|technical|executive|ats|all) ;;
            *)
                log_error "Invalid template: $TEMPLATE"
                log_info "Valid templates: resume, technical, executive, ats, all"
                exit 1
                ;;
        esac
    fi

    # Validate format
    case "$FORMAT" in
        markdown|json|both) ;;
        *)
            log_error "Invalid format: $FORMAT"
            log_info "Valid formats: markdown, json, both"
            exit 1
            ;;
    esac
}

# Build the Go application if needed
build_analyzer() {
    local binary_path="./github-user-analyzer"
    local source_path="./cmd/github-user-analyzer"

    # Check if binary exists and source is newer
    if [[ ! -f "$binary_path" ]] || [[ "$source_path" -nt "$binary_path" ]]; then
        log_info "Building GitHub User Analyzer..."

        if ! go build -o "$binary_path" "$source_path"; then
            log_error "Failed to build GitHub User Analyzer"
            exit 1
        fi

        log_success "Built GitHub User Analyzer"
    fi
}

# Run the analysis
run_analysis() {
    local start_time=$(date +%s)

    log_info "Analyzing GitHub user: $USERNAME"
    log_info "Template: $TEMPLATE | Format: $FORMAT | Output: $OUTPUT_DIR"

    # Create output directory if it doesn't exist
    mkdir -p "$OUTPUT_DIR"

    # Run the analyzer
    local cmd="./github-user-analyzer -user \"$USERNAME\" -format \"$FORMAT\" -output \"$OUTPUT_DIR\" -token \"$TOKEN\" $VERBOSE"
    if [[ $TEMPLATE_SPECIFIED -eq 1 ]]; then
        cmd+=" -template \"$TEMPLATE\""
    fi

    if [[ -n "$VERBOSE" ]]; then
        log_info "Running: $cmd"
    fi

    if eval "$cmd"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_success "Analysis completed in ${duration} seconds"

        # Show generated files
        echo ""
        log_info "Generated files in $OUTPUT_DIR:"

        if [[ "$FORMAT" == "json" || "$FORMAT" == "both" ]]; then
            if [[ -f "$OUTPUT_DIR/${USERNAME}_profile.json" ]]; then
                echo "  📄 ${USERNAME}_profile.json (raw data)"
            fi
        fi

        if [[ "$FORMAT" == "markdown" || "$FORMAT" == "both" ]]; then
            if [[ -f "$OUTPUT_DIR/${USERNAME}_profile_${TEMPLATE}.md" ]]; then
                echo "  📝 ${USERNAME}_profile_${TEMPLATE}.md (${TEMPLATE} template)"
            fi
        fi

        echo ""
        log_success "Ready to enhance your resume! 🚀"

    else
        log_error "Analysis failed"
        exit 1
    fi
}

# Main execution
main() {
    echo "GitHub User Analyzer v1.0.0"
    echo "========================================="
    echo ""

    parse_args "$@"
    check_dependencies
    build_analyzer
    run_analysis
}

# Run main function with all arguments
main "$@"