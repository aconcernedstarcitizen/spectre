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
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
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

func (f *FastCheckout) LoadSessionFromBrowser(automation *Automation) error {
	fmt.Println("🔐 Extracting session from browser...")

	if automation == nil || automation.page == nil {
		return fmt.Errorf("browser not initialized")
	}

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

	fmt.Printf("✓ Extracted %d cookies from browser\n", len(f.cookies))

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
		fmt.Printf("✓ Extracted CSRF token: %s...\n", f.csrfToken[:16])
	} else {
		fmt.Println("⚠️  CSRF token not found, will try without it")
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
	fmt.Printf("🔍 Extracting SKU slug from %s...\n", itemURL)

	req, err := http.NewRequest("GET", itemURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", f.userAgent)

	for _, cookie := range f.cookies {
		req.AddCookie(cookie)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch product page: %w", err)
	}
	defer resp.Body.Close()

	if f.config.DebugMode {
		fmt.Printf("[DEBUG] Product page HTTP status: %d\n", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if f.config.DebugMode {
		fmt.Printf("[DEBUG] Product page body length: %d bytes\n", len(body))
	}

	re := regexp.MustCompile(`"skuSlug":\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		fmt.Printf("✓ Found SKU slug: %s\n", matches[1])
		return matches[1], nil
	}

	return "", fmt.Errorf("could not find SKU slug in product page")
}

func (f *FastCheckout) GetSKUIDFromSlug(skuSlug string) (string, error) {
	fmt.Printf("🔍 Converting SKU slug to numeric ID...\n")

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

	resp, err := f.graphqlRequest(request)
	if err != nil {
		return "", fmt.Errorf("failed to query SKU: %w", err)
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
		return "", fmt.Errorf("failed to parse SKU query response: %w", err)
	}

	if len(responses) == 0 || len(responses[0].Data.Store.Listing.Skus) == 0 {
		return "", fmt.Errorf("no SKU found for slug: %s", skuSlug)
	}

	skuID := responses[0].Data.Store.Listing.Skus[0].ID.String()
	fmt.Printf("✓ Found SKU ID: %s (%s)\n", skuID, responses[0].Data.Store.Listing.Skus[0].Title)

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

func (f *FastCheckout) GetSKUFromBrowser(automation *Automation, itemURL string) (string, error) {
	fmt.Println("🕵️  Opening incognito browser to extract SKU slug...")

	incognitoLauncher := launcher.New().Headless(true).Leakless(true)
	incognitoURL, err := incognitoLauncher.Launch()
	if err != nil {
		return "", fmt.Errorf("failed to launch incognito browser: %w", err)
	}

	incognitoBrowser := rod.New().ControlURL(incognitoURL).MustConnect()
	defer func() {
		incognitoBrowser.Close()
		incognitoLauncher.Cleanup()
	}()

	incognitoPage, err := stealth.Page(incognitoBrowser)
	if err != nil {
		return "", fmt.Errorf("failed to create stealth incognito page: %w", err)
	}
	defer incognitoPage.Close()

	err = incognitoPage.Navigate(itemURL)
	if err != nil {
		return "", fmt.Errorf("failed to navigate to item page: %w", err)
	}

	if err := incognitoPage.WaitLoad(); err != nil {
		return "", fmt.Errorf("incognito page failed to load: %w", err)
	}

	skuSlug, err := incognitoPage.Eval(`() => {
		const div = document.querySelector('[data-rsi-component="SkuDetailPage"]');
		if (!div) return null;
		const props = div.getAttribute('data-rsi-component-props');
		if (!props) return null;
		try {
			const parsed = JSON.parse(props);
			return parsed.skuSlug || null;
		} catch (e) {
			return null;
		}
	}`)

	if err != nil || skuSlug.Value.Str() == "" || skuSlug.Value.Str() == "null" {
		htmlSource, htmlErr := incognitoPage.HTML()
		if htmlErr == nil {
			re := regexp.MustCompile(`"skuSlug":\s*"([^"]+)"`)
			matches := re.FindStringSubmatch(htmlSource)
			if len(matches) > 1 {
				skuSlugStr := matches[1]
				fmt.Printf("✓ Extracted SKU slug (incognito/HTML): %s\n", skuSlugStr)
				return f.getSKUIDFromSlug(skuSlugStr)
			}
		}
		return "", fmt.Errorf("SKU slug not found in incognito page")
	}

	skuSlugStr := skuSlug.Value.Str()
	fmt.Printf("✓ Extracted SKU slug (incognito): %s\n", skuSlugStr)

	return f.getSKUIDFromSlug(skuSlugStr)
}

func (f *FastCheckout) getSKUIDFromSlug(skuSlugStr string) (string, error) {
	fmt.Printf("🔍 Querying SKU ID for slug: %s\n", skuSlugStr)

	if f.config.DebugMode {
		fmt.Printf("[DEBUG] SKU slug value: '%s' (length: %d)\n", skuSlugStr, len(skuSlugStr))
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

	resp, err := f.graphqlRequest(request)
	if err != nil {
		return "", fmt.Errorf("GetSkus query failed: %w", err)
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
		return "", fmt.Errorf("failed to parse GetSkus response: %w", err)
	}

	if len(responses) == 0 || len(responses[0].Data.Store.Search.Resources) == 0 {
		return "", fmt.Errorf("no SKU found for slug: %s", skuSlugStr)
	}

	skuID := responses[0].Data.Store.Search.Resources[0].ID
	fmt.Printf("✓ Found SKU ID: %s\n", skuID)

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
		strings.Contains(errStr, "throttle") ||
		strings.Contains(errStr, "4227")
}

func isOutOfStockError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "4226") ||
		strings.Contains(errStr, "out of stock") ||
		strings.Contains(errStr, "not available") ||
		strings.Contains(errStr, "unavailable")
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
		return "", fmt.Errorf("reCAPTCHA not loaded on page")
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
		return "", fmt.Errorf("failed to start reCAPTCHA execution: %w", err)
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
						fmt.Printf("[DEBUG] ✅ Successfully read reCAPTCHA token: %s (len=%d)\n", tokenPreview, len(tokenStr))
					}
					return tokenStr, nil
				} else if f.config.DebugMode && i == 0 {
					fmt.Printf("[DEBUG] Debug state is 'success' but token is invalid: '%s' (len=%d)\n", tokenStr, len(tokenStr))
				}
			}
		} else if f.config.DebugMode && i == 0 {
			fmt.Printf("[DEBUG] Waiting for token generation... debug_state='%s' (need 'success')\n", debugStr)
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

			fmt.Printf("[DEBUG] Token check %d: debug_state='%s', typeof=%s, callbackInvoked=%v\n",
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
			return "", fmt.Errorf("reCAPTCHA execution error: %s", errMsg)
		}
	}

	debugState, _ := page.Eval(`() => window.__specterDebug`)
	debugStr := "unknown"
	if debugState != nil {
		debugStr = debugState.Value.Str()
	}

	if debugStr == "null_token" {
		return "", fmt.Errorf("reCAPTCHA v3 Enterprise detected automation and returned null token (stealth mode insufficient)")
	}

	return "", fmt.Errorf("reCAPTCHA token generation timeout after 5s (debug state: %s)", debugStr)
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
			fmt.Printf("[DEBUG] 🔄 Reusing cached reCAPTCHA token (age: %v, valid for: %v more)\n",
				tokenAge.Round(time.Second),
				(60*time.Second - tokenAge).Round(time.Second))
		}
		return f.cachedRecaptchaToken, nil
	}

	// Token is missing or expired (>60s) - generate fresh token
	if f.cachedRecaptchaToken != "" {
		fmt.Printf("♻️  reCAPTCHA token expired (age: %v) - generating fresh token...\n", tokenAge.Round(time.Second))
	} else {
		fmt.Println("🔐 Generating initial reCAPTCHA token...")
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
		fmt.Printf("✅ Fresh reCAPTCHA token cached (valid for 60s): %s\n", tokenPreview)
	}

	return token, nil
}

func (f *FastCheckout) AddToCart(skuID string, automation *Automation) error {
	fmt.Printf("🛒 Adding to cart (API) with retry mechanism...\n")
	fmt.Printf("[DEBUG] SKU ID: %s\n", skuID)

	retryDuration := f.config.RetryDurationSeconds
	fmt.Printf("⏱️  Will retry for up to %d seconds\n", retryDuration)

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
					fmt.Printf("[DEBUG] Attempt %d: Got reCAPTCHA token: %s (len=%d)\n", attemptNum, tokenPreview, len(recaptchaToken))
				} else {
					fmt.Printf("[DEBUG] Attempt %d: reCAPTCHA returned invalid/empty token: '%s'\n", attemptNum, recaptchaToken)
				}
			}
		case err := <-tokenErrChan:
			if f.config.DebugMode && attemptNum <= 3 {
				fmt.Printf("[DEBUG] Attempt %d: reCAPTCHA error (continuing): %v\n", attemptNum, err)
			}
			if strings.Contains(err.Error(), "automation detected") && attemptNum == 1 {
				fmt.Println("⚠️  WARNING: reCAPTCHA v3 Enterprise is detecting automation")
				fmt.Println("   The script will continue without tokens, but may fail with CFUException")
			}
		case <-time.After(5 * time.Second):
			if f.config.DebugMode && attemptNum <= 3 {
				fmt.Printf("[DEBUG] Attempt %d: reCAPTCHA timeout after 5s (continuing without token)\n", attemptNum)
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
				fmt.Printf("[DEBUG] Attempt %d: Generated reCAPTCHA token but NOT using it for AddCartMultiItemMutation (len=%d)\n", attemptNum, len(recaptchaToken))
			} else {
				fmt.Printf("[DEBUG] Attempt %d: No reCAPTCHA token generated\n", attemptNum)
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
			fmt.Printf("[DEBUG] Request body:\n%s\n", string(jsonData))
		}

		resp, err := f.graphqlRequest(request)

		if f.config.DebugMode && attemptNum <= 3 {
			fmt.Printf("[DEBUG] Attempt %d response: %s\n", attemptNum, resp)
			if err != nil {
				fmt.Printf("[DEBUG] Attempt %d error: %v\n", attemptNum, err)
			}
		}

		if err == nil {
			elapsed := time.Since(startTime)
			fmt.Printf("✓ Added to cart successfully!\n")
			if attemptNum > 1 {
				fmt.Printf("🎉 Success after %d attempt(s) in %v\n", attemptNum, elapsed)
			}
			return nil
		}

		remaining := retryDeadline.Sub(time.Now())

		if remaining <= 0 {
			elapsed := time.Since(startTime)
			fmt.Printf("❌ Sale window expired after %d attempts in %v\n", attemptNum, elapsed)
			return fmt.Errorf("add to cart failed after %d attempts: %w", attemptNum, err)
		}

		var delay time.Duration
		isRateLimited := isRateLimitError(err)
		isOutOfStock := isOutOfStockError(err)
		isCaptchaFail := isCaptchaError(err)

		if isCaptchaFail {
			// Minimal delay for CAPTCHA - just retry immediately with new token
			delayMs := 5 + rand.Intn(15) // 5-20ms - extremely fast
			delay = time.Duration(delayMs) * time.Millisecond

			if attemptNum%10 == 0 || attemptNum <= 3 {
				fmt.Printf("🔐 Attempt %d: CAPTCHA verification needed - fast retry in %dms (remaining: %v)...\n",
					attemptNum, delayMs, remaining.Round(time.Second))
			}
		} else if isRateLimited {
			// Aggressive rate limit handling - minimal backoff
			delayMs := 50 + rand.Intn(100) // 50-150ms instead of 2000-5000ms
			delay = time.Duration(delayMs) * time.Millisecond
			fmt.Printf("⚠️  Attempt %d: Rate limited (4227) - fast retry in %dms (remaining: %v)...\n",
				attemptNum, delayMs, remaining.Round(time.Second))
		} else if isOutOfStock {
			// Out of stock - minimal delay, keep hammering
			delayMs := 5 + rand.Intn(15) // 5-20ms - extremely fast
			delay = time.Duration(delayMs) * time.Millisecond

			if attemptNum%10 == 0 {
				fmt.Printf("⏳ Attempt %d: Out of stock (4226) - fast retry (remaining: %v)...\n",
					attemptNum, remaining.Round(time.Second))
			}
		} else {
			// Generic error - minimal delay
			delayMs := 5 + rand.Intn(25) // 5-30ms
			delay = time.Duration(delayMs) * time.Millisecond

			attemptDuration := time.Since(attemptStart)
			if attemptNum <= 5 || attemptNum%20 == 0 {
				fmt.Printf("⚠️  Attempt %d failed (%v) - fast retry in %dms (remaining: %v)...\n",
					attemptNum, attemptDuration, delayMs, remaining.Round(time.Second))
			}
		}

		if delay > remaining {
			delay = remaining
		}

		time.Sleep(delay)
	}
}

func (f *FastCheckout) GetCartTotals() (cartTotal float64, maxCredit float64, err error) {
	fmt.Println("📊 Querying cart totals...")

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

	resp, err := f.graphqlRequest(request)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query cart totals: %w", err)
	}

	var responses []map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &responses); err != nil {
		return 0, 0, fmt.Errorf("failed to parse cart totals: %w", err)
	}

	if len(responses) == 0 {
		return 0, 0, fmt.Errorf("empty response from cart totals query")
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

	fmt.Printf("✓ Cart total: $%.2f\n", cartTotal)
	fmt.Printf("✓ Available store credit: $%.2f\n", availableCredit)
	fmt.Printf("✓ Max applicable to this cart: $%.2f\n", maxCredit)

	return cartTotal, maxCredit, nil
}

type CartItem struct {
	Name     string
	Price    float64
	SKUID    string
	Quantity int
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

	resp, err := f.graphqlRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to query cart items: %w", err)
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
		return nil, fmt.Errorf("failed to parse cart items response: %w", err)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("empty response from cart items query")
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
func (f *FastCheckout) ValidateCartContents(expectedSKUID string, cartTotal float64) (bool, error) {
	items, err := f.GetCartItems()
	if err != nil {
		return false, fmt.Errorf("failed to get cart items: %w", err)
	}

	// Empty cart is normal - proceed with adding
	if len(items) == 0 {
		fmt.Println("✓ Cart is empty, will add item")
		return true, nil // Add to cart
	}

	// Check if cart already has exactly what we want: 1 item, correct SKU, quantity=1
	if len(items) == 1 && items[0].SKUID == expectedSKUID && items[0].Quantity == 1 {
		expectedTotal := items[0].Price
		if cartTotal == expectedTotal {
			// Perfect! Cart already has correct item at full price, don't add again
			fmt.Printf("✓ Cart already contains target item: %s ($%.2f)\n", items[0].Name, items[0].Price)
			fmt.Println("  Skipping add-to-cart step (would create duplicate)")
			return false, nil // Don't add, proceed with existing cart
		} else if cartTotal == 0 {
			// Cart total is $0 - store credit already applied from previous run
			fmt.Printf("✓ Cart already contains target item: %s ($%.2f)\n", items[0].Name, items[0].Price)
			fmt.Println("  Store credit already applied (cart total: $0.00)")
			fmt.Println("  Skipping add-to-cart and credit steps")
			return false, nil // Don't add, proceed with existing cart
		}
		// If price doesn't match and isn't $0, fall through to show warning
	}

	// Cart has issues - multiple items, wrong items, quantity > 1, or price mismatch

	// Cart has issues - either multiple items, wrong items, or quantity > 1
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                    ⚠️  CART WARNING                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Calculate total items across all line items (considering quantities)
	totalItems := 0
	for _, item := range items {
		totalItems += item.Quantity
	}

	if len(items) == 1 {
		fmt.Printf("Your cart contains %d × %s:\n\n", items[0].Quantity, items[0].Name)
	} else {
		fmt.Printf("Your cart contains %d item(s) across %d line item(s):\n\n", totalItems, len(items))
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

		fmt.Printf("   Price: $%.2f × %d = $%.2f\n", item.Price, item.Quantity, item.Price*float64(item.Quantity))

		if item.SKUID == expectedSKUID {
			fmt.Printf("   (This is your target item)\n")
			if item.Quantity > 1 {
				fmt.Printf("   ⚠️  WARNING: Buying %d copies of this ship!\n", item.Quantity)
			}
		}
		fmt.Println()
	}

	fmt.Printf("Cart Total: $%.2f\n", cartTotal)

	// Check if cart total matches expected for single item
	if len(items) == 1 && items[0].SKUID == expectedSKUID && items[0].Quantity == 1 {
		expectedTotal := items[0].Price
		if cartTotal != expectedTotal {
			fmt.Printf("Expected Total: $%.2f (for 1 × %s)\n", expectedTotal, items[0].Name)
		}
	}
	fmt.Println()

	// Determine the warning message
	if len(items) > 1 {
		fmt.Println("⚠️  You have MULTIPLE DIFFERENT items in your cart!")
		fmt.Println("   This checkout will purchase ALL items shown above.")
	} else if len(items) == 1 && items[0].SKUID == expectedSKUID && items[0].Quantity > 1 {
		fmt.Printf("⚠️  You are buying %d copies of the SAME ship!\n", items[0].Quantity)
		fmt.Printf("   This will purchase %d × %s for $%.2f total.\n",
			items[0].Quantity, items[0].Name, items[0].Price*float64(items[0].Quantity))
		fmt.Println()
		fmt.Println("   NOTE: RSI limits purchases to max 5 of any item per order.")
	} else if len(items) == 1 && items[0].SKUID == expectedSKUID && items[0].Quantity == 1 {
		// Single correct item but cart total doesn't match
		expectedTotal := items[0].Price
		fmt.Printf("⚠️  Cart total ($%.2f) doesn't match expected price ($%.2f)!\n", cartTotal, expectedTotal)
		fmt.Println("   This could be due to taxes, fees, or cart calculation issues.")
	} else {
		fmt.Println("⚠️  The cart contains a different item than expected!")
	}

	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  1. Press ENTER to continue with the CURRENT cart contents")
	fmt.Println("  2. Press ESC to cancel and manually edit your cart")
	fmt.Println()
	fmt.Print("⏳ Your choice: ")

	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadByte()
		if err != nil {
			return false, fmt.Errorf("failed to read input: %w", err)
		}

		if input == '\n' || input == '\r' {
			fmt.Println()
			fmt.Println("✓ User confirmed to proceed with CURRENT cart contents (will not add another item)")
			return false, nil // Don't add to cart, use existing cart
		}

		if input == 27 { // ESC key
			fmt.Println()
			fmt.Println("⚠️  User requested cancellation")
			return false, fmt.Errorf("user canceled due to unexpected cart contents")
		}
	}
}

func (f *FastCheckout) ApplyStoreCredit(amount float64) error {
	fmt.Printf("💰 Applying $%.2f store credit (API)...\n", amount)

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

	resp, err := f.graphqlRequest(request)
	if err != nil {
		return fmt.Errorf("apply credit failed: %w", err)
	}

	fmt.Printf("✓ Store credit applied: %s\n", resp)
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

	resp, err := f.graphqlRequest(request)
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

		fmt.Printf("✓ Moved to next step: %s\n", activeStep)

		if flow.Current.OrderCreated {
			fmt.Println("✓ Order has been created!")
		}
	}

	return nil
}

func (f *FastCheckout) GetDefaultBillingAddress() (string, error) {
	fmt.Println("📋 Fetching billing address...")

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

	resp, err := f.graphqlRequest(request)
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

	fmt.Printf("✓ Found billing address: %s (ID: %s)\n", addressName, addressID)
	return addressID, nil
}

func (f *FastCheckout) AssignBillingAddress(addressID string) error {
	fmt.Printf("📍 Assigning billing address (ID: %s)...\n", addressID)

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

	resp, err := f.graphqlRequest(request)
	if err != nil {
		return fmt.Errorf("failed to assign address: %w", err)
	}

	fmt.Printf("✓ Billing address assigned: %s\n", resp)
	return nil
}

func (f *FastCheckout) ValidateCart(automation *Automation) error {
	return f.ValidateCartWithDeadline(automation, time.Time{})
}

func (f *FastCheckout) ValidateCartWithDeadline(automation *Automation, deadline time.Time) error {
	fmt.Println("✅ Validating and completing order...")

	startTime := time.Now()

	// Use provided deadline or create one based on config
	var retryDeadline time.Time
	if !deadline.IsZero() {
		retryDeadline = deadline
		remaining := deadline.Sub(startTime)
		fmt.Printf("⏱️  Will retry validation until sale window ends: %v remaining\n", remaining.Round(time.Second))
	} else {
		retryDeadline = startTime.Add(time.Duration(f.config.RetryDurationSeconds) * time.Second)
		fmt.Printf("⏱️  Will retry validation for up to %d seconds\n", f.config.RetryDurationSeconds)
	}

	// Generate mark ONCE and reuse for all retry attempts (matches browser behavior)
	// The browser uses a constant mark throughout the validation session
	// Mark is a random 10-digit number (range: 1000000000 to 9999999999)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	mark := fmt.Sprintf("%d", 1000000000+rng.Intn(9000000000))

	attemptNum := 0

	for {
		attemptNum++
		attemptStart := time.Now()

		// Show progress for aggressive retry mode (every 50 attempts)
		remaining := retryDeadline.Sub(time.Now())
		if attemptNum == 1 || attemptNum%50 == 0 {
			if !deadline.IsZero() {
				// Timed sale mode - show time remaining in window
				fmt.Printf("🔄 Validation Attempt %d - Time remaining in sale window: %v\n", attemptNum, remaining.Round(time.Second))
			} else if attemptNum%50 == 0 {
				// Normal mode - only show every 50 attempts to reduce spam
				fmt.Printf("🔄 Validation Attempt %d - Time remaining: %v\n", attemptNum, remaining.Round(time.Second))
			}
		}

		// Use cached token (refreshed automatically every 60 seconds)
		recaptchaToken, err := f.GetOrRefreshCachedRecaptchaToken(automation, "store/cart/validate")
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to get reCAPTCHA token: %v\n", err)
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

		resp, err := f.graphqlRequest(request)

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

				fmt.Printf("✓ Order validated! Order slug: %s\n", orderSlug)

				if orderCreated {
					fmt.Println("✓ Order has been created!")
				}

				elapsed := time.Since(startTime)
				fmt.Printf("🎉 Success after %d attempt(s) in %v\n", attemptNum, elapsed)
				return nil
			}
		}

		remaining = retryDeadline.Sub(time.Now())

		if remaining <= 0 {
			elapsed := time.Since(startTime)
			if !deadline.IsZero() {
				fmt.Printf("❌ Sale window expired after %d validation attempts in %v\n", attemptNum, elapsed)
				return fmt.Errorf("cart validation failed after %d attempts - sale window expired: %w", attemptNum, err)
			} else {
				fmt.Printf("❌ Retry timeout reached after %d attempts in %v\n", attemptNum, elapsed)
				return fmt.Errorf("cart validation failed after %d attempts: %w", attemptNum, err)
			}
		}

		var delay time.Duration
		isRateLimited := isRateLimitError(err)
		isOutOfStock := isOutOfStockError(err)

		if isRateLimited {
			// Minimal backoff for rate limits during validation
			delayMs := 50 + rand.Intn(100) // 50-150ms instead of 2000-5000ms
			delay = time.Duration(delayMs) * time.Millisecond
			fmt.Printf("⚠️  Attempt %d: Rate limited (4227) during validation - fast retry in %dms (remaining: %v)...\n",
				attemptNum, delayMs, remaining.Round(time.Second))
		} else if isOutOfStock {
			// Minimal delay for out of stock during validation
			delayMs := 5 + rand.Intn(15) // 5-20ms
			delay = time.Duration(delayMs) * time.Millisecond

			if attemptNum%10 == 0 {
				fmt.Printf("⏳ Attempt %d: Item unavailable (4226) during validation - fast retry (remaining: %v)...\n",
					attemptNum, remaining.Round(time.Second))
			}
		} else {
			// Generic error during validation - minimal delay
			delayMs := 5 + rand.Intn(25) // 5-30ms
			delay = time.Duration(delayMs) * time.Millisecond

			attemptDuration := time.Since(attemptStart)
			if attemptNum <= 5 || attemptNum%20 == 0 {
				fmt.Printf("⚠️  Attempt %d failed (%v) - fast retry in %dms (remaining: %v)...\n",
					attemptNum, attemptDuration, delayMs, remaining.Round(time.Second))
			}
		}

		if delay > remaining {
			delay = remaining
		}

		time.Sleep(delay)
	}
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

func (f *FastCheckout) RunFastCheckout(automation *Automation) error {
	startTime := time.Now()

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║           FAST CHECKOUT - API MODE                        ║")
	fmt.Println("║           (Browser-Free Lightning Speed)                  ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	if err := f.LoadSessionFromBrowser(automation); err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Always get SKU ID for validation, even if skipping add to cart
	skuID, err := f.GetSKUFromBrowser(automation, f.config.ItemURL)
	if err != nil {
		return fmt.Errorf("failed to get SKU ID: %w", err)
	}

	// Check cart state BEFORE trying to add to cart
	fmt.Println("🔍 Checking current cart state...")
	cartTotal, maxCredit, err := f.GetCartTotals()
	if err != nil {
		return fmt.Errorf("failed to query cart totals: %w", err)
	}

	// Validate existing cart contents before adding
	shouldAdd, err := f.ValidateCartContents(skuID, cartTotal)
	if err != nil {
		return fmt.Errorf("cart validation failed: %w", err)
	}

	// Now add to cart if not skipping AND if cart validation says it's safe to add
	if !f.config.SkipAddToCart && shouldAdd {
		if err := f.AddToCart(skuID, automation); err != nil {
			return fmt.Errorf("failed to add to cart: %w", err)
		}

		// Re-query cart totals after adding
		cartTotal, maxCredit, err = f.GetCartTotals()
		if err != nil {
			return fmt.Errorf("failed to query cart totals after add: %w", err)
		}

		// Validate again after adding to cart
		fmt.Println("🔍 Validating cart after adding item...")
		_, err = f.ValidateCartContents(skuID, cartTotal)
		if err != nil {
			return fmt.Errorf("post-add cart validation failed: %w", err)
		}
	} else if !shouldAdd {
		fmt.Println("⏭️  Skipping add to cart (proceeding with current cart contents)")
	} else {
		fmt.Println("⏭️  Skipping add to cart (item already in cart)")
	}

	if f.config.AutoApplyCredit {
		creditToApply := cartTotal
		if creditToApply > maxCredit {
			fmt.Printf("⚠️  Cart total ($%.2f) exceeds max applicable credit ($%.2f)\n", cartTotal, maxCredit)
			fmt.Printf("   Applying maximum credit of $%.2f\n", maxCredit)
			creditToApply = maxCredit
		}

		if creditToApply > 0 {
			if err := f.ApplyStoreCredit(creditToApply); err != nil {
				return fmt.Errorf("failed to apply credit: %w", err)
			}
			// OPTIMIZATION: We know the total is $0 after applying credit, no need to re-query
			cartTotal = 0
			if f.config.DebugMode {
				fmt.Printf("[DEBUG] Cart total after credit: $%.2f (optimized - not re-queried)\n", cartTotal)
			}
		} else {
			fmt.Println("ℹ️  No credit needed (cart total is $0)")
		}
	}

	if cartTotal == 0 {
		fmt.Println("➡️  Moving to billing/addresses step...")
		if err := f.NextStep(); err != nil {
			return fmt.Errorf("failed to move to billing/addresses: %w", err)
		}

		// OPTIMIZATION: Cache address ID if not already cached
		if f.cachedAddressID == "" {
			addressID, err := f.GetDefaultBillingAddress()
			if err != nil {
				return fmt.Errorf("failed to get billing address: %w", err)
			}
			f.cachedAddressID = addressID
			if f.config.DebugMode {
				fmt.Printf("[DEBUG] Cached billing address: %s\n", addressID)
			}
		} else if f.config.DebugMode {
			fmt.Printf("[DEBUG] Using cached billing address: %s\n", f.cachedAddressID)
		}

		if err := f.AssignBillingAddress(f.cachedAddressID); err != nil {
			return fmt.Errorf("failed to assign billing address: %w", err)
		}

		if !f.config.DryRun {
			fmt.Println("🎯 Completing order (validating cart)...")
			if err := f.ValidateCart(automation); err != nil {
				return fmt.Errorf("failed to validate cart: %w", err)
			}
			fmt.Println("✓ ORDER COMPLETED!")
		} else {
			fmt.Println("🧪 DRY RUN - Stopping before final submission")
		}
	} else {
		fmt.Printf("➡️  Moving to payment step (remaining balance: $%.2f)...\n", cartTotal)
		if err := f.NextStep(); err != nil {
			return fmt.Errorf("failed to move to payment: %w", err)
		}

		if !f.config.DryRun {
			fmt.Println("🎯 Completing order with payment...")
			if err := f.NextStep(); err != nil {
				return fmt.Errorf("failed to complete order: %w", err)
			}
			fmt.Println("✓ ORDER COMPLETED!")
		} else {
			fmt.Println("🧪 DRY RUN - Stopping before final submission")
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n⚡ Total checkout time: %v\n", elapsed)
	fmt.Printf("🎯 Target: <1 second | Actual: %v\n", elapsed)

	if elapsed.Milliseconds() < 1000 {
		fmt.Println("🏆 ACHIEVED SUB-SECOND CHECKOUT!")
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

	// Wait until it's time to start
	now := time.Now()
	if now.Before(startRetryTime) {
		waitDuration := startRetryTime.Sub(now)
		fmt.Printf("⏳ Waiting %v until retry window starts...\n", waitDuration.Round(time.Second))
		time.Sleep(waitDuration)
		fmt.Println("✓ Retry window started!")
	} else if now.After(endRetryTime) {
		return fmt.Errorf("sale window has already passed (ended at %s)", endRetryTime.Local().Format(time.RFC1123))
	} else {
		fmt.Println("⚡ Already in retry window - starting immediately!")
	}

	if err := f.LoadSessionFromBrowser(automation); err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	skuID, err := f.GetSKUFromBrowser(automation, f.config.ItemURL)
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

	// Check cart state BEFORE starting aggressive add-to-cart retries
	fmt.Println("🔍 Checking current cart state before starting...")
	preCheckCartTotal, _, err := f.GetCartTotals()
	if err != nil {
		return fmt.Errorf("failed to query initial cart totals: %w", err)
	}

	// Validate existing cart contents before Phase 1
	shouldAdd, err := f.ValidateCartContents(skuID, preCheckCartTotal)
	if err != nil {
		return fmt.Errorf("pre-flight cart validation failed: %w", err)
	}

	// Track overall timing
	startTime := time.Now()

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

			// Ultra-fast retry delays
			var delay time.Duration
			if isOutOfStockError(err) {
				delay = time.Duration(5+rand.Intn(15)) * time.Millisecond // 5-20ms
			} else if isRateLimitError(err) {
				delay = time.Duration(50+rand.Intn(100)) * time.Millisecond // 50-150ms
			} else {
				delay = time.Duration(5+rand.Intn(25)) * time.Millisecond // 5-30ms
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

	// Query cart totals after Phase 1
	cartTotal, maxCredit, err := f.GetCartTotals()
	if err != nil {
		return fmt.Errorf("failed to query cart totals: %w", err)
	}

	// Validate cart contents after Phase 1 (add-to-cart) and before applying credit
	fmt.Println("🔍 Validating cart before proceeding with purchase...")
	_, err = f.ValidateCartContents(skuID, cartTotal)
	if err != nil {
		return fmt.Errorf("pre-purchase cart validation failed: %w", err)
	}

	if f.config.AutoApplyCredit {
		creditToApply := cartTotal
		if creditToApply > maxCredit {
			fmt.Printf("⚠️  Cart total ($%.2f) exceeds max credit ($%.2f) - applying max\n", cartTotal, maxCredit)
			creditToApply = maxCredit
		}

		if creditToApply > 0 {
			if err := f.ApplyStoreCredit(creditToApply); err != nil {
				return fmt.Errorf("failed to apply credit: %w", err)
			}
			cartTotal = 0
		}
	}

	if cartTotal == 0 {
		fmt.Println("➡️  Moving to billing/addresses step...")
		if err := f.NextStep(); err != nil {
			return fmt.Errorf("failed to move to billing/addresses: %w", err)
		}

		if err := f.AssignBillingAddress(f.cachedAddressID); err != nil {
			return fmt.Errorf("failed to assign billing address: %w", err)
		}

		if !f.config.DryRun {
			fmt.Println("🎯 Completing order with aggressive retries until sale window ends...")
			// Use the same endRetryTime from the sale window for validation
			if err := f.ValidateCartWithDeadline(automation, endRetryTime); err != nil {
				return fmt.Errorf("failed to validate cart: %w", err)
			}
			fmt.Println("\n✅ ORDER COMPLETED!")
		} else {
			fmt.Println("🧪 DRY RUN - Stopping before final submission")
		}
	} else {
		fmt.Printf("➡️  Moving to payment step (balance: $%.2f)...\n", cartTotal)
		if err := f.NextStep(); err != nil {
			return fmt.Errorf("failed to move to payment: %w", err)
		}

		if !f.config.DryRun {
			fmt.Println("🎯 Completing order with payment...")
			if err := f.NextStep(); err != nil {
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

	resp, err := f.graphqlRequest(request)
	if err != nil {
		return err
	}

	// Success - item added to cart
	_ = resp
	return nil
}
