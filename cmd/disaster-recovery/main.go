package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	BaseURL          string
	OfflineToken     string
	SSOURL           string
	QuickstartName   string
	Interval         int
	ProxyURL         string
	httpClient       *http.Client
	accessToken      string
	accountID        string
	favoriteState    bool
	tokenRefreshTime time.Time
}

type SSOTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type JWTPayload struct {
	AccountID string `json:"account_id"`
	Sub       string `json:"sub"`
}

type QuickstartsResponse struct {
	Data []Quickstart `json:"data"`
}

type Quickstart struct {
	Name string `json:"name"`
}

type FavoriteRequest struct {
	QuickstartName string `json:"quickstartName"`
	Favorite       bool   `json:"favorite"`
}

type FavoriteResponse struct {
	Data FavoriteData `json:"data"`
}

type FavoriteData struct {
	ID             int    `json:"id"`
	AccountID      string `json:"accountId"`
	QuickstartName string `json:"quickstartName"`
	Favorite       bool   `json:"favorite"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

func main() {
	config := parseFlags()

	// Setup HTTP client
	config.httpClient = createHTTPClient(config.ProxyURL)

	// Exchange offline token for access token
	fmt.Println("[INFO] Exchanging offline token for access token...")
	if err := config.refreshAccessToken(); err != nil {
		log.Fatalf("[ERROR] Failed to refresh access token: %v", err)
	}

	// Extract account ID from JWT
	if err := config.extractAccountFromJWT(); err != nil {
		log.Fatalf("[ERROR] Failed to extract account from JWT: %v", err)
	}

	// Fetch a quickstart if not provided
	if config.QuickstartName == "" {
		if err := config.fetchAvailableQuickstart(); err != nil {
			log.Fatalf("[ERROR] Failed to fetch quickstart: %v", err)
		}
	}

	// Run continuous test
	config.runContinuous()
}

func parseFlags() *Config {
	config := &Config{}

	offlineTokenFile := flag.String("offline-token-file", "", "Path to file containing offline token")
	flag.StringVar(&config.BaseURL, "base-url", getEnv("CONSOLEDOT_BASE_URL", "https://console.stage.redhat.com"), "Base URL for the quickstarts API")
	flag.StringVar(&config.OfflineToken, "offline-token", os.Getenv("OFFLINE_TOKEN"), "Offline token from access.redhat.com/management/api")
	flag.StringVar(&config.SSOURL, "sso-url", getEnv("SSO_REFRESH_TOKEN_URL", "https://sso.stage.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token"), "SSO token refresh URL")
	flag.StringVar(&config.QuickstartName, "quickstart", "", "Quickstart name to toggle (default: fetch first available)")
	flag.IntVar(&config.Interval, "interval", 15, "Interval in seconds between toggles")
	flag.StringVar(&config.ProxyURL, "proxy", "", "HTTP/HTTPS proxy URL")

	flag.Parse()

	// Read offline token from file if provided
	if *offlineTokenFile != "" {
		data, err := os.ReadFile(*offlineTokenFile)
		if err != nil {
			log.Fatalf("[ERROR] Failed to read offline token file: %v", err)
		}
		config.OfflineToken = strings.TrimSpace(string(data))
		fmt.Printf("[INFO] Read offline token from %s\n", *offlineTokenFile)
	}

	// Validate offline token
	if config.OfflineToken == "" {
		fmt.Println("[ERROR] No offline token provided.")
		fmt.Println("[ERROR] Use --offline-token-file or set $OFFLINE_TOKEN environment variable")
		fmt.Println("\nHow to get an offline token:")
		fmt.Println("1. Visit https://access.stage.redhat.com/management/api")
		fmt.Println("2. Generate an offline token")
		fmt.Println("3. Save it to a file and run:")
		fmt.Println("   ./disaster-recovery --offline-token-file /path/to/token.txt")
		os.Exit(1)
	}

	config.BaseURL = strings.TrimRight(config.BaseURL, "/")

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func createHTTPClient(proxyURL string) *http.Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			log.Printf("[WARN] Invalid proxy URL '%s': %v. Continuing without proxy.\n", proxyURL, err)
		} else {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxy),
			}
			fmt.Printf("[INFO] Using proxy: %s\n", proxyURL)
		}
	}

	return client
}

func (c *Config) refreshAccessToken() error {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", "rhsm-api")
	data.Set("refresh_token", c.OfflineToken)

	fmt.Printf("[INFO] Requesting access token from %s\n", c.SSOURL)

	req, err := http.NewRequest("POST", c.SSOURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SSO returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp SSOTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	if tokenResp.AccessToken == "" {
		return fmt.Errorf("no access_token in SSO response")
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenRefreshTime = time.Now()
	fmt.Println("[INFO] Successfully obtained access token")
	return nil
}

func (c *Config) extractAccountFromJWT() error {
	parts := strings.Split(c.accessToken, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWT token format")
	}

	// Decode the payload (second part) using base64url encoding (RFC 7515)
	payload := parts[1]

	// Add padding if needed
	if padding := len(payload) % 4; padding > 0 {
		payload += strings.Repeat("=", 4-padding)
	}

	decoded, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var jwtPayload JWTPayload
	if err := json.Unmarshal(decoded, &jwtPayload); err != nil {
		return fmt.Errorf("failed to unmarshal JWT payload: %w", err)
	}

	c.accountID = jwtPayload.AccountID
	if c.accountID == "" {
		c.accountID = jwtPayload.Sub
	}

	fmt.Printf("[INFO] Extracted account_id from JWT: %s\n", c.accountID)
	return nil
}

func (c *Config) fetchAvailableQuickstart() error {
	url := fmt.Sprintf("%s/api/quickstarts/v1/quickstarts?account=%s", c.BaseURL, c.accountID)

	fmt.Println("[INIT] Fetching available quickstarts from API...")
	fmt.Printf("[REQUEST] GET %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.printRequestID(resp)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch quickstarts: %d: %s", resp.StatusCode, string(body))
	}

	var quickstartsResp QuickstartsResponse
	if err := json.NewDecoder(resp.Body).Decode(&quickstartsResp); err != nil {
		return err
	}

	if len(quickstartsResp.Data) == 0 {
		return fmt.Errorf("no quickstarts found in API response")
	}

	c.QuickstartName = quickstartsResp.Data[0].Name
	if c.QuickstartName == "" {
		return fmt.Errorf("quickstart data missing 'name' field")
	}

	fmt.Printf("[INIT] Selected quickstart: %s\n", c.QuickstartName)
	return nil
}

func (c *Config) toggleFavorite() error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	c.favoriteState = !c.favoriteState
	action := "UNFAVORITE"
	if c.favoriteState {
		action = "FAVORITE"
	}

	url := fmt.Sprintf("%s/api/quickstarts/v1/favorites?account=%s", c.BaseURL, c.accountID)

	payload := FavoriteRequest{
		QuickstartName: c.QuickstartName,
		Favorite:       c.favoriteState,
	}

	payloadBytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("\n[%s] %s: %s\n", timestamp, action, c.QuickstartName)
	fmt.Printf("[REQUEST] POST %s\n", url)
	fmt.Printf("[PAYLOAD] %s\n", string(payloadBytes))

	req, err := http.NewRequest("POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.printRequestID(resp)

	fmt.Printf("[RESPONSE] Status: %d\n", resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var formatted bytes.Buffer
		if err := json.Indent(&formatted, body, "", "  "); err == nil {
			fmt.Printf("[SUCCESS] %s\n", formatted.String())
		} else {
			fmt.Printf("[SUCCESS] %s\n", string(body))
		}
	} else {
		fmt.Printf("[ERROR] %d: %s\n", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Config) printRequestID(resp *http.Response) {
	headers := []string{"x-rh-insights-request-id", "x-rh-request-id", "x-request-id"}
	for _, header := range headers {
		if reqID := resp.Header.Get(header); reqID != "" {
			fmt.Printf("[REQUEST-ID] %s\n", reqID)
			return
		}
	}
}

func (c *Config) runContinuous() {
	fmt.Println("=== Quickstarts Disaster Recovery Test ===")
	fmt.Printf("Target: %s\n", c.BaseURL)
	fmt.Printf("Quickstart: %s\n", c.QuickstartName)
	fmt.Printf("Account: %s\n", c.accountID)
	fmt.Printf("Interval: %ds\n", c.Interval)
	fmt.Println("===")

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	iteration := 0
	ticker := time.NewTicker(time.Duration(c.Interval) * time.Second)
	defer ticker.Stop()

	// Refresh token every 4 minutes (tokens expire in ~5 minutes)
	tokenRefreshTicker := time.NewTicker(4 * time.Minute)
	defer tokenRefreshTicker.Stop()

	// Run first iteration immediately
	iteration++
	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("Iteration #%d\n", iteration)
	fmt.Printf("%s\n", strings.Repeat("=", 60))

	if err := c.toggleFavorite(); err != nil {
		fmt.Printf("[ERROR] Request failed: %v\n", err)
	}

	for {
		select {
		case <-sigChan:
			fmt.Println("\n\n[STOPPED] Test interrupted by user")
			return
		case <-tokenRefreshTicker.C:
			fmt.Println("\n[INFO] Refreshing access token (tokens expire after ~5 minutes)...")
			if err := c.refreshAccessToken(); err != nil {
				log.Printf("[ERROR] Failed to refresh token: %v\n", err)
				fmt.Println("[WARN] Continuing with existing token, may fail on next request")
			}
		case <-ticker.C:
			iteration++
			fmt.Printf("\n%s\n", strings.Repeat("=", 60))
			fmt.Printf("Iteration #%d\n", iteration)
			fmt.Printf("%s\n", strings.Repeat("=", 60))

			if err := c.toggleFavorite(); err != nil {
				fmt.Printf("[ERROR] Request failed: %v\n", err)
			}

			fmt.Printf("\n[SLEEP] Waiting %d seconds until next toggle...\n", c.Interval)
		}
	}
}
