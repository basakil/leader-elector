#!/bin/bash

# This script is designed to be sourced to set temporary AWS credentials
# using an MFA token.

set -e

print_help() {
  cat <<EOF
Obtain and export temporary AWS credentials using MFA (Multi-Factor Authentication).

Usage:
  source $0 [<MFA_DEVICE_ARN>] [--token <MFA_TOKEN_CODE>] [--help|-h]

Parameters:
  <MFA_DEVICE_ARN>   (positional) The ARN or serial number of your MFA device (optional if MFA_DEVICE_ARN env var is set)
  --token <code>     The MFA token code (if not provided, you will be prompted)
  -h, --help         Show this help message and exit/return

Examples:
  source $0 arn:aws:iam::123456789012:mfa/my-user --token 123456
  source $0 --token 123456
  export MFA_DEVICE_ARN=arn:aws:iam::123456789012:mfa/my-user; source $0
EOF
}

# Find project root (parent of script directory) without cd
SCRIPT=$(readlink -f "$0")
SCRIPTPATH=$(dirname "$SCRIPT")
PROJECT_ROOT="$SCRIPTPATH/.."

# --- Configuration ---
# Maximum duration for sts:GetSessionToken is 12 hours (43200 seconds)
SESSION_DURATION_SECONDS=43100
# --- End Configuration ---

MFA_TOKEN_CODE=""
_positional_mfa_arn_provided="" # Flag to indicate if MFA_DEVICE_ARN was given as a positional arg

# Parse arguments
while [[ "$#" -gt 0 ]]; do
    case "$1" in
        --token)
            if [ -n "$2" ]; then
                MFA_TOKEN_CODE="$2"
                shift # past argument
            else
                echo "Error: --token requires a value."
                return 1
            fi
            ;;
        -h|--help)
            print_help
            return 0
            ;;
        *) # Positional argument (MFA_DEVICE_ARN)
            # Check if it looks like an ARN or a serial number
            if [[ "$1" =~ ^arn:aws:iam::[0-9]{12}:mfa/.*$ ]] || [[ "$1" =~ ^[0-9]{9,}$ ]]; then
                if [ -n "$_positional_mfa_arn_provided" ]; then
                    echo "Warning: Multiple positional arguments for MFA_DEVICE_ARN found. Using the last one: '$1'."
                fi
                _positional_mfa_arn_provided="$1"
            else
                echo "Error: Unrecognized argument or invalid MFA Device ARN format: '$1'."
                print_help
                return 1
            fi
            ;;
    esac
    shift # past argument or value
done

# Determine the MFA_DEVICE_ARN to use
# If a positional argument was provided, it takes precedence.
if [ -n "$_positional_mfa_arn_provided" ]; then
    if [ -n "$MFA_DEVICE_ARN" ] && [ "$MFA_DEVICE_ARN" != "$_positional_mfa_arn_provided" ]; then
        echo "Info: Overriding environment variable MFA_DEVICE_ARN ($MFA_DEVICE_ARN) with positional argument '$_positional_mfa_arn_provided'."
    fi
    MFA_DEVICE_ARN="$_positional_mfa_arn_provided"
elif [ -z "$MFA_DEVICE_ARN" ]; then
    # If no positional arg and env var is not set, then it's an error
    print_help
    echo "Error: Please provide your MFA Device ARN as the first argument or set it as an environment variable (e.g., export MFA_DEVICE_ARN='arn:aws:iam::...')."
    return 1 # Use return for sourced scripts to exit the sourcing
else
    echo "Info: Using MFA_DEVICE_ARN from environment variable: $MFA_DEVICE_ARN"
fi

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo "Error: AWS CLI is not installed. Please install it to use this script."
    return 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: 'jq' is not installed. Please install 'jq' (a JSON processor) to use this script."
    echo "  For Debian/Ubuntu: sudo apt-get install jq"
    echo "  For macOS (Homebrew): brew install jq"
    return 1
fi

# If MFA_TOKEN_CODE is not provided, prompt the user
if [ -z "$MFA_TOKEN_CODE" ]; then
    read -r -p "Enter MFA Token for $MFA_DEVICE_ARN: " MFA_TOKEN_CODE
    if [ -z "$MFA_TOKEN_CODE" ]; then
        echo "Error: MFA Token cannot be empty."
        return 1
    fi
    # Optional: Basic validation for 6-digit token
    if ! [[ "$MFA_TOKEN_CODE" =~ ^[0-9]{6}$ ]]; then
        echo "Warning: MFA token '$MFA_TOKEN_CODE' does not appear to be a 6-digit number."
        # You might choose to exit here, but a warning allows the user to proceed if their token format is different.
    fi
fi

# Call the logout function if it exists to clear previous credentials
if command -v aws_mfa_logout &> /dev/null; then
    aws_mfa_logout
else
    echo "Info: 'aws_mfa_logout' function not yet defined or available; skipping explicit logout."
fi


echo "Attempting to get temporary AWS credentials using MFA..."

# Attempt to get session token from STS
# We're using default AWS_PROFILE and AWS_REGION if they are set in the environment,
# otherwise, it falls back to the AWS CLI's default configuration.
AWS_CREDENTIALS=$(aws sts get-session-token \
    --serial-number "$MFA_DEVICE_ARN" \
    --token-code "$MFA_TOKEN_CODE" \
    --duration-seconds "$SESSION_DURATION_SECONDS" \
    --output json 2>&1) # Redirect stderr to stdout for error capture

# Check for errors from the AWS CLI command
if [ $? -ne 0 ]; then
    echo "Error getting temporary AWS credentials:"
    echo "$AWS_CREDENTIALS" # This will contain the error message from aws cli
    return 1
fi

# Parse the JSON output and set environment variables
AWS_ACCESS_KEY_ID=$(echo "$AWS_CREDENTIALS" | jq -r '.Credentials.AccessKeyId')
AWS_SECRET_ACCESS_KEY=$(echo "$AWS_CREDENTIALS" | jq -r '.Credentials.SecretAccessKey')
AWS_SESSION_TOKEN=$(echo "$AWS_CREDENTIALS" | jq -r '.Credentials.SessionToken')
CREDENTIALS_EXPIRATION=$(echo "$AWS_CREDENTIALS" | jq -r '.Credentials.Expiration')


# Check if parsing was successful
if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ] || [ -z "$AWS_SESSION_TOKEN" ]; then
    echo "Error: Failed to parse AWS credentials from STS response. Ensure 'jq' is installed and response format is as expected."
    echo "Full STS response (for debugging):"
    echo "$AWS_CREDENTIALS"
    return 1
fi

# Set the environment variables
export AWS_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY
export AWS_SESSION_TOKEN

echo "---"
echo "AWS temporary credentials set successfully!"
echo "Access Key ID: $AWS_ACCESS_KEY_ID"
echo "Expiration:    $CREDENTIALS_EXPIRATION (Maximum 12 hours)"
echo "---"

# Optional: Add a function to clear credentials if you like
# This function will only be available in the shell if sourced.
aws_mfa_logout() {
    unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN
    echo "AWS temporary credentials cleared."
}
export -f aws_mfa_logout # Make the function available in the shell

return 0