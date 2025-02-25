#!/usr/bin/env bash

# Function to print information messages in blue
info() {
  echo -e "\033[0;34m[INFO] $1\033[0m"
}

# Function to print warning messages in yellow
warning() {
  echo -e "\033[0;33m[WARNING] $1\033[0m"
}

# Function to print success messages in green
success() {
  echo -e "\033[0;32m[SUCCESS] $1\033[0m"
}

# Function to print error messages in red
error() {
  echo -e "\033[0;31m[ERROR] $1\033[0m" >&2
}

# Function to print debug messages in purple
# Only prints messages if the DEBUG_MODE environment variable is set to true
debug() {
  if [ "$DEBUG_MODE" = "true" ]; then
    echo -e "\033[0;35m[DEBUG] $1\033[0m"
  fi
}

export -f info warning success error debug
