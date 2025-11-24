package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application
type Config struct {
	// ItemURL is the direct URL to the ship/item to purchase
	ItemURL string `yaml:"item_url"`

	// BrowserProfilePath is where browser data/cookies are stored
	// Leave empty to use default location
	BrowserProfilePath string `yaml:"browser_profile_path"`

	// Browser type: chrome, edge, or firefox
	BrowserType string `yaml:"browser_type"`

	// Timeouts and delays
	PageLoadTimeout    int     `yaml:"page_load_timeout"`     // seconds
	MinDelayBetween    float64 `yaml:"min_delay_between"`     // seconds
	MaxDelayBetween    float64 `yaml:"max_delay_between"`     // seconds
	CheckoutReadyDelay int     `yaml:"checkout_ready_delay"`  // seconds to wait before starting checkout

	// Retry configuration
	MaxRetries       int     `yaml:"max_retries"`        // maximum number of retry attempts (0 = no retries)
	RetryDelayMin    float64 `yaml:"retry_delay_min"`    // minimum delay between retries in seconds
	RetryDelayMax    float64 `yaml:"retry_delay_max"`    // maximum delay between retries in seconds

	// Viewport configuration
	ViewportWidth  int `yaml:"viewport_width"`   // browser viewport width (desktop mode recommended)
	ViewportHeight int `yaml:"viewport_height"`  // browser viewport height

	// Store credit configuration
	AutoApplyCredit bool `yaml:"auto_apply_credit"` // Automatically apply store credit (amount auto-detected from item price)

	// Cart behavior
	SkipAddToCart bool `yaml:"skip_add_to_cart"` // Skip adding to cart (item already in cart from previous attempt)

	// Behavior
	Headless        bool `yaml:"headless"`           // Run browser in headless mode
	KeepBrowserOpen bool `yaml:"keep_browser_open"`  // Keep browser open after completion

	// Testing and Debug
	DryRun      bool `yaml:"dry_run"`       // Stop before final purchase (test mode)
	Interactive bool `yaml:"interactive"`   // Pause and wait for Enter at each step
	DebugMode   bool `yaml:"debug_mode"`    // Show detailed debug information

	// Selectors (can be customized if site changes)
	Selectors SelectorConfig `yaml:"selectors"`
}

// SelectorConfig holds CSS selectors for various elements
type SelectorConfig struct {
	AddToCartButton    string `yaml:"add_to_cart_button"`
	CartIcon           string `yaml:"cart_icon"`
	CheckoutButton     string `yaml:"checkout_button"`
	ApplyCreditButton  string `yaml:"apply_credit_button"`
	AgreeCheckbox      string `yaml:"agree_checkbox"`
	FinalCheckoutButton string `yaml:"final_checkout_button"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	userDataDir := getUserDataDir()

	return &Config{
		ItemURL:            "",
		BrowserProfilePath: filepath.Join(userDataDir, "browser-profile"),
		BrowserType:        "chrome",
		PageLoadTimeout:    30,
		MinDelayBetween:    0.5,
		MaxDelayBetween:    1.0,
		CheckoutReadyDelay: 2,
		MaxRetries:         30,
		RetryDelayMin:      0.4,
		RetryDelayMax:      0.8,
		ViewportWidth:      1920,
		ViewportHeight:     1080,
		Headless:           false,
		KeepBrowserOpen:    true,
		AutoApplyCredit:    true,
		SkipAddToCart:      false,
		DryRun:             false,
		Interactive:        false,
		DebugMode:          false,
		Selectors: SelectorConfig{
			AddToCartButton:    ".add-to-cart, .js-add-to-cart, button[data-action='add-to-cart']",
			CartIcon:           ".cart-icon, .shopping-cart, [data-testid='cart']",
			CheckoutButton:     ".checkout-button, .btn-checkout, button:contains('Checkout')",
			ApplyCreditButton:  ".apply-credit, .store-credit, button:contains('Apply Credit')",
			AgreeCheckbox:      "input[type='checkbox'][name*='agree'], input[type='checkbox'][name*='terms']",
			FinalCheckoutButton: ".final-checkout, .confirm-order, button:contains('Confirm')",
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	// If file doesn't exist, create it with defaults
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := config.Save(path); err != nil {
			return nil, err
		}
		return config, nil
	}

	// Read existing config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	// Ensure browser profile path exists
	if config.BrowserProfilePath != "" {
		if err := os.MkdirAll(config.BrowserProfilePath, 0755); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// Save writes the configuration to a YAML file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
