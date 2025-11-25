package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
)

type FastCheckout struct {
	client    *http.Client
	config    *Config
	baseURL   string
	graphqlURL string
	cookies   []*http.Cookie
	csrfToken string
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

	return nil
}

func (f *FastCheckout) GetSKUSlugFromURL(itemURL string) (string, error) {
	fmt.Printf("🔍 Extracting SKU slug from %s...\n", itemURL)

	req, err := http.NewRequest("GET", itemURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

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
	movements := 3 + rand.Intn(4)
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

		time.Sleep(time.Duration(30+rand.Intn(70)) * time.Millisecond)
	}

	scrollAmount := 50 + rand.Intn(100)
	page.Eval(fmt.Sprintf(`() => window.scrollBy(0, %d)`, scrollAmount))

	time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)

	page.Eval(`() => {
		var event = new Event('mousedown', {bubbles: true});
		document.dispatchEvent(event);
	}`)

	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

	page.Eval(`() => {
		var event = new Event('mouseup', {bubbles: true});
		document.dispatchEvent(event);
	}`)

	time.Sleep(time.Duration(150+rand.Intn(250)) * time.Millisecond)
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

	maxAttempts := 25
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(200 * time.Millisecond)

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

func (f *FastCheckout) AddToCart(skuID string, automation *Automation) error {
	fmt.Printf("🛒 Adding to cart (API) with retry mechanism...\n")
	fmt.Printf("[DEBUG] SKU ID: %s\n", skuID)

	totalWindow := f.config.StartBeforeSaleSeconds + f.config.ContinueAfterSaleSeconds
	fmt.Printf("⏱️  Will retry for up to %d seconds (sale window: -%ds to +%ds)\n",
		totalWindow, f.config.StartBeforeSaleSeconds, f.config.ContinueAfterSaleSeconds)

	startTime := time.Now()
	retryDeadline := startTime.Add(time.Duration(totalWindow) * time.Second)
	attemptNum := 0

	mutation := `mutation AddCartMultiItemMutation($query: [CartAddInput!], $captcha: String) {
  store(name: "pledge") {
    cart {
      mutations {
        addMany(query: $query, captcha: $captcha) {
          count
          resources {
            id
            title
          }
        }
      }
    }
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

		variables := map[string]interface{}{
			"query": []map[string]interface{}{
				{
					"qty":   1,
					"skuId": skuID,
				},
			},
		}

		if recaptchaToken != "" && recaptchaToken != "null" && recaptchaToken != "undefined" && len(recaptchaToken) > 10 {
			variables["captcha"] = recaptchaToken
			if f.config.DebugMode && attemptNum <= 3 {
				fmt.Printf("[DEBUG] Attempt %d: Including reCAPTCHA token in request\n", attemptNum)
			}
		} else {
			if f.config.DebugMode && attemptNum <= 3 {
				fmt.Printf("[DEBUG] Attempt %d: NOT including reCAPTCHA (invalid or missing)\n", attemptNum)
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
			delayMs := f.config.RetryDelayMinMs + rand.Intn(f.config.RetryDelayMaxMs-f.config.RetryDelayMinMs+1)
			delay = time.Duration(delayMs) * time.Millisecond

			if attemptNum%10 == 0 || attemptNum <= 3 {
				fmt.Printf("🔐 Attempt %d: CAPTCHA verification needed - retrying in %dms (remaining: %v)...\n",
					attemptNum, delayMs, remaining.Round(time.Second))
			}
		} else if isRateLimited {
			delayMs := 2000 + rand.Intn(3000)
			delay = time.Duration(delayMs) * time.Millisecond
			fmt.Printf("⚠️  Attempt %d: Rate limited (4227) - waiting %dms before retry (remaining: %v)...\n",
				attemptNum, delayMs, remaining.Round(time.Second))
		} else if isOutOfStock {
			delayMs := f.config.RetryDelayMinMs + rand.Intn(f.config.RetryDelayMaxMs-f.config.RetryDelayMinMs+1)
			delay = time.Duration(delayMs) * time.Millisecond

			if attemptNum%10 == 0 {
				fmt.Printf("⏳ Attempt %d: Out of stock (4226) - waiting for sale to start (remaining: %v)...\n",
					attemptNum, remaining.Round(time.Second))
			}
		} else {
			delayMs := f.config.RetryDelayMinMs + rand.Intn(f.config.RetryDelayMaxMs-f.config.RetryDelayMinMs+1)
			delay = time.Duration(delayMs) * time.Millisecond

			attemptDuration := time.Since(attemptStart)
			fmt.Printf("⚠️  Attempt %d failed (%v) - retrying in %dms (remaining: %v)...\n",
				attemptNum, attemptDuration, delayMs, remaining.Round(time.Second))
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
	fmt.Println("✅ Validating and completing order...")

	startTime := time.Now()
	retryDeadline := startTime.Add(time.Duration(f.config.RetryDurationSeconds) * time.Second)
	attemptNum := 0

	for {
		attemptNum++
		attemptStart := time.Now()

		recaptchaToken, err := f.GetRecaptchaToken(automation, "store/cart/validate")
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to get reCAPTCHA token: %v\n", err)
		}

		mark := fmt.Sprintf("%d", time.Now().Unix())

		mutation := `mutation CartValidateCartMutation($storeFront: String, $token: String, $mark: String) {
  store(name: $storeFront) {
    cart {
      mutations {
        validate(mark: $mark, token: $token)
      }
      flow {
        steps {
          step
          action
          finalStep
          active
        }
        current {
          orderCreated
        }
      }
    }
    order {
      slug
    }
  }
}`

		variables := map[string]interface{}{
			"storeFront": "pledge",
			"mark":       mark,
		}

		if recaptchaToken != "" {
			variables["token"] = recaptchaToken
		}

		request := []GraphQLRequest{
			{
				OperationName: "CartValidateCartMutation",
				Variables:     variables,
				Query:         mutation,
			},
		}

		resp, err := f.graphqlRequest(request)

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

		remaining := retryDeadline.Sub(time.Now())

		if remaining <= 0 {
			elapsed := time.Since(startTime)
			fmt.Printf("❌ Retry timeout reached after %d attempts in %v\n", attemptNum, elapsed)
			return fmt.Errorf("cart validation failed after %d attempts: %w", attemptNum, err)
		}

		var delay time.Duration
		isRateLimited := isRateLimitError(err)
		isOutOfStock := isOutOfStockError(err)

		if isRateLimited {
			delayMs := 2000 + rand.Intn(3000)
			delay = time.Duration(delayMs) * time.Millisecond
			fmt.Printf("⚠️  Attempt %d: Rate limited (4227) during validation - waiting %dms before retry (remaining: %v)...\n",
				attemptNum, delayMs, remaining.Round(time.Second))
		} else if isOutOfStock {
			delayMs := f.config.RetryDelayMinMs + rand.Intn(f.config.RetryDelayMaxMs-f.config.RetryDelayMinMs+1)
			delay = time.Duration(delayMs) * time.Millisecond

			if attemptNum%10 == 0 {
				fmt.Printf("⏳ Attempt %d: Item unavailable (4226) during validation - retrying (remaining: %v)...\n",
					attemptNum, remaining.Round(time.Second))
			}
		} else {
			delayMs := f.config.RetryDelayMinMs + rand.Intn(f.config.RetryDelayMaxMs-f.config.RetryDelayMinMs+1)
			delay = time.Duration(delayMs) * time.Millisecond

			attemptDuration := time.Since(attemptStart)
			fmt.Printf("⚠️  Attempt %d failed (%v) - retrying in %dms (remaining: %v)...\n",
				attemptNum, attemptDuration, delayMs, remaining.Round(time.Second))
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://robertsspaceindustries.com")
	req.Header.Set("Referer", "https://robertsspaceindustries.com/")

	for _, cookie := range f.cookies {
		req.AddCookie(cookie)
	}

	if f.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", f.csrfToken)
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

	if !f.config.SkipAddToCart {
		skuID, err := f.GetSKUFromBrowser(automation, f.config.ItemURL)
		if err != nil {
			return fmt.Errorf("failed to get SKU ID: %w", err)
		}

		if err := f.AddToCart(skuID, automation); err != nil {
			return fmt.Errorf("failed to add to cart: %w", err)
		}
	} else {
		fmt.Println("⏭️  Skipping add to cart (item already in cart)")
	}

	fmt.Println("➡️  Moving to billing step...")
	if err := f.NextStep(); err != nil {
		return fmt.Errorf("failed to move to billing: %w", err)
	}

	if f.config.AutoApplyCredit {
		cartTotal, maxCredit, err := f.GetCartTotals()
		if err != nil {
			return fmt.Errorf("failed to query cart totals: %w", err)
		}

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
		} else {
			fmt.Println("ℹ️  No credit needed (cart total is $0)")
		}
	}

	cartTotal, _, err := f.GetCartTotals()
	if err != nil {
		return fmt.Errorf("failed to query updated cart totals: %w", err)
	}

	if cartTotal == 0 {
		fmt.Println("➡️  Moving to addresses step (total is $0)...")
		if err := f.NextStep(); err != nil {
			return fmt.Errorf("failed to move to addresses: %w", err)
		}

		addressID, err := f.GetDefaultBillingAddress()
		if err != nil {
			return fmt.Errorf("failed to get billing address: %w", err)
		}

		if err := f.AssignBillingAddress(addressID); err != nil {
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
