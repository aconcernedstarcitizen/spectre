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

	targetURL := "https://robertsspaceindustries.com"
	if a.config.ItemURL != "" {
		targetURL = a.config.ItemURL
		fmt.Printf(T("loading_product_page")+"\n", targetURL)
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

		fmt.Println(T("product_page_loaded"))
	}

	fmt.Println(T("browser_configured"))

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

	if a.config.RecaptchaSiteKey != "" {
		fmt.Println(T("recaptcha_preloading"))
		a.preloadRecaptcha()

		fmt.Println(T("building_interaction_history"))
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
