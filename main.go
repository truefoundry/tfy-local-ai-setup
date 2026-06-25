package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const httpTimeout = 10 * time.Second

var httpClient = &http.Client{Timeout: httpTimeout}

// ---- API types ----

type deviceAuthResponse struct {
	DeviceCode        string `json:"deviceCode"`
	UserCode          string `json:"userCode"`
	ExpiresInSeconds  int    `json:"expiresInSeconds"`
	IntervalInSeconds int    `json:"intervalInSeconds"`
}

type tokenResponse struct {
	StatusCode   int    `json:"statusCode"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Error        string `json:"error"`
	Message      string `json:"message"`
}

// ---- logging ----

func logf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

// ---- entry point ----

func main() {
	url          := flag.String("url",           "", "Base URL of the TrueFoundry control plane (required)")
	tenant       := flag.String("tenant",        "", "Tenant name (required)")
	gateway      := flag.String("gateway",       "", "Gateway URL (defaults to --url)")
	opusModel    := flag.String("opus-model",    "claude-code/claude-opus",   "Model ID for ANTHROPIC_DEFAULT_OPUS_MODEL (Claude Code)")
	sonnetModel  := flag.String("sonnet-model",  "claude-code/claude-sonnet", "Model ID for ANTHROPIC_DEFAULT_SONNET_MODEL (Claude Code)")
	haikuModel   := flag.String("haiku-model",   "claude-code/claude-haiku",  "Model ID for ANTHROPIC_DEFAULT_HAIKU_MODEL (Claude Code)")
	settingsFile := flag.String("settings-file", "", "Path to a JSON template for managed-settings.json (Claude Code; falls back to existing file on disk, then built-in default)")
	claudeCode   := flag.Bool("claude-code",     false, "Configure Claude Code managed settings (default: auto-detect)")
	codex        := flag.Bool("codex",           false, "Configure Codex managed settings (default: auto-detect)")
	dryRun       := flag.Bool("dry-run",         false, "Print config to stdout instead of writing files")
	getToken     := flag.Bool("_get-token",      false, "") // internal: device auth only, prints token to stdout
	flag.Parse()

	if *url == "" || *tenant == "" {
		fmt.Fprintln(os.Stderr, "error: --url and --tenant are required")
		flag.Usage()
		os.Exit(1)
	}
	*url = strings.TrimRight(*url, "/")

	gatewayURL := *gateway
	if gatewayURL == "" {
		gatewayURL = *url
	}
	gatewayURL = strings.TrimRight(gatewayURL, "/")

	if *getToken {
		token := doGetToken(*url, *tenant)
		fmt.Println(token)
		return
	}

	// If neither --claude-code nor --codex is set, auto-detect what's installed.
	configureClaude := *claudeCode
	configureCodex  := *codex
	if !configureClaude && !configureCodex {
		configureClaude = toolInstalled("claude")
		configureCodex  = toolInstalled("codex")
		if configureClaude {
			logf("Detected Claude Code — will configure.")
		}
		if configureCodex {
			logf("Detected Codex — will configure.")
		}
		if !configureClaude && !configureCodex {
			fmt.Fprintln(os.Stderr, "error: neither Claude Code nor Codex detected; install one or pass --claude-code / --codex explicitly")
			os.Exit(1)
		}
	}

	models := modelIDs{opus: *opusModel, sonnet: *sonnetModel, haiku: *haikuModel}
	doSetup(*url, *tenant, gatewayURL, *settingsFile, models, configureClaude, configureCodex, *dryRun)
}

type modelIDs struct {
	opus, sonnet, haiku string
}

func toolInstalled(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// ---- setup mode (root) ----

func doSetup(controlPlaneURL, tenant, gatewayURL, settingsFile string, models modelIDs, configureClaude, configureCodex bool, dryRun bool) {
	checkRoot()

	user, err := loggedInUser()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not determine logged-in user: %v\n", err)
		os.Exit(1)
	}
	logf("Fetching auth token for %s...", user)

	token, err := tokenAsUser(user, controlPlaneURL, tenant)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not obtain token: %v\n", err)
		os.Exit(1)
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: empty token returned")
		os.Exit(1)
	}
	logf("Token obtained.")

	if configureClaude {
		setupClaudeCode(gatewayURL, settingsFile, token, models, dryRun)
	}
	if configureCodex {
		setupCodex(gatewayURL, token, dryRun)
	}
}

func setupClaudeCode(gatewayURL, settingsFile, token string, models modelIDs, dryRun bool) {
	destDir := managedSettingsDir()
	dest := filepath.Join(destDir, "managed-settings.json")

	logf("Building Claude Code managed-settings.json...")
	jsonContent, err := buildManagedSettings(dest, settingsFile, gatewayURL, token, models)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if dryRun {
		logf("Dry-run — Claude Code managed-settings.json:")
		fmt.Println()
		fmt.Println(jsonContent)
		return
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error: could not create directory %s: %v\n", destDir, err)
		os.Exit(1)
	}
	unlockFile(dest)
	if err := os.WriteFile(dest, []byte(jsonContent+"\n"), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: could not write %s: %v\n", dest, err)
		os.Exit(1)
	}
	if err := chownRoot(dest); err != nil {
		logf("WARNING: could not set ownership: %v", err)
	}
	lockFile(dest)
	logf("Claude Code managed-settings.json updated and locked.")
	logf("Verifying:")
	verifyFile(dest)
}

func setupCodex(gatewayURL, token string, dryRun bool) {
	dest := codexConfigPath()
	if dest == "" {
		logf("WARNING: Codex system config path not supported on this platform — skipping.")
		return
	}

	logf("Building Codex config.toml...")
	tomlContent := buildCodexConfig(gatewayURL, token)

	if dryRun {
		logf("Dry-run — Codex config.toml (%s):", dest)
		fmt.Println()
		fmt.Println(tomlContent)
		return
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error: could not create directory %s: %v\n", filepath.Dir(dest), err)
		os.Exit(1)
	}
	unlockFile(dest)
	if err := os.WriteFile(dest, []byte(tomlContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: could not write %s: %v\n", dest, err)
		os.Exit(1)
	}
	if err := chownRoot(dest); err != nil {
		logf("WARNING: could not set ownership: %v", err)
	}
	lockFile(dest)
	logf("Codex config.toml updated and locked.")
	logf("Verifying:")
	verifyFile(dest)
}

// ---- device auth (get-token mode) ----

// doGetToken performs the OAuth2 device flow and returns the access token.
// It attempts a silent refresh from ~/.tfy-refresh-token first, falling back
// to the full browser flow. Exits on error.
func doGetToken(baseURL, tenant string) string {
	tokenFile := defaultTokenFile()

	// Try silent refresh first.
	if rt := loadRefreshToken(tokenFile); rt != "" {
		fmt.Fprintln(os.Stderr, "Found cached refresh token, attempting silent refresh...")
		if tok, err := tryRefresh(baseURL, tenant, rt); err == nil {
			persistRefreshToken(tokenFile, tok.RefreshToken)
			return tok.AccessToken
		} else {
			fmt.Fprintf(os.Stderr, "Silent refresh failed (%v), falling back to device flow...\n", err)
		}
	}

	// Full device flow.
	tok, err := deviceFlow(baseURL, tenant)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	persistRefreshToken(tokenFile, tok.RefreshToken)
	return tok.AccessToken
}

func defaultTokenFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".tfy-refresh-token"
	}
	return filepath.Join(home, ".tfy-refresh-token")
}

func loadRefreshToken(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func persistRefreshToken(path, token string) {
	if token == "" {
		return
	}
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not create token directory: %v\n", err)
			return
		}
	}
	if err := os.WriteFile(path, []byte(token), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save refresh token: %v\n", err)
	}
}

func tryRefresh(baseURL, tenant, refreshToken string) (*tokenResponse, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"grantType":    "refresh_token",
		"tenantName":   tenant,
		"refreshToken": refreshToken,
		"returnJWT":    true,
	})
	resp, err := httpClient.Post(baseURL+"/api/svc/v1/oauth2/token", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("refresh token rejected: HTTP %d empty body", resp.StatusCode)
	}
	var tok tokenResponse
	if err := json.Unmarshal(raw, &tok); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}
	if tok.AccessToken == "" {
		msg := firstNonEmpty(tok.Message, tok.Error, fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil, fmt.Errorf("refresh token rejected: %s", msg)
	}
	return &tok, nil
}

func deviceFlow(baseURL, tenant string) (*tokenResponse, error) {
	authBody, _ := json.Marshal(map[string]string{"tenantName": tenant})
	resp, err := httpClient.Post(baseURL+"/api/svc/v1/oauth2/device-authorize", "application/json", bytes.NewReader(authBody))
	if err != nil {
		return nil, fmt.Errorf("device authorize request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("device authorize returned HTTP %d", resp.StatusCode)
	}

	var auth deviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return nil, fmt.Errorf("failed to parse device authorize response: %w", err)
	}
	if auth.DeviceCode == "" || auth.UserCode == "" || auth.ExpiresInSeconds <= 0 {
		return nil, fmt.Errorf("invalid device authorize response from server")
	}

	verificationURI := baseURL + "/authorize/device?user_code=" + auth.UserCode
	fmt.Fprintf(os.Stderr, "Opening: %s\n", verificationURI)
	fmt.Fprintln(os.Stderr, "If the browser did not open, visit the URL above manually.")
	openBrowser(verificationURI)

	interval := time.Duration(auth.IntervalInSeconds) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(auth.ExpiresInSeconds) * time.Second)

	pollBody, _ := json.Marshal(map[string]interface{}{
		"tenantName": tenant,
		"grantType":  "device_code",
		"returnJWT":  true,
		"deviceCode": auth.DeviceCode,
	})

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		pollResp, err := httpClient.Post(baseURL+"/api/svc/v1/oauth2/token", "application/json", bytes.NewReader(pollBody))
		if err != nil {
			return nil, fmt.Errorf("token poll request failed: %w", err)
		}
		var tok tokenResponse
		decodeErr := json.NewDecoder(pollResp.Body).Decode(&tok)
		pollResp.Body.Close()

		if decodeErr != nil {
			return nil, fmt.Errorf("failed to parse token response: %w", decodeErr)
		}
		if tok.AccessToken != "" {
			return &tok, nil
		}
		if tok.StatusCode == 202 {
			continue
		}
		msg := firstNonEmpty(tok.Message, tok.Error, fmt.Sprintf("status %d", tok.StatusCode))
		return nil, fmt.Errorf("authorization failed: %s", msg)
	}

	return nil, fmt.Errorf("login timed out — device code expired before user authorized")
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not open browser: %v\n", err)
	}
}

// ---- JSON build / patch ----

func buildManagedSettings(existingPath, settingsFile, gatewayURL, token string, models modelIDs) (string, error) {
	const headerKey = "X-TFY-API-KEY"
	newEntry := headerKey + ": " + token

	var settings map[string]interface{}

	// Priority: --settings-file flag > existing file on disk > built-in default.
	templatePath := settingsFile
	if templatePath == "" {
		templatePath = existingPath
	}

	if data, err := os.ReadFile(templatePath); err == nil {
		// Patch template: preserve all existing keys, update only what we own.
		if err := json.Unmarshal(data, &settings); err != nil {
			return "", fmt.Errorf("failed to parse %s: %v", templatePath, err)
		}
		delete(settings, "apiKeyHelper")

		env := getOrMakeEnv(settings)
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"]   = models.opus
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = models.sonnet
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"]  = models.haiku
		env["ANTHROPIC_CUSTOM_HEADERS"] = upsertHeader(
			mapStrVal(env, "ANTHROPIC_CUSTOM_HEADERS"),
			headerKey, newEntry,
		)
		settings["env"] = env
	} else {
		// No template or existing file: write built-in default config.
		settings = defaultConfig(gatewayURL, newEntry, models)
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// upsertHeader replaces the value for `key` in a comma-separated header string,
// or appends it if not present.
func upsertHeader(existing, key, newEntry string) string {
	if existing == "" {
		return newEntry
	}
	re := regexp.MustCompile(`(?i)^` + regexp.QuoteMeta(key) + `\s*:`)
	var updated []string
	found := false
	for _, p := range strings.Split(existing, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if re.MatchString(p) {
			updated = append(updated, newEntry)
			found = true
		} else {
			updated = append(updated, p)
		}
	}
	if !found {
		updated = append(updated, newEntry)
	}
	return strings.Join(updated, ",")
}

func getOrMakeEnv(settings map[string]interface{}) map[string]interface{} {
	if v, ok := settings["env"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	m := make(map[string]interface{})
	settings["env"] = m
	return m
}

func mapStrVal(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func defaultConfig(gatewayURL, customHeaders string, models modelIDs) map[string]interface{} {
	return map[string]interface{}{
		"permissions": map[string]interface{}{
			"disableBypassPermissionsMode": "disable",
			"deny": []string{
				"Bash(curl:*)", "Bash(wget:*)",
				"Read(**/.env)", "Read(**/.env.*)",
				"Read(**/secrets/**)", "Read(**/.ssh/**)", "Read(**/credentials/**)",
			},
			"ask": []string{"Bash(git push:*)", "Write(**)"},
		},
		"allowManagedPermissionRulesOnly": true,
		"allowManagedHooksOnly":           true,
		"transcriptRetentionDays":         14,
		"sandbox": map[string]interface{}{
			"enabled": true,
			"network": map[string]interface{}{
				"httpProxyPort":  8080,
				"socksProxyPort": 8081,
			},
		},
		"strictKnownMarketplaces": []interface{}{},
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL":                     gatewayURL,
			"ANTHROPIC_DEFAULT_OPUS_MODEL":           models.opus,
			"ANTHROPIC_DEFAULT_SONNET_MODEL":         models.sonnet,
			"ANTHROPIC_DEFAULT_HAIKU_MODEL":          models.haiku,
			"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS": "1",
			"ANTHROPIC_CUSTOM_HEADERS":               customHeaders,
		},
	}
}

// ---- platform: root check ----

func checkRoot() {
	switch runtime.GOOS {
	case "windows":
		out, _ := exec.Command("whoami", "/groups").Output()
		if !strings.Contains(string(out), "S-1-16-12288") {
			fmt.Fprintln(os.Stderr, "error: must be run as Administrator")
			os.Exit(1)
		}
	default:
		if os.Getuid() != 0 {
			fmt.Fprintln(os.Stderr, "error: must be run as root")
			os.Exit(1)
		}
	}
}

// ---- platform: logged-in user ----

func loggedInUser() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return loggedInUserDarwin()
	case "windows":
		return loggedInUserWindows()
	default:
		return loggedInUserLinux()
	}
}

func loggedInUserDarwin() (string, error) {
	cmd := exec.Command("/usr/sbin/scutil")
	cmd.Stdin = strings.NewReader("show State:/Users/ConsoleUser\n")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("scutil failed: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "Name :") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		user := strings.TrimSpace(parts[1])
		if at := strings.Index(user, "@"); at >= 0 {
			user = user[:at]
		}
		if user == "" || user == "loginwindow" {
			continue
		}
		return user, nil
	}
	return "", fmt.Errorf("no logged-in user found in scutil output")
}

func loggedInUserLinux() (string, error) {
	if user := os.Getenv("SUDO_USER"); user != "" {
		return user, nil
	}
	out, err := exec.Command("logname").Output()
	if err != nil {
		return "", fmt.Errorf("logname failed: %v", err)
	}
	user := strings.TrimSpace(string(out))
	if user == "" {
		return "", fmt.Errorf("logname returned empty output")
	}
	return user, nil
}

func loggedInUserWindows() (string, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		`(Get-WmiObject -Class Win32_ComputerSystem).UserName`).Output()
	if err != nil {
		return "", fmt.Errorf("WMI query failed: %v", err)
	}
	user := strings.TrimSpace(string(out))
	// Strip DOMAIN\ prefix.
	if i := strings.LastIndex(user, `\`); i >= 0 {
		user = user[i+1:]
	}
	if user == "" {
		return "", fmt.Errorf("WMI returned empty user name")
	}
	return user, nil
}

// ---- platform: get token as user ----

// tokenAsUser re-execs the binary as the logged-in (non-root) user so the
// device auth flow (and browser open) happens in the user's session.
func tokenAsUser(user, controlPlaneURL, tenant string) (string, error) {
	selfPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not determine self path: %v", err)
	}
	if runtime.GOOS == "windows" {
		return tokenAsUserWindows(user, selfPath, controlPlaneURL, tenant)
	}
	return tokenAsUserUnix(user, selfPath, controlPlaneURL, tenant)
}

func tokenAsUserUnix(user, selfPath, controlPlaneURL, tenant string) (string, error) {
	cmd := exec.Command(
		"sudo", "-u", user,
		selfPath,
		"--_get-token",
		"--url="+controlPlaneURL,
		"--tenant="+tenant,
	)
	cmd.Stderr = os.Stderr // forward device-flow messages to our stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("sudo re-exec failed: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// tokenAsUserWindows runs the binary as the logged-in user via a scheduled
// task (the only reliable way to open a browser in the user's session when
// running as SYSTEM/Administrator).
func tokenAsUserWindows(user, selfPath, controlPlaneURL, tenant string) (string, error) {
	taskName := fmt.Sprintf("TfyMdmToken_%d", time.Now().UnixNano())
	tokenFile := filepath.Join(os.TempDir(), taskName+".txt")
	defer os.Remove(tokenFile)

	cmdArg := fmt.Sprintf(`cmd.exe /c "%s" --_get-token --url="%s" --tenant="%s" > "%s" 2>nul`,
		selfPath, controlPlaneURL, tenant, tokenFile)

	createArgs := []string{
		"/create", "/tn", taskName,
		"/tr", cmdArg,
		"/sc", "once", "/st", "00:00",
		"/ru", user,
		"/it", // only when user is logged on interactively
		"/f",  // overwrite if exists
	}
	if out, err := exec.Command("schtasks", createArgs...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("schtasks /create failed: %v — %s", err, out)
	}
	defer exec.Command("schtasks", "/delete", "/tn", taskName, "/f").Run() //nolint:errcheck

	if out, err := exec.Command("schtasks", "/run", "/tn", taskName).CombinedOutput(); err != nil {
		return "", fmt.Errorf("schtasks /run failed: %v — %s", err, out)
	}

	// Wait up to 5 minutes for the task to finish.
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		out, err := exec.Command("schtasks", "/query", "/tn", taskName, "/fo", "csv", "/nh").Output()
		if err == nil && strings.Contains(string(out), "Ready") {
			break
		}
	}

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", fmt.Errorf("could not read token output: %v", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// ---- platform: dest dir ----

// ---- Codex config ----

// buildCodexConfig returns a config.toml that points Codex at the TrueFoundry
// gateway. Only the provider section is written — model selection and all other
// settings come from the user's own config or a separately managed requirements.toml.
func buildCodexConfig(gatewayURL, token string) string {
	return fmt.Sprintf(`model_provider = "truefoundry"

[model_providers.truefoundry]
name     = "TrueFoundry Gateway"
base_url = %q
wire_api = "responses"

[model_providers.truefoundry.http_headers]
Authorization = "Bearer %s"
`, gatewayURL, token)
}

// codexConfigPath returns the system-level Codex config path for the current OS.
// Returns an empty string on platforms without a system config path.
func codexConfigPath() string {
	switch runtime.GOOS {
	case "windows":
		// Codex has no documented system-level config path on Windows.
		return ""
	default:
		return "/etc/codex/config.toml"
	}
}

func managedSettingsDir() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/ClaudeCode"
	case "windows":
		return `C:\Program Files\ClaudeCode`
	default:
		return "/etc/claude-code"
	}
}

// ---- platform: file locking ----

func unlockFile(path string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("chflags", "noschg", path).Run() //nolint:errcheck
	case "linux":
		exec.Command("chattr", "-i", path).Run() //nolint:errcheck
	case "windows":
		exec.Command("icacls", path, "/grant", "Administrators:(W)").Run() //nolint:errcheck
	}
}

func chownRoot(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("chown", "root:wheel", path).Run()
	case "linux":
		return exec.Command("chown", "root:root", path).Run()
	default:
		return nil // Windows: ACLs set in lockFile
	}
}

func lockFile(path string) {
	switch runtime.GOOS {
	case "darwin":
		if err := exec.Command("chflags", "schg", path).Run(); err != nil {
			logf("WARNING: chflags schg failed: %v", err)
		}
	case "linux":
		if err := exec.Command("chattr", "+i", path).Run(); err != nil {
			logf("WARNING: chattr not available — file not immutable-locked")
		}
	case "windows":
		exec.Command("icacls", path, //nolint:errcheck
			"/inheritance:r",
			"/grant", "SYSTEM:(F)",
			"/grant", "Administrators:(R)",
			"/grant", "Users:(R)",
		).Run()
	}
}

func verifyFile(path string) {
	var out []byte
	switch runtime.GOOS {
	case "darwin":
		out, _ = exec.Command("ls", "-lO", path).Output()
	case "linux":
		out, _ = exec.Command("ls", "-l", path).Output()
	case "windows":
		out, _ = exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf(`Get-Item "%s" | Select-Object FullName, LastWriteTime`, path)).Output()
	}
	fmt.Print(string(out))
}

// ---- misc ----

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return "unknown error"
}
