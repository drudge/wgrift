//go:build mage

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/magefile/mage/mg"
)

const (
	screenshotDir  = "docs/screenshots"
	screenshotPort = "18080"
	baseURL        = "http://localhost:" + screenshotPort
	wasmLoadWait   = 3 * time.Second
)

// Screenshots builds the project, starts a demo server, and captures UI screenshots.
func Screenshots() error {
	mg.Deps(Build)

	// Remove stale DB so demo seed runs fresh
	os.Remove("wgrift.db")

	if err := os.MkdirAll(screenshotDir, 0o755); err != nil {
		return fmt.Errorf("creating screenshot dir: %w", err)
	}

	// Write temporary config to use a non-default port
	cfgContent := fmt.Sprintf("server:\n  listen: \"0.0.0.0:%s\"\n", screenshotPort)
	cfgPath := "wgrift-screenshots.yaml"
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		return fmt.Errorf("writing temp config: %w", err)
	}
	defer os.Remove(cfgPath)

	// Start demo server
	cmd := exec.Command("./bin/wgrift", "serve", "--config", cfgPath)
	cmd.Env = append(os.Environ(),
		"WGRIFT_MASTER_KEY=dev-master-key",
		"WGRIFT_DEMO_MODE=true",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}
	defer func() {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	}()

	// Wait for server to be ready
	if err := waitForServer(baseURL+"/api/v1/auth/session", 15*time.Second); err != nil {
		return err
	}

	if err := captureScreenshots(); err != nil {
		return fmt.Errorf("capturing screenshots: %w", err)
	}

	fmt.Println("\nScreenshots saved to", screenshotDir)
	return nil
}

func waitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("server did not become ready within %s", timeout)
}

func captureScreenshots() error {
	// Get session cookie and peer ID via API before launching browser
	cookie, err := apiLogin()
	if err != nil {
		return fmt.Errorf("API login: %w", err)
	}
	peerID, err := findFirstPeerID(cookie)
	if err != nil {
		return fmt.Errorf("finding peer ID: %w", err)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(1440, 900),
		chromedp.Flag("force-dark-mode", true),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	// 1. Login page (unauthenticated)
	log.Println("Capturing login page...")
	if err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL+"/login"),
		chromedp.Sleep(wasmLoadWait),
	); err != nil {
		return fmt.Errorf("login page: %w", err)
	}
	if err := screenshotCurrent(ctx, "login.png"); err != nil {
		return fmt.Errorf("login screenshot: %w", err)
	}

	// 2. Inject session cookie for authenticated pages
	if err := chromedp.Run(ctx,
		network.SetCookie(cookie.Name, cookie.Value).
			WithDomain("localhost").
			WithPath("/"),
	); err != nil {
		return fmt.Errorf("setting cookie: %w", err)
	}

	// 3. Dashboard
	log.Println("Capturing dashboard...")
	if err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL+"/"),
		chromedp.Sleep(wasmLoadWait),
	); err != nil {
		return fmt.Errorf("dashboard: %w", err)
	}
	if err := screenshotCurrent(ctx, "dashboard.png"); err != nil {
		return fmt.Errorf("dashboard screenshot: %w", err)
	}

	// 4. Interface detail (wg0)
	log.Println("Capturing interface detail...")
	if err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL+"/interfaces/wg0"),
		chromedp.Sleep(wasmLoadWait),
	); err != nil {
		return fmt.Errorf("interface detail: %w", err)
	}
	if err := screenshotCurrent(ctx, "interface-detail.png"); err != nil {
		return fmt.Errorf("interface detail screenshot: %w", err)
	}

	// 5. Add Peer — click the button on the current page
	log.Println("Capturing add peer form...")
	if err := chromedp.Run(ctx,
		// Use JavaScript to find and click the button by text content
		chromedp.Evaluate(`document.querySelectorAll('button').forEach(b => { if (b.textContent.trim() === 'Add Peer') b.click(); })`, nil),
		chromedp.Sleep(wasmLoadWait),
	); err != nil {
		return fmt.Errorf("clicking add peer: %w", err)
	}
	if err := screenshotCurrent(ctx, "add-peer.png"); err != nil {
		return fmt.Errorf("add peer screenshot: %w", err)
	}

	// 6. Peer config
	log.Printf("Capturing peer config (peer: %s)...\n", peerID[:8])
	if err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL+"/interfaces/wg0/peers/"+peerID+"/config"),
		chromedp.Sleep(wasmLoadWait),
	); err != nil {
		return fmt.Errorf("peer config: %w", err)
	}
	if err := screenshotCurrent(ctx, "peer-config.png"); err != nil {
		return fmt.Errorf("peer config screenshot: %w", err)
	}

	// 7. Connection logs
	log.Println("Capturing connection logs...")
	if err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL+"/logs"),
		chromedp.Sleep(wasmLoadWait),
	); err != nil {
		return fmt.Errorf("logs: %w", err)
	}
	if err := screenshotCurrent(ctx, "logs.png"); err != nil {
		return fmt.Errorf("logs screenshot: %w", err)
	}

	// 8. Settings — use a tall viewport so SMTP + OIDC sections are both visible
	log.Println("Capturing settings page...")
	if err := chromedp.Run(ctx,
		emulation.SetDeviceMetricsOverride(1440, 1400, 1, false),
		chromedp.Navigate(baseURL+"/settings"),
		chromedp.Sleep(wasmLoadWait),
	); err != nil {
		return fmt.Errorf("settings: %w", err)
	}
	if err := screenshotCurrent(ctx, "settings.png"); err != nil {
		return fmt.Errorf("settings screenshot: %w", err)
	}
	// Reset viewport back to default
	if err := chromedp.Run(ctx,
		emulation.SetDeviceMetricsOverride(1440, 900, 1, false),
	); err != nil {
		return fmt.Errorf("resetting viewport: %w", err)
	}

	// 9. Edit peer — click edit on first peer to show email alerts section
	log.Println("Capturing edit peer form...")
	if err := chromedp.Run(ctx,
		// Use a tall viewport so the full edit form including email alerts is visible
		emulation.SetDeviceMetricsOverride(1440, 1600, 1, false),
		chromedp.Navigate(baseURL+"/interfaces/wg0"),
		chromedp.Sleep(wasmLoadWait),
		// Click the first "Edit peer" pencil button
		chromedp.Evaluate(`document.querySelector('button[title="Edit peer"]').click()`, nil),
		chromedp.Sleep(wasmLoadWait),
	); err != nil {
		return fmt.Errorf("edit peer: %w", err)
	}
	if err := screenshotCurrent(ctx, "edit-peer.png"); err != nil {
		return fmt.Errorf("edit peer screenshot: %w", err)
	}

	// 10. Mobile view
	log.Println("Capturing mobile view...")
	if err := chromedp.Run(ctx,
		emulation.SetDeviceMetricsOverride(390, 844, 2, true),
		chromedp.Navigate(baseURL+"/"),
		chromedp.Sleep(wasmLoadWait),
	); err != nil {
		return fmt.Errorf("mobile: %w", err)
	}
	if err := screenshotCurrent(ctx, "mobile.png"); err != nil {
		return fmt.Errorf("mobile screenshot: %w", err)
	}

	return nil
}

func screenshotCurrent(ctx context.Context, filename string) error {
	var buf []byte
	if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 90)); err != nil {
		return err
	}
	path := filepath.Join(screenshotDir, filename)
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return err
	}
	log.Printf("  → %s (%d KB)\n", path, len(buf)/1024)
	return nil
}

// apiLogin authenticates via the REST API and returns the session cookie.
func apiLogin() (*http.Cookie, error) {
	body := `{"username":"admin","password":"admin"}`
	resp, err := http.Post(baseURL+"/api/v1/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	for _, c := range resp.Cookies() {
		if c.Name == "wgrift_session" {
			return c, nil
		}
	}
	return nil, fmt.Errorf("no session cookie in login response")
}

// findFirstPeerID returns the ID of the first enabled peer on wg0.
func findFirstPeerID(cookie *http.Cookie) (string, error) {
	req, _ := http.NewRequest("GET", baseURL+"/api/v1/interfaces/wg0/peers", nil)
	req.AddCookie(cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			Enabled bool   `json:"enabled"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	for _, p := range result.Data {
		if p.Enabled {
			return p.ID, nil
		}
	}
	return "", fmt.Errorf("no enabled peers found for wg0")
}
