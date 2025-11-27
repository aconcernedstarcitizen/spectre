# Claude Context - Internal Guidelines for AI Assistant

> **Purpose**: This document provides context and guidelines for Claude (AI assistant) when working on the Specter codebase across multiple sessions. It captures institutional knowledge, decision rationale, and patterns to maintain consistency.

---

## Project Overview

**Specter** is an automated checkout tool for Star Citizen ships on the Roberts Space Industries (RSI) store. It uses browser automation + direct API calls to achieve sub-second checkout speeds for limited-availability ship sales.

**Primary Goal**: Complete checkout in <1 second (currently achieving 300-850ms)

**Tech Stack**:
- Go 1.21+
- `github.com/go-rod/rod` - Browser automation via Chrome DevTools Protocol
- `github.com/go-rod/stealth` - Anti-bot detection
- GraphQL API - RSI store backend
- YAML - Configuration and localization

**Key Files**:
- `main.go` - Entry point
- `automation.go` - Browser automation (Rod)
- `fast_checkout.go` - API-based checkout logic (2256 lines)
- `locale.go` - i18n system
- `config.go` - Configuration management
- `lang/*.yaml` - Localization files

---

## Critical Context to Maintain Across Sessions

### 1. Performance is Sacred

**Target**: Sub-second checkout (<1000ms)

**Current Performance**: 300-850ms (achieved through 7 optimizations)

**When making ANY change, ask**:
- Does this add latency?
- Can I eliminate a round trip?
- Can I cache this?
- Can I combine multiple operations?

**Never**:
- Add redundant GraphQL queries
- Launch additional browser instances
- Add synchronous waits unless absolutely necessary
- Introduce blocking I/O in critical path

### 2. GraphQL Operation Knowledge

**Critical Understanding**: RSI uses GraphQL. Key operations:

**AddCartMultiItemMutation** (Add to cart):
- Does NOT require reCAPTCHA token (this was tested and confirmed)
- Do NOT send reCAPTCHA token with this mutation
- Works with just cookies + CSRF token

**CartValidateCartMutation** (Finalize order):
- REQUIRES reCAPTCHA v3 Enterprise token
- REQUIRES random "mark" parameter matching RSI's logic: `Math.floor(Math.random() * 10000000000)` (0-9999999999)
- Mark should be generated ONCE per validation session and reused across retries

**Error Codes** (CRITICAL - UPDATED 2025-01-27):
- `4226` = **Payment auth error** → Retry with 500-700ms delay (configurable)
- `4227` = **Payment auth error** → Retry with 1300-2100ms delay (configurable)
- `CFUException` = CAPTCHA required → Get reCAPTCHA token
- `TyUnknownCustomerException` = Not logged in → Prompt user to login
- HTTP 429 or "rate limit" text = Actual rate limit → 50-150ms delay (configurable)
- "out of stock" / "not available" = Out of stock → 100ms delay (configurable)

**IMPORTANT**: 4226 and 4227 are NOT stock/rate limit errors. They are payment authorization errors from RSI's payment processor.

### 3. Optimization History (DO NOT REVERT)

These optimizations are PERMANENT. Do not revert them:

**Optimization 1**: Removed redundant post-add cart validation
- Why: Successful add guarantees cart state
- Saved: 100-300ms
- Location: After AddToCart() calls

**Optimization 2**: Combined GetCartTotals() + GetCartItems() into GetCartTotalsAndItems()
- Why: Two queries can be one
- Saved: 100-300ms per call (used 2x)
- Function: GetCartTotalsAndItems() returns CartInfo struct

**Optimization 3**: HTTP-based SKU extraction (UPDATED 2025-01-27)
- Why: Use HTTP client instead of opening incognito browser window
- Method: HTTP GET + regex extraction of SKU slug before login
- Saved: 140-400ms (HTTP request ~10-50ms vs incognito window ~150-450ms)
- Function: extractAndCacheSKU() in automation.go
- Timing: Runs BEFORE login prompt, validates page is valid ship page
- Caches: SKU slug in automation.cachedSKU

**Optimization 4**: Cached billing address
- Why: Address doesn't change between checkouts
- Saved: 50-100ms
- Field: cachedAddressID in FastCheckout struct

**Optimization 5**: Cached reCAPTCHA tokens
- Why: Tokens valid 60s, generating takes ~1-2s
- Saved: 500-1500ms in aggressive retry loops
- Function: GetOrRefreshCachedRecaptchaToken()

**Optimization 6**: Mathematical cart total calculation (UPDATED 2025-01-27)
- Why: Use item.Price instead of cartTotal for credit calculation (handles tax in regions like Ukraine)
- Rationale: Store credit items never have tax; item price is the true amount for single items
- Saved: Prevents incorrect credit amounts in taxed regions
- Location: RunFastCheckout() and RunTimedSaleCheckout()

**Optimization 7**: Pre-flight checks in timed sale mode
- Why: Catch errors BEFORE waiting for sale window
- Benefit: Better UX, errors caught early
- Location: RunTimedSaleCheckout() - checks run before sleep

**Optimization 8**: Configurable retry delays (ADDED 2025-01-27)
- Why: Allow users to tune delays based on their network/testing
- Config fields: payment_4227_min_ms, payment_4227_max_ms, payment_4226_min_ms, payment_4226_max_ms, rate_limit_min_ms, rate_limit_max_ms, out_of_stock_delay_ms, generic_error_delay_ms
- Default values optimized for production use
- Location: config.go defaults, used in AddToCart() and ValidateCartWithDeadline()

### 4. Design Patterns to Follow

**Pattern 1: Login Retry Wrapper**

All GraphQL requests MUST use `graphqlRequestWithLoginRetry()`:

```go
resp, err := f.graphqlRequestWithLoginRetry(request)
// NOT: resp, err := f.graphqlRequest(request)
```

This automatically detects "not logged in" errors and prompts user.

**Pattern 2: Three-State Validation**

Cart validation returns `(bool, error)`:
- `(true, nil)` = Safe to add item
- `(false, nil)` = Skip add, proceed with current cart
- `(false, error)` = User cancelled or validation error

**Pattern 3: Error Classification** (UPDATED 2025-01-27)

Always classify errors by type to determine retry strategy:
- `isPaymentAuthError(err)` → Returns (is4226, is4227) for payment auth errors
  - 4227: 1.3-2.1s delay (configurable)
  - 4226: 500-700ms delay (configurable)
- `isOutOfStockError()` → 100ms delay (configurable)
- `isRateLimitError()` → 50-150ms delay (configurable, only for actual HTTP 429/rate limit text)
- `isCaptchaError()` → Get reCAPTCHA token
- `isNotLoggedInError()` → Prompt user login

**CRITICAL**: Do NOT check for 4226/4227 in isOutOfStockError() or isRateLimitError(). They are payment auth errors.

**Pattern 4: Smart Retry Based on Error Type** (UPDATED 2025-01-27)

Use appropriate delays based on error classification:
```go
is4226, is4227 := isPaymentAuthError(err)
if is4227 {
    delay = time.Duration(f.config.Payment4227MinMs + rand.Intn(f.config.Payment4227MaxMs-f.config.Payment4227MinMs+1)) * time.Millisecond
} else if is4226 {
    delay = time.Duration(f.config.Payment4226MinMs + rand.Intn(f.config.Payment4226MaxMs-f.config.Payment4226MinMs+1)) * time.Millisecond
} else if isOutOfStockError(err) {
    delay = time.Duration(f.config.OutOfStockDelayMs) * time.Millisecond
}
```

Rationale: Different errors need different backoff strategies. Payment auth errors need longer delays.

### 5. Localization is Mandatory

**CRITICAL RULE**: ALL user-facing messages MUST be localized.

**Pattern**:
```go
// BAD
fmt.Println("Extracting session from browser...")

// GOOD
fmt.Println(T("session_extracting"))
```

**When adding new messages**:
1. Add key to `lang/en_US.yaml`
2. Add Russian translation to `lang/ru_RU.yaml`
3. Use `T("key_name")` in code
4. Add `+"\n"` for printf: `fmt.Printf(T("key")+"\n", arg)`

**Naming convention**: `category_subcategory_description`
- Examples: `session_extracting`, `cart_adding_api_retry`, `error_failed_create_request`

### 6. Browser Automation Context

**Rod Library**: Chrome DevTools Protocol wrapper

**Stealth Mode**: ALWAYS use `stealth.MustPage(browser)` to avoid detection

**Leakless Mode**:
- Enabled on macOS/Linux
- DISABLED on Windows (causes issues)
- Already handled in code with runtime.GOOS check

**Session Extraction**:
- Cookies extracted from browser
- CSRF token from meta tag or cookies
- User-Agent from navigator.userAgent
- All used for API requests

**SKU Extraction** (UPDATED 2025-01-27):
- **When**: BEFORE login prompt (validates page is valid ship page)
- **Method**: HTTP GET request + regex extraction from HTML
- **Why**: HTTP client is faster (~10-50ms) than browser automation (~150-450ms)
- **Flow**:
  1. Browser navigates to item URL (user sees page)
  2. extractAndCacheSKU() uses HTTP client to fetch same URL
  3. Regex extracts SKU slug from HTML: `"skuSlug":\s*"([^"]+)"`
  4. If SKU not found → Clear error message with instructions
  5. If valid → Cache in automation.cachedSKU
  6. Show login prompt to user
  7. After login → Convert SKU slug to SKU ID via GraphQL
- **Function**: extractAndCacheSKU() in automation.go
- **Error Handling**: Shows detailed error if page is not a valid ship page

### 7. Timed Sale Mode Context

**Purpose**: Handle limited-time ship sales with aggressive retries

**Key Insight**: Pre-flight checks run BEFORE waiting for sale window

**Flow**:
```
1. Pre-flight checks (load session, get SKU, validate cart)
2. Display "All pre-flight checks passed!"
3. Wait until (sale_time - start_before_minutes)
4. Phase 1: Aggressive add-to-cart retries
5. Phase 2: Aggressive checkout retries
```

**Why pre-flight checks first**:
- Catches login errors immediately (not right before sale!)
- User can take a break knowing everything is ready
- Better UX

### 8. Testing Approach

**Test Coverage**:
- Locale system (detection, loading, translation)
- Error classification functions
- Cart validation logic
- Combined query optimization
- NOT integration tests (requires real RSI account)

**Run tests**: `go test -v`

**When adding features**:
- Add unit tests for pure functions
- Add test cases to locale_test.go for new keys
- Run tests before committing

### 9. Configuration Context

**config.yaml** controls behavior:
- `item_url` - Product to purchase
- `skip_add_to_cart` - For testing (skip add step)
- `auto_apply_credit` - Auto-apply store credit
- `dry_run` - Stop before final submission
- `timed_sale_enabled` - Enable timed sale mode
- `retry_duration_seconds` - Retry timeout

**User may set these** - respect config values!

### 10. Common Pitfalls to Avoid

**❌ DON'T**:
- Add redundant GraphQL queries (check optimization history first)
- Use hardcoded English strings (MUST localize)
- Launch incognito browser for SKU extraction (use GetSKUFromActivePage)
- Send reCAPTCHA token with AddCartMultiItemMutation (not needed!)
- Revert performance optimizations
- Add long sleep/wait times in critical path
- Skip login retry wrapper for GraphQL requests

**✅ DO**:
- Cache reusable data (addresses, tokens)
- Combine multiple queries when possible
- Use error classification for retry strategy
- Localize all user-facing messages
- Maintain sub-second performance target
- Follow existing design patterns

### 11. Code Organization Context

**FastCheckout struct** (main checkout logic):
- `client *http.Client` - For GraphQL requests
- `cookies []*http.Cookie` - Session cookies from browser
- `csrfToken string` - CSRF protection token
- `cachedAddressID string` - Cached billing address
- `cachedRecaptchaToken string` - Cached reCAPTCHA token
- `automation *Automation` - Reference for login retry

**Automation struct** (browser control):
- `browser *rod.Browser` - Rod browser instance
- `page *rod.Page` - Active page
- `config *Config` - Configuration reference

### 12. GraphQL Request Structure

**Standard pattern**:
```go
request := []GraphQLRequest{{
    OperationName: "MutationName",
    Variables: map[string]interface{}{
        "storeFront": "pledge",
        // ... other vars
    },
    Query: `mutation MutationName($var: Type) { ... }`,
}}

resp, err := f.graphqlRequestWithLoginRetry(request)
```

**Headers required**:
- `Content-Type: application/json`
- `User-Agent: <browser user agent>`
- `Accept: */*`
- `Accept-Language: en`
- `Origin: https://robertsspaceindustries.com`
- `Referer: https://robertsspaceindustries.com/`
- `x-csrf-token: <csrf token>` (if available)
- Cookies: All session cookies

### 13. When User Asks for Changes

**Always consider**:
1. Performance impact (is this on critical path?)
2. Localization (new user-facing messages?)
3. Error handling (can this fail? how to retry?)
4. Timed sale mode (does this affect aggressive retry?)
5. Testing (can we test this? add tests?)

**If change adds >50ms**: Discuss with user, explain trade-offs

**If change removes optimizations**: Push back, explain why optimization exists

**If change adds new messages**: Require localization keys

### 14. Session Handoff Strategy

**At end of each session, ensure**:
1. All changes documented (commit message or notes)
2. Todo list updated if tasks incomplete
3. Performance impact noted if applicable
4. New patterns documented if introduced

**At start of each session, review**:
1. This CLAUDE_CONTEXT.md file
2. Recent commit history
3. Todo list status
4. LOCALIZATION.md if dealing with messages
5. TECHNICAL_GUIDE.md if user needs reference

### 15. User's Priorities (in order)

1. **Performance** - Sub-second checkout is non-negotiable
2. **Reliability** - Must work for timed sales
3. **Localization** - All messages in English + Russian
4. **User Experience** - Clear messages, error handling
5. **Code Quality** - Clean, maintainable, well-tested

**If there's a conflict**: Performance > Reliability > Everything else

### 16. reCAPTCHA Specifics (Critical to Remember)

**Site Key**: `6LcPU38UAAAAAFI0AlOu_UWIr6_etMQXCubwvnQO`

**Key Discovery**: AddCartMultiItemMutation does NOT need reCAPTCHA
- We tested this extensively
- Sending token with add-to-cart can cause issues
- Only CartValidateCartMutation needs reCAPTCHA

**Token Generation**:
- Takes 1-2 seconds
- Tokens valid ~2 minutes
- We cache for 60 seconds (safety margin)
- Stealth mode helps avoid detection

**If reCAPTCHA fails**: Script continues without token, may fail at validation step with CFUException

### 17. Cart Validation Logic Context

**Why it's complex**: Need to handle multiple edge cases:
- Empty cart (normal - add item)
- Cart with correct item already (skip add to avoid duplicate)
- Cart with correct item + store credit applied ($0 total) (skip add + credit)
- Cart with wrong item (warn user)
- Cart with multiple items (warn user)
- Cart with quantity >1 (warn user about buying multiple)

**User interaction**: If cart has issues, user sees warning and can:
- Press ENTER to continue with current cart
- Press ESC to cancel and manually edit cart

**Return value meaning**:
- `(true, nil)` means "go ahead and add to cart"
- `(false, nil)` means "skip add, cart is already correct"
- `(false, error)` means "stop, user cancelled or error occurred"

### 18. Debugging Context

**Debug Mode**: Enabled via `debug_mode: true` in config

**When debug enabled**:
- Show HTTP response status codes
- Show GraphQL request/response bodies (first few attempts)
- Show reCAPTCHA token generation details
- Show SKU slug extraction details

**Debug messages should**:
- Use `[DEBUG]` prefix
- Be localized (use `debug_*` keys)
- Only appear when `f.config.DebugMode` is true

### 19. Error Message Guidelines for Claude

**User-facing errors** (shown to user):
- MUST be localized
- Should be actionable (tell user what to do)
- Include emoji for visual clarity (⚠️ ❌ ✅)
- Example: `error_not_logged_in_detected`

**Internal errors** (for developers):
- Can be in English
- Should include technical details
- Wrap with `%w` for error chains
- Example: `"failed to marshal JSON: %w"`

**Mixed errors** (shown to user + wrapped):
- Localize the user-visible part
- Keep technical error in wrap
- Example: `fmt.Errorf(T("error_failed_create_request"), err)`

### 20. Git Workflow Context

**User commits**: User handles git operations

**When making changes**:
- Make atomic, logical changes
- Don't mix optimizations with features
- Don't mix localization with logic changes
- Keep related changes together

**If user asks to commit**:
- Follow git safety protocol from system message
- Write clear, descriptive commit messages
- Include "Generated with Claude Code" footer
- NEVER force push to main/master

### 21. Key Numbers to Remember

**Performance Targets**:
- Total checkout: <1000ms (1 second)
- Current best: ~300ms
- Current worst: ~850ms
- GraphQL query: 50-150ms typical
- Browser launch: 150-450ms (ELIMINATED via optimization)

**Retry Timings**:
- Out of stock: 5-20ms delay
- Rate limited: 50-150ms delay
- Other errors: 5-30ms delay

**reCAPTCHA**:
- Token valid: ~2 minutes
- Cache duration: 60 seconds
- Generation time: 1-2 seconds

**Timed Sale**:
- Start before sale: 5 minutes (configurable)
- Continue after sale: 10 minutes (configurable)

### 22. When User Says "Optimize"

**Checklist**:
1. Profile current performance (where's the time spent?)
2. Look for redundant operations (especially GraphQL queries)
3. Can we cache anything?
4. Can we combine operations?
5. Can we eliminate browser interactions?
6. Can we parallelize?

**Don't**:
- Prematurely optimize complex code
- Sacrifice readability for <10ms gains
- Add caching for data that changes frequently
- Remove error handling for speed

### 23. Locale System Context

**Detection**: Auto-detect from `LANG`, `LC_ALL`, `LC_MESSAGES` env vars

**Fallback**: Always fallback to `en_US` if locale missing

**Loading**: Happens at startup via `InitLocale()`

**Usage**: `T("key_name", params...)`

**Files**:
- `locale.go` - Implementation
- `locale_test.go` - Tests
- `lang/en_US.yaml` - English translations
- `lang/ru_RU.yaml` - Russian translations

**Adding language**: Create `lang/<locale>.yaml` with all keys

### 24. What Success Looks Like

**Successful session completion**:
- ✅ All tests passing (`go test -v`)
- ✅ Code compiles (`go build`)
- ✅ Performance maintained or improved
- ✅ All user-facing messages localized
- ✅ Changes documented (in code comments or this file)
- ✅ Todo list updated

**If session interrupted**:
- Update todo list with current status
- Note any in-progress work
- Document any discoveries/insights

### 25. Quick Reference - File Purposes

- `main.go` - Entry point, config loading, mode selection
- `automation.go` - Browser setup, login flow, reCAPTCHA injection
- `fast_checkout.go` - Core checkout logic, GraphQL operations
- `config.go` - Configuration struct and loading
- `locale.go` - i18n system implementation
- `locale_test.go` - i18n tests
- `fast_checkout_test.go` - Checkout logic tests
- `config.yaml` - User configuration
- `lang/*.yaml` - Translation files
- `LOCALIZATION.md` - User-facing localization guide
- `TECHNICAL_GUIDE.md` - User-facing technical reference
- `CLAUDE_CONTEXT.md` - This file (for Claude)

---

## Remember

- **Performance is sacred** - protect the sub-second target
- **User asked for it this way** - optimizations exist for a reason
- **Localization is mandatory** - no exceptions
- **Patterns exist for a reason** - follow them
- **Test your changes** - run `go test -v`
- **Document decisions** - future Claude (and user) will thank you

When in doubt, read this file first. When making changes, update this file if introducing new patterns or insights.

---

**Last Updated**: Session with critical bug fixes + error code reclassification + HTTP-based SKU validation + configurable retry delays (2025-01-27)
