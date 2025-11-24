#!/bin/bash

# Selector Discovery Helper Script
# This script helps you find the correct CSS selectors for RSI website

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║         RSI Selector Discovery Tool                      ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "This script will help you find the correct CSS selectors"
echo "for the Aurora ES page on robertsspaceindustries.com"
echo ""
echo "INSTRUCTIONS:"
echo "1. The browser will open and navigate to the Aurora ES page"
echo "2. When you see 'Page loaded. Review the page if needed.'"
echo "3. Type: inspect"
echo "4. Look through the list of buttons for 'Add to Cart' or similar"
echo "5. Note the Class and ID values"
echo "6. Press Enter to continue"
echo ""
echo "At the end, this script will help you update config.yaml"
echo ""
read -p "Press Enter to start..."

# Run specter in interactive mode
./specter -interactive -debug -dry-run

echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║         Update Your Selectors                            ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "Based on what you saw in the 'inspect' output:"
echo ""
read -p "What was the Class of the 'Add to Cart' button? (e.g., .js-buy-btn): " add_to_cart_class
read -p "What was the cart icon Class? (e.g., .cart-link): " cart_class
echo ""

if [ ! -z "$add_to_cart_class" ]; then
    echo "Updating config.yaml with your selectors..."

    # Backup original config
    cp config.yaml config.yaml.backup

    # Update the add_to_cart_button selector
    sed -i.tmp "s|add_to_cart_button:.*|add_to_cart_button: $add_to_cart_class|" config.yaml

    if [ ! -z "$cart_class" ]; then
        sed -i.tmp "s|cart_icon:.*|cart_icon: $cart_class|" config.yaml
    fi

    rm -f config.yaml.tmp

    echo "✓ Config updated!"
    echo ""
    echo "Your old config was saved to config.yaml.backup"
    echo ""
    echo "Try running the test again:"
    echo "  ./specter -dry-run"
else
    echo "No changes made. You can manually edit config.yaml"
fi
