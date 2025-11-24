package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Automation handles the browser automation logic
type Automation struct {
	config       *Config
	browser      *rod.Browser
	page         *rod.Page
	launcher     *launcher.Launcher
	rand         *rand.Rand
	itemPrice    string // Auto-detected price from item page
	stopChan     chan bool // Channel to signal shutdown
	itemInCart   bool   // Flag to track if item is already in cart (for smart retries)
}

// NewAutomation creates a new automation instance
func NewAutomation(config *Config) *Automation {
	return &Automation{
		config:   config,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		stopChan: make(chan bool, 1),
	}
}

// Close closes the browser and destroys the browser session
func (a *Automation) Close() {
	// Signal shutdown to watcher
	select {
	case a.stopChan <- true:
	default:
	}

	fmt.Println("\n🔒 Cleaning up browser session...")

	if a.page != nil {
		a.page.Close()
	}

	if a.browser != nil {
		a.browser.Close()
	}

	if a.launcher != nil {
		a.launcher.Cleanup()
	}

	fmt.Println("✓ Browser session destroyed")
}

// isBrowserAlive checks if the browser is still alive and connected
func (a *Automation) isBrowserAlive() bool {
	if a.browser == nil {
		return false
	}

	// Try multiple checks to detect if browser is closed
	// Check 1: Try to get browser version
	_, err := a.browser.Version()
	if err != nil {
		a.debugLog("Browser version check failed: %v", err)
		return false
	}

	// Check 2: Try to get page info if page exists
	if a.page != nil {
		_, err := a.page.Info()
		if err != nil {
			a.debugLog("Page info check failed: %v", err)
			return false
		}
	}

	return true
}

// checkBrowserOrExit checks if browser is alive, exits if not
func (a *Automation) checkBrowserOrExit() {
	if !a.isBrowserAlive() {
		fmt.Println("\n⚠️  Browser window was closed by user")
		fmt.Println("🛑 Shutting down gracefully...")
		os.Exit(0)
	}
}

// watchBrowser monitors if the browser is closed and shuts down the app gracefully
func (a *Automation) watchBrowser() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopChan:
			// Shutdown signal received
			return
		case <-ticker.C:
			// Check if browser is still alive
			a.checkBrowserOrExit()
		}
	}
}

// randomDelay waits for a random duration between min and max delays
func (a *Automation) randomDelay() {
	min := a.config.MinDelayBetween
	max := a.config.MaxDelayBetween
	duration := min + a.rand.Float64()*(max-min)

	if !a.config.DebugMode {
		fmt.Printf("⏱  Waiting %.2f seconds...\n", duration)
	}
	time.Sleep(time.Duration(duration * float64(time.Second)))
}

// getTimeout returns a randomized timeout duration (200-380ms) to avoid detection
func (a *Automation) getTimeout() time.Duration {
	timeoutMs := 200 + a.rand.Intn(180) // Random 200-380ms
	return time.Duration(timeoutMs) * time.Millisecond
}

// getClickTimeout returns a randomized timeout for click operations (700-1100ms) to avoid detection
func (a *Automation) getClickTimeout() time.Duration {
	timeoutMs := 700 + a.rand.Intn(400) // Random 700-1100ms
	return time.Duration(timeoutMs) * time.Millisecond
}

// reliableClick performs a click using JavaScript (faster and more reliable than Rod's native click)
func (a *Automation) reliableClick(elem *rod.Element) error {
	// Use JavaScript click with randomized timeout (700-1100ms) - fast and avoids detection
	_, err := elem.Timeout(a.getClickTimeout()).Eval(`() => this.click()`)
	if err != nil {
		return fmt.Errorf("click failed: %w", err)
	}
	return nil
}

// sleepWithBrowserCheck sleeps for the given duration but checks browser status periodically
func (a *Automation) sleepWithBrowserCheck(duration time.Duration) {
	checkInterval := 500 * time.Millisecond // Check every 500ms
	elapsed := time.Duration(0)

	for elapsed < duration {
		// Check if browser is still alive
		a.checkBrowserOrExit()

		// Sleep for check interval or remaining time, whichever is shorter
		sleepTime := checkInterval
		remaining := duration - elapsed
		if remaining < sleepTime {
			sleepTime = remaining
		}

		time.Sleep(sleepTime)
		elapsed += sleepTime
	}
}

// retryDelay waits for a random duration between retry delay min and max
func (a *Automation) retryDelay() {
	min := a.config.RetryDelayMin
	max := a.config.RetryDelayMax
	duration := min + a.rand.Float64()*(max-min)

	fmt.Printf("⏳ Waiting %.2f seconds before retry...\n", duration)
	a.sleepWithBrowserCheck(time.Duration(duration * float64(time.Second)))
}

// debugLog prints a message only if debug mode is enabled
func (a *Automation) debugLog(format string, args ...interface{}) {
	if a.config.DebugMode {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// waitForUser pauses and waits for user to press Enter (if interactive mode is enabled)
func (a *Automation) waitForUser(message string) {
	if !a.config.Interactive {
		return
	}

	fmt.Printf("\n⏸  %s\n", message)
	fmt.Print("Press Enter to continue (or type 'inspect' to see page elements, 'screenshot' to save image)... ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "inspect" {
		a.inspectPage()
		fmt.Print("\nPress Enter to continue... ")
		reader.ReadString('\n')
	} else if input == "screenshot" {
		a.takeScreenshot()
		fmt.Print("\nPress Enter to continue... ")
		reader.ReadString('\n')
	}
}

// inspectPage shows all interactive elements on the current page
func (a *Automation) inspectPage() {
	if a.page == nil {
		fmt.Println("No page loaded")
		return
	}

	fmt.Println("\n=== PAGE INSPECTION ===")

	// Get URL
	info, err := a.page.Info()
	if err != nil {
		fmt.Printf("URL: <unable to get>\n")
	} else {
		fmt.Printf("URL: %s\n", info.URL)
	}

	// Use JavaScript to return JSON string, then parse it
	fmt.Println("\nButtons:")

	// Execute JavaScript to get button info as JSON string
	buttonsJS := `JSON.stringify(
		Array.from(document.querySelectorAll('button')).slice(0, 20).map((btn, i) => ({
			index: i,
			text: (btn.innerText || '').trim().substring(0, 50),
			className: (btn.className || '').substring(0, 80),
			id: btn.id || '',
			dataCy: btn.getAttribute('data-cy-id') || ''
		}))
	)`

	buttonsResult, err := a.page.Eval(buttonsJS)
	if err != nil {
		fmt.Printf("  ⚠ Error fetching buttons: %v\n", err)
	} else {
		jsonStr := buttonsResult.Value.Str()
		// Manual JSON parsing for simple structure
		if jsonStr != "" && jsonStr != "[]" {
			// Simple display - just show we have buttons
			buttonCount := strings.Count(jsonStr, "\"index\"")
			fmt.Printf("  Found %d buttons. Use browser DevTools (F12) for detailed inspection.\n", buttonCount)
			fmt.Println("  Tip: Right-click element in browser → Inspect → Copy selector")
		} else {
			fmt.Println("  No buttons found on page")
		}
	}

	// Try a simpler approach - just get text content
	fmt.Println("\nButtons with text (simplified view):")
	simpleButtonsJS := `
		Array.from(document.querySelectorAll('button'))
			.slice(0, 20)
			.map((b, i) => i + ': "' + (b.innerText || '').trim().substring(0, 40) + '" [class: ' + (b.className || '').split(' ')[0] + ']')
			.join('\\n')
	`

	simpleResult, err := a.page.Eval(simpleButtonsJS)
	if err != nil {
		fmt.Printf("  ⚠ Error: %v\n", err)
	} else {
		output := simpleResult.Value.Str()
		if output != "" {
			fmt.Println(output)
		}
	}

	fmt.Println("\nInput fields:")
	inputsJS := `
		Array.from(document.querySelectorAll('input'))
			.slice(0, 15)
			.map((inp, i) => i + ': type=' + (inp.type || '') + ' name=' + (inp.name || '') + ' id=' + (inp.id || '') + ' placeholder="' + (inp.placeholder || '') + '"')
			.join('\\n')
	`

	inputsResult, err := a.page.Eval(inputsJS)
	if err != nil {
		fmt.Printf("  ⚠ Error: %v\n", err)
	} else {
		output := inputsResult.Value.Str()
		if output != "" {
			fmt.Println(output)
		} else {
			fmt.Println("  No input fields found")
		}
	}

	fmt.Println("\n💡 TIP: For detailed element inspection:")
	fmt.Println("   1. Look at the browser window")
	fmt.Println("   2. Press F12 to open DevTools")
	fmt.Println("   3. Click the inspector tool (top-left)")
	fmt.Println("   4. Click the element you want")
	fmt.Println("   5. Right-click in DevTools → Copy → Copy selector")
	fmt.Println("======================")
}

// takeScreenshot saves a screenshot of the current page
func (a *Automation) takeScreenshot() {
	if a.page == nil {
		fmt.Println("No page loaded")
		return
	}

	filename := fmt.Sprintf("screenshot_%d.png", time.Now().Unix())
	data, err := a.page.Screenshot(true, nil)
	if err != nil {
		fmt.Printf("Failed to take screenshot: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf("Failed to save screenshot: %v\n", err)
		return
	}

	fmt.Printf("✓ Screenshot saved to: %s\n", filename)
}

// setupBrowser initializes the browser instance
func (a *Automation) setupBrowser() error {
	fmt.Println("🚀 Launching browser...")

	// Configure launcher with leakless mode to prevent "Opening in existing browser session" errors
	// Leakless mode ensures the browser closes properly when the program exits
	a.launcher = launcher.New().
		Headless(a.config.Headless).
		UserDataDir(a.config.BrowserProfilePath).
		Leakless(true)

	// Note: Browser type selection (Chrome/Edge/Firefox) is handled automatically
	// by the launcher finding the default browser. For advanced usage, you can
	// set specific browser paths in the launcher.

	url, err := a.launcher.Launch()
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "Opening in existing browser session") {
			return fmt.Errorf("failed to launch browser: Chrome is already running. Please close all Chrome windows and try again")
		}
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	// Connect to browser
	a.browser = rod.New().ControlURL(url).MustConnect()

	// Note: We don't set a default timeout on browser/page level
	// Instead, we use explicit timeouts for each operation to avoid timeout cascade issues

	// Start browser watcher in background
	go a.watchBrowser()
	a.debugLog("Browser watcher started")

	fmt.Println("✓ Browser launched successfully")
	return nil
}

// waitForLogin opens a login page and waits for user to confirm they're logged in and ready
func (a *Automation) waitForLogin() error {
	fmt.Println("🔐 Opening browser for login...")

	// Create a page for the RSI main page or login page
	loginURL := "https://robertsspaceindustries.com"

	var err error
	a.page, err = a.browser.Page(proto.TargetCreateTarget{URL: loginURL})
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	// Note: No default page-level timeout - use explicit timeouts per operation

	// Set realistic Chrome User-Agent to avoid detection (BEFORE page loads)
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	err = a.page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: userAgent,
	})
	if err != nil {
		a.debugLog("Warning: Failed to set User-Agent: %v", err)
	} else {
		a.debugLog("User-Agent set to Chrome desktop")
	}

	// Set viewport to desktop size BEFORE page loads
	// Critical for RSI - store credit only appears in desktop view
	err = a.page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  a.config.ViewportWidth,
		Height: a.config.ViewportHeight,
		DeviceScaleFactor: 1,
		Mobile: false,
	})
	if err != nil {
		a.debugLog("Warning: Failed to set viewport: %v", err)
	} else {
		a.debugLog("Viewport set to %dx%d (desktop)", a.config.ViewportWidth, a.config.ViewportHeight)
	}

	// Wait for page to load
	if err := a.page.WaitLoad(); err != nil {
		return fmt.Errorf("page failed to load: %w", err)
	}

	fmt.Println("✓ Browser configured with desktop viewport")

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                      LOGIN REQUIRED                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("📋 Instructions:")
	fmt.Println("   1. Log in to your RSI account in the browser window")
	fmt.Println("   2. WAIT until the ship is AVAILABLE on the RSI store")
	fmt.Println("   3. When ready, press ENTER to start the automated checkout")
	fmt.Println()
	fmt.Println("   ⚠️  Press ESC to exit if you need to cancel")
	fmt.Println()
	fmt.Print("⏳ Press ENTER when logged in and ship is available (or ESC to exit): ")

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		// Check for ENTER key (newline)
		if input == '\n' || input == '\r' {
			fmt.Println()
			fmt.Println("✓ User confirmed ready to proceed")
			break
		}

		// Check for ESC key (ASCII 27)
		if input == 27 {
			fmt.Println()
			fmt.Println("⚠️  User requested exit")
			return fmt.Errorf("user canceled operation")
		}
	}

	return nil
}

// navigateToItem navigates to the item URL
// detectItemPrice extracts the item price from the store page
func (a *Automation) detectItemPrice() error {
	fmt.Println("💵 Detecting item price...")

	// Use JavaScript to find the price element
	priceJS := `() => {
		// Try RSI's specific price selector
		var priceElem = document.querySelector('.a-priceUnit__amount[data-cy-id="price_unit__value"]');
		if (priceElem) {
			return priceElem.textContent.trim();
		}

		// Fallback: try other common price selectors
		var fallbacks = [
			'.a-priceUnit__amount',
			'[data-cy-id="price_unit__value"]',
			'.price-amount',
			'.item-price'
		];

		for (var i = 0; i < fallbacks.length; i++) {
			var elem = document.querySelector(fallbacks[i]);
			if (elem) {
				return elem.textContent.trim();
			}
		}

		return '';
	}`

	result, err := a.page.Timeout(5 * time.Second).Eval(priceJS)
	if err != nil {
		return fmt.Errorf("failed to detect price: %w", err)
	}

	priceText := result.Value.Str()
	if priceText == "" {
		return fmt.Errorf("could not find price on item page")
	}

	// Remove dollar sign and extract just the number (e.g., "$20.00" -> "20.00")
	priceText = strings.TrimSpace(priceText)
	priceText = strings.TrimPrefix(priceText, "$")

	a.itemPrice = priceText
	fmt.Printf("✓ Detected item price: $%s\n", a.itemPrice)

	return nil
}

func (a *Automation) navigateToItem() error {
	fmt.Printf("🌐 Navigating to item: %s\n", a.config.ItemURL)

	var err error

	// Reuse existing page from waitForLogin() and navigate to item URL
	if a.page != nil {
		err = a.page.Navigate(a.config.ItemURL)
		if err != nil {
			return fmt.Errorf("failed to navigate to item: %w", err)
		}

		// Wait for navigation to complete
		err = a.page.WaitLoad()
		if err != nil {
			return fmt.Errorf("failed to wait for page load: %w", err)
		}
	} else {
		// Fallback: create page if somehow it doesn't exist
		a.page, err = a.browser.Page(proto.TargetCreateTarget{URL: a.config.ItemURL})
		if err != nil {
			return fmt.Errorf("failed to create page: %w", err)
		}

		// Note: No default page-level timeout - use explicit timeouts per operation

		// Set realistic Chrome User-Agent to avoid detection
		userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
		err = a.page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: userAgent,
		})
		if err != nil {
			a.debugLog("Warning: Failed to set User-Agent: %v", err)
		}

		// Set viewport to desktop size
		err = a.page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width:  a.config.ViewportWidth,
			Height: a.config.ViewportHeight,
			DeviceScaleFactor: 1,
			Mobile: false,
		})
		if err != nil {
			a.debugLog("Warning: Failed to set viewport: %v", err)
		}

		err = a.page.WaitLoad()
		if err != nil {
			return fmt.Errorf("failed to wait for page load: %w", err)
		}
	}

	fmt.Println("✓ Page loaded successfully")

	// Detect the item price from the page
	if err := a.detectItemPrice(); err != nil {
		fmt.Printf("⚠  Warning: %v\n", err)
		fmt.Println("⚠  Will use fallback amount for store credit")
	}

	a.waitForUser("Page loaded. Review the page if needed.")
	return nil
}

// addToCart adds the item to the cart
func (a *Automation) addToCart() error {
	fmt.Println("🛒 Looking for 'Add to Cart' button...")

	a.randomDelay()

	// Try to find the add to cart button with retry loop
	selectors := []string{
		a.config.Selectors.AddToCartButton,
		"button.js-add-to-cart",
		"button[data-action='add-to-cart']",
		"input[type='submit'][value*='Add']",
		"a.add-to-cart",
	}

	var addButton *rod.Element
	maxAttempts := 10

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Small delay between retry attempts
			delayMs := 200 + a.rand.Intn(300) // 200-500ms
			a.debugLog("Retry %d/%d - waiting %dms...", attempt-1, maxAttempts-1, delayMs)
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}

		// Try each selector
		for _, selector := range selectors {
			a.debugLog("Trying add to cart selector: %s", selector)
			elem, err := a.page.Timeout(800 * time.Millisecond).Element(selector)
			if err != nil {
				a.debugLog("Selector failed: %v", err)
			}
			if err == nil && elem != nil {
				addButton = elem
				fmt.Printf("✓ Found add to cart button: %s (attempt %d)\n", selector, attempt)
				break
			}
		}

		if addButton != nil {
			break
		}
	}

	if addButton == nil {
		return fmt.Errorf("could not find add to cart button")
	}

	// Check if button is visible and enabled
	visible, err := addButton.Visible()
	if err != nil || !visible {
		return fmt.Errorf("add to cart button is not visible")
	}

	// Click the button using direct JavaScript (most reliable approach)
	a.waitForUser("About to add item to cart.")
	a.randomDelay()

	// Use JavaScript to click - more reliable than element-based click
	clickJS := `() => { document.querySelector('.m-storeAction__button').click() }`
	_, err = a.page.Timeout(2 * time.Second).Eval(clickJS)
	if err != nil {
		return fmt.Errorf("failed to click add to cart: %w", err)
	}

	fmt.Println("✓ Item added to cart")

	// Mark item as in cart for smart retries
	a.itemInCart = true
	a.debugLog("Item marked as in cart - future retries will skip add-to-cart step")

	return nil
}

// goToCart navigates to the shopping cart
func (a *Automation) goToCart() error {
	fmt.Println("🛍  Navigating to cart...")

	a.randomDelay()

	// Ensure viewport is set to desktop BEFORE navigating to cart
	// Critical: RSI hides store credit input on mobile viewport
	err := a.page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             a.config.ViewportWidth,
		Height:            a.config.ViewportHeight,
		DeviceScaleFactor: 1,
		Mobile:            false,
	})
	if err != nil {
		a.debugLog("Warning: Failed to set viewport: %v", err)
	} else {
		a.debugLog("Viewport confirmed: %dx%d (desktop mode)", a.config.ViewportWidth, a.config.ViewportHeight)
	}

	// For RSI, navigate directly to the cart URL
	// Note: Must use the full store URL path
	cartURL := "https://robertsspaceindustries.com/en/store/pledge/cart"

	a.debugLog("Navigating directly to cart URL: %s", cartURL)
	if err := a.page.Navigate(cartURL); err != nil {
		return fmt.Errorf("failed to navigate to cart: %w", err)
	}

	// Wait for cart page to load with longer timeout
	if err := a.page.Timeout(time.Duration(a.config.PageLoadTimeout) * time.Second).WaitLoad(); err != nil {
		return fmt.Errorf("failed to load cart page: %w", err)
	}

	// Brief wait for dynamic content to render (RSI uses client-side rendering)
	a.randomDelay()

	// Minimal wait for store credit section to render - speed is critical
	time.Sleep(time.Duration(300+rand.Intn(200)) * time.Millisecond) // 300-500ms

	fmt.Println("✓ In shopping cart")

	a.waitForUser("Cart page loaded. Verify item is in cart.")
	return nil
}

// proceedToCheckout clicks the checkout button
func (a *Automation) proceedToCheckout() error {
	fmt.Println("💳 Proceeding to checkout...")

	a.randomDelay()

	// Find checkout button
	selectors := []string{
		a.config.Selectors.CheckoutButton,
		"button[type='submit']:contains('Checkout')",
		"a.checkout-button",
		"button.js-checkout",
		"input[type='submit'][value*='Checkout']",
	}

	var checkoutButton *rod.Element
	for _, selector := range selectors {
		elem, err := a.page.Element(selector)
		if err == nil && elem != nil {
			checkoutButton = elem
			fmt.Printf("✓ Found checkout button: %s\n", selector)
			break
		}
	}

	if checkoutButton == nil {
		return fmt.Errorf("could not find checkout button")
	}

	// Click checkout
	a.randomDelay()
	if err := a.reliableClick(checkoutButton); err != nil {
		return fmt.Errorf("failed to click checkout: %w", err)
	}

	// Minimal wait for checkout page transition - speed is critical
	time.Sleep(time.Duration(400+rand.Intn(200)) * time.Millisecond) // 400-600ms

	fmt.Println("✓ On checkout page")
	return nil
}

// checkIfTotalIsZero checks if the cart total is already $0
func (a *Automation) checkIfTotalIsZero() bool {
	totalJS := `() => {
		var labels = document.querySelectorAll('p[data-cy-id="__label"]');
		for (var i = 0; i < labels.length; i++) {
			if (labels[i].textContent.indexOf('Total') !== -1) {
				if (labels[i].parentElement) {
					return labels[i].parentElement.textContent.trim();
				}
			}
		}
		return '';
	}`

	totalResult, err := a.page.Timeout(3 * time.Second).Eval(totalJS)
	if err != nil || totalResult == nil {
		a.debugLog("Could not check total: %v", err)
		return false
	}

	totalText := totalResult.Value.Str()
	a.debugLog("Total check: '%s'", totalText)

	if totalText == "" {
		return false
	}

	// Check if $0
	isZero := false
	totalLower := strings.ToLower(totalText)

	if strings.Contains(totalText, "Total0") ||
	   strings.Contains(totalText, "Total $0") ||
	   strings.Contains(totalText, "Total$0") ||
	   (strings.Contains(totalLower, "total") &&
	    (strings.HasSuffix(totalText, "0") || strings.HasSuffix(totalText, "0.00") || strings.HasSuffix(totalText, "$0"))) {

		hasOtherDigits := false
		for _, r := range totalText {
			if r >= '1' && r <= '9' {
				hasOtherDigits = true
				break
			}
		}

		if !hasOtherDigits {
			isZero = true
		}
	}

	return isZero
}

// detectCartSubtotal extracts the subtotal from the cart page
func (a *Automation) detectCartSubtotal() (string, error) {
	fmt.Println("💵 Detecting cart subtotal...")

	// Use JavaScript to find the subtotal amount
	subtotalJS := `() => {
		// Find the element with "Subtotal" label
		var labels = document.querySelectorAll('[data-cy-id="__label"]');
		for (var i = 0; i < labels.length; i++) {
			if (labels[i].textContent.trim() === 'Subtotal') {
				// Found the Subtotal label, now find the price in the same row
				var row = labels[i].closest('.m-summaryLineItem__row');
				if (row) {
					var priceElem = row.querySelector('[data-cy-id="price_unit__value"]');
					if (priceElem) {
						return priceElem.textContent.trim();
					}
				}
			}
		}
		return '';
	}`

	result, err := a.page.Timeout(5 * time.Second).Eval(subtotalJS)
	if err != nil {
		return "", fmt.Errorf("failed to detect subtotal: %w", err)
	}

	subtotalText := result.Value.Str()
	if subtotalText == "" {
		return "", fmt.Errorf("could not find subtotal on cart page")
	}

	// Remove dollar sign and extract just the number (e.g., "$1,950.00" -> "1950.00")
	subtotalText = strings.TrimSpace(subtotalText)
	subtotalText = strings.TrimPrefix(subtotalText, "$")
	subtotalText = strings.ReplaceAll(subtotalText, ",", "") // Remove comma separators

	fmt.Printf("✓ Detected cart subtotal: $%s\n", subtotalText)
	return subtotalText, nil
}

// applyStoreCredit applies store credit to the order
func (a *Automation) applyStoreCredit() error {
	if !a.config.AutoApplyCredit {
		fmt.Println("⏭  Skipping store credit (disabled in config)")
		return nil
	}

	// Step 0: Check if store credit already applied (from previous attempt/session)
	fmt.Println("🔍 Checking if store credit already applied...")

	// Check both: Is total $0? OR does credit chip exist?
	totalIsZero := a.checkIfTotalIsZero()

	// Also check if credit chip exists (indicates credit was applied before)
	checkChipJS := `() => {
		var chipStack = document.querySelector('.m-chipStack[data-cy-id="__chip-stack"]');
		if (chipStack) {
			var chip = chipStack.querySelector('.a-chip[data-cy-id="chip"]');
			if (chip) {
				return chip.textContent.trim();
			}
		}
		return '';
	}`

	chipResult, _ := a.page.Timeout(2 * time.Second).Eval(checkChipJS)
	hasChip := false
	chipText := ""
	if chipResult != nil {
		chipText = chipResult.Value.Str()
		hasChip = chipText != ""
	}

	a.debugLog("Credit check - Total is zero: %v, Has chip: %v (%s)", totalIsZero, hasChip, chipText)

	if totalIsZero {
		fmt.Println("✓ Store credit already applied - Total is already $0!")
		a.debugLog("Skipping store credit entry, going straight to Proceed button")
	} else if hasChip {
		fmt.Println("✓ Store credit chip detected - credit was already applied!")
		fmt.Printf("   Credit applied: %s\n", chipText)
		a.debugLog("Skipping store credit entry (chip exists), going to Proceed button")
	} else {
		// Total is not $0 and no chip exists, need to apply store credit
		fmt.Println("💰 Applying store credit...")

	// Step 1: Find the store credit input field with retry logic
	// RSI uses data-cy-id="input" within the "Add store credits" section
	// The element loads dynamically, so we wait for it to appear
	a.debugLog("Waiting for store credit input to appear...")

	// We need to find the input within the "Add store credits" section specifically
	// because there's also a "Add a coupon" section with identical selectors

	var creditInput *rod.Element
	maxAttempts := 10  // Try up to 10 times

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			// Randomized delay between 200-600ms to avoid detection
			delayMs := 200 + a.rand.Intn(400)
			a.debugLog("Retry %d/%d - waiting %dms...", attempt-1, maxAttempts-1, delayMs)
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}

		// Use JavaScript to check if the store credit input exists
		checkInputJS := `() => {
			var labels = document.querySelectorAll('[data-cy-id="__label"]');
			for (var i = 0; i < labels.length; i++) {
				if (labels[i].textContent.trim() === 'Add store credits') {
					var summaryUnit = labels[i].closest('.a-summaryUnitBlock');
					if (summaryUnit) {
						var input = summaryUnit.querySelector('input[data-cy-id="input"]');
						if (input && input.offsetParent !== null) {
							return 'found';
						}
					}
				}
			}
			return '';
		}`

		result, err := a.page.Timeout(2 * time.Second).Eval(checkInputJS)
		if err == nil && result != nil && result.Value.Str() == "found" {
			creditInput = &rod.Element{} // Dummy element, we'll use JavaScript for interaction
			fmt.Printf("✓ Found store credit input field (attempt %d)\n", attempt)
			break
		}
	}

	if creditInput == nil {
		fmt.Println("⚠  Store credit input field not found")
		a.waitForUser("Store credit input not found. Please manually enter amount and click Add, then press Enter.")
		return nil
	}

	// Step 2: Detect cart subtotal to determine store credit amount
	var amountToApply string

	// In interactive mode, pause to let user enter amount manually
	if a.config.Interactive {
		fmt.Println("\n⏸  Store credit input found!")
		fmt.Println("Please manually:")
		fmt.Println("  1. Look at the browser window")
		fmt.Println("  2. Enter the store credit amount you want to use")
		fmt.Println("  3. Press Enter here when you've entered the amount (don't click Add yet)")

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Press Enter when amount is entered... ")
		reader.ReadString('\n')
	} else {
		// In non-interactive mode, detect the cart subtotal
		subtotal, err := a.detectCartSubtotal()
		if err == nil && subtotal != "" {
			amountToApply = subtotal
			fmt.Printf("💰 Using cart subtotal: $%s\n", amountToApply)
		} else {
			a.debugLog("Failed to detect subtotal: %v", err)
			// Fallback: use a large amount that will be capped
			amountToApply = "999999"
			fmt.Println("⚠  No subtotal detected, using fallback amount")
		}

		// Use JavaScript to find the input and get its selector for Rod
		findInputJS := `() => {
			var labels = document.querySelectorAll('[data-cy-id="__label"]');
			for (var i = 0; i < labels.length; i++) {
				if (labels[i].textContent.trim() === 'Add store credits') {
					var summaryUnit = labels[i].closest('.a-summaryUnitBlock');
					if (summaryUnit) {
						var input = summaryUnit.querySelector('input[data-cy-id="input"]');
						if (input) {
							// Mark it with a unique attribute so we can find it
							input.setAttribute('data-store-credit-input', 'true');
							return 'found';
						}
					}
				}
			}
			return '';
		}`

		result, err := a.page.Timeout(5 * time.Second).Eval(findInputJS)
		if err != nil || result == nil || result.Value.Str() != "found" {
			fmt.Printf("⚠  Failed to locate input: %v\n", err)
			if a.config.Interactive {
				a.waitForUser("Failed to enter amount. Please do it manually and press Enter.")
			} else {
				return fmt.Errorf("failed to locate store credit input: %w", err)
			}
		} else {
			// Now find the element with Rod and use keyboard input
			storeInput, inputErr := a.page.Timeout(5 * time.Second).Element(`input[data-store-credit-input="true"]`)
			if inputErr != nil {
				fmt.Printf("⚠  Failed to get input element: %v\n", inputErr)
				if a.config.Interactive {
					a.waitForUser("Failed to enter amount. Please do it manually and press Enter.")
				} else {
					return fmt.Errorf("failed to get store credit input element: %w", inputErr)
				}
			} else {
				// Focus and select using JavaScript (avoids pointer-events:none issue)
				focusJS := `() => { this.focus(); this.select(); }`
				_, focusErr := storeInput.Timeout(3 * time.Second).Eval(focusJS)
				if focusErr != nil {
					fmt.Printf("⚠  Failed to focus input: %v\n", focusErr)
				}

				// Small wait for field to be ready
				time.Sleep(200 * time.Millisecond)

				// Type the amount using keyboard input with extended timeout
				a.debugLog("Typing amount into input field...")
				err := storeInput.Timeout(10 * time.Second).Input(amountToApply)
				if err != nil {
					fmt.Printf("⚠  Failed to input amount: %v\n", err)
					if a.config.Interactive {
						a.waitForUser("Failed to enter amount. Please do it manually and press Enter.")
					} else {
						return fmt.Errorf("failed to input store credit amount: %w", err)
					}
				} else {
					fmt.Printf("✓ Entered amount: $%s\n", amountToApply)
				}
			}
		}
	}

	// Step 4: Click the "Add" button in the store credits section using JavaScript
	fmt.Println("🔍 Looking for 'Add' button in store credits section...")
	a.waitForUser("About to click the 'Add' button to apply store credit.")
	a.randomDelay()

	// Use JavaScript to click the Add button specifically in the store credits section
	clickAddJS := `() => {
		var labels = document.querySelectorAll('[data-cy-id="__label"]');
		for (var i = 0; i < labels.length; i++) {
			if (labels[i].textContent.trim() === 'Add store credits') {
				var summaryUnit = labels[i].closest('.a-summaryUnitBlock');
				if (summaryUnit) {
					var button = summaryUnit.querySelector('button[data-cy-id="button"]');
					if (button) {
						button.click();
						return 'success';
					}
				}
			}
		}
		return 'failed';
	}`

	result, err := a.page.Timeout(3 * time.Second).Eval(clickAddJS)
	if err != nil || result == nil || result.Value.Str() != "success" {
		fmt.Printf("⚠  Failed to click Add button: %v\n", err)
		if a.config.Interactive {
			a.waitForUser("Failed to click Add. Please click it manually, then press Enter.")
		} else {
			return fmt.Errorf("failed to click Add button for store credit")
		}
	}

	fmt.Println("✓ Clicked 'Add' button")

	// Wait for React/GraphQL to process the store credit and update the UI
	// Optimized for speed - every millisecond counts
	fmt.Println("⏳ Waiting for store credit to be processed...")
	waitTime := time.Duration(600+rand.Intn(300)) * time.Millisecond // 600-900ms
	a.debugLog("Waiting %v for GraphQL processing", waitTime)
	time.Sleep(waitTime)

	a.debugLog("=== Starting $0 verification ===")
	// Step 6: VERIFY the total is EXACTLY $0 (REQUIRED)
	fmt.Println("🔍 Verifying total is $0...")

	// JavaScript to find and extract the total amount (using arrow function format for Rod)
	totalJS := `() => {
		var labels = document.querySelectorAll('p[data-cy-id="__label"]');
		for (var i = 0; i < labels.length; i++) {
			if (labels[i].textContent.indexOf('Total') !== -1) {
				if (labels[i].parentElement) {
					return labels[i].parentElement.textContent.trim();
				}
			}
		}
		return '';
	}`

	totalResult, err := a.page.Timeout(3 * time.Second).Eval(totalJS)
	if err != nil {
		fmt.Printf("❌ ERROR checking total: %v\n", err)
		return fmt.Errorf("failed to check total amount: %w", err)
	}

	totalText := totalResult.Value.Str()
	a.debugLog("Total text extracted: '%s'", totalText)

	if totalText == "" {
		fmt.Println("❌ ERROR: Could not find total on page")
		return fmt.Errorf("could not find total amount on page")
	}

	fmt.Printf("  Total found: %s\n", totalText)

	// Check if total is exactly $0
	// RSI format is "Total0" or "Total$0.00" or similar
	isZero := false
	totalLower := strings.ToLower(totalText)

	// Check for various $0 formats
	if strings.Contains(totalText, "Total0") ||
	   strings.Contains(totalText, "Total $0") ||
	   strings.Contains(totalText, "Total$0") ||
	   (strings.Contains(totalLower, "total") &&
	    (strings.HasSuffix(totalText, "0") || strings.HasSuffix(totalText, "0.00") || strings.HasSuffix(totalText, "$0"))) {

		// Make sure it doesn't contain other digits
		hasOtherDigits := false
		for _, r := range totalText {
			if r >= '1' && r <= '9' {
				hasOtherDigits = true
				break
			}
		}

		if !hasOtherDigits {
			isZero = true
		}
	}

	if !isZero {
		fmt.Printf("❌ ERROR: Total is NOT $0 (found: %s)\n", totalText)
		return fmt.Errorf("total is NOT $0 (found: %s) - insufficient store credit or credit not applied", totalText)
	}

	a.debugLog("=== $0 verification PASSED ===")
	fmt.Println("✓ VERIFIED: Total is $0 - Store credit fully applied!")
	} // end of else block for applying store credit

	// Step 7: Check if we're already at Step 2 (address selection)
	fmt.Println("🔍 Checking current checkout step...")

	// Check for "Step 1" button (indicates we're at Step 2)
	checkStep2JS := `() => {
		var summaryUnits = document.querySelectorAll('.a-summaryUnitBlock');
		for (var i = 0; i < summaryUnits.length; i++) {
			var button = summaryUnits[i].querySelector('button[data-cy-id="button"]');
			if (button) {
				var buttonText = button.querySelector('[data-cy-id="button__text"]');
				if (buttonText && buttonText.textContent.trim() === 'Step 1') {
					return 'found';
				}
			}
		}
		return 'not_found';
	}`

	step2Check, _ := a.page.Timeout(2 * time.Second).Eval(checkStep2JS)
	if step2Check != nil && step2Check.Value.Str() == "found" {
		// We're at Step 2 - check if total is $0
		fmt.Println("📍 Currently at Step 2 (address selection)")

		atStep2Total := a.checkIfTotalIsZero()

		if atStep2Total {
			fmt.Println("✓ At Step 2 with $0 total - continuing forward")
			a.debugLog("Store credit already applied, proceeding with checkout")
		} else {
			// Total is not $0, need to go back to Step 1 to apply credit
			fmt.Println("⏪ At Step 2 but total is NOT $0 - going back to Step 1...")

			clickStep1JS := `() => {
				var summaryUnits = document.querySelectorAll('.a-summaryUnitBlock');
				for (var i = 0; i < summaryUnits.length; i++) {
					var button = summaryUnits[i].querySelector('button[data-cy-id="button"]');
					if (button) {
						var buttonText = button.querySelector('[data-cy-id="button__text"]');
						if (buttonText && buttonText.textContent.trim() === 'Step 1') {
							button.click();
							return 'clicked';
						}
					}
				}
				return 'not_found';
			}`

			clickResult, _ := a.page.Timeout(3 * time.Second).Eval(clickStep1JS)
			if clickResult != nil && clickResult.Value.Str() == "clicked" {
				fmt.Println("✓ Clicked 'Step 1' - returning to cart page")
				time.Sleep(time.Duration(400+rand.Intn(300)) * time.Millisecond) // 400-700ms for page transition
			}
		}
	} else {
		a.debugLog("Currently at Step 1 (cart page)")
	}

	// Step 8: Click the "Proceed to pay" button to proceed
	a.debugLog("=== Looking for Proceed to pay button ===")
	fmt.Println("🔘 Looking for 'Proceed to pay' button...")
	a.randomDelay()

	a.waitForUser("About to click 'Proceed to pay' button to proceed to checkout.")

	// Try JavaScript click first (more reliable for React apps)
	fmt.Println("🖱  Attempting to click 'Proceed to pay' button...")

	clickProceedJS := `() => {
		var button = document.querySelector('button[data-cy-id="__place-order-button"]');
		if (button && !button.disabled) {
			// Scroll into view first
			button.scrollIntoView({behavior: 'instant', block: 'center'});
			// Direct click (synchronous)
			button.click();
			return 'clicked';
		}
		return 'not_found';
	}`

	maxProceedAttempts := 5
	var clickSuccess bool

	for attempt := 1; attempt <= maxProceedAttempts; attempt++ {
		a.debugLog("Proceed button click attempt %d/%d...", attempt, maxProceedAttempts)

		result, err := a.page.Timeout(5 * time.Second).Eval(clickProceedJS)
		if err == nil && result != nil && result.Value.Str() == "clicked" {
			fmt.Println("✓ Clicked 'Proceed to pay' button (JavaScript click)")
			clickSuccess = true
			break
		}

		a.debugLog("JavaScript click failed on attempt %d, retrying...", attempt)

		if attempt < maxProceedAttempts {
			time.Sleep(time.Duration(500+rand.Intn(300)) * time.Millisecond)
		}
	}

	if !clickSuccess {
		fmt.Println("❌ ERROR: Failed to click 'Proceed to pay' button after multiple attempts")
		return fmt.Errorf("could not click Proceed to pay button")
	}

	// Wait for modal to appear (reduced from 2s + 10s to 1s + shorter load wait)
	fmt.Println("⏳ Waiting for modal to appear...")
	a.debugLog("Waiting for page transition after Proceed click...")

	// Variable wait: 800-1200ms
	modalWait := time.Duration(800+rand.Intn(400)) * time.Millisecond
	a.debugLog("Waiting %v for modal", modalWait)
	time.Sleep(modalWait)

	a.debugLog("=== applyStoreCredit() completed successfully ===")
	return nil
}

// acceptTerms handles the modal after clicking "Proceed to pay"
// It clicks "Jump to bottom" and checks the agreement checkbox
func (a *Automation) acceptTerms() error {
	a.debugLog("=== ENTERED acceptTerms() ===")
	fmt.Println("📋 Handling disclaimer modal...")

	// Wait for modal to appear after clicking "Proceed to pay" (already waited in applyStoreCredit)
	// Additional small wait for modal content to render - variable 100-200ms
	modalRenderWait := time.Duration(100+rand.Intn(100)) * time.Millisecond
	a.debugLog("Waiting %v for modal content", modalRenderWait)
	time.Sleep(modalRenderWait)

	// Step 1: Click "Jump to bottom" button in the modal
	fmt.Println("🔽 Looking for 'Jump to bottom' button...")
	a.randomDelay()

	jumpJS := `() => {
		// Find the jumpToButton div and click the button inside it
		var jumpDiv = document.querySelector('.jumpToButton');
		if (jumpDiv) {
			var button = jumpDiv.querySelector('button[data-cy-id="button"]');
			if (button) {
				button.click();
				return 'success';
			}
		}
		return 'not_found';
	}`

	a.debugLog("Executing jump to bottom click...")
	jumpResult, jumpErr := a.page.Timeout(3 * time.Second).Eval(jumpJS)
	if jumpErr != nil {
		fmt.Printf("⚠  Could not find 'Jump to bottom' button: %v\n", jumpErr)
		// Continue anyway - maybe it's not needed
	} else if jumpResult.Value.Str() == "success" {
		fmt.Println("✓ Clicked 'Jump to bottom' button")
		// Wait for scroll animation - variable 150-250ms
		scrollWait := time.Duration(150+rand.Intn(100)) * time.Millisecond
		a.debugLog("Waiting %v for scroll", scrollWait)
		time.Sleep(scrollWait)
	} else {
		fmt.Println("⚠  'Jump to bottom' button not found (might not be needed)")
	}

	// Step 2: Find and check the agreement checkbox
	fmt.Println("📝 Looking for agreement checkbox...")
	a.randomDelay()

	// Use JavaScript to find and click the checkbox using the exact selector from RSI
	checkJS := `() => {
		// Find checkbox by data-cy-id
		var checkbox = document.querySelector('input[data-cy-id="checkbox__input"]');
		if (checkbox && !checkbox.checked) {
			checkbox.click();
			return 'success';
		} else if (checkbox && checkbox.checked) {
			return 'already_checked';
		}
		return 'not_found';
	}`

	a.debugLog("Looking for checkbox with data-cy-id='checkbox__input'...")

	// Retry a few times in case the modal content is still loading
	var checkResult *proto.RuntimeRemoteObject
	var checkErr error
	maxAttempts := 10

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		a.debugLog("Checkbox attempt %d/%d...", attempt, maxAttempts)
		checkResult, checkErr = a.page.Timeout(2 * time.Second).Eval(checkJS)
		if checkErr == nil && checkResult != nil {
			resultStr := checkResult.Value.Str()
			a.debugLog("Checkbox result: %s", resultStr)
			if resultStr == "success" || resultStr == "already_checked" {
				break
			}
		}
		if attempt < maxAttempts {
			time.Sleep(100 * time.Millisecond)
		}
	}

	if checkErr != nil {
		fmt.Printf("❌ ERROR: Could not find checkbox: %v\n", checkErr)
		return fmt.Errorf("failed to find agreement checkbox")
	}

	resultStr := checkResult.Value.Str()
	if resultStr == "not_found" {
		fmt.Println("❌ ERROR: Agreement checkbox not found in modal")
		return fmt.Errorf("could not find agreement checkbox in modal")
	} else if resultStr == "already_checked" {
		fmt.Println("✓ Agreement checkbox already checked")
	} else {
		fmt.Println("✓ Checked agreement checkbox")
	}

	// Wait a moment for the "I agree" button to become enabled (React state update)
	// Variable wait: 150-250ms
	fmt.Println("⏳ Waiting for 'I agree' button to enable...")
	enableWait := time.Duration(150+rand.Intn(100)) * time.Millisecond
	a.debugLog("Waiting %v for button to enable", enableWait)
	time.Sleep(enableWait)

	return nil
}

// finalizeCheckout clicks the final "I agree" button in the modal
func (a *Automation) finalizeCheckout() error {
	a.debugLog("=== ENTERED finalizeCheckout() ===")
	fmt.Println("🎯 Looking for 'I agree' button...")

	a.randomDelay()

	// Look for the "I agree" button in the modal (using the exact selector from RSI)
	// This button is enabled after checking the agreement checkbox
	agreeJS := `() => {
		// Find the primary button in modal footer by data-cy-id
		var button = document.querySelector('button[data-cy-id="modal_footer__primary_button"]');
		if (button) {
			var isDisabled = button.disabled || button.classList.contains('-disabled');
			return {
				found: true,
				enabled: !isDisabled,
				text: button.textContent.trim()
			};
		}
		return {found: false, enabled: false};
	}`

	a.debugLog("Looking for 'I agree' button with data-cy-id='modal_footer__primary_button'...")
	agreeResult, agreeErr := a.page.Timeout(3 * time.Second).Eval(agreeJS)
	if agreeErr != nil {
		fmt.Printf("❌ ERROR: Could not check for 'I agree' button: %v\n", agreeErr)
		return fmt.Errorf("failed to find 'I agree' button: %w", agreeErr)
	}

	// Parse the result
	resultMap := agreeResult.Value.Map()
	found := resultMap["found"]
	if !found.Bool() {
		fmt.Println("❌ ERROR: 'I agree' button not found in modal")
		return fmt.Errorf("could not find 'I agree' button")
	}

	enabled := resultMap["enabled"]
	buttonText := resultMap["text"].Str()

	if !enabled.Bool() {
		fmt.Printf("⚠️  WARNING: 'I agree' button found but is disabled (text: '%s')\n", buttonText)
		fmt.Println("⚠️  This might mean the checkbox didn't register. Retrying...")

		// Try clicking the checkbox again
		recheckJS := `() => {
			var checkbox = document.querySelector('input[data-cy-id="checkbox__input"]');
			if (checkbox && !checkbox.checked) {
				checkbox.click();
				return 'clicked';
			}
			return 'already_checked';
		}`
		recheckResult, _ := a.page.Timeout(2 * time.Second).Eval(recheckJS)
		if recheckResult != nil {
			a.debugLog("Recheck result: %s", recheckResult.Value.Str())
		}

		// Wait for button to enable - variable 700-1000ms
		retryWait := time.Duration(700+rand.Intn(300)) * time.Millisecond
		a.debugLog("Waiting %v for retry", retryWait)
		time.Sleep(retryWait)

		// Check again
		agreeResult2, _ := a.page.Timeout(2 * time.Second).Eval(agreeJS)
		if agreeResult2 != nil {
			resultMap2 := agreeResult2.Value.Map()
			if !resultMap2["enabled"].Bool() {
				return fmt.Errorf("'I agree' button is still disabled after retry")
			}
		}
	}

	fmt.Printf("✓ Found enabled 'I agree' button: '%s'\n", buttonText)

	// Check if dry-run mode
	if a.config.DryRun {
		fmt.Println()
		fmt.Println("🧪 DRY RUN MODE - Stopping before clicking 'I agree'")
		fmt.Println("✓ Would click: 'I agree' button")
		fmt.Println("✓ Checkout flow verified successfully!")
		return nil
	}

	// FINAL STEP: Click "I agree" to complete the purchase
	fmt.Println()
	fmt.Println("⚠️  FINAL STEP: About to click 'I agree' and complete purchase!")
	a.waitForUser("This will submit the order. Press Enter to continue.")
	a.randomDelay()

	// Click using JavaScript with the exact selector
	clickAgreeJS := `() => {
		var button = document.querySelector('button[data-cy-id="modal_footer__primary_button"]');
		if (button && !button.disabled && !button.classList.contains('-disabled')) {
			button.click();
			return 'success';
		}
		return 'not_found_or_disabled';
	}`

	a.debugLog("Clicking 'I agree' button...")
	clickResult, clickErr := a.page.Timeout(3 * time.Second).Eval(clickAgreeJS)
	if clickErr != nil {
		fmt.Printf("❌ ERROR: Failed to click 'I agree' button: %v\n", clickErr)
		return fmt.Errorf("failed to click 'I agree' button: %w", clickErr)
	}

	if clickResult.Value.Str() != "success" {
		fmt.Printf("❌ ERROR: 'I agree' button not found or disabled: %s\n", clickResult.Value.Str())
		return fmt.Errorf("failed to click 'I agree' button - button not found or disabled")
	}

	fmt.Println("✓ Clicked 'I agree' button")
	fmt.Println("✓ Order submitted!")
	return nil
}

// handleDisclaimerModal handles the RSI disclaimer modal that appears before final purchase
func (a *Automation) handleDisclaimerModal() error {
	fmt.Println("📋 Handling disclaimer modal...")

	// Step 1: Check if modal is present
	modalSelectors := []string{
		"dialog[data-cy-id='modal']",
		".c-modal",
		"dialog[aria-modal='true']",
	}

	var modalPresent bool
	for _, selector := range modalSelectors {
		elem, err := a.page.Element(selector)
		if err == nil && elem != nil {
			modalPresent = true
			fmt.Printf("✓ Found disclaimer modal: %s\n", selector)
			break
		}
	}

	if !modalPresent {
		fmt.Println("⚠  No disclaimer modal found - order may have already been placed")
		return nil
	}

	a.waitForUser("Disclaimer modal appeared. About to handle modal steps.")
	a.randomDelay()

	// Step 2: Click "Jump to bottom" button
	fmt.Println("🔽 Looking for 'Jump to bottom' button...")

	jumpSelectors := []string{
		"button:contains('Jump to bottom')",
		".jumpToButton button",
		"button[data-cy-id='button']:contains('Jump')",
	}

	var jumpButton *rod.Element
	for _, selector := range jumpSelectors {
		a.debugLog("Trying jump to bottom selector: %s", selector)
		elem, err := a.page.Element(selector)
		if err == nil && elem != nil {
			text, _ := elem.Text()
			if strings.Contains(toLower(text), "jump") {
				jumpButton = elem
				fmt.Printf("✓ Found 'Jump to bottom' button: %s\n", selector)
				break
			}
		}
	}

	if jumpButton == nil {
		fmt.Println("⚠  'Jump to bottom' button not found, trying to proceed anyway...")
	} else {
		a.randomDelay()
		if err := a.reliableClick(jumpButton); err != nil {
			fmt.Printf("⚠  Failed to click 'Jump to bottom': %v\n", err)
		} else {
			fmt.Println("✓ Clicked 'Jump to bottom'")
			a.waitForUser("Scrolled to bottom of disclaimer. About to find checkbox.")
		}
	}

	// Step 3: Find and check the "I agree to ToS and Privacy Policy" checkbox
	fmt.Println("☑️  Looking for agreement checkbox...")

	a.randomDelay()

	checkboxSelectors := []string{
		"input[data-cy-id='checkbox__input']",
		".a-checkbox__input",
		"input[type='checkbox']",
	}

	var checkbox *rod.Element
	for _, selector := range checkboxSelectors {
		a.debugLog("Trying checkbox selector: %s", selector)
		elem, err := a.page.Element(selector)
		if err == nil && elem != nil {
			checkbox = elem
			fmt.Printf("✓ Found agreement checkbox: %s\n", selector)
			break
		}
	}

	if checkbox == nil {
		return fmt.Errorf("could not find agreement checkbox in modal")
	}

	// Check if already checked
	checked, err := checkbox.Property("checked")
	if err == nil && checked.Bool() {
		fmt.Println("✓ Checkbox already checked")
	} else {
		a.randomDelay()
		if err := a.reliableClick(checkbox); err != nil {
			return fmt.Errorf("failed to check agreement checkbox: %w", err)
		}
		fmt.Println("✓ Checked agreement checkbox")
		a.waitForUser("Agreement checkbox checked. 'I agree' button should now be enabled.")
	}

	// Step 4: Click the "I agree" button (becomes enabled after checkbox is checked)
	fmt.Println("✅ Looking for 'I agree' button...")

	a.randomDelay()

	agreeSelectors := []string{
		"button[data-cy-id='modal_footer__primary_button']",
		".m-modalFooter__primaryButton",
		"button:contains('I agree')",
	}

	var agreeButton *rod.Element
	for _, selector := range agreeSelectors {
		a.debugLog("Trying I agree button selector: %s", selector)
		elem, err := a.page.Element(selector)
		if err == nil && elem != nil {
			// Check if button is disabled
			disabled, _ := elem.Property("disabled")

			if !disabled.Bool() {
				agreeButton = elem
				text, _ := elem.Text()
				fmt.Printf("✓ Found 'I agree' button: %s (text: '%s')\n", selector, text)
				break
			} else {
				a.debugLog("Button found but is disabled, skipping...")
			}
		}
	}

	if agreeButton == nil {
		return fmt.Errorf("could not find enabled 'I agree' button in modal")
	}

	// FINAL STEP: Click "I agree" to complete the purchase
	fmt.Println("\n⚠️  FINAL STEP: 'I agree' button found and ready to click...")

	// Check if dry run mode is enabled
	if a.config.DryRun {
		fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
		fmt.Println("║              DRY RUN MODE - COMPLETE!                     ║")
		fmt.Println("╚═══════════════════════════════════════════════════════════╝")
		fmt.Println()
		fmt.Println("✓ Successfully navigated through entire checkout flow:")
		fmt.Println("  ✓ Added item to cart")
		fmt.Println("  ✓ Went to cart")
		fmt.Println("  ✓ Proceeded to checkout")
		fmt.Println("  ✓ Applied store credit")
		fmt.Println("  ✓ Verified total is $0")
		fmt.Println("  ✓ Clicked Continue")
		fmt.Println("  ✓ Accepted terms")
		fmt.Println("  ✓ Clicked 'Proceed to pay'")
		fmt.Println("  ✓ Handled disclaimer modal")
		fmt.Println("  ✓ Clicked 'Jump to bottom'")
		fmt.Println("  ✓ Checked agreement checkbox")
		fmt.Println("  ✓ Found 'I agree' button")
		fmt.Println()
		fmt.Println("🛑 STOPPED before final 'I agree' click (would place order)")
		fmt.Println()
		fmt.Println("In LIVE mode, clicking 'I agree' will complete the purchase!")
		a.waitForUser("Dry run complete! You can inspect the disclaimer modal.")
		return nil
	}

	// LIVE MODE: Actually click the button
	fmt.Println("🔴 LIVE MODE: Clicking 'I agree' to complete purchase...")
	a.waitForUser("This is the FINAL click! Order will be placed. Press Enter to proceed.")

	a.randomDelay()

	if err := a.reliableClick(agreeButton); err != nil {
		return fmt.Errorf("failed to click 'I agree' button: %w", err)
	}

	fmt.Println("✓ Clicked 'I agree' button")

	return nil
}

// performCheckout executes a single checkout attempt
func (a *Automation) performCheckout() error {
	// Add to cart (skip if already in cart from config or previous successful attempt)
	if a.config.SkipAddToCart || a.itemInCart {
		if a.itemInCart {
			fmt.Println("⏭️  Skipping add to cart (item already in cart from previous attempt)")
		} else {
			fmt.Println("⏭️  Skipping add to cart (item already in cart)")
		}
		// Note: We're already at cart page from RunCheckout() or retry navigation
	} else {
		// First attempt - add item to cart
		if err := a.addToCart(); err != nil {
			return fmt.Errorf("failed to add to cart: %w", err)
		}

		// Go to cart (only when we just added the item)
		if err := a.goToCart(); err != nil {
			return fmt.Errorf("failed to navigate to cart: %w", err)
		}
	}

	// Apply store credit on cart page
	// This includes entering credit, clicking Add, verifying $0, and clicking Continue
	a.debugLog("=== CALLING applyStoreCredit() ===")
	if err := a.applyStoreCredit(); err != nil {
		fmt.Printf("❌ ERROR in applyStoreCredit(): %v\n", err)
		return fmt.Errorf("failed to apply store credit: %w", err)
	}
	a.debugLog("=== applyStoreCredit() RETURNED SUCCESS ===")

	// Accept terms
	a.debugLog("=== CALLING acceptTerms() ===")
	if err := a.acceptTerms(); err != nil {
		fmt.Printf("❌ ERROR in acceptTerms(): %v\n", err)
		return fmt.Errorf("failed to accept terms: %w", err)
	}
	a.debugLog("=== acceptTerms() RETURNED SUCCESS ===")

	// Finalize checkout
	a.debugLog("=== CALLING finalizeCheckout() ===")
	if err := a.finalizeCheckout(); err != nil {
		fmt.Printf("❌ ERROR in finalizeCheckout(): %v\n", err)
		return fmt.Errorf("failed to finalize checkout: %w", err)
	}
	a.debugLog("=== finalizeCheckout() RETURNED SUCCESS ===")


	return nil
}

// RunCheckout executes the full checkout process with retry logic
func (a *Automation) RunCheckout() error {
	// Setup browser
	if err := a.setupBrowser(); err != nil {
		return err
	}

	// Wait for user to log in and confirm ready
	if err := a.waitForLogin(); err != nil {
		return err
	}

	// Navigate to item or cart depending on mode
	if a.config.SkipAddToCart {
		// Skip-cart mode: go directly to cart (item already there)
		fmt.Println("⏭️  Skip-cart mode: Going directly to cart page")
		if err := a.goToCart(); err != nil {
			return fmt.Errorf("failed to navigate to cart: %w", err)
		}
	} else {
		// Normal mode: navigate to item page first
		if err := a.navigateToItem(); err != nil {
			return err
		}
	}

	// Attempt checkout with retries
	maxAttempts := a.config.MaxRetries + 1 // +1 for initial attempt
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Check if browser is still alive before each attempt
		a.checkBrowserOrExit()

		if attempt > 1 {
			fmt.Printf("\n╔═══════════════════════════════════════════════════════════╗\n")
			fmt.Printf("║  RETRY ATTEMPT %d of %d                                   \n", attempt-1, a.config.MaxRetries)
			fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n")

			// Wait before retry with random delay
			a.retryDelay()

			// Check browser again after delay
			a.checkBrowserOrExit()

			// Navigate back to start point depending on mode and cart status
			if a.config.SkipAddToCart || a.itemInCart {
				// Item is in cart (either from config or previous attempt) - go to cart page
				if a.itemInCart {
					fmt.Println("🔄 Resetting: Navigating back to cart page (item already added)...")
				} else {
					fmt.Println("🔄 Resetting: Navigating back to cart page...")
				}

				// Navigate to cart page
				navErr := a.page.Navigate("https://robertsspaceindustries.com/en/store/pledge/cart")
				if navErr != nil {
					fmt.Printf("⚠  Failed to navigate back to cart: %v\n", navErr)
					// Continue anyway, try to recover
				} else {
					// Wait for page load with explicit timeout
					loadErr := a.page.Timeout(time.Duration(a.config.PageLoadTimeout) * time.Second).WaitLoad()
					if loadErr != nil {
						a.debugLog("Page load timeout (continuing anyway): %v", loadErr)
					}
					time.Sleep(time.Duration(300+rand.Intn(200)) * time.Millisecond) // 300-500ms for dynamic content
				}
			} else {
				// First attempt failed before adding to cart - go back to item page
				fmt.Println("🔄 Resetting: Navigating back to item page...")

				// Navigate to item page
				navErr := a.page.Navigate(a.config.ItemURL)
				if navErr != nil {
					fmt.Printf("⚠  Failed to navigate back to item: %v\n", navErr)
					// Continue anyway, try to recover
				} else {
					loadErr := a.page.Timeout(time.Duration(a.config.PageLoadTimeout) * time.Second).WaitLoad()
					if loadErr != nil {
						a.debugLog("Page load timeout (continuing anyway): %v", loadErr)
					}
				}
			}
		} else {
			fmt.Printf("╔═══════════════════════════════════════════════════════════╗\n")
			fmt.Printf("║  CHECKOUT ATTEMPT 1 of %d                                 \n", maxAttempts)
			fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n\n")
		}

		// Attempt the checkout
		err := a.performCheckout()

		if err == nil {
			// Success!
			if attempt > 1 {
				fmt.Printf("\n🎉 SUCCESS on retry attempt %d!\n", attempt-1)
			}
			return nil
		}

		// Checkout failed
		lastErr = err
		fmt.Printf("\n❌ Attempt %d failed: %v\n", attempt, err)

		// If this was the last attempt, don't retry
		if attempt >= maxAttempts {
			fmt.Printf("\n⛔ All %d attempts exhausted.\n", maxAttempts)
			break
		}

		// Check if error is retryable
		errStr := strings.ToLower(err.Error())
		isRetryable := true

		// Some errors should not be retried
		if strings.Contains(errStr, "context canceled") ||
		   strings.Contains(errStr, "browser closed") {
			fmt.Println("⚠  Non-retryable error detected. Stopping retries.")
			isRetryable = false
		}

		if !isRetryable {
			break
		}

		fmt.Printf("♻️  Will retry in a moment... (%d attempts remaining)\n", maxAttempts-attempt)
	}

	return fmt.Errorf("checkout failed after %d attempts: %w", maxAttempts, lastErr)
}

// contains checks if a string contains any of the substrings (case-insensitive)
func contains(s string, substrs ...string) bool {
	s = toLower(s)
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == toLower(substr) {
					return true
				}
			}
		}
	}
	return false
}

// toLower converts a string to lowercase
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
