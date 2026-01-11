#!/bin/bash

echo "🔒 Security Test - Checking for API key leaks"
echo "=============================================="
echo ""

# Check if .env is ignored by git
if git check-ignore -q .env; then
    echo "✅ .env file is properly ignored by git"
else
    echo "❌ WARNING: .env file is NOT ignored by git!"
    exit 1
fi

# Check if .env exists
if [ -f .env ]; then
    echo "✅ .env file exists with API keys"
else
    echo "❌ .env file not found"
    exit 1
fi

# Check if .env.example exists
if [ -f .env.example ]; then
    echo "✅ .env.example template exists"
else
    echo "❌ .env.example not found"
    exit 1
fi

# Search for potential API key patterns in tracked files
echo ""
echo "Searching for API key patterns in git-tracked files..."
# Check for patterns like DEVTO_API_KEY= or GEMINI_API_KEY= followed by actual values
if git ls-files | xargs grep -E "DEVTO_API_KEY=[^y]|GEMINI_API_KEY=[^y]" 2>/dev/null | grep -v ".env.example" | grep -v "test_security.sh"; then
    echo "❌ WARNING: Potential API keys found in tracked files!"
    exit 1
else
    echo "✅ No API keys found in tracked files"
fi

echo ""
echo "=============================================="
echo "✅ All security checks passed!"
echo "Your API keys are safe and protected."
