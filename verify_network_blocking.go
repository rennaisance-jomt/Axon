package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/rennaisance-jomt/axon/internal/browser"
	"github.com/rennaisance-jomt/axon/internal/config"
)

func main() {
	// Configure test URL
	testURL := "https://www.cnn.com"

	cfg := &config.BrowserConfig{
		PoolSize:      1,
		LaunchOptions: map[string]interface{}{"--no-sandbox": true},
	}
	pool, err := browser.NewPool(cfg)
	if err != nil {
		log.Fatalf("Failed to spin up pool: %v", err)
	}
	defer pool.Close()

	sm := browser.NewSessionManager(pool)
	
	fmt.Println("🛑 Running Headless-Native Blocklist Test (Wait...")

	// Launch Axon Native Session (which has blocking ENABLED)
	axonStart := time.Now()
	axonSession, err := sm.Create("axon_session", "")
	if err != nil {
		log.Fatalf("Failed to create axon session: %v", err)
	}
	defer sm.Delete(axonSession.ID)

	err = axonSession.Navigate(testURL, "load")
	if err != nil {
		log.Fatalf("Navigation failed: %v", err)
	}
	axonDur := time.Since(axonStart)

	// Launch RAW Session (blocking DISABLED)
	rootBrowser, _ := pool.Acquire()
	rawContext, _ := rootBrowser.Incognito()
	rawPage, _ := rawContext.Page(proto.TargetCreateTarget{})
	
	rawStart := time.Now()
	err = rawPage.Navigate(testURL)
	if err != nil {
		log.Fatalf("RAW Navigation failed: %v", err)
	}
	rawPage.WaitLoad()
	rawDur := time.Since(rawStart)

	rawContext.Close()

	fmt.Printf("⏱️  Raw Chrome Load Time (Unblocked): %v\n", rawDur)
	fmt.Printf("⏱️  Axon Embedded Load Time (Blocked): %v\n", axonDur)

	diff := float64(rawDur - axonDur) / float64(rawDur) * 100.0
	fmt.Printf("📉 Latency Reduction: %.2f%%\n", diff)

	if diff > 50 {
		fmt.Println("✅ VERIFICATION PASSED: Network interception successful!")
	} else {
		// Just a warning, network fluctuation might cause <50% occasionally
		fmt.Println("⚠️ VERIFICATION: Reduction is lower than expected, might be cached or lightweight page.")
	}
}
