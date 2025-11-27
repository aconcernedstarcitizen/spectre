# Localization Guidelines

## Overview

Specter supports full internationalization (i18n) with multi-language support. All user-facing messages must be localized to ensure a consistent experience for users in different regions.

## Supported Languages

- **en_US** - English (United States)
- **ru_RU** - Russian (Russia)

Additional languages can be added by creating new YAML files in the `lang/` directory following the same structure.

## Directory Structure

```
specter/
├── lang/
│   ├── en_US.yaml    # English translations
│   ├── ru_RU.yaml    # Russian translations
│   └── ...           # Additional language files
├── locale.go         # Localization system implementation
└── locale_test.go    # Localization tests
```

## How Localization Works

### 1. System Locale Detection

The app automatically detects the system locale from environment variables:
- `LANG` (primary)
- `LC_ALL` (fallback)
- `LC_MESSAGES` (fallback)

If no locale is detected or the locale file doesn't exist, it defaults to `en_US`.

### 2. Loading Translations

On startup, the app:
1. Detects system locale via `DetectSystemLocale()`
2. Loads the corresponding YAML file from `lang/<locale>.yaml`
3. Falls back to `en_US` if the primary locale fails to load
4. Stores translations in a global `Locale` struct

### 3. Translation Function

Use the `T()` function to get localized strings:

```go
// Simple translation
fmt.Println(T("session_extracting"))

// Translation with parameters
fmt.Printf(T("session_cookies_extracted")+"\n", len(cookies))

// Translation with multiple parameters
fmt.Printf(T("cart_item_price_line")+"\n", price, quantity, total)
```

## Adding New Localized Strings

### Step 1: Add Translation Keys

Add the key to **ALL** locale files (`en_US.yaml`, `ru_RU.yaml`, etc.):

**lang/en_US.yaml:**
```yaml
# Session Extraction
session_extracting: "🔐 Extracting session from browser..."
session_cookies_extracted: "✓ Extracted %d cookies from browser"
```

**lang/ru_RU.yaml:**
```yaml
# Session Extraction
session_extracting: "🔐 Извлечение сеанса из браузера..."
session_cookies_extracted: "✓ Извлечено %d cookie из браузера"
```

### Step 2: Use T() in Code

Replace hardcoded strings with `T()` calls:

**Before:**
```go
fmt.Println("🔐 Extracting session from browser...")
fmt.Printf("✓ Extracted %d cookies from browser\n", len(cookies))
```

**After:**
```go
fmt.Println(T("session_extracting"))
fmt.Printf(T("session_cookies_extracted")+"\n", len(cookies))
```

### Step 3: Test

Run locale tests to ensure all keys exist:
```bash
go test -v -run TestLocalizationKeysExist
```

## Localization Key Naming Conventions

Use descriptive, hierarchical keys grouped by feature/context:

### Pattern: `<category>_<subcategory>_<description>`

Examples:
- `session_extracting` - Session extraction status
- `session_cookies_extracted` - Session cookie extraction result
- `cart_adding_api_retry` - Cart add operation with retry
- `cart_totals_result` - Cart total query result
- `error_failed_create_request` - Error: failed to create request
- `debug_product_page_status` - Debug: product page HTTP status

### Categories

- `session_*` - Session and authentication
- `sku_*` - SKU extraction and lookup
- `cart_*` - Cart operations (add, query, validate)
- `credit_*` - Store credit operations
- `address_*` - Billing address operations
- `validation_*` - Order validation
- `checkout_*` - Checkout flow
- `timed_sale_*` - Timed sale mode
- `recaptcha_*` - reCAPTCHA operations
- `error_*` - Error messages
- `debug_*` - Debug messages
- `browser_*` - Browser operations
- `login_*` - Login flow

## Format Specifiers

Use Go's `fmt` package format specifiers in translation strings:

- `%s` - String
- `%d` - Integer
- `%f` - Float
- `%.2f` - Float with 2 decimal places
- `%v` - Any value (duration, time, etc.)
- `%w` - Error wrapping

Example:
```yaml
cart_item_price_line: "   Price: $%.2f × %d = $%.2f"
error_failed_create_request: "failed to create request: %w"
```

## Error Messages

### User-Facing Errors

Use localized error messages for all user-facing errors:

```go
// BAD - Hardcoded English
return fmt.Errorf("browser not initialized")

// GOOD - Localized
return fmt.Errorf(T("session_browser_not_initialized"))
```

### Internal/Technical Errors

For errors that are primarily for debugging (wrapped errors, internal errors), you can keep them in English to maintain stack trace readability:

```go
// Technical error - can stay in English
if err != nil {
    return fmt.Errorf("failed to marshal JSON: %w", err)
}
```

However, the **user-visible portion** should still be localized:

```go
// BAD
return fmt.Errorf("failed to create request: %w", err)

// GOOD
return fmt.Errorf(T("error_failed_create_request"), err)
```

## Debug Messages

All `[DEBUG]` messages should be localized for consistency:

```go
// BAD
if f.config.DebugMode {
    fmt.Printf("[DEBUG] Product page HTTP status: %d\n", statusCode)
}

// GOOD
if f.config.DebugMode {
    fmt.Printf(T("debug_product_page_status")+"\n", statusCode)
}
```

## Multi-Line Messages

For multi-line messages (headers, banners), use separate keys for each line or include newlines in the translation:

**Option 1: Separate keys**
```yaml
checkout_fast_header_line1: "╔═══════════════════════════════════════════════════════════╗"
checkout_fast_header_line2: "║           FAST CHECKOUT - API MODE                        ║"
checkout_fast_header_line3: "║           (Browser-Free Lightning Speed)                  ║"
checkout_fast_header_line4: "╚═══════════════════════════════════════════════════════════╝"
```

```go
fmt.Println(T("checkout_fast_header_line1"))
fmt.Println(T("checkout_fast_header_line2"))
fmt.Println(T("checkout_fast_header_line3"))
fmt.Println(T("checkout_fast_header_line4"))
```

**Option 2: Single key with newlines**
```yaml
checkout_fast_header: "╔═══════════════════════════════════════════════════════════╗\n║           FAST CHECKOUT - API MODE                        ║\n║           (Browser-Free Lightning Speed)                  ║\n╚═══════════════════════════════════════════════════════════╝"
```

```go
fmt.Println(T("checkout_fast_header"))
```

## Testing Localization

### Add Tests for New Keys

Update `locale_test.go` to include new required keys:

```go
func TestLocalizationKeysExist(t *testing.T) {
    requiredKeys := []string{
        "session_extracting",
        "session_cookies_extracted",
        // ... add new keys here
    }

    // Test would load actual locale files and verify keys exist
}
```

### Run Tests

```bash
# Run all locale tests
go test -v -run TestLocalization

# Run specific test
go test -v -run TestTranslationFunction
```

## Translation Workflow

1. **Identify hardcoded strings** in code
2. **Create descriptive keys** following naming conventions
3. **Add to English locale** (`lang/en_US.yaml`) first
4. **Translate to other languages** (`lang/ru_RU.yaml`, etc.)
5. **Update code** to use `T()` function
6. **Add test coverage** for new keys
7. **Test** with different locales

## Emoji Usage

Emojis are allowed and encouraged in localized strings for visual clarity:

```yaml
session_extracting: "🔐 Extracting session from browser..."
cart_adding_api_retry: "🛒 Adding to cart (API) with retry mechanism..."
validation_completing: "✅ Validating and completing order..."
```

Emojis help users quickly identify message types:
- 🔐 - Security/authentication
- 🛒 - Cart operations
- ✅ - Success/validation
- ⚠️ - Warnings
- ❌ - Errors
- 🔍 - Search/lookup
- ⏱️ - Timing/waiting

## Common Mistakes

### ❌ Hardcoded strings
```go
fmt.Println("🔐 Extracting session from browser...")
```

### ✅ Use T() function
```go
fmt.Println(T("session_extracting"))
```

### ❌ Missing newline handling
```go
fmt.Printf(T("session_cookies_extracted"), len(cookies))  // Missing \n
```

### ✅ Add newline explicitly
```go
fmt.Printf(T("session_cookies_extracted")+"\n", len(cookies))
```

### ❌ Key missing in some locales
```yaml
# en_US.yaml has key
new_feature_message: "New feature added"

# ru_RU.yaml missing key - will cause fallback to key name
```

### ✅ Add to ALL locale files
```yaml
# en_US.yaml
new_feature_message: "New feature added"

# ru_RU.yaml
new_feature_message: "Добавлена новая функция"
```

## Rule: Everything Must Be Localized

**IMPORTANT**: From now on, **ALL user-facing messages** in the application must be localized. This includes:

- ✅ Status messages
- ✅ Progress indicators
- ✅ Success messages
- ✅ Warning messages
- ✅ Error messages
- ✅ Debug messages
- ✅ Headers and banners
- ✅ User prompts
- ✅ Informational messages

**The ONLY exceptions are:**
- ❌ Code comments
- ❌ Variable names
- ❌ Function names
- ❌ Log messages that are purely for internal debugging (if any)

## Getting Current Locale

To get the current locale being used:

```go
currentLocale := GetLocale()  // Returns "en_US", "ru_RU", etc.
```

## Changing Locale at Runtime

The locale is set at startup based on system settings. To change it programmatically:

```go
// Detect system locale
locale := DetectSystemLocale()

// Initialize with detected locale
err := InitLocale(locale)
if err != nil {
    // Falls back to en_US automatically
    log.Printf("Failed to load locale %s: %v", locale, err)
}
```

## Future Enhancements

Potential future improvements to the localization system:

1. **Runtime locale switching** - Allow users to change language without restarting
2. **Locale-specific formatting** - Date/time/currency formatting per locale
3. **Pluralization support** - Handle plural forms correctly (e.g., "1 item" vs "2 items")
4. **Context-aware translations** - Same word with different meanings in different contexts
5. **Translation validation** - Automated checks for missing keys across locales
6. **Hot reload** - Reload locale files without restarting the app

## Resources

- Locale files: `lang/*.yaml`
- Localization code: `locale.go`
- Tests: `locale_test.go`
- Go `fmt` package: https://pkg.go.dev/fmt
