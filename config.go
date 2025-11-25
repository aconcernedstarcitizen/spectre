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

	StartBeforeSaleSeconds   int `yaml:"start_before_sale_seconds"`
	ContinueAfterSaleSeconds int `yaml:"continue_after_sale_seconds"`

	RecaptchaSiteKey string `yaml:"recaptcha_site_key"`
	RecaptchaAction  string `yaml:"recaptcha_action"`

	ViewportWidth  int `yaml:"viewport_width"`
	ViewportHeight int `yaml:"viewport_height"`

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
		MinDelayBetween:      0.5,
		MaxDelayBetween:      1.0,
		CheckoutReadyDelay:   2,
		RetryDurationSeconds:     300,
		RetryDelayMinMs:          29,
		RetryDelayMaxMs:          107,
		StartBeforeSaleSeconds:   600,
		ContinueAfterSaleSeconds: 900,
		RecaptchaSiteKey:         "6LcZ-cUpAAAAABTy47-ryVJAsZFocXguqi_FgLlJ",
		RecaptchaAction:          "store/cart/add",
		ViewportWidth:            1920,
		ViewportHeight:           1080,
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
