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
	// Command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	url := flag.String("url", "", "Direct URL to the ship/item to purchase (overrides config)")
	dryRun := flag.Bool("dry-run", false, "Test mode: stop before final purchase")
	interactive := flag.Bool("interactive", false, "Pause at each step for review")
	debug := flag.Bool("debug", false, "Enable detailed debug logging")
	skipCart := flag.Bool("skip-cart", false, "Skip adding to cart (item already in cart)")
	flag.Parse()

	// Load configuration
	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override URL if provided via command line
	if *url != "" {
		config.ItemURL = *url
	}

	// Override test/debug flags if provided
	if *dryRun {
		config.DryRun = true
	}
	if *interactive {
		config.Interactive = true
	}
	if *debug {
		config.DebugMode = true
	}
	if *skipCart {
		config.SkipAddToCart = true
	}

	// Validate we have a URL to work with (not required in skip-cart mode)
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

	// Show mode indicators
	if config.DryRun {
		fmt.Println("🧪 DRY RUN MODE - Will stop before final purchase")
	}
	if config.Interactive {
		fmt.Println("⏸️  INTERACTIVE MODE - Will pause at each step")
	}
	if config.DebugMode {
		fmt.Println("🔍 DEBUG MODE - Detailed logging enabled")
	}
	if config.SkipAddToCart {
		fmt.Println("⏭️  SKIP CART MODE - Item already in cart, skipping add step")
	}
	fmt.Println()

	// Initialize automation
	automation := NewAutomation(config)
	defer automation.Close()

	// Run the checkout process
	if err := automation.RunCheckout(); err != nil {
		log.Fatalf("Checkout failed: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ Checkout process completed successfully!")
	fmt.Println()

	// Keep browser open for a bit so user can see the result
	if config.KeepBrowserOpen {
		fmt.Println("Keeping browser open for 30 seconds...")
		time.Sleep(30 * time.Second)
	}
}

func init() {
	// Ensure user data directory exists
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
