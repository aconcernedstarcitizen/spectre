package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
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
		fmt.Println("\n⚠️  Browser window was closed by user")
		fmt.Println("🛑 Shutting down gracefully...")
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
		fmt.Printf("⏱  Waiting %.2f seconds...\n", duration)
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
	fmt.Println("🚀 Launching browser...")

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
		a.debugLog("Browser profile path: %s", a.config.BrowserProfilePath)
	}

	if chromeExists {
		a.launcher = a.launcher.Bin(chromePath)
		fmt.Println("✓ Using system Chrome browser")
		a.debugLog("Chrome path: %s", chromePath)
	} else {
		fmt.Println("ℹ️  System Chrome not found, will download Chromium")
		// Will use automatic Chromium download (default behavior)
	}

	if runtime.GOOS == "windows" {
		fmt.Println("ℹ️  Running on Windows - leakless mode disabled")
	}

	url, err := a.launcher.Launch()
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "Opening in existing browser session") ||
			strings.Contains(errMsg, "ProcessSingleton") ||
			strings.Contains(errMsg, "SingletonLock") {
			fmt.Println("\n❌ Chrome is already running with the same profile")
			fmt.Println("\n📋 To fix this issue:")
			fmt.Println("   1. Close ALL Chrome/Chromium windows completely")
			if runtime.GOOS == "darwin" {
				fmt.Println("   2. On Mac: Check Activity Monitor for any Chrome processes")
				fmt.Println("   3. Or run in Terminal: killall 'Google Chrome'")
			} else if runtime.GOOS == "windows" {
				fmt.Println("   2. Check Task Manager for any Chrome processes")
				fmt.Println("   3. End all Chrome.exe processes")
			}
			fmt.Println("   4. Try running Specter again")
			return fmt.Errorf("browser already running - please close Chrome completely")
		}

		// Check for permission/access errors during download
		if strings.Contains(errMsg, "Access is denied") || strings.Contains(errMsg, "permission denied") {
			fmt.Println("\n❌ Browser download failed due to file permissions")
			fmt.Println("\n📋 To fix this issue:")
			fmt.Println("   1. Close ALL Chrome/Chromium processes (check Task Manager)")
			if runtime.GOOS == "windows" {
				fmt.Println("   2. Delete folder: %APPDATA%\\rod\\browser")
				fmt.Println("   3. Add antivirus exclusion for: %APPDATA%\\rod")
			} else {
				fmt.Println("   2. Delete folder: ~/Library/Caches/rod/browser")
			}
			fmt.Println("   4. Try running again")
			fmt.Println("\n💡 Alternative: Install Google Chrome and restart the app")
			fmt.Println("   Download from: https://www.google.com/chrome")
			return fmt.Errorf("browser setup failed: %w", err)
		}

		return fmt.Errorf("failed to launch browser: %w", err)
	}

	a.browser = rod.New().ControlURL(url).MustConnect()


	go a.watchBrowser()
	a.debugLog("Browser watcher started")

	fmt.Println("✓ Browser launched successfully")
	return nil
}

func (a *Automation) waitForLogin() error {
	fmt.Println("🔐 Opening browser for login...")

	targetURL := "https://robertsspaceindustries.com"
	if a.config.ItemURL != "" {
		targetURL = a.config.ItemURL
		fmt.Printf("📄 Navigating to product page: %s\n", targetURL)
	}

	var err error
	a.page, err = stealth.Page(a.browser)
	if err != nil {
		return fmt.Errorf("failed to create stealth page: %w", err)
	}

	a.debugLog("✓ Stealth mode enabled (anti-bot detection)")

	err = a.page.Navigate(targetURL)
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
		a.debugLog("User-Agent set to Chrome desktop")
	}

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

	if err := a.page.WaitLoad(); err != nil {
		return fmt.Errorf("page failed to load: %w", err)
	}

	if a.config.ItemURL != "" {
		statusCode, err := a.page.Eval(`() => {
			return window.performance?.getEntriesByType?.('navigation')?.[0]?.responseStatus || 200;
		}`)
		if err == nil {
			status := statusCode.Value.Int()
			if status == 404 {
				return fmt.Errorf("product page not found (404). Please verify the URL is correct: %s", a.config.ItemURL)
			} else if status >= 400 {
				return fmt.Errorf("failed to load product page (HTTP %d)", status)
			}
			a.debugLog("Product page HTTP status: %d", status)
		}

		hasSKUData, err := a.page.Eval(`() => {
			const skuDetailDiv = document.querySelector('[data-rsi-component="SkuDetailPage"]');
			return skuDetailDiv !== null;
		}`)
		if err == nil && !hasSKUData.Value.Bool() {
			return fmt.Errorf("page does not appear to be a valid product page (no SKU data found). URL: %s", a.config.ItemURL)
		}

		fmt.Println("✓ Product page loaded successfully")
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

	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		if input == '\n' || input == '\r' {
			fmt.Println()
			fmt.Println("✓ User confirmed ready to proceed")
			break
		}

		if input == 27 {
			fmt.Println()
			fmt.Println("⚠️  User requested exit")
			return fmt.Errorf("user canceled operation")
		}
	}

	if a.config.RecaptchaSiteKey != "" {
		fmt.Println("🔐 Pre-loading reCAPTCHA Enterprise...")
		a.preloadRecaptcha()

		fmt.Println("🎭 Building interaction history for reCAPTCHA scoring...")
		a.buildInteractionHistory()
	}

	return nil
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

	a.debugLog("Built ~200ms of interaction history (optimized for speed)")
	fmt.Println("✓ Interaction history established (fast mode)")
}

func (a *Automation) preloadRecaptcha() {
	checkExisting, err := a.page.Eval(`() => typeof grecaptcha !== 'undefined' && typeof grecaptcha.enterprise !== 'undefined'`)
	if err == nil && checkExisting.Value.Bool() {
		fmt.Println("✓ reCAPTCHA Enterprise already loaded")
		a.debugLog("reCAPTCHA already present on page")

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
			a.debugLog("Detected reCAPTCHA site key from page: %s", detectedKey)
			if detectedKey != a.config.RecaptchaSiteKey {
				fmt.Printf("⚠️  WARNING: Config key differs from page key!\n")
				fmt.Printf("   Config:   %s\n", a.config.RecaptchaSiteKey)
				fmt.Printf("   Detected: %s\n", detectedKey)
				fmt.Printf("   Update your config.yaml to use the detected key\n")
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
		a.debugLog("Warning: Failed to inject reCAPTCHA script: %v", err)
		fmt.Println("⚠️  reCAPTCHA injection failed (will retry during checkout)")
		return
	}

	a.debugLog("reCAPTCHA script injected, waiting for load...")

	// OPTIMIZED: Reduced wait time and check intervals for speed
	maxWait := 20 // 20 checks of 100ms = 2 seconds max instead of 5 seconds
	for i := 0; i < maxWait; i++ {
		time.Sleep(100 * time.Millisecond) // Check every 100ms instead of 1 second

		readyCheck, err := a.page.Eval(`() => typeof grecaptcha !== 'undefined' && typeof grecaptcha.enterprise !== 'undefined'`)
		if err == nil && readyCheck.Value.Bool() {
			fmt.Println("✓ reCAPTCHA Enterprise ready")
			a.debugLog("reCAPTCHA loaded successfully after %dms", (i+1)*100)
			return
		}
	}

	fmt.Println("⚠️  reCAPTCHA did not load in time (will retry during checkout)")
	a.debugLog("reCAPTCHA load timeout after 2 seconds")
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
