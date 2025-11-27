package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	url := flag.String("url", "", "Direct URL to the ship/item to purchase (overrides config)")
	dryRun := flag.Bool("dry-run", false, "Test mode: stop before final purchase")
	debug := flag.Bool("debug", false, "Enable detailed debug logging")
	skipCart := flag.Bool("skip-cart", false, "Skip adding to cart (item already in cart)")
	saleTime := flag.String("sale-time", "", "Sale start time in RFC3339 format (e.g., 2025-01-15T18:00:00Z) - enables timed sale mode")
	startBefore := flag.Int("start-before", 10, "Minutes to start retrying before sale (default: 10)")
	continueAfter := flag.Int("continue-after", 20, "Minutes to continue retrying after sale (default: 20)")
	flag.Parse()

	// Initialize localization
	if err := InitLocale(); err != nil {
		log.Printf("Warning: Locale initialization failed, using default English: %v", err)
	}

	// Check for user data directory permission issues (after locale is loaded)
	checkUserDataDirPermissions()

	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *url != "" {
		config.ItemURL = *url
	}

	if *dryRun {
		config.DryRun = true
	}
	if *debug {
		config.DebugMode = true
	}
	if *skipCart {
		config.SkipAddToCart = true
	}

	// Timed sale mode configuration
	if *saleTime != "" {
		config.EnableSaleTiming = true
		config.SaleStartTime = *saleTime
		config.StartBeforeSaleMinutes = *startBefore
		config.ContinueAfterSaleMinutes = *continueAfter
	}

	if config.ItemURL == "" && !config.SkipAddToCart {
		log.Fatal("No item URL specified. Use -url flag or set it in config.yaml")
	}

	fmt.Println(T("app_header"))
	fmt.Println()
	if config.ItemURL != "" {
		fmt.Printf(T("target_url")+"\n", config.ItemURL)
	}
	fmt.Printf(T("browser_profile")+"\n", config.BrowserProfilePath)

	if config.DryRun {
		fmt.Println(T("dry_run_mode"))
	}
	if config.DebugMode {
		fmt.Println(T("debug_mode"))
	}
	if config.SkipAddToCart {
		fmt.Println(T("skip_cart_mode"))
	}

	if config.EnableSaleTiming && config.SaleStartTime != "" {
		fmt.Println(T("timed_sale_mode"))
		fmt.Printf(T("timed_sale_time")+"\n", config.SaleStartTime)
		fmt.Printf(T("timed_sale_window")+"\n",
			config.StartBeforeSaleMinutes, config.ContinueAfterSaleMinutes)
	}

	fmt.Println(T("fast_api_mode"))
	fmt.Println()

	fmt.Println(T("step1_browser_setup"))
	automation := NewAutomation(config)
	defer automation.Close()

	if err := automation.setupBrowser(); err != nil {
		log.Fatalf("Failed to setup browser: %v", err)
	}

	if err := automation.waitForLogin(); err != nil {
		log.Fatalf("Failed to wait for login: %v", err)
	}

	fmt.Println(T("step2_init_fast_checkout"))
	fastCheckout, err := NewFastCheckout(config)
	if err != nil {
		log.Fatalf("Failed to initialize fast checkout: %v", err)
	}

	fmt.Println(T("step3_running_checkout"))

	// Use timed sale mode if enabled and sale time is configured
	if config.EnableSaleTiming && config.SaleStartTime != "" {
		if err := fastCheckout.RunTimedSaleCheckout(automation); err != nil {
			log.Fatalf("Timed sale checkout failed: %v", err)
		}
	} else {
		if err := fastCheckout.RunFastCheckout(automation); err != nil {
			log.Fatalf("Fast checkout failed: %v", err)
		}
	}

	fmt.Println()
	fmt.Println(T("checkout_completed"))
	fmt.Println()

	if config.KeepBrowserOpen {
		fmt.Println(T("keeping_browser_open"))
		time.Sleep(30 * time.Second)
	}
}

// Store init error for later display (after locale is loaded)
var initUserDataDirError error

func init() {
	userDataDir := getUserDataDir()
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		initUserDataDirError = err
	}
}

func checkUserDataDirPermissions() {
	if initUserDataDirError != nil {
		userDataDir := getUserDataDir()
		// Check if this is a macOS permission issue
		if runtime.GOOS == "darwin" && strings.Contains(initUserDataDirError.Error(), "operation not permitted") {
			fmt.Println(T("error_macos_permission_header"))
			fmt.Printf(T("error_macos_permission_location"), userDataDir)
			fmt.Println(T("error_macos_permission_fix_instructions"))
			fmt.Println(T("error_macos_permission_step1"))
			fmt.Println(T("error_macos_permission_step2"))
			fmt.Println(T("error_macos_permission_step3"))
			fmt.Println(T("error_macos_permission_step4"))
			fmt.Println(T("error_macos_permission_alternative"))
			fmt.Println()
		}
		log.Printf(T("error_macos_user_data_dir_warning"), initUserDataDirError)
	}
}

func getUserDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./specter-data"
	}
	return filepath.Join(home, ".specter")
}
