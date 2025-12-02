package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ItemURL string `yaml:"item_url"`

	BrowserProfilePath string `yaml:"browser_profile_path"`

	BrowserType string `yaml:"browser_type"`

	PageLoadTimeout    int     `yaml:"page_load_timeout"`
	MinDelayBetween    float64 `yaml:"min_delay_between"`
	MaxDelayBetween    float64 `yaml:"max_delay_between"`
	CheckoutReadyDelay int     `yaml:"checkout_ready_delay"`

	RetryDurationSeconds int `yaml:"retry_duration_seconds"`
	RetryDelayMinMs      int `yaml:"retry_delay_min_ms"`
	RetryDelayMaxMs      int `yaml:"retry_delay_max_ms"`

	// Error-specific retry delays (in milliseconds)
	Payment4227MinMs    int `yaml:"payment_4227_min_ms"`    // Payment auth error 4227 min delay
	Payment4227MaxMs    int `yaml:"payment_4227_max_ms"`    // Payment auth error 4227 max delay
	Payment4226MinMs    int `yaml:"payment_4226_min_ms"`    // Payment auth error 4226 min delay
	Payment4226MaxMs    int `yaml:"payment_4226_max_ms"`    // Payment auth error 4226 max delay
	RateLimitMinMs      int `yaml:"rate_limit_min_ms"`      // Rate limit min delay
	RateLimitMaxMs      int `yaml:"rate_limit_max_ms"`      // Rate limit max delay
	OutOfStockDelayMs   int `yaml:"out_of_stock_delay_ms"`  // Out of stock delay
	GenericErrorDelayMs int `yaml:"generic_error_delay_ms"` // Generic error delay

	// Sale wave configuration
	SaleWindows              []string `yaml:"sale_windows"`                // Sale times in RFC3339 format (e.g., ["2025-01-15T16:00:00Z", "2025-01-15T20:00:00Z"])
	PreWaveActivationMinutes int      `yaml:"pre_wave_activation_minutes"` // Minutes before wave to start polling for product page (default: 2)
	PostWaveTimeoutMinutes   int      `yaml:"post_wave_timeout_minutes"`   // Minutes after wave to keep trying before moving to next wave (default: 5)
	PollingDelayMinMs        int      `yaml:"polling_delay_min_ms"`        // Minimum delay between polling attempts (default: 29ms)
	PollingDelayMaxMs        int      `yaml:"polling_delay_max_ms"`        // Maximum delay between polling attempts (default: 139ms)

	RecaptchaSiteKey string `yaml:"recaptcha_site_key"`
	RecaptchaAction  string `yaml:"recaptcha_action"`

	AutoApplyCredit bool `yaml:"auto_apply_credit"`

	SkipAddToCart bool `yaml:"skip_add_to_cart"`

	Headless        bool `yaml:"headless"`
	KeepBrowserOpen bool `yaml:"keep_browser_open"`

	DryRun    bool `yaml:"dry_run"`
	DebugMode bool `yaml:"debug_mode"`

	Selectors SelectorConfig `yaml:"selectors"`
}

type SelectorConfig struct {
	AddToCartButton     string `yaml:"add_to_cart_button"`
	CartIcon            string `yaml:"cart_icon"`
	CheckoutButton      string `yaml:"checkout_button"`
	ApplyCreditButton   string `yaml:"apply_credit_button"`
	AgreeCheckbox       string `yaml:"agree_checkbox"`
	FinalCheckoutButton string `yaml:"final_checkout_button"`
}

func DefaultConfig() *Config {
	userDataDir := getUserDataDir()

	return &Config{
		ItemURL:              "",
		BrowserProfilePath:   filepath.Join(userDataDir, "browser-profile"),
		BrowserType:          "chrome",
		PageLoadTimeout:      30,
		MinDelayBetween:      0.1,  // Reduced from 0.5 for speed
		MaxDelayBetween:      0.3,  // Reduced from 1.0 for speed
		CheckoutReadyDelay:   1,    // Reduced from 2 for speed
		RetryDurationSeconds:     300,
		RetryDelayMinMs:          5,    // Ultra-fast retries
		RetryDelayMaxMs:          20,   // Ultra-fast retries
		Payment4227MinMs:    1300,  // Payment auth 4227: 1.3-2.1 seconds
		Payment4227MaxMs:    2100,
		Payment4226MinMs:    500,   // Payment auth 4226: 500-700ms
		Payment4226MaxMs:    700,
		RateLimitMinMs:           50,         // Rate limit: 50-150ms
		RateLimitMaxMs:           150,
		OutOfStockDelayMs:        100,        // Out of stock: 100ms
		GenericErrorDelayMs:      100,        // Generic errors: 100ms
		SaleWindows:              []string{}, // Sale windows (required: use --waves-date YYYY-MM-DD or configure in config.yaml)
		PreWaveActivationMinutes: 2,          // Start polling 2 minutes before wave
		PostWaveTimeoutMinutes:   5,          // Continue 5 minutes after wave before moving to next
		PollingDelayMinMs:        29,         // Polling delay: 29-139ms (human-like, variable timing)
		PollingDelayMaxMs:        139,
		RecaptchaSiteKey:         "6LcZ-cUpAAAAABTy47-ryVJAsZFocXguqi_FgLlJ",
		RecaptchaAction:          "store/cart/add",
		Headless:             false,
		KeepBrowserOpen:      true,
		AutoApplyCredit:      true,
		SkipAddToCart:        false,
		DryRun:               false,
		DebugMode:            false,
		Selectors: SelectorConfig{
			AddToCartButton:     ".add-to-cart, .js-add-to-cart, button[data-action='add-to-cart']",
			CartIcon:            ".cart-icon, .shopping-cart, [data-testid='cart']",
			CheckoutButton:      ".checkout-button, .btn-checkout, button:contains('Checkout')",
			ApplyCreditButton:   ".apply-credit, .store-credit, button:contains('Apply Credit')",
			AgreeCheckbox:       "input[type='checkbox'][name*='agree'], input[type='checkbox'][name*='terms']",
			FinalCheckoutButton: ".final-checkout, .confirm-order, button:contains('Confirm')",
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := config.Save(path); err != nil {
			return nil, err
		}
		return config, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	if config.BrowserProfilePath != "" {
		if err := os.MkdirAll(config.BrowserProfilePath, 0755); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
