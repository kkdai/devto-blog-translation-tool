#!/bin/bash
# Translate and publish Dev.to draft articles using Gemini
# Usage: ./translate.sh

# Ensure API keys are set
if [ -z "$DEVTO_API_KEY" ]; then
  echo "Error: DEVTO_API_KEY environment variable is not set"
  exit 1
fi

if [ -z "$GEMINI_API_KEY" ]; then
  echo "Error: GEMINI_API_KEY environment variable is not set"
  exit 1
fi

# Navigate to the Go tool directory
cd "$(dirname "$0")/translator" || exit 1

# Run the translator tool
# Non-interactive mode (defaulting to 'yes' for the prompt if needed)
# In this skill's context, the agent will confirm first anyway.
# We'll use 'yes' to pipe to the Go tool's confirmation prompt.
echo "yes" | go run cmd/translator/main.go
