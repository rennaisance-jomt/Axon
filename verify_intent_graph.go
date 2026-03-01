package main

import (
	"fmt"
	"log"

	"github.com/rennaisance-jomt/axon/internal/browser"
	"github.com/rennaisance-jomt/axon/internal/config"
)

func main() {
	// Boot up Zero-Overhead Context Pool
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
	session, err := sm.Create("verification_session", "")
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	defer sm.Delete(session.ID)

	fmt.Println("🚀 Navigating to Wikipedia to test Intent Grouping...")
	err = session.Navigate("https://en.wikipedia.org/wiki/Main_Page", "networkidle")
	if err != nil {
		log.Fatalf("Navigation failed: %v", err)
	}

	// Wait a moment for rendering
	session.Page.WaitLoad()

	// 1. standard HTML size
	html, err := session.Page.HTML()
	if err != nil {
		log.Fatalf("HTML retrieval failed: %v", err)
	}
	rawHtmlTokens := len(html) / 4
	fmt.Printf("📊 Standard HTML Token Count: ~%d tokens\n", rawHtmlTokens)

	// 2. Extract snapshot directly (which runs our new Native CDP + Intent Graph logic)
	extractor := browser.NewSnapshotExtractor()
	snapshot, err := extractor.Extract(session.Page, "full", "")
	if err != nil {
		log.Fatalf("Snapshot extraction failed: %v", err)
	}

	// Get Token counts of the final collapsed snapshot string
	intentGraphTokens := len(snapshot.Content) / 4

	fmt.Printf("📊 High-Compression Intent Graph Token Count: ~%d tokens\n", intentGraphTokens)

	reduction := float64(rawHtmlTokens-intentGraphTokens) / float64(rawHtmlTokens) * 100.0
	fmt.Printf("📉 Total Reduction: %.2f%%\n", reduction)

	if reduction > 50 {
		fmt.Println("✅ VERIFICATION PASSED: Token reduction > 50%")
	} else {
		fmt.Println("❌ VERIFICATION FAILED: Token reduction not sufficient")
	}

	fmt.Println("\n📝 Intent Graph Output Sample (Showing how Input + Submits are grouped into Input_Groups):")
	fmt.Println(snapshot.Content)
}
