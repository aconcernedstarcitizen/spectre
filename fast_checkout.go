package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
)

type FastCheckout struct {
	client           *http.Client
	config           *Config
	baseURL          string
	graphqlURL       string
	cookies          []*http.Cookie
	csrfToken        string
	userAgent        string
	cachedAddressID  string // Cached billing address for speed
	automation       *Automation // Reference to automation for login retry

	// reCAPTCHA token caching (tokens valid for 2 minutes, we refresh every 1 minute)
	cachedRecaptchaToken     string
	cachedRecaptchaTimestamp time.Time
	recaptchaMutex           sync.Mutex
}

type GraphQLRequest struct {
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
	Query         string                 `json:"query"`
}

type GraphQLResponse struct {
	Data   json.RawMessage   `json:"data"`
	Errors []GraphQLError    `json:"errors"`
}

type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []GraphQLLocation      `json:"locations"`
	Path       []interface{}          `json:"path"`
	Extensions map[string]interface{} `json:"extensions"`
	Code       string                 `json:"code"`
	Kind       string                 `json:"kind"`
	Category   string                 `json:"category"`
}

type GraphQLLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func NewFastCheckout(config *Config) (*FastCheckout, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &FastCheckout{
		client:     client,
		config:     config,
		baseURL:    "https://robertsspaceindustries.com",
		graphqlURL: "https://robertsspaceindustries.com/graphql",
	}, nil
}

func (f *FastCheckout) promptForLogin(automation *Automation) error {
	fmt.Println(T("error_not_logged_in_detected"))
	fmt.Println(T("error_not_logged_in_instructions"))
	fmt.Println(T("error_not_logged_in_step1"))
	fmt.Println(T("error_not_logged_in_step2"))
	fmt.Println(T("error_not_logged_in_step3"))
	fmt.Println()
	fmt.Print(T("error_not_logged_in_prompt"))

	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		if input == '\n' || input == '\r' {
			fmt.Println()
			fmt.Println(T("error_not_logged_in_retrying"))
			break
		}

		if input == 27 { // ESC key
			fmt.Println()
			return fmt.Errorf(T("error_not_logged_in_user_canceled"))
		}
	}

	// Reload session after login
	return f.LoadSessionFromBrowser(automation)
}

func (f *FastCheckout) LoadSessionFromBrowser(automation *Automation) error {
	fmt.Println(T("session_extracting"))

	if automation == nil || automation.page == nil {
		return fmt.Errorf(T("session_browser_not_initialized"))
	}

	// Store automation reference for login retry
	f.automation = automation

	cookies, err := automation.page.Cookies([]string{f.baseURL})
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	for _, cookie := range cookies {
		var expires time.Time
		if cookie.Expires > 0 {
			expires = time.Unix(int64(cookie.Expires), 0)
		}

		f.cookies = append(f.cookies, &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			Expires:  expires,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HTTPOnly,
		})
	}

	fmt.Printf(T("session_cookies_extracted")+"\n", len(f.cookies))

	csrfToken, err := automation.page.Eval(`() => {
		const meta = document.querySelector('meta[name="csrf-token"]');
		if (meta) return meta.getAttribute('content');

		const cookies = document.cookie.split(';');
		for (const cookie of cookies) {
			const [name, value] = cookie.trim().split('=');
			if (name === 'Rsi-XSRF' || name === 'XSRF-TOKEN') {
				return value;
			}
		}
		return null;
	}`)
	if err == nil && csrfToken.Value.Str() != "" {
		f.csrfToken = csrfToken.Value.Str()
		fmt.Printf(T("session_csrf_extracted")+"\n", f.csrfToken[:16])
	} else {
		fmt.Println(T("session_csrf_not_found"))
	}

	userAgentResult, err := automation.page.Eval(`() => navigator.userAgent`)
	if err == nil && userAgentResult.Value.Str() != "" {
		f.userAgent = userAgentResult.Value.Str()
	} else {
		f.userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	}

	return nil
}

func (f *FastCheckout) GetSKUSlugFromURL(itemURL string) (string, error) {
	fmt.Printf(T("sku_extracting_from_url")+"\n", itemURL)

	req, err := http.NewRequest("GET", itemURL, nil)
	if err != nil {
		return "", fmt.Errorf(T("error_failed_create_request"), err)
	}

	req.Header.Set("User-Agent", f.userAgent)

	for _, cookie := range f.cookies {
		req.AddCookie(cookie)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf(T("error_failed_fetch_product"), err)
	}
	defer resp.Body.Close()

	if f.config.DebugMode {
		fmt.Printf(T("debug_product_page_status")+"\n", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf(T("error_failed_read_response"), err)
	}

	if f.config.DebugMode {
		fmt.Printf(T("debug_product_page_length")+"\n", len(body))
	}

	re := regexp.MustCompile(`"skuSlug":\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		fmt.Printf(T("sku_found_slug")+"\n", matches[1])
		return matches[1], nil
	}

	return "", fmt.Errorf(T("sku_could_not_find"))
}

func (f *FastCheckout) GetSKUIDFromSlug(skuSlug string) (string, error) {
	fmt.Println(T("sku_converting_slug"))

	query := `query GetSkuQuery($slug: String!, $storeFront: String!) {
  store(name: $storeFront) {
    listing(slug: $slug) {
      skus {
        id
        title
      }
    }
  }
}`

	request := []GraphQLRequest{
		{
			OperationName: "GetSkuQuery",
			Variables: map[string]interface{}{
				"slug":       skuSlug,
				"storeFront": "pledge",
			},
			Query: query,
		},
	}

	resp, err := f.graphqlRequestWithLoginRetry(request)
	if err != nil {
		return "", fmt.Errorf(T("error_failed_query_sku"), err)
	}

	var responses []struct {
		Data struct {
			Store struct {
				Listing struct {
					Skus []struct {
						ID    json.Number `json:"id"`
						Title string      `json:"title"`
					} `json:"skus"`
				} `json:"listing"`
			} `json:"store"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(resp), &responses); err != nil {
		return "", fmt.Errorf(T("error_failed_parse_sku"), err)
	}

	if len(responses) == 0 || len(responses[0].Data.Store.Listing.Skus) == 0 {
		return "", fmt.Errorf(T("sku_no_sku_found"), skuSlug)
	}

	skuID := responses[0].Data.Store.Listing.Skus[0].ID.String()
	fmt.Printf(T("sku_id_found_with_title")+"\n", skuID, responses[0].Data.Store.Listing.Skus[0].Title)

	return skuID, nil
}

func (f *FastCheckout) GetSKUFromURL(itemURL string) (string, error) {
	skuSlug, err := f.GetSKUSlugFromURL(itemURL)
	if err != nil {
		return "", err
	}

	skuID, err := f.GetSKUIDFromSlug(skuSlug)
	if err != nil {
		return "", err
	}

	return skuID, nil
}

// GetSKUFromActivePage extracts SKU ID from the SKU slug (cached from before login)
// The slug was already extracted and validated before login, so we just convert it to ID
// If not cached, falls back to HTTP-based extraction (no incognito browser needed)
func (f *FastCheckout) GetSKUFromActivePage(automation *Automation) (string, error) {
	// Use cached SKU slug if available (extracted before login via HTTP)
	if automation != nil && automation.cachedSKU != "" {
		fmt.Printf(T("sku_using_cached_slug")+"\n", automation.cachedSKU)
		return f.getSKUIDFromSlug(automation.cachedSKU)
	}

	// Fallback: Extract fresh using HTTP if not cached
	fmt.Println(T("sku_extracting_via_http"))

	if automation == nil || automation.page == nil {
		return "", fmt.Errorf(T("sku_browser_not_available"))
	}

	// Get current URL from the active page
	currentURL, err := automation.page.Eval(`() => window.location.href`)
	if err != nil || currentURL.Value.Str() == "" {
		return "", fmt.Errorf(T("sku_could_not_get_url"))
	}
	itemURL := currentURL.Value.Str()

	// Use HTTP client to fetch page (works regardless of login state)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", itemURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers to mimic browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page via HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("page returned HTTP %d: %s", resp.StatusCode, itemURL)
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read page body: %w", err)
	}
	htmlContent := string(bodyBytes)

	// Extract SKU slug using regex
	re := regexp.MustCompile(`"skuSlug":\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(htmlContent)

	if len(matches) > 1 {
		skuSlugStr := matches[1]
		fmt.Printf(T("sku_extracted_http")+"\n", skuSlugStr)
		// Cache for future use
		if automation != nil {
			automation.cachedSKU = skuSlugStr
		}
		return f.getSKUIDFromSlug(skuSlugStr)
	}

	return "", fmt.Errorf(T("sku_slug_not_found"))
}

func (f *FastCheckout) getSKUIDFromSlug(skuSlugStr string) (string, error) {
	fmt.Printf(T("sku_querying_for_slug")+"\n", skuSlugStr)

	if f.config.DebugMode {
		fmt.Printf(T("debug_sku_slug_details")+"\n", skuSlugStr, len(skuSlugStr))
	}

	query := `query GetSkus($query: SearchQuery!) {
  store(name: "pledge", browse: true) {
    search(query: $query) {
      resources {
        id
        slug
        __typename
      }
      __typename
    }
    __typename
  }
}`

	request := []GraphQLRequest{
		{
			OperationName: "GetSkus",
			Variables: map[string]interface{}{
				"query": map[string]interface{}{
					"skus": map[string]interface{}{
						"slugs": []string{skuSlugStr},
					},
				},
			},
			Query: query,
		},
	}

	resp, err := f.graphqlRequestWithLoginRetry(request)
	if err != nil {
		return "", fmt.Errorf(T("error_getskus_failed"), err)
	}

	var responses []struct {
		Data struct {
			Store struct {
				Search struct {
					Resources []struct {
						ID string `json:"id"`
					} `json:"resources"`
				} `json:"search"`
			} `json:"store"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(resp), &responses); err != nil {
		return "", fmt.Errorf(T("error_failed_parse_getskus"), err)
	}

	if len(responses) == 0 || len(responses[0].Data.Store.Search.Resources) == 0 {
		return "", fmt.Errorf(T("sku_no_sku_found"), skuSlugStr)
	}

	skuID := responses[0].Data.Store.Search.Resources[0].ID
	fmt.Printf(T("sku_id_found")+"\n", skuID)

	return skuID, nil
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "rate-limit") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "throttle")
}

func isOutOfStockError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "out of stock") ||
		strings.Contains(errStr, "not available") ||
		strings.Contains(errStr, "unavailable")
}

func isPaymentAuthError(err error) (is4226 bool, is4227 bool) {
	if err == nil {
		return false, false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "4226"), strings.Contains(errStr, "4227")
}

func isCaptchaError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "CFUException") ||
		strings.Contains(errStr, "captcha") ||
		strings.Contains(errStr, "CAPTCHA")
}

func isNotLoggedInError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "TyUnknownCustomerException") ||
		strings.Contains(errStr, "Customer not logged in") ||
		strings.Contains(errStr, "not logged in") ||
		strings.Contains(errStr, "authentication required") ||
		strings.Contains(errStr, "unauthorized")
}

func (f *FastCheckout) simulateHumanBehavior(page *rod.Page) {
	// OPTIMIZED: Minimal human simulation for maximum speed
	// Reduced from ~600ms average to ~50ms while maintaining basic variety
	movements := 1 + rand.Intn(2) // Reduced from 3-6 to 1-2 movements
	for i := 0; i < movements; i++ {
		x := rand.Intn(800) + 100
		y := rand.Intn(600) + 100

		page.Eval(fmt.Sprintf(`() => {
			var event = new MouseEvent('mousemove', {
				view: window,
				bubbles: true,
				cancelable: true,
				clientX: %d,
				clientY: %d
			});
			document.dispatchEvent(event);
		}`, x, y))

		time.Sleep(time.Duration(5+rand.Intn(10)) * time.Millisecond) // Reduced from 30-100ms to 5-15ms
	}

	// Quick scroll
	scrollAmount := 50 + rand.Intn(100)
	page.Eval(fmt.Sprintf(`() => window.scrollBy(0, %d)`, scrollAmount))

	time.Sleep(time.Duration(5+rand.Intn(10)) * time.Millisecond) // Reduced from 100-300ms to 5-15ms

	// Quick click
	page.Eval(`() => {
		var event = new Event('mousedown', {bubbles: true});
		document.dispatchEvent(event);
	}`)

	time.Sleep(time.Duration(5+rand.Intn(10)) * time.Millisecond) // Reduced from 50-150ms to 5-15ms

	page.Eval(`() => {
		var event = new Event('mouseup', {bubbles: true});
		document.dispatchEvent(event);
	}`)

	time.Sleep(time.Duration(10+rand.Intn(20)) * time.Millisecond) // Reduced from 150-400ms to 10-30ms
}

func (f *FastCheckout) GetRecaptchaToken(automation *Automation, action string) (string, error) {
	if f.config.RecaptchaSiteKey == "" {
		return "", nil
	}

	if automation == nil || automation.browser == nil {
		return "", nil
	}

	page := automation.page
	if page == nil {
		return "", nil
	}

	checkReady, err := page.Eval(`() => typeof grecaptcha !== 'undefined' && typeof grecaptcha.enterprise !== 'undefined'`)
	if err != nil || !checkReady.Value.Bool() {
		return "", fmt.Errorf(T("error_recaptcha_not_loaded"))
	}

	f.simulateHumanBehavior(page)


	startScript := fmt.Sprintf(`() => {
		window.__specterToken = null;
		window.__specterError = null;
		window.__specterDebug = 'init';
		window.__specterCallbackInvoked = false;

		console.log('[Specter] Starting reCAPTCHA token generation...');
		console.log('[Specter] Site key:', '%s');
		console.log('[Specter] Action:', '%s');

		grecaptcha.enterprise.ready(function() {
			console.log('[Specter] grecaptcha.enterprise ready, calling execute...');
			window.__specterDebug = 'ready';

			try {
				grecaptcha.enterprise.execute('%s', {action: '%s'}).then(function(token) {
				window.__specterCallbackInvoked = true;
				console.log('[Specter] ===== reCAPTCHA CALLBACK =====');
				console.log('[Specter] Token type:', typeof token);
				console.log('[Specter] Token value:', token);
				console.log('[Specter] Token is null?', token === null);
				console.log('[Specter] Token is undefined?', token === undefined);
				console.log('[Specter] Token length:', token ? token.length : 'N/A');

				if (token && typeof token === 'string' && token.length > 10) {
					console.log('[Specter] ✓ Valid token received:', token.substring(0, 50) + '...');
					window.__specterDebug = 'success';
					window.__specterToken = token;
				} else {
					console.log('[Specter] ✗ Invalid/null token - likely low reCAPTCHA score (automation detected)');
					window.__specterDebug = 'null_token';
					window.__specterError = 'reCAPTCHA returned null/empty token (low score - automation detected)';
				}
			}).catch(function(err) {
				window.__specterCallbackInvoked = true;
				console.log('[Specter] ===== reCAPTCHA ERROR (Promise rejected) =====');
				console.log('[Specter] Error:', err);
				console.log('[Specter] Error toString:', err.toString());
				console.log('[Specter] Error message:', err.message);
				console.log('[Specter] Error stack:', err.stack);
				window.__specterDebug = 'error: ' + err.toString();
				window.__specterError = err.toString();
			});
			} catch (syncErr) {
				window.__specterCallbackInvoked = true;
				console.log('[Specter] ===== reCAPTCHA ERROR (Synchronous) =====');
				console.log('[Specter] Caught synchronous error:', syncErr);
				console.log('[Specter] Error toString:', syncErr.toString());
				console.log('[Specter] Error message:', syncErr.message);
				console.log('[Specter] Error stack:', syncErr.stack);
				window.__specterDebug = 'sync_error: ' + syncErr.toString();
				window.__specterError = syncErr.toString();
			}
		});

		return 'started';
	}`, f.config.RecaptchaSiteKey, action, f.config.RecaptchaSiteKey, action)

	_, err = page.Eval(startScript)
	if err != nil {
		return "", fmt.Errorf(T("error_recaptcha_execution_failed"), err)
	}

	// OPTIMIZED: Faster token checking for speed
	maxAttempts := 30 // 30 attempts at 50ms = 1.5 seconds max
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(50 * time.Millisecond) // Reduced from 200ms to 50ms for faster polling

		debugState, _ := page.Eval(`() => window.__specterDebug`)
		debugStr := ""
		if debugState != nil {
			debugStr = debugState.Value.Str()
		}

		if debugStr == "success" {
			checkToken, err := page.Eval(`() => {
				if (window.__specterToken && typeof window.__specterToken === 'string') {
					return window.__specterToken;
				}
				return '';
			}`)
			if err == nil {
				tokenStr := checkToken.Value.Str()
				if tokenStr != "" && len(tokenStr) > 100 {
					if f.config.DebugMode {
						tokenPreview := tokenStr
						if len(tokenStr) > 50 {
							tokenPreview = tokenStr[:50] + "..."
						}
						fmt.Printf(T("debug_recaptcha_token_success")+"\n", tokenPreview, len(tokenStr))
					}
					return tokenStr, nil
				} else if f.config.DebugMode && i == 0 {
					fmt.Printf(T("debug_recaptcha_token_invalid")+"\n", tokenStr, len(tokenStr))
				}
			}
		} else if f.config.DebugMode && i == 0 {
			fmt.Printf(T("debug_recaptcha_waiting_state")+"\n", debugStr)
		}

		if f.config.DebugMode && i == 0 {
			tokenTypeCheck, _ := page.Eval(`() => typeof window.__specterToken`)
			tokenType := "unknown"
			if tokenTypeCheck != nil {
				tokenType = tokenTypeCheck.Value.Str()
			}

			callbackCheck, _ := page.Eval(`() => window.__specterCallbackInvoked`)
			callbackInvoked := false
			if callbackCheck != nil {
				callbackInvoked = callbackCheck.Value.Bool()
			}

			fmt.Printf(T("debug_recaptcha_token_check")+"\n",
				i, debugStr, tokenType, callbackInvoked)
		}

		checkError, err := page.Eval(`() => {
			if (window.__specterError && typeof window.__specterError === 'string' && window.__specterError.length > 0) {
				return window.__specterError;
			}
			return '';
		}`)
		if err == nil && checkError.Value.Str() != "" {
			errMsg := checkError.Value.Str()
			return "", fmt.Errorf(T("error_recaptcha_execution_error"), errMsg)
		}
	}

	debugState, _ := page.Eval(`() => window.__specterDebug`)
	debugStr := "unknown"
	if debugState != nil {
		debugStr = debugState.Value.Str()
	}

	if debugStr == "null_token" {
		return "", fmt.Errorf(T("error_recaptcha_automation_detected"))
	}

	return "", fmt.Errorf(T("error_recaptcha_timeout_debug"), debugStr)
}

// GetOrRefreshCachedRecaptchaToken returns a cached reCAPTCHA token if it's less than 60 seconds old,
// otherwise generates a fresh token and caches it. This improves performance during aggressive retry loops.
func (f *FastCheckout) GetOrRefreshCachedRecaptchaToken(automation *Automation, action string) (string, error) {
	f.recaptchaMutex.Lock()
	defer f.recaptchaMutex.Unlock()

	now := time.Now()
	tokenAge := now.Sub(f.cachedRecaptchaTimestamp)

	// Use cached token if it's less than 60 seconds old (well under 2-minute expiration)
	if f.cachedRecaptchaToken != "" && tokenAge < 60*time.Second {
		if f.config.DebugMode {
			fmt.Printf(T("debug_recaptcha_reusing")+"\n",
				tokenAge.Round(time.Second),
				(60*time.Second - tokenAge).Round(time.Second))
		}
		return f.cachedRecaptchaToken, nil
	}

	// Token is missing or expired (>60s) - generate fresh token
	if f.cachedRecaptchaToken != "" {
		fmt.Printf(T("recaptcha_expired_generating")+"\n", tokenAge.Round(time.Second))
	} else {
		fmt.Println(T("recaptcha_generating_initial"))
	}

	token, err := f.GetRecaptchaToken(automation, action)
	if err != nil {
		return "", err
	}

	// Cache the new token with current timestamp
	f.cachedRecaptchaToken = token
	f.cachedRecaptchaTimestamp = now

	if token != "" && len(token) > 10 {
		tokenPreview := token
		if len(token) > 30 {
			tokenPreview = token[:30] + "..."
		}
		fmt.Printf(T("recaptcha_cached_fresh")+"\n", tokenPreview)
	}

	return token, nil
}

func (f *FastCheckout) AddToCart(skuID string, automation *Automation) error {
	fmt.Println(T("cart_adding_api_retry"))
	fmt.Printf(T("cart_debug_sku_id")+"\n", skuID)

	retryDuration := f.config.RetryDurationSeconds
	fmt.Printf(T("cart_will_retry_seconds")+"\n", retryDuration)

	startTime := time.Now()
	retryDeadline := startTime.Add(time.Duration(retryDuration) * time.Second)
	attemptNum := 0

	mutation := `mutation AddCartMultiItemMutation($query: [CartAddInput!]) {
  store(name: "pledge") {
    cart {
      mutations {
        addMany(query: $query) {
          count
          resources {
            id
            title
            __typename
          }
          __typename
        }
        __typename
      }
      __typename
    }
    __typename
  }
}`

	for {
		attemptNum++
		attemptStart := time.Now()

		tokenChan := make(chan string, 1)
		tokenErrChan := make(chan error, 1)

		go func() {
			token, err := f.GetRecaptchaToken(automation, f.config.RecaptchaAction)
			if err != nil {
				tokenErrChan <- err
			} else {
				tokenChan <- token
			}
		}()

		var recaptchaToken string
		select {
		case token := <-tokenChan:
			recaptchaToken = token
			if f.config.DebugMode && attemptNum <= 3 {
				if recaptchaToken != "" && recaptchaToken != "null" && recaptchaToken != "undefined" {
					tokenPreview := recaptchaToken
					if len(recaptchaToken) > 50 {
						tokenPreview = recaptchaToken[:50] + "..."
					}
					fmt.Printf(T("debug_attempt_got_token")+"\n", attemptNum, tokenPreview, len(recaptchaToken))
				} else {
					fmt.Printf(T("debug_attempt_invalid_token")+"\n", attemptNum, recaptchaToken)
				}
			}
		case err := <-tokenErrChan:
			if f.config.DebugMode && attemptNum <= 3 {
				fmt.Printf(T("debug_attempt_error")+"\n", attemptNum, err)
			}
			if strings.Contains(err.Error(), "automation detected") && attemptNum == 1 {
				fmt.Println(T("recaptcha_warning_automation_detected"))
				fmt.Println(T("recaptcha_warning_may_fail"))
			}
		case <-time.After(5 * time.Second):
			if f.config.DebugMode && attemptNum <= 3 {
				fmt.Printf(T("debug_attempt_timeout")+"\n", attemptNum)
			}
		}

		// Match the exact structure from manual browser add-to-cart
		variables := map[string]interface{}{
			"query": []map[string]interface{}{
				{
					"qty":   1,
					"skuId": skuID, // Keep as string, like browser does
				},
			},
		}

		if f.config.DebugMode && attemptNum <= 3 {
			if recaptchaToken != "" && recaptchaToken != "null" && recaptchaToken != "undefined" && len(recaptchaToken) > 10 {
				fmt.Printf(T("debug_attempt_not_using_token")+"\n", attemptNum, len(recaptchaToken))
			} else {
				fmt.Printf(T("debug_attempt_no_token")+"\n", attemptNum)
			}
		}

		request := []GraphQLRequest{
			{
				OperationName: "AddCartMultiItemMutation",
				Variables:     variables,
				Query:         mutation,
			},
		}

		if f.config.DebugMode && attemptNum == 1 {
			jsonData, _ := json.MarshalIndent(request, "", "  ")
			fmt.Printf(T("debug_request_body")+"\n", string(jsonData))
		}

		resp, err := f.graphqlRequestWithLoginRetry(request)

		if f.config.DebugMode && attemptNum <= 3 {
			fmt.Printf(T("debug_attempt_response")+"\n", attemptNum, resp)
			if err != nil {
				fmt.Printf(T("debug_attempt_error_details")+"\n", attemptNum, err)
			}
		}

		if err == nil {
			elapsed := time.Since(startTime)
			fmt.Println(T("cart_added_successfully"))
			if attemptNum > 1 {
				fmt.Printf(T("cart_success_after_attempts")+"\n", attemptNum, elapsed)
			}
			return nil
		}

		remaining := retryDeadline.Sub(time.Now())

		if remaining <= 0 {
			elapsed := time.Since(startTime)
			fmt.Printf(T("cart_sale_window_expired")+"\n", attemptNum, elapsed)
			return fmt.Errorf(T("error_add_cart_attempts"), attemptNum, err)
		}

		var delay time.Duration
		is4226, is4227 := isPaymentAuthError(err)
		isRateLimited := isRateLimitError(err)
		isOutOfStock := isOutOfStockError(err)
		isCaptchaFail := isCaptchaError(err)

		if is4227 {
			// Payment auth error 4227 - configurable backoff
			delayMs := f.config.Payment4227MinMs + rand.Intn(f.config.Payment4227MaxMs-f.config.Payment4227MinMs+1)
			delay = time.Duration(delayMs) * time.Millisecond
			if attemptNum%10 == 0 || attemptNum <= 3 {
				fmt.Printf(T("cart_payment_auth_4227_retry")+"\n",
					attemptNum, delayMs, remaining.Round(time.Second))
			}
		} else if is4226 {
			// Payment auth error 4226 - configurable backoff
			delayMs := f.config.Payment4226MinMs + rand.Intn(f.config.Payment4226MaxMs-f.config.Payment4226MinMs+1)
			delay = time.Duration(delayMs) * time.Millisecond
			if attemptNum%10 == 0 || attemptNum <= 3 {
				fmt.Printf(T("cart_payment_auth_4226_retry")+"\n",
					attemptNum, delayMs, remaining.Round(time.Second))
			}
		} else if isCaptchaFail {
			// Minimal delay for CAPTCHA - just retry immediately with new token
			delayMs := 5 + rand.Intn(15) // 5-20ms - extremely fast
			delay = time.Duration(delayMs) * time.Millisecond

			if attemptNum%10 == 0 || attemptNum <= 3 {
				fmt.Printf(T("cart_captcha_fast_retry")+"\n",
					attemptNum, delayMs, remaining.Round(time.Second))
			}
		} else if isRateLimited {
			// Rate limit handling - configurable
			delayMs := f.config.RateLimitMinMs + rand.Intn(f.config.RateLimitMaxMs-f.config.RateLimitMinMs+1)
			delay = time.Duration(delayMs) * time.Millisecond
			fmt.Printf(T("cart_rate_limited_retry")+"\n",
				attemptNum, delayMs, remaining.Round(time.Second))
		} else if isOutOfStock {
			// Out of stock - configurable delay
			delay = time.Duration(f.config.OutOfStockDelayMs) * time.Millisecond

			if attemptNum%10 == 0 {
				fmt.Printf(T("cart_out_of_stock_retry")+"\n",
					attemptNum, remaining.Round(time.Second))
			}
		} else {
			// Generic/other errors - configurable delay
			delay = time.Duration(f.config.GenericErrorDelayMs) * time.Millisecond

			attemptDuration := time.Since(attemptStart)
			if attemptNum <= 5 || attemptNum%20 == 0 {
				fmt.Printf(T("cart_attempt_failed_retry")+"\n",
					attemptNum, attemptDuration, f.config.GenericErrorDelayMs, remaining.Round(time.Second))
			}
		}

		if delay > remaining {
			delay = remaining
		}

		time.Sleep(delay)
	}
}

func (f *FastCheckout) GetCartTotals() (cartTotal float64, maxCredit float64, err error) {
	fmt.Println(T("cart_querying_totals"))

	query := `query CartSummaryViewQuery($storeFront: String) {
  store(name: $storeFront) {
    cart {
      totals {
        total
        credits {
          amount
          maxApplicable
        }
      }
    }
  }
  customer {
    ledger(ledgerCode: "credit") {
      amount {
        value
      }
    }
  }
}`

	request := []GraphQLRequest{
		{
			OperationName: "CartSummaryViewQuery",
			Variables: map[string]interface{}{
				"storeFront": "pledge",
			},
			Query: query,
		},
	}

	resp, err := f.graphqlRequestWithLoginRetry(request)
	if err != nil {
		return 0, 0, fmt.Errorf(T("error_failed_query_cart_totals"), err)
	}

	var responses []map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &responses); err != nil {
		return 0, 0, fmt.Errorf(T("error_failed_parse_cart_totals"), err)
	}

	if len(responses) == 0 {
		return 0, 0, fmt.Errorf(T("error_empty_cart_totals"))
	}

	data := responses[0]["data"].(map[string]interface{})
	store := data["store"].(map[string]interface{})
	cart := store["cart"].(map[string]interface{})
	totals := cart["totals"].(map[string]interface{})

	cartTotalCents := totals["total"].(float64)
	cartTotal = cartTotalCents / 100.0

	credits := totals["credits"].(map[string]interface{})
	maxCreditCents := credits["maxApplicable"].(float64)
	maxCredit = maxCreditCents / 100.0

	customer := data["customer"].(map[string]interface{})
	ledger := customer["ledger"].(map[string]interface{})
	ledgerAmount := ledger["amount"].(map[string]interface{})
	availableCreditCents := ledgerAmount["value"].(float64)
	availableCredit := availableCreditCents / 100.0

	fmt.Printf(T("cart_totals_result")+"\n", cartTotal)
	fmt.Printf(T("cart_available_credit_result")+"\n", availableCredit)
	fmt.Printf(T("cart_max_credit_result")+"\n", maxCredit)

	return cartTotal, maxCredit, nil
}

type CartItem struct {
	Name     string
	Price    float64
	SKUID    string
	Quantity int
}

type CartInfo struct {
	Total      float64
	MaxCredit  float64
	Items      []CartItem
}

// GetCartTotalsAndItems combines GetCartTotals and GetCartItems into a single query
// for performance optimization (saves 50-150ms per call)
func (f *FastCheckout) GetCartTotalsAndItems() (*CartInfo, error) {
	query := `query CombinedCartQuery($storeFront: String) {
  store(name: $storeFront) {
    cart {
      totals {
        total
        credits {
          amount
          maxApplicable
        }
      }
      lineItems {
        id
        skuId
        sku {
          title
        }
        unitPriceWithTax {
          amount
        }
        qty
      }
    }
  }
  customer {
    ledger(ledgerCode: "credit") {
      amount {
        value
      }
    }
  }
}`

	request := []GraphQLRequest{
		{
			OperationName: "CombinedCartQuery",
			Variables: map[string]interface{}{
				"storeFront": "pledge",
			},
			Query: query,
		},
	}

	resp, err := f.graphqlRequestWithLoginRetry(request)
	if err != nil {
		return nil, fmt.Errorf(T("error_failed_query_cart_info"), err)
	}

	var responses []struct {
		Data struct {
			Store struct {
				Cart struct {
					Totals struct {
						Total   float64 `json:"total"`
						Credits struct {
							Amount        float64 `json:"amount"`
							MaxApplicable float64 `json:"maxApplicable"`
						} `json:"credits"`
					} `json:"totals"`
					LineItems []struct {
						ID     string      `json:"id"`
						SkuID  json.Number `json:"skuId"`
						Sku    struct {
							Title string `json:"title"`
						} `json:"sku"`
						UnitPriceWithTax struct {
							Amount float64 `json:"amount"`
						} `json:"unitPriceWithTax"`
						Qty int `json:"qty"`
					} `json:"lineItems"`
				} `json:"cart"`
			} `json:"store"`
			Customer struct {
				Ledger struct {
					Amount struct {
						Value float64 `json:"value"`
					} `json:"amount"`
				} `json:"ledger"`
			} `json:"customer"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(resp), &responses); err != nil {
		return nil, fmt.Errorf(T("error_failed_parse_cart_info"), err)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf(T("error_empty_cart_info"))
	}

	data := responses[0].Data
	cart := data.Store.Cart
	totals := cart.Totals

	cartTotal := totals.Total / 100.0
	maxCredit := totals.Credits.MaxApplicable / 100.0

	items := make([]CartItem, 0, len(cart.LineItems))
	for _, lineItem := range cart.LineItems {
		skuIDStr := string(lineItem.SkuID)
		price := lineItem.UnitPriceWithTax.Amount / 100.0

		items = append(items, CartItem{
			Name:     lineItem.Sku.Title,
			Price:    price,
			SKUID:    skuIDStr,
			Quantity: lineItem.Qty,
		})
	}

	return &CartInfo{
		Total:     cartTotal,
		MaxCredit: maxCredit,
		Items:     items,
	}, nil
}

func (f *FastCheckout) GetCartItems() ([]CartItem, error) {
	query := `query StepperQuery($storeFront: String) {
  store(name: $storeFront) {
    cart {
      lineItems {
        id
        skuId
        sku {
          title
        }
        unitPriceWithTax {
          amount
        }
        qty
      }
    }
  }
}`

	request := []GraphQLRequest{
		{
			OperationName: "StepperQuery",
			Variables: map[string]interface{}{
				"storeFront": "pledge",
			},
			Query: query,
		},
	}

	resp, err := f.graphqlRequestWithLoginRetry(request)
	if err != nil {
		return nil, fmt.Errorf(T("error_failed_query_cart_items"), err)
	}

	var responses []struct {
		Data struct {
			Store struct {
				Cart struct {
					LineItems []struct {
						ID     string      `json:"id"`
						SkuID  json.Number `json:"skuId"`
						Sku    struct {
							Title string `json:"title"`
						} `json:"sku"`
						UnitPriceWithTax struct {
							Amount float64 `json:"amount"`
						} `json:"unitPriceWithTax"`
						Qty int `json:"qty"`
					} `json:"lineItems"`
				} `json:"cart"`
			} `json:"store"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(resp), &responses); err != nil {
		return nil, fmt.Errorf(T("error_failed_parse_cart_items"), err)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf(T("error_empty_cart_items"))
	}

	items := []CartItem{}
	for _, lineItem := range responses[0].Data.Store.Cart.LineItems {
		items = append(items, CartItem{
			Name:     lineItem.Sku.Title,
			Price:    lineItem.UnitPriceWithTax.Amount / 100.0, // Convert cents to dollars
			SKUID:    lineItem.SkuID.String(),
			Quantity: lineItem.Qty,
		})
	}

	return items, nil
}

// ValidateCartContents checks cart state and returns:
// - (true, nil): Cart is valid, safe to add item to cart
// - (false, nil): Cart has issues but user chose to continue with current contents (skip adding)
// - (false, error): User cancelled or validation error occurred
//
// OPTIMIZATION: Now accepts items as parameter to avoid redundant GraphQL query
func (f *FastCheckout) ValidateCartContents(expectedSKUID string, cartTotal float64, items []CartItem) (bool, error) {

	// Empty cart is normal - proceed with adding
	if len(items) == 0 {
		fmt.Println(T("cart_empty_will_add"))
		return true, nil // Add to cart
	}

	// Check if cart already has exactly what we want: 1 item, correct SKU, quantity=1
	if len(items) == 1 && items[0].SKUID == expectedSKUID && items[0].Quantity == 1 {
		expectedTotal := items[0].Price
		if cartTotal == expectedTotal {
			// Perfect! Cart already has correct item at full price, don't add again
			fmt.Printf(T("cart_already_contains_target")+"\n", items[0].Name, items[0].Price)
			fmt.Println(T("cart_skip_duplicate"))
			return false, nil // Don't add, proceed with existing cart
		} else if cartTotal == 0 {
			// Cart total is $0 - store credit already applied from previous run
			fmt.Printf(T("cart_already_contains_target")+"\n", items[0].Name, items[0].Price)
			fmt.Println(T("cart_credit_already_applied"))
			fmt.Println(T("cart_skip_add_and_credit"))
			return false, nil // Don't add, proceed with existing cart
		}
		// If price doesn't match and isn't $0, fall through to show warning
	}

	// Cart has issues - multiple items, wrong items, quantity > 1, or price mismatch

	// Cart has issues - either multiple items, wrong items, or quantity > 1
	fmt.Println()
	fmt.Println(T("cart_warning_header"))
	fmt.Println()

	// Calculate total items across all line items (considering quantities)
	totalItems := 0
	for _, item := range items {
		totalItems += item.Quantity
	}

	if len(items) == 1 {
		fmt.Printf(T("cart_warning_single_quantity")+"\n\n", items[0].Quantity, items[0].Name)
	} else {
		fmt.Printf(T("cart_warning_multiple_items")+"\n\n", totalItems, len(items))
	}

	for i, item := range items {
		marker := "  "
		if item.SKUID == expectedSKUID {
			marker = "→ "
		}

		if item.Quantity > 1 {
			fmt.Printf("%s%d. %s (Quantity: %d)\n", marker, i+1, item.Name, item.Quantity)
		} else {
			fmt.Printf("%s%d. %s\n", marker, i+1, item.Name)
		}

		fmt.Printf(T("cart_item_price_line")+"\n", item.Price, item.Quantity, item.Price*float64(item.Quantity))

		if item.SKUID == expectedSKUID {
			fmt.Println(T("cart_item_target_marker"))
			if item.Quantity > 1 {
				fmt.Printf(T("cart_item_quantity_warning")+"\n", item.Quantity)
			}
		}
		fmt.Println()
	}

	fmt.Printf(T("cart_total_label")+"\n", cartTotal)

	// Check if cart total matches expected for single item
	if len(items) == 1 && items[0].SKUID == expectedSKUID && items[0].Quantity == 1 {
		expectedTotal := items[0].Price
		if cartTotal != expectedTotal {
			fmt.Printf(T("cart_expected_total")+"\n", expectedTotal, items[0].Name)
		}
	}
	fmt.Println()

	// Determine the warning message
	if len(items) > 1 {
		fmt.Println(T("cart_multiple_items_warning"))
	} else if len(items) == 1 && items[0].SKUID == expectedSKUID && items[0].Quantity > 1 {
		fmt.Printf(T("cart_quantity_warning")+"\n", items[0].Quantity)
		fmt.Printf(T("cart_quantity_purchase_details")+"\n",
			items[0].Quantity, items[0].Name, items[0].Price*float64(items[0].Quantity))
		fmt.Println()
		fmt.Println(T("cart_quantity_limit_note"))
	} else if len(items) == 1 && items[0].SKUID == expectedSKUID && items[0].Quantity == 1 {
		// Single correct item but cart total doesn't match
		expectedTotal := items[0].Price
		fmt.Printf(T("cart_total_mismatch")+"\n", cartTotal, expectedTotal)
		fmt.Println(T("cart_total_mismatch_reason"))
	} else {
		fmt.Println(T("wrong_item_warning"))
	}

	fmt.Println()
	fmt.Println(T("cart_options_header"))
	fmt.Println(T("cart_option_continue"))
	fmt.Println(T("cart_option_cancel"))
	fmt.Println()
	fmt.Print(T("cart_choice_prompt"))

	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadByte()
		if err != nil {
			return false, fmt.Errorf(T("cart_error_read_input"), err)
		}

		if input == '\n' || input == '\r' {
			fmt.Println()
			fmt.Println(T("cart_user_confirmed_current"))
			return false, nil // Don't add to cart, use existing cart
		}

		if input == 27 { // ESC key
			fmt.Println()
			fmt.Println(T("cart_user_canceled"))
			return false, fmt.Errorf(T("cart_error_user_canceled"))
		}
	}
}

func (f *FastCheckout) ApplyStoreCredit(amount float64) error {
	fmt.Printf(T("credit_applying_api")+"\n", amount)

	mutation := `mutation AddCreditMutation($amount: Float!, $storeFront: String) {
  store(name: $storeFront) {
    cart {
      mutations {
        credit_update(amount: $amount)
      }
      totals {
        total
        credits {
          amount
        }
      }
    }
  }
}`

	request := []GraphQLRequest{
		{
			OperationName: "AddCreditMutation",
			Variables: map[string]interface{}{
				"amount":     amount,
				"storeFront": "pledge",
			},
			Query: mutation,
		},
	}

	resp, err := f.graphqlRequestWithLoginRetry(request)
	if err != nil {
		// Check for insufficient credit error
		errStr := err.Error()
		if strings.Contains(errStr, "You don't have that many credits available") ||
			strings.Contains(errStr, "CFUValidationException") && strings.Contains(errStr, "amount:") {
			// User-friendly error message for insufficient credits
			fmt.Println()
			fmt.Println(T("credit_insufficient_error_header"))
			fmt.Println()
			fmt.Println(T("credit_insufficient_error_message"))
			fmt.Printf(T("credit_insufficient_attempted_amount")+"\n", amount)
			fmt.Println()
			fmt.Println(T("credit_insufficient_instructions"))
			fmt.Println()
			return fmt.Errorf("insufficient store credits available")
		}
		return fmt.Errorf("apply credit failed: %w", err)
	}

	fmt.Printf(T("credit_applied_response")+"\n", resp)
	return nil
}

func (f *FastCheckout) NextStep() error {
	mutation := `mutation NextStepMutation($storeFront: String) {
  store(name: $storeFront) {
    cart {
      mutations {
        flow {
          moveNext
        }
      }
      flow {
        steps {
          step
          action
          active
        }
        current {
          orderCreated
        }
      }
    }
  }
}`

	request := []GraphQLRequest{
		{
			OperationName: "NextStepMutation",
			Variables: map[string]interface{}{
				"storeFront": "pledge",
			},
			Query: mutation,
		},
	}

	resp, err := f.graphqlRequestWithLoginRetry(request)
	if err != nil {
		return fmt.Errorf("next step failed: %w", err)
	}

	var responses []struct {
		Data struct {
			Store struct {
				Cart struct {
					Flow struct {
						Steps []struct {
							Step   string `json:"step"`
							Action string `json:"action"`
							Active bool   `json:"active"`
						} `json:"steps"`
						Current struct {
							OrderCreated bool `json:"orderCreated"`
						} `json:"current"`
					} `json:"flow"`
				} `json:"cart"`
			} `json:"store"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(resp), &responses); err != nil {
		return fmt.Errorf("failed to parse NextStep response: %w", err)
	}

	if len(responses) > 0 {
		flow := responses[0].Data.Store.Cart.Flow

		activeStep := "unknown"
		for _, step := range flow.Steps {
			if step.Active {
				activeStep = step.Step
				break
			}
		}

		fmt.Printf(T("step_moved_to")+"\n", activeStep)

		if flow.Current.OrderCreated {
			fmt.Println(T("validation_order_created"))
		}
	}

	return nil
}

func (f *FastCheckout) GetDefaultBillingAddress() (string, error) {
	fmt.Println(T("address_fetching"))

	query := `query AddressBookQuery($storeFront: String) {
  store(name: $storeFront) {
    addressBook {
      id
      defaultBilling
      defaultShipping
      company
      firstname
      lastname
      addressLine
      postalCode
      phone
      city
      country {
        id
        name
        code
      }
      region {
        id
        code
        name
      }
    }
  }
}`

	request := []GraphQLRequest{
		{
			OperationName: "AddressBookQuery",
			Variables: map[string]interface{}{
				"storeFront": "pledge",
			},
			Query: query,
		},
	}

	resp, err := f.graphqlRequestWithLoginRetry(request)
	if err != nil {
		return "", fmt.Errorf("failed to query address book: %w", err)
	}

	var responses []struct {
		Data struct {
			Store struct {
				AddressBook []struct {
					ID             string `json:"id"`
					DefaultBilling bool   `json:"defaultBilling"`
					Firstname      string `json:"firstname"`
					Lastname       string `json:"lastname"`
					City           string `json:"city"`
				} `json:"addressBook"`
			} `json:"store"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(resp), &responses); err != nil {
		return "", fmt.Errorf("failed to parse address book: %w", err)
	}

	if len(responses) == 0 || len(responses[0].Data.Store.AddressBook) == 0 {
		return "", fmt.Errorf("no addresses found in address book")
	}

	var addressID string
	var addressName string
	for _, addr := range responses[0].Data.Store.AddressBook {
		if addr.DefaultBilling {
			addressID = addr.ID
			addressName = fmt.Sprintf("%s %s, %s", addr.Firstname, addr.Lastname, addr.City)
			break
		}
	}

	if addressID == "" {
		addr := responses[0].Data.Store.AddressBook[0]
		addressID = addr.ID
		addressName = fmt.Sprintf("%s %s, %s", addr.Firstname, addr.Lastname, addr.City)
	}

	fmt.Printf(T("address_found")+"\n", addressName, addressID)
	return addressID, nil
}

func (f *FastCheckout) AssignBillingAddress(addressID string) error {
	fmt.Printf(T("address_assigning")+"\n", addressID)

	mutation := `mutation CartAddressAssignMutation($billing: ID, $shipping: ID, $storeFront: String) {
  store(name: $storeFront) {
    cart {
      mutations {
        assignAddresses(assign: {billing: $billing, shipping: $shipping})
      }
      billingAddress {
        id
        firstname
        lastname
        city
      }
    }
  }
}`

	request := []GraphQLRequest{
		{
			OperationName: "CartAddressAssignMutation",
			Variables: map[string]interface{}{
				"billing":    addressID,
				"storeFront": "pledge",
			},
			Query: mutation,
		},
	}

	resp, err := f.graphqlRequestWithLoginRetry(request)
	if err != nil {
		return fmt.Errorf("failed to assign address: %w", err)
	}

	fmt.Printf(T("address_assigned")+"\n", resp)
	return nil
}

func (f *FastCheckout) ValidateCart(automation *Automation) error {
	return f.ValidateCartWithDeadline(automation, time.Time{})
}

func (f *FastCheckout) ValidateCartWithDeadline(automation *Automation, deadline time.Time) error {
	fmt.Println(T("validation_completing"))

	startTime := time.Now()

	// Use provided deadline or create one based on config
	var retryDeadline time.Time
	if !deadline.IsZero() {
		retryDeadline = deadline
		remaining := deadline.Sub(startTime)
		fmt.Printf(T("validation_retry_until_end")+"\n", remaining.Round(time.Second))
	} else {
		retryDeadline = startTime.Add(time.Duration(f.config.RetryDurationSeconds) * time.Second)
		fmt.Printf(T("validation_retry_for_seconds")+"\n", f.config.RetryDurationSeconds)
	}

	// Generate mark ONCE and reuse for all retry attempts (matches browser behavior)
	// The browser uses a constant mark throughout the validation session
	// Mark matches RSI's logic: Math.floor(Math.random() * 10000000000)
	// This generates a random integer from 0 to 9999999999 (not guaranteed to be 10 digits)
	mark := fmt.Sprintf("%d", rand.Intn(10000000000))

	attemptNum := 0

	for {
		attemptNum++
		attemptStart := time.Now()

		// Show progress for aggressive retry mode (every 50 attempts)
		remaining := retryDeadline.Sub(time.Now())
		if attemptNum == 1 || attemptNum%50 == 0 {
			if !deadline.IsZero() {
				// Timed sale mode - show time remaining in window
				fmt.Printf(T("validation_attempt_sale_window")+"\n", attemptNum, remaining.Round(time.Second))
			} else if attemptNum%50 == 0 {
				// Normal mode - only show every 50 attempts to reduce spam
				fmt.Printf(T("validation_attempt_regular")+"\n", attemptNum, remaining.Round(time.Second))
			}
		}

		// Use cached token (refreshed automatically every 60 seconds)
		recaptchaToken, err := f.GetOrRefreshCachedRecaptchaToken(automation, "store/cart/validate")
		if err != nil {
			fmt.Printf(T("validation_recaptcha_warning")+"\n", err)
		}

		mutation := `mutation CartValidateCartMutation($storeFront: String, $token: String, $mark: String) {
  store(name: $storeFront) {
    cart {
      mutations {
        validate(mark: $mark, token: $token)
        __typename
      }
      flow {
        steps {
          step
          action
          finalStep
          active
          __typename
        }
        current {
          orderCreated
          __typename
        }
        __typename
      }
      __typename
    }
    order {
      slug
      __typename
    }
    __typename
  }
}`

		variables := map[string]interface{}{
			"storeFront": "pledge",
			"mark":       mark,
		}

		if recaptchaToken != "" {
			variables["token"] = recaptchaToken
		}

		// Debug logging for validation mutation
		if f.config.DebugMode && attemptNum <= 3 {
			if recaptchaToken != "" && recaptchaToken != "null" && recaptchaToken != "undefined" && len(recaptchaToken) > 10 {
				fmt.Printf("[DEBUG] Validation Attempt %d: Using reCAPTCHA token (len=%d) with mark=%s\n", attemptNum, len(recaptchaToken), mark)
			} else {
				fmt.Printf("[DEBUG] Validation Attempt %d: No valid reCAPTCHA token! mark=%s\n", attemptNum, mark)
			}
		}

		request := []GraphQLRequest{
			{
				OperationName: "CartValidateCartMutation",
				Variables:     variables,
				Query:         mutation,
			},
		}

		if f.config.DebugMode && attemptNum == 1 {
			jsonData, _ := json.MarshalIndent(request, "", "  ")
			fmt.Printf("[DEBUG] CartValidateCartMutation request body:\n%s\n", string(jsonData))
		}

		resp, err := f.graphqlRequestWithLoginRetry(request)

		if f.config.DebugMode && attemptNum <= 3 {
			fmt.Printf("[DEBUG] Validation Attempt %d response: %s\n", attemptNum, resp)
			if err != nil {
				fmt.Printf("[DEBUG] Validation Attempt %d error: %v\n", attemptNum, err)
			}
		}

		if err == nil {
			var responses []struct {
				Data struct {
					Store struct {
						Cart struct {
							Flow struct {
								Current struct {
									OrderCreated bool `json:"orderCreated"`
								} `json:"current"`
							} `json:"flow"`
						} `json:"cart"`
						Order struct {
							Slug string `json:"slug"`
						} `json:"order"`
					} `json:"store"`
				} `json:"data"`
			}

			if err := json.Unmarshal([]byte(resp), &responses); err == nil && len(responses) > 0 {
				orderSlug := responses[0].Data.Store.Order.Slug
				orderCreated := responses[0].Data.Store.Cart.Flow.Current.OrderCreated

				fmt.Printf(T("validation_order_slug")+"\n", orderSlug)

				if orderCreated {
					fmt.Println(T("validation_order_created"))
				}

				elapsed := time.Since(startTime)
				fmt.Printf(T("validation_success_attempts")+"\n", attemptNum, elapsed)
				return nil
			}
		}

		remaining = retryDeadline.Sub(time.Now())

		if remaining <= 0 {
			elapsed := time.Since(startTime)
			if !deadline.IsZero() {
				fmt.Printf(T("validation_window_expired")+"\n", attemptNum, elapsed)
				return fmt.Errorf("cart validation failed after %d attempts - sale window expired: %w", attemptNum, err)
			} else {
				fmt.Printf(T("validation_timeout")+"\n", attemptNum, elapsed)
				return fmt.Errorf("cart validation failed after %d attempts: %w", attemptNum, err)
			}
		}

		var delay time.Duration
		is4226, is4227 := isPaymentAuthError(err)
		isOutOfStock := isOutOfStockError(err)

		if is4227 {
			// Payment auth error 4227 - configurable backoff
			delayMs := f.config.Payment4227MinMs + rand.Intn(f.config.Payment4227MaxMs-f.config.Payment4227MinMs+1)
			delay = time.Duration(delayMs) * time.Millisecond
			if attemptNum%10 == 0 || attemptNum <= 3 {
				fmt.Printf(T("validation_payment_auth_4227")+"\n",
					attemptNum, delayMs, remaining.Round(time.Second))
			}
		} else if is4226 {
			// Payment auth error 4226 - configurable backoff
			delayMs := f.config.Payment4226MinMs + rand.Intn(f.config.Payment4226MaxMs-f.config.Payment4226MinMs+1)
			delay = time.Duration(delayMs) * time.Millisecond
			if attemptNum%10 == 0 || attemptNum <= 3 {
				fmt.Printf(T("validation_payment_auth_4226")+"\n",
					attemptNum, delayMs, remaining.Round(time.Second))
			}
		} else if isOutOfStock {
			// Out of stock - configurable delay
			delay = time.Duration(f.config.OutOfStockDelayMs) * time.Millisecond

			if attemptNum%10 == 0 {
				fmt.Printf(T("validation_out_of_stock")+"\n",
					attemptNum, remaining.Round(time.Second))
			}
		} else {
			// Generic/other errors - configurable delay
			delay = time.Duration(f.config.GenericErrorDelayMs) * time.Millisecond

			attemptDuration := time.Since(attemptStart)
			if attemptNum <= 5 || attemptNum%20 == 0 {
				fmt.Printf(T("validation_failed_retry")+"\n",
					attemptNum, attemptDuration, f.config.GenericErrorDelayMs, remaining.Round(time.Second))
			}
		}

		if delay > remaining {
			delay = remaining
		}

		time.Sleep(delay)
	}
}

// graphqlRequestWithLoginRetry wraps graphqlRequest and handles "not logged in" errors
// by prompting the user to login and retrying the request
func (f *FastCheckout) graphqlRequestWithLoginRetry(requests []GraphQLRequest) (string, error) {
	result, err := f.graphqlRequest(requests)

	// If we get a "not logged in" error, prompt user to login and retry
	if err != nil && isNotLoggedInError(err) {
		// Prompt user to login
		if loginErr := f.promptForLogin(f.automation); loginErr != nil {
			return "", fmt.Errorf("login failed: %w", loginErr)
		}

		// Retry the request after login
		result, err = f.graphqlRequest(requests)
	}

	return result, err
}

func (f *FastCheckout) graphqlRequest(requests []GraphQLRequest) (string, error) {
	jsonData, err := json.Marshal(requests)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", f.graphqlURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Origin", "https://robertsspaceindustries.com")
	req.Header.Set("Referer", "https://robertsspaceindustries.com/")

	for _, cookie := range f.cookies {
		req.AddCookie(cookie)
	}

	if f.csrfToken != "" {
		req.Header.Set("x-csrf-token", f.csrfToken)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var responses []GraphQLResponse
	if err := json.Unmarshal(body, &responses); err != nil {
		return "", fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	for i, gqlResp := range responses {
		if len(gqlResp.Errors) > 0 {
			errMsg := fmt.Sprintf("GraphQL error in operation %d:\n", i+1)
			for _, gqlErr := range gqlResp.Errors {
				errMsg += fmt.Sprintf("  ❌ %s", gqlErr.Message)

				if details, ok := gqlErr.Extensions["details"].(map[string]interface{}); ok {
					errMsg += "\n     Details:"
					for key, value := range details {
						errMsg += fmt.Sprintf("\n       • %s: %v", key, value)
					}
				}

				if gqlErr.Code != "" {
					errMsg += fmt.Sprintf("\n     Code: %s", gqlErr.Code)
				}

				if len(gqlErr.Path) > 0 {
					errMsg += fmt.Sprintf("\n     Path: %v", gqlErr.Path)
				}

				errMsg += "\n"
			}
			return "", fmt.Errorf(errMsg)
		}
	}

	return string(body), nil
}

// isNetworkError checks if an error is a network/timeout error that should be retried
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "Client.Timeout") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "no route to host")
}

// retryOnNetworkError wraps an operation with retry logic for network/timeout errors
// Retries indefinitely until success or non-network error
func retryOnNetworkError(operation func() error, operationName string) error {
	attemptNum := 0
	for {
		attemptNum++
		err := operation()

		if err == nil {
			// Success!
			return nil
		}

		// Check if this is a network error we should retry
		if isNetworkError(err) {
			// Network error - retry after a short delay
			delay := time.Duration(500+rand.Intn(1000)) * time.Millisecond // 500-1500ms
			if attemptNum%10 == 0 || attemptNum <= 3 {
				fmt.Printf("⚠️  %s failed (attempt %d): network error - retrying in %dms...\n",
					operationName, attemptNum, delay.Milliseconds())
				if attemptNum <= 3 {
					fmt.Printf("   Error: %v\n", err)
				}
			}
			time.Sleep(delay)
			continue
		}

		// Non-network error - return immediately
		return err
	}
}

func (f *FastCheckout) RunFastCheckout(automation *Automation) error {
	startTime := time.Now()

	fmt.Println(T("checkout_fast_header_line1"))
	fmt.Println(T("checkout_fast_header_line2"))
	fmt.Println(T("checkout_fast_header_line3"))
	fmt.Println(T("checkout_fast_header_line4"))
	fmt.Println()

	if err := f.LoadSessionFromBrowser(automation); err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Always get SKU ID for validation, even if skipping add to cart
	// OPTIMIZATION: Extract SKU from already-open page (no incognito browser needed)
	// This saves 150-450ms by eliminating the incognito browser launch + navigation
	skuID, err := f.GetSKUFromActivePage(automation)
	if err != nil {
		return fmt.Errorf("failed to get SKU ID: %w", err)
	}

	// Check cart state BEFORE trying to add to cart
	fmt.Println(T("cart_checking_state"))
	// OPTIMIZATION: Use combined query to get totals and items in single round trip (saves 50-150ms)
	cartInfo, err := f.GetCartTotalsAndItems()
	if err != nil {
		return fmt.Errorf("failed to query cart info: %w", err)
	}

	// FAST PATH: If cart already has correct item with credits applied ($0 total),
	// skip directly to final validation step. This supports --skip-cart for repeat runs.
	if len(cartInfo.Items) == 1 &&
		cartInfo.Items[0].SKUID == skuID &&
		cartInfo.Items[0].Quantity == 1 &&
		cartInfo.Total == 0 {

		fmt.Println(T("cart_ready_to_checkout"))
		fmt.Printf(T("cart_ready_item")+"\n", cartInfo.Items[0].Name)
		fmt.Println(T("cart_ready_skipping_to_validation"))

		// Get/cache the billing address
		if f.cachedAddressID == "" {
			var addressID string
			err := retryOnNetworkError(func() error {
				var err error
				addressID, err = f.GetDefaultBillingAddress()
				return err
			}, "Get Default Billing Address")
			if err != nil {
				return fmt.Errorf("failed to get billing address: %w", err)
			}
			f.cachedAddressID = addressID
		}

		// Move to billing/addresses step
		fmt.Println(T("checkout_moving_billing"))
		err := retryOnNetworkError(func() error {
			return f.NextStep()
		}, "Move to Billing Step")
		if err != nil {
			return fmt.Errorf("failed to move to billing/addresses: %w", err)
		}

		// Assign billing address
		err = retryOnNetworkError(func() error {
			return f.AssignBillingAddress(f.cachedAddressID)
		}, "Assign Billing Address")
		if err != nil {
			return fmt.Errorf("failed to assign billing address: %w", err)
		}

		// Complete the order
		if !f.config.DryRun {
			fmt.Println(T("checkout_completing_order"))
			err := retryOnNetworkError(func() error {
				return f.ValidateCart(automation)
			}, "Validate Cart")
			if err != nil {
				return fmt.Errorf("failed to validate cart: %w", err)
			}
			fmt.Println(T("checkout_order_completed"))
		} else {
			fmt.Println(T("checkout_dry_run_stop"))
		}

		elapsed := time.Since(startTime)
		fmt.Println(T("checkout_total_time"), elapsed)
		fmt.Printf(T("checkout_target_vs_actual")+"\n", elapsed)
		if elapsed.Milliseconds() < 1000 {
			fmt.Println(T("checkout_achieved_subsecond"))
		}
		return nil
	}

	// Validate existing cart contents before adding
	shouldAdd, err := f.ValidateCartContents(skuID, cartInfo.Total, cartInfo.Items)
	if err != nil {
		return fmt.Errorf("cart validation failed: %w", err)
	}

	cartTotal := cartInfo.Total
	maxCredit := cartInfo.MaxCredit
	var itemPrice float64

	// Calculate item price from cart (for tax handling)
	// Store credit items don't have tax, so we use item price as the true total
	if len(cartInfo.Items) > 0 {
		itemPrice = cartInfo.Items[0].Price
	}

	// Now add to cart if not skipping AND if cart validation says it's safe to add
	if !f.config.SkipAddToCart && shouldAdd {
		if err := f.AddToCart(skuID, automation); err != nil {
			return fmt.Errorf("failed to add to cart: %w", err)
		}

		// Re-query cart info after adding
		cartInfo, err = f.GetCartTotalsAndItems()
		if err != nil {
			return fmt.Errorf("failed to query cart info after add: %w", err)
		}
		cartTotal = cartInfo.Total
		maxCredit = cartInfo.MaxCredit
		if len(cartInfo.Items) > 0 {
			itemPrice = cartInfo.Items[0].Price
		}

		// OPTIMIZATION: Skip post-add validation - we just successfully added the item,
		// so we know the cart state. This saves 50-150ms by avoiding an extra GraphQL query.
		// The pre-add validation already ensured cart was in a good state.
		fmt.Println(T("cart_item_added_success"))
	} else if !shouldAdd {
		fmt.Println(T("checkout_skip_add_cart_current"))
	} else {
		fmt.Println(T("checkout_skip_add_cart_exists"))
	}

	if f.config.AutoApplyCredit {
		// Use item price instead of cart total (handles tax in some regions)
		// Store credit items don't have tax, item price is the correct amount
		creditToApply := itemPrice
		if creditToApply > maxCredit {
			fmt.Printf(T("credit_total_exceeds_max_apply")+"\n", itemPrice, maxCredit)
			fmt.Printf(T("credit_applying_maximum")+"\n", maxCredit)
			creditToApply = maxCredit
		}

		if creditToApply > 0 {
			err := retryOnNetworkError(func() error {
				return f.ApplyStoreCredit(creditToApply)
			}, "Apply Store Credit")
			if err != nil {
				return fmt.Errorf("failed to apply credit: %w", err)
			}
			// OPTIMIZATION: We know the total is $0 after applying credit, no need to re-query
			cartTotal = 0
			if f.config.DebugMode {
				fmt.Printf(T("debug_cart_total_optimized")+"\n", cartTotal)
			}
		} else {
			fmt.Println(T("checkout_no_credit_needed"))
		}
	}

	if cartTotal == 0 {
		fmt.Println(T("checkout_moving_billing"))
		err := retryOnNetworkError(func() error {
			return f.NextStep()
		}, "Move to Billing Step")
		if err != nil {
			return fmt.Errorf("failed to move to billing/addresses: %w", err)
		}

		// OPTIMIZATION: Cache address ID if not already cached
		if f.cachedAddressID == "" {
			var addressID string
			err := retryOnNetworkError(func() error {
				var err error
				addressID, err = f.GetDefaultBillingAddress()
				return err
			}, "Get Default Billing Address")
			if err != nil {
				return fmt.Errorf("failed to get billing address: %w", err)
			}
			f.cachedAddressID = addressID
			if f.config.DebugMode {
				fmt.Printf(T("debug_cached_address")+"\n", addressID)
			}
		} else if f.config.DebugMode {
			fmt.Printf(T("debug_using_cached_address")+"\n", f.cachedAddressID)
		}

		err = retryOnNetworkError(func() error {
			return f.AssignBillingAddress(f.cachedAddressID)
		}, "Assign Billing Address")
		if err != nil {
			return fmt.Errorf("failed to assign billing address: %w", err)
		}

		if !f.config.DryRun {
			fmt.Println(T("checkout_completing_order"))
			err := retryOnNetworkError(func() error {
				return f.ValidateCart(automation)
			}, "Validate Cart")
			if err != nil {
				return fmt.Errorf("failed to validate cart: %w", err)
			}
			fmt.Println(T("checkout_order_completed"))
		} else {
			fmt.Println(T("checkout_dry_run_stop"))
		}
	} else {
		fmt.Printf(T("checkout_moving_payment")+"\n", cartTotal)
		err := retryOnNetworkError(func() error {
			return f.NextStep()
		}, "Move to Payment Step")
		if err != nil {
			return fmt.Errorf("failed to move to payment: %w", err)
		}

		if !f.config.DryRun {
			fmt.Println(T("checkout_completing_payment"))
			err := retryOnNetworkError(func() error {
				return f.NextStep()
			}, "Complete Order")
			if err != nil {
				return fmt.Errorf("failed to complete order: %w", err)
			}
			fmt.Println(T("checkout_order_completed"))
		} else {
			fmt.Println(T("checkout_dry_run_stop"))
		}
	}

	elapsed := time.Since(startTime)
	fmt.Println(T("checkout_total_time"), elapsed)
	fmt.Printf(T("checkout_target_vs_actual")+"\n", elapsed)

	if elapsed.Milliseconds() < 1000 {
		fmt.Println(T("checkout_achieved_subsecond"))
	}

	return nil
}

// RunTimedSaleCheckout handles timed sale scenarios with aggressive retry logic
func (f *FastCheckout) RunTimedSaleCheckout(automation *Automation) error {
	// Parse sale start time
	saleTime, err := time.Parse(time.RFC3339, f.config.SaleStartTime)
	if err != nil {
		return fmt.Errorf("invalid sale start time format (use RFC3339): %w", err)
	}

	startRetryTime := saleTime.Add(-time.Duration(f.config.StartBeforeSaleMinutes) * time.Minute)
	endRetryTime := saleTime.Add(time.Duration(f.config.ContinueAfterSaleMinutes) * time.Minute)

	// Track overall timing from the start
	startTime := time.Now()

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║           TIMED SALE MODE - AGGRESSIVE RETRY              ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("⏰ Sale starts at: %s\n", saleTime.Local().Format(time.RFC1123))
	fmt.Printf("🚀 Will start retrying at: %s (%d min before)\n",
		startRetryTime.Local().Format(time.RFC1123), f.config.StartBeforeSaleMinutes)
	fmt.Printf("⏱️  Will stop retrying at: %s (%d min after)\n",
		endRetryTime.Local().Format(time.RFC1123), f.config.ContinueAfterSaleMinutes)
	fmt.Println()

	// OPTIMIZATION: Do all pre-checks BEFORE waiting for sale window
	// This catches login/config issues immediately, not right before the sale!
	fmt.Println("🔧 Running pre-flight checks...")
	fmt.Println()

	if err := f.LoadSessionFromBrowser(automation); err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// OPTIMIZATION: Extract SKU from already-open page (no incognito browser needed)
	// This saves 150-450ms by eliminating the incognito browser launch + navigation
	skuID, err := f.GetSKUFromActivePage(automation)
	if err != nil {
		return fmt.Errorf("failed to get SKU ID: %w", err)
	}

	// OPTIMIZATION: Pre-fetch and cache billing address after session is loaded
	// This saves time during checkout phase
	if f.cachedAddressID == "" {
		addressID, err := f.GetDefaultBillingAddress()
		if err != nil {
			return fmt.Errorf("failed to pre-fetch billing address: %w", err)
		}
		f.cachedAddressID = addressID
		fmt.Printf("✓ Cached billing address: %s\n", addressID)
	} else {
		fmt.Printf("✓ Using cached billing address: %s\n", f.cachedAddressID)
	}

	// Check cart state BEFORE waiting (validates login + cart state)
	fmt.Println("🔍 Checking current cart state...")
	// OPTIMIZATION: Use combined query to get totals and items in single round trip (saves 50-150ms)
	preCheckCartInfo, err := f.GetCartTotalsAndItems()
	if err != nil {
		return fmt.Errorf("failed to query initial cart info: %w", err)
	}

	// FAST PATH: If cart already has correct item with credits applied ($0 total),
	// skip directly to final validation step. This supports --skip-cart for repeat runs.
	if len(preCheckCartInfo.Items) == 1 &&
		preCheckCartInfo.Items[0].SKUID == skuID &&
		preCheckCartInfo.Items[0].Quantity == 1 &&
		preCheckCartInfo.Total == 0 {

		fmt.Println(T("cart_ready_to_checkout"))
		fmt.Printf(T("cart_ready_item")+"\n", preCheckCartInfo.Items[0].Name)
		fmt.Println(T("cart_ready_skipping_to_validation"))
		fmt.Println()

		// Move to billing/addresses step
		fmt.Println("➡️  Moving to billing/addresses step...")
		err := retryOnNetworkError(func() error {
			return f.NextStep()
		}, "Move to Billing Step")
		if err != nil {
			return fmt.Errorf("failed to move to billing/addresses: %w", err)
		}

		// Assign billing address
		err = retryOnNetworkError(func() error {
			return f.AssignBillingAddress(f.cachedAddressID)
		}, "Assign Billing Address")
		if err != nil {
			return fmt.Errorf("failed to assign billing address: %w", err)
		}

		// Complete the order (use deadline if in timed mode)
		if !f.config.DryRun {
			fmt.Println("🎯 Completing order with aggressive retries until sale window ends...")
			err := retryOnNetworkError(func() error {
				return f.ValidateCartWithDeadline(automation, endRetryTime)
			}, "Validate Cart")
			if err != nil {
				return fmt.Errorf("failed to validate cart: %w", err)
			}
			fmt.Println("\n✅ ORDER COMPLETED!")
		} else {
			fmt.Println("🧪 DRY RUN - Stopping before final submission")
		}

		totalElapsed := time.Since(startTime)
		fmt.Printf("\n⚡ Total time from start to completion: %v\n", totalElapsed)
		return nil
	}

	// Validate existing cart contents before Phase 1
	shouldAdd, err := f.ValidateCartContents(skuID, preCheckCartInfo.Total, preCheckCartInfo.Items)
	if err != nil {
		return fmt.Errorf("pre-flight cart validation failed: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ All pre-flight checks passed!")
	fmt.Println()

	// Wait until it's time to start the aggressive retry phase
	now := time.Now()
	if now.Before(startRetryTime) {
		waitDuration := startRetryTime.Sub(now)
		fmt.Printf("⏳ Waiting %v until retry window starts...\n", waitDuration.Round(time.Second))
		fmt.Println("💡 Tip: Everything is ready! You can take a break until the sale starts.")
		fmt.Println()
		time.Sleep(waitDuration)
		fmt.Println("✓ Retry window started!")
	} else if now.After(endRetryTime) {
		return fmt.Errorf("sale window has already passed (ended at %s)", endRetryTime.Local().Format(time.RFC1123))
	} else {
		fmt.Println("⚡ Already in retry window - starting immediately!")
	}

	// Phase 1: Aggressive add-to-cart retries (only if validation says to add)
	if shouldAdd {
		fmt.Println()
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Println("           PHASE 1: ADD TO CART (AGGRESSIVE RETRY)")
		fmt.Println("═══════════════════════════════════════════════════════════")

		attemptNum := 0

		for {
			attemptNum++
			now := time.Now()

			if now.After(endRetryTime) {
				elapsed := time.Since(startTime)
				return fmt.Errorf("failed to add to cart after %d attempts in %v - retry window expired", attemptNum, elapsed)
			}

			remaining := endRetryTime.Sub(now)

			if attemptNum == 1 || attemptNum%50 == 0 {
				fmt.Printf("🔄 Attempt %d - Time remaining: %v\n", attemptNum, remaining.Round(time.Second))
			}

			// Try to add to cart (without retry logic inside - we handle retries here)
			err := f.addToCartSingleAttempt(skuID, automation)
			if err == nil {
				elapsed := time.Since(startTime)
				fmt.Printf("\n✅ Successfully added to cart after %d attempts in %v!\n\n", attemptNum, elapsed)
				break
			}

			// Retry delays based on error type
			var delay time.Duration
			is4226, is4227 := isPaymentAuthError(err)

			if is4227 {
				delayMs := f.config.Payment4227MinMs + rand.Intn(f.config.Payment4227MaxMs-f.config.Payment4227MinMs+1)
				delay = time.Duration(delayMs) * time.Millisecond
			} else if is4226 {
				delayMs := f.config.Payment4226MinMs + rand.Intn(f.config.Payment4226MaxMs-f.config.Payment4226MinMs+1)
				delay = time.Duration(delayMs) * time.Millisecond
			} else if isRateLimitError(err) {
				delayMs := f.config.RateLimitMinMs + rand.Intn(f.config.RateLimitMaxMs-f.config.RateLimitMinMs+1)
				delay = time.Duration(delayMs) * time.Millisecond
			} else if isOutOfStockError(err) {
				delay = time.Duration(f.config.OutOfStockDelayMs) * time.Millisecond
			} else {
				delay = time.Duration(f.config.GenericErrorDelayMs) * time.Millisecond
			}

			time.Sleep(delay)
		}
	} else {
		fmt.Println("⏭️  Skipping Phase 1 (proceeding with current cart contents)")
	}

	// Phase 2: Aggressive checkout retries
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("           PHASE 2: CHECKOUT (AGGRESSIVE RETRY)")
	fmt.Println("═══════════════════════════════════════════════════════════")

	// Query cart info after Phase 1
	cartInfo, err := f.GetCartTotalsAndItems()
	if err != nil {
		return fmt.Errorf("failed to query cart info: %w", err)
	}
	cartTotal := cartInfo.Total
	maxCredit := cartInfo.MaxCredit
	var itemPrice float64
	if len(cartInfo.Items) > 0 {
		itemPrice = cartInfo.Items[0].Price
	}

	// OPTIMIZATION: Skip post-Phase-1 validation - Phase 1 successfully added the item,
	// so we know the cart state. This saves 50-150ms by avoiding an extra GraphQL query.
	// The pre-Phase-1 validation already ensured cart was in a good state.
	fmt.Println("✓ Phase 1 complete, proceeding to Phase 2 (validation skipped for speed)")

	if f.config.AutoApplyCredit {
		// Use item price instead of cart total (handles tax in some regions)
		// Store credit items don't have tax, item price is the correct amount
		creditToApply := itemPrice
		if creditToApply > maxCredit {
			fmt.Printf("⚠️  Item price ($%.2f) exceeds max credit ($%.2f) - applying max\n", itemPrice, maxCredit)
			creditToApply = maxCredit
		}

		if creditToApply > 0 {
			err := retryOnNetworkError(func() error {
				return f.ApplyStoreCredit(creditToApply)
			}, "Apply Store Credit")
			if err != nil {
				return fmt.Errorf("failed to apply credit: %w", err)
			}
			cartTotal = 0
		}
	}

	if cartTotal == 0 {
		fmt.Println("➡️  Moving to billing/addresses step...")
		err := retryOnNetworkError(func() error {
			return f.NextStep()
		}, "Move to Billing Step")
		if err != nil {
			return fmt.Errorf("failed to move to billing/addresses: %w", err)
		}

		err = retryOnNetworkError(func() error {
			return f.AssignBillingAddress(f.cachedAddressID)
		}, "Assign Billing Address")
		if err != nil {
			return fmt.Errorf("failed to assign billing address: %w", err)
		}

		if !f.config.DryRun {
			fmt.Println("🎯 Completing order with aggressive retries until sale window ends...")
			// Use the same endRetryTime from the sale window for validation
			err := retryOnNetworkError(func() error {
				return f.ValidateCartWithDeadline(automation, endRetryTime)
			}, "Validate Cart")
			if err != nil {
				return fmt.Errorf("failed to validate cart: %w", err)
			}
			fmt.Println("\n✅ ORDER COMPLETED!")
		} else {
			fmt.Println("🧪 DRY RUN - Stopping before final submission")
		}
	} else {
		fmt.Printf("➡️  Moving to payment step (balance: $%.2f)...\n", cartTotal)
		err := retryOnNetworkError(func() error {
			return f.NextStep()
		}, "Move to Payment Step")
		if err != nil {
			return fmt.Errorf("failed to move to payment: %w", err)
		}

		if !f.config.DryRun {
			fmt.Println("🎯 Completing order with payment...")
			err := retryOnNetworkError(func() error {
				return f.NextStep()
			}, "Complete Order")
			if err != nil {
				return fmt.Errorf("failed to complete order: %w", err)
			}
			fmt.Println("\n✅ ORDER COMPLETED!")
		} else {
			fmt.Println("🧪 DRY RUN - Stopping before final submission")
		}
	}

	totalElapsed := time.Since(startTime)
	fmt.Printf("\n⚡ Total time from first attempt to completion: %v\n", totalElapsed)

	return nil
}

// addToCartSingleAttempt tries to add to cart once without retrying
func (f *FastCheckout) addToCartSingleAttempt(skuID string, automation *Automation) error {
	// No reCAPTCHA needed for AddCartMultiItemMutation
	mutation := `mutation AddCartMultiItemMutation($query: [CartAddInput!]) {
  store(name: "pledge") {
    cart {
      mutations {
        addMany(query: $query) {
          count
          resources {
            id
            title
            __typename
          }
          __typename
        }
        __typename
      }
      __typename
    }
    __typename
  }
}`

	variables := map[string]interface{}{
		"query": []map[string]interface{}{
			{
				"qty":   1,
				"skuId": skuID,
			},
		},
	}

	request := []GraphQLRequest{
		{
			OperationName: "AddCartMultiItemMutation",
			Variables:     variables,
			Query:         mutation,
		},
	}

	resp, err := f.graphqlRequestWithLoginRetry(request)
	if err != nil {
		return err
	}

	// Success - item added to cart
	_ = resp
	return nil
}
