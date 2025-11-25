package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	url := flag.String("url", "", "Direct URL to the ship/item to purchase (overrides config)")
	dryRun := flag.Bool("dry-run", false, "Test mode: stop before final purchase")
	debug := flag.Bool("debug", false, "Enable detailed debug logging")
	skipCart := flag.Bool("skip-cart", false, "Skip adding to cart (item already in cart)")
	flag.Parse()

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

	if config.ItemURL == "" && !config.SkipAddToCart {
		log.Fatal("No item URL specified. Use -url flag or set it in config.yaml")
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║              RSI Store Checkout Assistant                ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	if config.ItemURL != "" {
		fmt.Printf("Target URL: %s\n", config.ItemURL)
	}
	fmt.Printf("Browser Profile: %s\n", config.BrowserProfilePath)

	if config.DryRun {
		fmt.Println("🧪 DRY RUN MODE - Will stop before final purchase")
	}
	if config.DebugMode {
		fmt.Println("🔍 DEBUG MODE - Detailed logging enabled")
	}
	if config.SkipAddToCart {
		fmt.Println("⏭️  SKIP CART MODE - Item already in cart, skipping add step")
	}

	fmt.Println("⚡ FAST API MODE - Hybrid authentication + API checkout")
	fmt.Println()

	fmt.Println("🌐 Step 1: Setting up browser for authentication...")
	automation := NewAutomation(config)
	defer automation.Close()

	if err := automation.setupBrowser(); err != nil {
		log.Fatalf("Failed to setup browser: %v", err)
	}

	if err := automation.waitForLogin(); err != nil {
		log.Fatalf("Failed to wait for login: %v", err)
	}

	fmt.Println("\n⚡ Step 2: Initializing fast API checkout...")
	fastCheckout, err := NewFastCheckout(config)
	if err != nil {
		log.Fatalf("Failed to initialize fast checkout: %v", err)
	}

	fmt.Println("\n🚀 Step 3: Running lightning-fast API checkout...")
	if err := fastCheckout.RunFastCheckout(automation); err != nil {
		log.Fatalf("Fast checkout failed: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ Checkout process completed successfully!")
	fmt.Println()

	if config.KeepBrowserOpen {
		fmt.Println("Keeping browser open for 30 seconds...")
		time.Sleep(30 * time.Second)
	}
}

func init() {
	userDataDir := getUserDataDir()
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		log.Printf("Warning: Could not create user data directory: %v", err)
	}
}

func getUserDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./specter-data"
	}
	return filepath.Join(home, ".specter")
}
