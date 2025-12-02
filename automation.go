package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

type Automation struct {
	config       *Config
	browser      *rod.Browser
	page         *rod.Page
	launcher     *launcher.Launcher
	rand         *rand.Rand
	stopChan     chan bool
	itemInCart   bool
	cachedSKU    string // SKU extracted and validated before login
}

func NewAutomation(config *Config) *Automation {
	return &Automation{
		config:   config,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		stopChan: make(chan bool, 1),
	}
}

func (a *Automation) Close() {
	select {
	case a.stopChan <- true:
	default:
	}

	fmt.Println(T("cleaning_up"))

	if a.page != nil {
		a.page.Close()
	}

	if a.browser != nil {
		a.browser.Close()
	}

	if a.launcher != nil {
		a.launcher.Cleanup()
	}

	fmt.Println(T("browser_destroyed"))
}

func (a *Automation) isBrowserAlive() bool {
	if a.browser == nil {
		return false
	}

	_, err := a.browser.Version()
	if err != nil {
		a.debugLog("Browser version check failed: %v", err)
		return false
	}

	if a.page != nil {
		_, err := a.page.Info()
		if err != nil {
			a.debugLog("Page info check failed: %v", err)
			return false
		}
	}

	return true
}

func (a *Automation) checkBrowserOrExit() {
	if !a.isBrowserAlive() {
		fmt.Println(T("browser_closed_by_user"))
		fmt.Println(T("shutting_down"))
		os.Exit(0)
	}
}

func (a *Automation) watchBrowser() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopChan:
			return
		case <-ticker.C:
			a.checkBrowserOrExit()
		}
	}
}

func (a *Automation) randomDelay() {
	min := a.config.MinDelayBetween
	max := a.config.MaxDelayBetween
	duration := min + a.rand.Float64()*(max-min)

	if !a.config.DebugMode {
		fmt.Printf(T("waiting_seconds")+"\n", duration)
	}
	time.Sleep(time.Duration(duration * float64(time.Second)))
}

func (a *Automation) getTimeout() time.Duration {
	timeoutMs := 200 + a.rand.Intn(180) // Random 200-380ms
	return time.Duration(timeoutMs) * time.Millisecond
}

func (a *Automation) getClickTimeout() time.Duration {
	timeoutMs := 700 + a.rand.Intn(400) // Random 700-1100ms
	return time.Duration(timeoutMs) * time.Millisecond
}


func (a *Automation) debugLog(format string, args ...interface{}) {
	if a.config.DebugMode {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

func (a *Automation) setupBrowser() error {
	fmt.Println(T("browser_launching"))

	// Disable leakless mode on Windows to prevent deadlock
	// See: https://github.com/go-rod/rod/issues/853
	useLeakless := runtime.GOOS != "windows"

	// Try to find system Chrome first (avoids download and permission issues)
	chromePath, chromeExists := launcher.LookPath()

	// Build launcher with proper configuration order
	a.launcher = launcher.New().
		Leakless(useLeakless).
		Headless(a.config.Headless)

	// Set custom user data dir to avoid conflicts with running Chrome
	// IMPORTANT: Must set this before Bin() to ensure it's applied
	if a.config.BrowserProfilePath != "" {
		a.launcher = a.launcher.UserDataDir(a.config.BrowserProfilePath)
		a.debugLog(T("browser_profile_path_set", a.config.BrowserProfilePath))
	}

	if chromeExists {
		a.launcher = a.launcher.Bin(chromePath)
		fmt.Println(T("browser_using_system_chrome"))
		a.debugLog(T("browser_chrome_path_set", chromePath))
	} else {
		fmt.Println(T("browser_chrome_not_found"))
		// Will use automatic Chromium download (default behavior)
	}

	if runtime.GOOS == "windows" {
		fmt.Println(T("windows_leakless_disabled"))
	}

	url, err := a.launcher.Launch()
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "Opening in existing browser session") ||
			strings.Contains(errMsg, "ProcessSingleton") ||
			strings.Contains(errMsg, "SingletonLock") {
			fmt.Println(T("error_chrome_already_running_header"))
			fmt.Println(T("error_chrome_fix_instructions"))
			fmt.Println(T("error_chrome_close_all"))
			if runtime.GOOS == "darwin" {
				fmt.Println(T("error_chrome_mac_activity_monitor"))
				fmt.Println(T("error_chrome_mac_killall"))
			} else if runtime.GOOS == "windows" {
				fmt.Println(T("error_chrome_windows_task_manager"))
				fmt.Println(T("error_chrome_windows_end_processes"))
			}
			fmt.Println(T("error_chrome_try_again"))
			return fmt.Errorf(T("error_chrome_already_running"))
		}

		// Check for permission/access errors during download
		if strings.Contains(errMsg, "Access is denied") || strings.Contains(errMsg, "permission denied") {
			fmt.Println(T("error_browser_download_permission"))
			fmt.Println(T("error_browser_download_fix"))
			fmt.Println(T("error_browser_download_close_chrome"))
			if runtime.GOOS == "windows" {
				fmt.Println(T("error_browser_download_delete_windows"))
				fmt.Println(T("error_browser_download_exclusion_windows"))
			} else {
				fmt.Println(T("error_browser_download_delete_mac"))
			}
			fmt.Println(T("error_browser_download_try_again"))
			fmt.Println(T("error_browser_download_alternative"))
			fmt.Println(T("error_browser_download_chrome_url"))
			return fmt.Errorf(T("error_browser_setup_failed"), err)
		}

		return fmt.Errorf("failed to launch browser: %w", err)
	}

	a.browser = rod.New().ControlURL(url).MustConnect()


	go a.watchBrowser()
	a.debugLog(T("browser_watcher_started"))

	fmt.Println(T("browser_launched"))
	return nil
}

func (a *Automation) waitForLogin() error {
	fmt.Println(T("opening_for_login"))

	// ALWAYS open homepage first for login (not the item URL)
	homepageURL := "https://robertsspaceindustries.com"
	fmt.Printf(T("loading_homepage")+"\n", homepageURL)

	var err error
	a.page, err = stealth.Page(a.browser)
	if err != nil {
		return fmt.Errorf("failed to create stealth page: %w", err)
	}

	a.debugLog("✓ Stealth mode enabled (anti-bot detection)")

	err = a.page.Navigate(homepageURL)
	if err != nil {
		return fmt.Errorf("failed to navigate: %w", err)
	}

	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	err = a.page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: userAgent,
	})
	if err != nil {
		a.debugLog("Warning: Failed to set User-Agent: %v", err)
	} else {
		a.debugLog("User-Agent set to Chrome")
	}

	if err := a.page.WaitLoad(); err != nil {
		return fmt.Errorf("page failed to load: %w", err)
	}

	fmt.Println(T("browser_configured"))

	// Prompt user to login BEFORE trying to load item page
	fmt.Println()
	fmt.Println(T("login_required_header"))
	fmt.Println()
	fmt.Println(T("login_instructions"))
	fmt.Println()
	fmt.Print(T("login_prompt"))

	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		if input == '\n' || input == '\r' {
			fmt.Println()
			fmt.Println(T("user_confirmed_ready"))
			break
		}

		if input == 27 {
			fmt.Println()
			fmt.Println(T("user_requested_exit"))
			return fmt.Errorf("user canceled operation")
		}
	}

	// AFTER login, navigate to item URL and retry until it's available (not 404)
	if a.config.ItemURL != "" {
		fmt.Println()
		fmt.Printf(T("navigating_to_product_page")+"\n", a.config.ItemURL)

		if err := a.navigateToProductPageWithRetry(); err != nil {
			return err
		}

		fmt.Println(T("product_page_loaded"))

		// Extract and validate SKU AFTER successful navigation
		if err := a.extractAndCacheSKU(); err != nil {
			return err
		}
	}

	if a.config.RecaptchaSiteKey != "" {
		fmt.Println(T("recaptcha_preloading"))
		a.preloadRecaptcha()

		fmt.Println(T("building_interaction_history"))
		a.buildInteractionHistory()
	}

	return nil
}

// navigateToProductPageWithRetry retries navigation to item URL until it's available (not 404)
// This is critical for pre-sale scenarios where the product page doesn't exist yet
func (a *Automation) navigateToProductPageWithRetry() error {
	attemptNum := 0
	for {
		attemptNum++

		// Navigate to the product page
		err := a.page.Navigate(a.config.ItemURL)
		if err != nil {
			// Network error - retry after delay
			if attemptNum%10 == 0 || attemptNum <= 3 {
				fmt.Printf("⚠️  Attempt %d: Navigation error - retrying in 2s...\n", attemptNum)
				if attemptNum <= 3 {
					fmt.Printf("   Error: %v\n", err)
				}
			}
			time.Sleep(2 * time.Second)
			continue
		}

		// Wait for page to load
		if err := a.page.WaitLoad(); err != nil {
			if attemptNum%10 == 0 || attemptNum <= 3 {
				fmt.Printf("⚠️  Attempt %d: Page load error - retrying in 2s...\n", attemptNum)
			}
			time.Sleep(2 * time.Second)
			continue
		}

		// Check HTTP status code
		statusCode, err := a.page.Eval(`() => {
			return window.performance?.getEntriesByType?.('navigation')?.[0]?.responseStatus || 200;
		}`)

		if err == nil {
			status := statusCode.Value.Int()

			if status == 404 {
				// 404 - Page doesn't exist yet, keep retrying
				if attemptNum == 1 {
					fmt.Println("⏳ Product page not available yet (404) - waiting for sale to go live...")
					fmt.Println("💡 The app will keep retrying until the page is available")
					fmt.Println()
				}
				if attemptNum%30 == 0 {
					fmt.Printf("   Still waiting... (attempt %d, checking every 2s)\n", attemptNum)
				}
				time.Sleep(2 * time.Second)
				continue
			} else if status >= 400 {
				// Other error status - this might be temporary
				if attemptNum%10 == 0 || attemptNum <= 3 {
					fmt.Printf("⚠️  Attempt %d: HTTP %d error - retrying in 2s...\n", attemptNum, status)
				}
				time.Sleep(2 * time.Second)
				continue
			}

			a.debugLog("Product page HTTP status: %d", status)
		}

		// Check if page has SKU data (validates it's actually a product page)
		hasSKUData, err := a.page.Eval(`() => {
			const skuDetailDiv = document.querySelector('[data-rsi-component="SkuDetailPage"]');
			return skuDetailDiv !== null;
		}`)

		if err == nil && !hasSKUData.Value.Bool() {
			// No SKU data - might be loading or wrong page
			if attemptNum%10 == 0 || attemptNum <= 3 {
				fmt.Printf("⚠️  Attempt %d: Page loaded but no SKU data found - retrying in 2s...\n", attemptNum)
			}
			time.Sleep(2 * time.Second)
			continue
		}

		// Success! Page is loaded and valid
		if attemptNum > 1 {
			fmt.Printf("✓ Product page is now available! (took %d attempts)\n", attemptNum)
		}
		return nil
	}
}

func (a *Automation) buildInteractionHistory() {
	// OPTIMIZED: Reduced from ~3 seconds to ~200ms while maintaining variety
	for round := 0; round < 2; round++ { // Reduced from 3 to 2 rounds
		movements := 2 + a.rand.Intn(3) // Reduced from 4-8 to 2-4 movements
		for i := 0; i < movements; i++ {
			x := a.rand.Intn(1200) + 100
			y := a.rand.Intn(700) + 100

			a.page.Eval(fmt.Sprintf(`() => {
				var event = new MouseEvent('mousemove', {
					view: window,
					bubbles: true,
					cancelable: true,
					clientX: %d,
					clientY: %d
				});
				document.dispatchEvent(event);
			}`, x, y))

			time.Sleep(time.Duration(5+a.rand.Intn(10)) * time.Millisecond) // Reduced from 40-120ms to 5-15ms
		}

		if round < 1 { // Only scroll once
			scrollAmount := (a.rand.Intn(3) - 1) * (100 + a.rand.Intn(150))
			a.page.Eval(fmt.Sprintf(`() => window.scrollBy(0, %d)`, scrollAmount))
		}

		// Simplified click - just do one quick click
		if a.rand.Float64() < 0.5 {
			clickX := a.rand.Intn(1000) + 200
			clickY := a.rand.Intn(600) + 200

			a.page.Eval(fmt.Sprintf(`() => {
				var down = new MouseEvent('mousedown', {
					view: window,
					bubbles: true,
					cancelable: true,
					clientX: %d,
					clientY: %d
				});
				document.dispatchEvent(down);
			}`, clickX, clickY))

			time.Sleep(time.Duration(10+a.rand.Intn(20)) * time.Millisecond) // Reduced from 60-140ms to 10-30ms

			a.page.Eval(fmt.Sprintf(`() => {
				var up = new MouseEvent('mouseup', {
					view: window,
					bubbles: true,
					cancelable: true,
					clientX: %d,
					clientY: %d
				});
				document.dispatchEvent(up);
			}`, clickX, clickY))
		}

		time.Sleep(time.Duration(30+a.rand.Intn(50)) * time.Millisecond) // Reduced from 300-800ms to 30-80ms
	}

	a.debugLog(T("interaction_history_built"))
	fmt.Println(T("interaction_history_complete"))
}

func (a *Automation) preloadRecaptcha() {
	checkExisting, err := a.page.Eval(`() => typeof grecaptcha !== 'undefined' && typeof grecaptcha.enterprise !== 'undefined'`)
	if err == nil && checkExisting.Value.Bool() {
		fmt.Println(T("recaptcha_already_loaded"))
		a.debugLog(T("debug_recaptcha_present"))

		actualKey, err := a.page.Eval(`() => {
			const scripts = document.querySelectorAll('script[src*="recaptcha"]');
			for (let script of scripts) {
				const match = script.src.match(/render=([^&]+)/);
				if (match) return match[1];
			}
			return null;
		}`)
		if err == nil && actualKey != nil && actualKey.Value.Str() != "" {
			detectedKey := actualKey.Value.Str()
			a.debugLog(T("recaptcha_detected_key"), detectedKey)
			if detectedKey != a.config.RecaptchaSiteKey {
				fmt.Printf(T("recaptcha_key_warning")+"\n", a.config.RecaptchaSiteKey, detectedKey)
			}
		}

		return
	}

	scriptURL := fmt.Sprintf("https://www.google.com/recaptcha/enterprise.js?render=%s", a.config.RecaptchaSiteKey)

	injectScript := fmt.Sprintf(`() => {
		var script = document.createElement('script');
		script.src = '%s';
		script.id = 'specter-recaptcha';
		document.head.appendChild(script);
		return true;
	}`, scriptURL)

	_, err = a.page.Eval(injectScript)
	if err != nil {
		a.debugLog(T("warning_recaptcha_script_inject"), err)
		fmt.Println(T("recaptcha_injection_failed"))
		return
	}

	a.debugLog(T("debug_recaptcha_waiting"))

	// OPTIMIZED: Reduced wait time and check intervals for speed
	maxWait := 20 // 20 checks of 100ms = 2 seconds max instead of 5 seconds
	for i := 0; i < maxWait; i++ {
		time.Sleep(100 * time.Millisecond) // Check every 100ms instead of 1 second

		readyCheck, err := a.page.Eval(`() => typeof grecaptcha !== 'undefined' && typeof grecaptcha.enterprise !== 'undefined'`)
		if err == nil && readyCheck.Value.Bool() {
			fmt.Println(T("recaptcha_ready"))
			a.debugLog(T("recaptcha_loaded_after"), (i+1)*100)
			return
		}
	}

	fmt.Println(T("recaptcha_timeout"))
	a.debugLog(T("recaptcha_timeout_after"))
}

// extractAndCacheSKU extracts the SKU from the current page and caches it
// This is called BEFORE login to validate the page is a valid ship page
// Uses HTTP client instead of browser for speed and simplicity
func (a *Automation) extractAndCacheSKU() error {
	fmt.Println(T("sku_extracting_validating"))

	// Get current URL
	currentURL, err := a.page.Eval(`() => window.location.href`)
	if err != nil || currentURL.Value.Str() == "" {
		return fmt.Errorf(T("sku_could_not_get_url"))
	}
	itemURL := currentURL.Value.Str()

	// Fetch page HTML using HTTP client (much faster than opening browser window)
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", itemURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set User-Agent to mimic browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("page returned HTTP %d: %s", resp.StatusCode, itemURL)
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read page body: %w", err)
	}
	htmlContent := string(bodyBytes)

	// Extract SKU slug using regex
	re := regexp.MustCompile(`"skuSlug":\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(htmlContent)

	var skuSlugStr string
	if len(matches) > 1 {
		skuSlugStr = matches[1]
	}

	// Validate SKU was found
	if skuSlugStr == "" || skuSlugStr == "null" || skuSlugStr == "<nil>" {
		fmt.Println()
		fmt.Println(T("sku_validation_failed_header"))
		fmt.Println()
		fmt.Printf(T("sku_validation_failed_url")+"\n", itemURL)
		fmt.Println(T("sku_validation_failed_reason"))
		fmt.Println()
		fmt.Println(T("sku_validation_failed_fix"))
		fmt.Println(T("sku_validation_failed_step1"))
		fmt.Println(T("sku_validation_failed_step2"))
		fmt.Println()
		return fmt.Errorf("SKU validation failed: page is not a valid ship page")
	}

	// Cache the SKU slug (we'll convert to SKU ID later in fast_checkout)
	a.cachedSKU = skuSlugStr
	fmt.Printf(T("sku_validated_cached")+"\n", skuSlugStr)

	return nil
}

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
