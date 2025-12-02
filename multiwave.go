package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// MultiWaveOrchestrator manages multiple sale windows throughout a sale day
type MultiWaveOrchestrator struct {
	config       *Config
	timeSync     *TimeSync
	automation   *Automation
	fastCheckout *FastCheckout
	rand         *rand.Rand
}

// NewMultiWaveOrchestrator creates a new multi-wave orchestrator
func NewMultiWaveOrchestrator(config *Config, automation *Automation, fastCheckout *FastCheckout) *MultiWaveOrchestrator {
	return &MultiWaveOrchestrator{
		config:       config,
		timeSync:     NewTimeSync(config.DebugMode),
		automation:   automation,
		fastCheckout: fastCheckout,
		rand:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Run executes the multi-wave sale workflow
func (mwo *MultiWaveOrchestrator) Run() error {
	// Step 1: Synchronize time with reliable time servers
	fmt.Println(T("multiwave_syncing_time"))
	if err := mwo.timeSync.Sync(); err != nil {
		return fmt.Errorf("failed to sync time: %w", err)
	}

	offset := mwo.timeSync.GetOffset()
	if offset > 0 {
		fmt.Printf(T("multiwave_time_synced_ahead")+"\n", offset)
	} else if offset < 0 {
		fmt.Printf(T("multiwave_time_synced_behind")+"\n", -offset)
	} else {
		fmt.Println(T("multiwave_time_synced_perfect"))
	}

	// Step 2: Parse and validate all sale windows
	saleWindows, err := mwo.parseSaleWindows()
	if err != nil {
		return err
	}

	if len(saleWindows) == 0 {
		return fmt.Errorf("no sale windows configured")
	}

	fmt.Printf(T("multiwave_configured_waves")+"\n", len(saleWindows))
	fmt.Println()

	// Display all waves
	for i, waveTime := range saleWindows {
		localTime := waveTime.Local()
		fmt.Printf(T("multiwave_wave_time")+"\n", i+1, waveTime.Format(time.RFC3339), localTime.Format("15:04:05 MST"))
	}
	fmt.Println()

	// Step 3: Determine which wave to start from based on current time
	now := mwo.timeSync.Now()
	postWaveDuration := time.Duration(mwo.config.PostWaveTimeoutMinutes) * time.Minute

	startWaveIndex := -1
	for i, waveTime := range saleWindows {
		waveEndTime := waveTime.Add(postWaveDuration)

		if now.Before(waveEndTime) {
			// This wave is still relevant (either upcoming or active)
			startWaveIndex = i
			break
		}
	}

	// Check if all waves have passed
	if startWaveIndex == -1 {
		fmt.Println()
		fmt.Println(T("multiwave_all_waves_passed"))
		fmt.Printf(T("multiwave_last_wave_was")+"\n", len(saleWindows), saleWindows[len(saleWindows)-1].Local().Format("15:04:05 MST"))
		fmt.Println(T("multiwave_exiting_no_waves"))
		return fmt.Errorf("all sale waves have ended")
	}

	// Inform user if we're skipping past waves
	if startWaveIndex > 0 {
		fmt.Println()
		fmt.Printf(T("multiwave_skipping_past_waves")+"\n", startWaveIndex)
		for i := 0; i < startWaveIndex; i++ {
			fmt.Printf("   • Wave %d (%s) - Ended\n", i+1, saleWindows[i].Local().Format("15:04:05 MST"))
		}
		fmt.Println()
	}

	// Step 4: Process waves starting from the first relevant wave
	for i := startWaveIndex; i < len(saleWindows); i++ {
		waveNum := i + 1
		waveTime := saleWindows[i]

		fmt.Printf(T("multiwave_wave_header")+"\n", waveNum, len(saleWindows))
		fmt.Println()

		success, err := mwo.processWave(waveNum, waveTime)
		if err != nil {
			return fmt.Errorf("wave %d failed: %w", waveNum, err)
		}

		if success {
			fmt.Println()
			fmt.Println(T("multiwave_purchase_success"))
			fmt.Println(T("multiwave_exiting_gracefully"))
			return nil
		}

		// Wave failed - check if there's a next wave
		if waveNum < len(saleWindows) {
			nextWaveTime := saleWindows[i+1]
			now := mwo.timeSync.Now()
			waitDuration := nextWaveTime.Sub(now)

			fmt.Println()
			fmt.Printf(T("multiwave_wave_failed")+"\n", waveNum)
			fmt.Printf(T("multiwave_moving_to_next")+"\n", waveNum+1)
			fmt.Printf(T("multiwave_next_wave_in")+"\n", waitDuration.Round(time.Second))
			fmt.Printf(T("multiwave_staying_dormant")+"\n")
			fmt.Println()
		} else {
			// This was the last wave
			fmt.Println()
			fmt.Printf(T("multiwave_wave_failed")+"\n", waveNum)
			fmt.Println(T("multiwave_was_last_wave"))
			fmt.Println()
		}
	}

	// All waves completed without success
	fmt.Println()
	fmt.Println(T("multiwave_all_waves_failed"))
	return fmt.Errorf("checkout failed for all %d waves", len(saleWindows))
}

// parseSaleWindows parses the configured sale window strings into time.Time objects
// Supports user-friendly formats like "2025-01-15 16:00" (assumes UTC)
func (mwo *MultiWaveOrchestrator) parseSaleWindows() ([]time.Time, error) {
	var windows []time.Time

	for i, timeStr := range mwo.config.SaleWindows {
		t, err := ParseSaleTime(timeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid sale window %d (%s): %w", i+1, timeStr, err)
		}
		windows = append(windows, t)
	}

	return windows, nil
}

// processWave handles a single sale wave
func (mwo *MultiWaveOrchestrator) processWave(waveNum int, waveTime time.Time) (bool, error) {
	// Calculate activation time (pre-wave polling starts)
	preWaveDuration := time.Duration(mwo.config.PreWaveActivationMinutes) * time.Minute
	activationTime := waveTime.Add(-preWaveDuration)

	// Wait until activation time
	now := mwo.timeSync.Now()
	if now.Before(activationTime) {
		waitDuration := activationTime.Sub(now)
		fmt.Printf(T("multiwave_waiting_for_activation")+"\n", waitDuration.Round(time.Second))
		fmt.Printf(T("multiwave_activation_time")+"\n", activationTime.Local().Format("15:04:05 MST"))
		fmt.Println()

		// Sleep until activation, checking periodically for time sync
		mwo.sleepUntilWithUpdates(activationTime)
	}

	// Start pre-wave polling
	fmt.Println(T("multiwave_prewave_polling_start"))
	fmt.Printf(T("multiwave_polling_url")+"\n", mwo.config.ItemURL)
	fmt.Println()

	pageAvailableTime, err := mwo.pollForProductPage(waveTime)
	if err != nil {
		return false, err
	}

	fmt.Println()
	fmt.Println(T("multiwave_product_page_available"))
	if pageAvailableTime.Before(waveTime) {
		earlyBy := waveTime.Sub(pageAvailableTime)
		fmt.Printf(T("multiwave_page_available_early")+"\n", earlyBy.Round(time.Second))
	}
	fmt.Println()

	// Navigate to product page (it's now available)
	fmt.Println(T("multiwave_navigating_to_product"))
	if err := mwo.automation.page.Navigate(mwo.config.ItemURL); err != nil {
		return false, fmt.Errorf("failed to navigate to product page: %w", err)
	}
	if err := mwo.automation.page.WaitLoad(); err != nil {
		return false, fmt.Errorf("failed to load product page: %w", err)
	}

	// Extract and cache SKU
	fmt.Println(T("multiwave_extracting_sku"))
	if err := mwo.automation.extractAndCacheSKU(); err != nil {
		return false, fmt.Errorf("failed to extract SKU: %w", err)
	}

	// Calculate timeout (post-wave duration after wave start)
	postWaveDuration := time.Duration(mwo.config.PostWaveTimeoutMinutes) * time.Minute
	timeoutTime := waveTime.Add(postWaveDuration)

	fmt.Println()
	fmt.Println(T("multiwave_attempting_checkout"))
	fmt.Printf(T("multiwave_timeout_at")+"\n", timeoutTime.Local().Format("15:04:05 MST"))
	fmt.Println()

	// Attempt checkout with timeout
	success := mwo.attemptCheckoutWithTimeout(timeoutTime)

	return success, nil
}

// pollForProductPage polls the product URL until it returns 200 (not 404)
func (mwo *MultiWaveOrchestrator) pollForProductPage(waveTime time.Time) (time.Time, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	attemptNum := 0
	lastProgressUpdate := time.Now()

	for {
		attemptNum++
		now := mwo.timeSync.Now()

		// Check if we've passed the wave time + post-wave timeout
		postWaveDuration := time.Duration(mwo.config.PostWaveTimeoutMinutes) * time.Minute
		if now.After(waveTime.Add(postWaveDuration)) {
			return time.Time{}, fmt.Errorf("product page never became available (timed out)")
		}

		req, err := http.NewRequest("HEAD", mwo.config.ItemURL, nil)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				// Success! Page is available
				return mwo.timeSync.Now(), nil
			}

			// Show progress every 10 seconds
			if time.Since(lastProgressUpdate) >= 10*time.Second {
				timeUntilWave := waveTime.Sub(now)
				if timeUntilWave > 0 {
					fmt.Printf(T("multiwave_polling_progress_before")+"\n", resp.StatusCode, timeUntilWave.Round(time.Second))
				} else {
					timeSinceWave := now.Sub(waveTime)
					fmt.Printf(T("multiwave_polling_progress_after")+"\n", resp.StatusCode, timeSinceWave.Round(time.Second))
				}
				lastProgressUpdate = time.Now()
			}
		}

		// Sleep before next attempt (variable millisecond delay for human-like timing)
		minDelay := mwo.config.PollingDelayMinMs
		maxDelay := mwo.config.PollingDelayMaxMs
		delayMs := minDelay + mwo.rand.Intn(maxDelay-minDelay+1)
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}
}

// attemptCheckoutWithTimeout attempts checkout until timeout or success
func (mwo *MultiWaveOrchestrator) attemptCheckoutWithTimeout(timeoutTime time.Time) bool {
	// Start checkout attempts
	checkoutErr := mwo.fastCheckout.RunFastCheckout(mwo.automation)

	if checkoutErr == nil {
		// Success!
		return true
	}

	// Checkout failed
	fmt.Printf(T("multiwave_checkout_failed")+"\n", checkoutErr)

	// Check if we should retry or if we've timed out
	now := mwo.timeSync.Now()
	if now.After(timeoutTime) {
		fmt.Println(T("multiwave_wave_timeout"))
		return false
	}

	// Could add retry logic here if needed
	// For now, single attempt per wave

	return false
}

// sleepUntilWithUpdates sleeps until target time, with periodic progress updates
func (mwo *MultiWaveOrchestrator) sleepUntilWithUpdates(targetTime time.Time) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		now := mwo.timeSync.Now()
		remaining := targetTime.Sub(now)

		if remaining <= 0 {
			return
		}

		// If less than 30 seconds remaining, just sleep the remainder
		if remaining < 30*time.Second {
			time.Sleep(remaining)
			return
		}

		// Wait for next tick or timeout
		select {
		case <-ticker.C:
			// Resync time if needed (every hour)
			if mwo.timeSync.ShouldResync() {
				fmt.Println(T("multiwave_resyncing_time"))
				if err := mwo.timeSync.Sync(); err != nil {
					fmt.Printf(T("multiwave_resync_failed")+"\n", err)
				}
			}

			// Show progress
			now = mwo.timeSync.Now()
			remaining = targetTime.Sub(now)
			if remaining > 0 {
				fmt.Printf(T("multiwave_waiting_update")+"\n", remaining.Round(time.Second))
			}
		}
	}
}
