#!/bin/bash

# Specter Testing Setup Script
# This script helps you set up Specter for testing with the Aurora ES

set -e

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║         Specter Testing Setup for Aurora ES              ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ first."
    echo "   macOS: brew install go"
    echo "   Or download from: https://golang.org/dl/"
    exit 1
fi

echo "✓ Go is installed: $(go version)"
echo ""

# Download dependencies
echo "📦 Installing dependencies..."
go mod download
echo "✓ Dependencies installed"
echo ""

# Build the application
echo "🔨 Building Specter..."
go build -o specter .
echo "✓ Specter built successfully"
echo ""

# Create config if it doesn't exist
if [ ! -f config.yaml ]; then
    echo "📝 Creating test configuration..."
    cat > config.yaml << 'EOF'
# Test configuration for Aurora ES 10-Year
item_url: "https://robertsspaceindustries.com/en/pledge/Standalone-Ships/Aurora-ES-10-Year"

browser_profile_path: ""
browser_type: "chrome"

# Timeouts
page_load_timeout: 30
min_delay_between: 0.5
max_delay_between: 1.0
checkout_ready_delay: 2

# Testing modes enabled
headless: false
keep_browser_open: true
auto_apply_credit: true

# Test modes
dry_run: true
interactive: true
debug_mode: true

# Selectors (may need to be updated after testing)
selectors:
  add_to_cart_button: ".add-to-cart, .js-add-to-cart, button[data-action='add-to-cart']"
  cart_icon: ".cart-icon, .shopping-cart, [data-testid='cart']"
  checkout_button: ".checkout-button, .btn-checkout, button:contains('Checkout')"
  apply_credit_button: ".apply-credit, .store-credit, button:contains('Apply Credit')"
  agree_checkbox: "input[type='checkbox'][name*='agree'], input[type='checkbox'][name*='terms']"
  final_checkout_button: ".final-checkout, .confirm-order, button:contains('Confirm')"
EOF
    echo "✓ Created config.yaml with test settings"
else
    echo "⚠  config.yaml already exists, skipping creation"
fi
echo ""

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║                    Setup Complete!                        ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "Next steps:"
echo ""
echo "1. Run the app to establish your login (with 2FA):"
echo "   ./specter"
echo ""
echo "2. In the browser window:"
echo "   - Navigate to robertsspaceindustries.com"
echo "   - Sign in and complete 2FA authentication"
echo "   - Wait 10 seconds, then Ctrl+C in the terminal"
echo ""
echo "3. Test the flow in interactive mode:"
echo "   ./specter -interactive -debug -dry-run"
echo ""
echo "4. Read the testing guide:"
echo "   cat HOW_TO_TEST.md"
echo ""
echo "Happy testing! 🚀"
