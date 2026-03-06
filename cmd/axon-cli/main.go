package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/viper"
)

var (
	apiURL string
)

// Config represents the CLI configuration
type Config struct {
	APIURL string `mapstructure:"api_url"`
}

func init() {
	// Set up configuration file support
	viper.SetConfigName("axon-cli")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.axon")
	viper.AddConfigPath("/etc/axon")

	// Set default values
	viper.SetDefault("api_url", "http://localhost:8020/api/v1")

	// Try to read config file (optional)
	_ = viper.ReadInConfig()

	// Allow environment variable override
	viper.BindEnv("api_url", "AXON_API_URL")
	apiURL = viper.GetString("api_url")
}

func main() {
	// Parse base flags first
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Get the command
	command := os.Args[1]

	// Handle special commands
	switch command {
	case "help", "--help", "-h":
		printUsage()
		os.Exit(0)
	case "version", "--version", "-v":
		fmt.Println("Axon CLI v1.0.0")
		os.Exit(0)
	}

	// Reset args for subcommand parsing
	os.Args = os.Args[1:]

	switch command {
	case "snapshot":
		handleSnapshot()
	case "navigate":
		handleNavigate()
	case "act":
		handleAct()
	case "session":
		handleSession()
	case "vault":
		handleVault()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`Axon CLI - Command-line interface for Axon browser automation

Usage:
  axon <command> [arguments]

Commands:
  snapshot <session> [options]    Take a snapshot of the current page
  navigate <session> <url>        Navigate to a URL
  act <session> <action> [opts]   Perform an action on an element
  session <subcommand>            Manage sessions
  vault <subcommand>              Manage the Intelligence Vault

Session subcommands:
  session list                    List all sessions
  session create <id>             Create a new session
  session delete <id>             Delete a session
  session info <id>               Get session details

Vault subcommands:
  vault list                      List all secrets
  vault add <name> <url>          Add a secret to the vault
    --user <username>             Optional username
    --pass <password>             Optional password
    --value <value>               Optional generic secret value
  vault delete <name>             Delete a secret from the vault
  vault fill <session> <secret>   Fill a field using a vault secret
    --ref <id>                    Element reference ID
    --intent <desc>               Semantic element description
    --field <name>                Field to inject (default: password)

Options:
  --api-url <url>                 Override API URL (or use AXON_API_URL env var)
  -h, --help                      Show help

Configuration:
  Axon CLI looks for config in:
    - ./axon-cli.yaml
    - $HOME/.axon/axon-cli.yaml
    - /etc/axon/axon-cli.yaml

Examples:
  axon session create mysession
  axon navigate mysession https://github.com
  axon snapshot mysession
  axon act mysession click --ref e1
  axon session list
`)
}

// handleSnapshot handles the snapshot command
func handleSnapshot() {
	flag := flag.NewFlagSet("snapshot", flag.ExitOnError)
	sessionID := flag.String("session", "", "Session ID (required)")
	ref := flag.String("ref", "", "Element reference to focus on")
	apiURLFlag := flag.String("api-url", apiURL, "API URL")
	flag.Parse(os.Args[2:])

	if *sessionID == "" {
		// Try positional argument
		if len(flag.Args()) > 0 {
			*sessionID = flag.Args()[0]
		}
		if *sessionID == "" {
			fmt.Println("Error: session ID required")
			fmt.Println("Usage: axon snapshot <session>")
			os.Exit(1)
		}
	}

	reqBody := map[string]interface{}{}
	if *ref != "" {
		reqBody["ref"] = *ref
	}

	*apiURLFlag = getAPIURL(*apiURLFlag)
	resp, err := http.Post(fmt.Sprintf("%s/sessions/%s/snapshot", *apiURLFlag, *sessionID), "application/json", bytes.NewBuffer(mustMarshal(reqBody)))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	printResponse(resp)
}

// handleNavigate handles the navigate command
func handleNavigate() {
	flag := flag.NewFlagSet("navigate", flag.ExitOnError)
	sessionID := flag.String("session", "", "Session ID")
	url := flag.String("url", "", "URL to navigate to")
	apiURLFlag := flag.String("api-url", apiURL, "API URL")
	flag.Parse(os.Args[2:])

	// Handle positional arguments: axon navigate <session> <url>
	if *sessionID == "" && len(flag.Args()) > 0 {
		*sessionID = flag.Args()[0]
	}
	if *url == "" && len(flag.Args()) > 1 {
		*url = flag.Args()[1]
	}

	if *sessionID == "" || *url == "" {
		fmt.Println("Usage: axon navigate <session> <url>")
		os.Exit(1)
	}

	*apiURLFlag = getAPIURL(*apiURLFlag)
	reqBody := map[string]string{"url": *url}
	resp, err := http.Post(fmt.Sprintf("%s/sessions/%s/navigate", *apiURLFlag, *sessionID), "application/json", bytes.NewBuffer(mustMarshal(reqBody)))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	printResponse(resp)
}

// handleAct handles the act command
func handleAct() {
	flag := flag.NewFlagSet("act", flag.ExitOnError)
	sessionID := flag.String("session", "", "Session ID")
	action := flag.String("action", "", "Action to perform (click, fill, hover, select, etc.)")
	ref := flag.String("ref", "", "Element reference ID")
	value := flag.String("value", "", "Value for fill/select actions")
	confirm := flag.Bool("confirm", false, "Confirm irreversible action")
	apiURLFlag := flag.String("api-url", apiURL, "API URL")
	flag.Parse(os.Args[2:])

	// Handle positional arguments
	if *sessionID == "" && len(flag.Args()) > 0 {
		*sessionID = flag.Args()[0]
	}
	if *action == "" && len(flag.Args()) > 1 {
		*action = flag.Args()[1]
	}

	if *sessionID == "" || *action == "" {
		fmt.Println("Usage: axon act <session> <action> [--ref <ref>] [--value <value>]")
		os.Exit(1)
	}

	*apiURLFlag = getAPIURL(*apiURLFlag)
	reqBody := map[string]interface{}{
		"action":  *action,
		"ref":     *ref,
		"value":   *value,
		"confirm": *confirm,
	}
	resp, err := http.Post(fmt.Sprintf("%s/sessions/%s/act", *apiURLFlag, *sessionID), "application/json", bytes.NewBuffer(mustMarshal(reqBody)))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	printResponse(resp)
}

// handleSession handles session management commands
func handleSession() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: axon session <subcommand>")
		fmt.Println("Subcommands: list, create, delete, info")
		os.Exit(1)
	}

	subcommand := os.Args[2]
	os.Args = os.Args[2:]

	switch subcommand {
	case "list":
		handleSessionList()
	case "create":
		handleSessionCreate()
	case "delete":
		handleSessionDelete()
	case "info":
		handleSessionInfo()
	default:
		fmt.Printf("Unknown subcommand: %s\n", subcommand)
		fmt.Println("Usage: axon session <subcommand>")
		fmt.Println("Subcommands: list, create, delete, info")
		os.Exit(1)
	}
}

func handleSessionList() {
	flag := flag.NewFlagSet("session list", flag.ExitOnError)
	apiURLFlag := flag.String("api-url", apiURL, "API URL")
	flag.Parse(os.Args[2:])
	*apiURLFlag = getAPIURL(*apiURLFlag)

	resp, err := http.Get(fmt.Sprintf("%s/sessions", *apiURLFlag))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	// Pretty print JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		fmt.Println(string(body))
		return
	}
	fmt.Println(prettyJSON.String())
}

func handleSessionCreate() {
	flag := flag.NewFlagSet("session create", flag.ExitOnError)
	id := flag.String("id", "", "Session ID")
	profile := flag.String("profile", "", "Browser profile name")
	apiURLFlag := flag.String("api-url", apiURL, "API URL")
	flag.Parse(os.Args[2:])

	// Handle positional argument
	if *id == "" && len(flag.Args()) > 0 {
		*id = flag.Args()[0]
	}

	if *id == "" {
		fmt.Println("Usage: axon session create <id> [--profile <name>]")
		os.Exit(1)
	}

	*apiURLFlag = getAPIURL(*apiURLFlag)
	reqBody := map[string]interface{}{
		"id": *id,
	}
	if *profile != "" {
		reqBody["profile"] = *profile
	}

	resp, err := http.Post(fmt.Sprintf("%s/sessions", *apiURLFlag), "application/json", bytes.NewBuffer(mustMarshal(reqBody)))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	printResponse(resp)
}

func handleSessionDelete() {
	flag := flag.NewFlagSet("session delete", flag.ExitOnError)
	apiURLFlag := flag.String("api-url", apiURL, "API URL")
	flag.Parse(os.Args[2:])
	*apiURLFlag = getAPIURL(*apiURLFlag)

	if len(flag.Args()) < 1 {
		fmt.Println("Usage: axon session delete <id>")
		os.Exit(1)
	}

	id := flag.Args()[0]
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/sessions/%s", *apiURLFlag, id), nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		fmt.Printf("Session '%s' deleted successfully\n", id)
	} else {
		printResponse(resp)
	}
}

func handleSessionInfo() {
	flag := flag.NewFlagSet("session info", flag.ExitOnError)
	apiURLFlag := flag.String("api-url", apiURL, "API URL")
	flag.Parse(os.Args[2:])
	*apiURLFlag = getAPIURL(*apiURLFlag)

	if len(flag.Args()) < 1 {
		fmt.Println("Usage: axon session info <id>")
		os.Exit(1)
	}

	id := flag.Args()[0]
	resp, err := http.Get(fmt.Sprintf("%s/sessions/%s", *apiURLFlag, id))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	printResponse(resp)
}

func getAPIURL(flagURL string) string {
	if flagURL != "" && flagURL != apiURL {
		return flagURL
	}
	// Check environment variable again
	if envURL := os.Getenv("AXON_API_URL"); envURL != "" {
		return envURL
	}
	return apiURL
}

func printResponse(resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Pretty print JSON
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
			fmt.Println(string(body))
			return
		}
		fmt.Println(prettyJSON.String())
	} else {
		// Print error
		fmt.Printf("Error (Status %d): %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}
}

func handleVault() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: axon vault <subcommand>")
		fmt.Println("Subcommands: list, add, delete, fill")
		os.Exit(1)
	}

	subcommand := os.Args[2]
	os.Args = os.Args[2:]

	switch subcommand {
	case "list":
		handleVaultList()
	case "add":
		handleVaultAdd()
	case "delete":
		handleVaultDelete()
	case "fill":
		handleVaultFill()
	default:
		fmt.Printf("Unknown subcommand: %s\n", subcommand)
		fmt.Println("Usage: axon vault <subcommand>")
		fmt.Println("Subcommands: list, add, delete, fill")
		os.Exit(1)
	}
}

func handleVaultList() {
	flag := flag.NewFlagSet("vault list", flag.ExitOnError)
	apiURLFlag := flag.String("api-url", apiURL, "API URL")
	flag.Parse(os.Args[2:])
	*apiURLFlag = getAPIURL(*apiURLFlag)

	resp, err := http.Get(fmt.Sprintf("%s/vault/secrets", *apiURLFlag))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		fmt.Println(string(body))
		return
	}
	fmt.Println(prettyJSON.String())
}

func handleVaultAdd() {
	flag := flag.NewFlagSet("vault add", flag.ExitOnError)
	name := flag.String("name", "", "Secret name")
	url := flag.String("url", "", "Target URL/Domain")
	username := flag.String("user", "", "Username")
	password := flag.String("pass", "", "Password")
	value := flag.String("value", "", "Generic secret value")
	apiURLFlag := flag.String("api-url", apiURL, "API URL")
	flag.Parse(os.Args[2:])

	// Positional arguments support: axon vault add <name> <url>
	if *name == "" && len(flag.Args()) > 0 {
		*name = flag.Args()[0]
	}
	if *url == "" && len(flag.Args()) > 1 {
		*url = flag.Args()[1]
	}

	if *name == "" || *url == "" {
		fmt.Println("Usage: axon vault add <name> <url> [--user <user>] [--pass <pass>] [--value <val>]")
		os.Exit(1)
	}

	*apiURLFlag = getAPIURL(*apiURLFlag)
	reqBody := map[string]interface{}{
		"name":     *name,
		"url":      *url,
		"username": *username,
		"password": *password,
		"value":    *value,
	}

	resp, err := http.Post(fmt.Sprintf("%s/vault/secrets", *apiURLFlag), "application/json", bytes.NewBuffer(mustMarshal(reqBody)))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	printResponse(resp)
}

func handleVaultDelete() {
	flag := flag.NewFlagSet("vault delete", flag.ExitOnError)
	apiURLFlag := flag.String("api-url", apiURL, "API URL")
	flag.Parse(os.Args[2:])
	*apiURLFlag = getAPIURL(*apiURLFlag)

	if len(flag.Args()) < 1 {
		fmt.Println("Usage: axon vault delete <name>")
		os.Exit(1)
	}

	name := flag.Args()[0]
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/vault/secrets/%s", *apiURLFlag, name), nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		fmt.Printf("Secret '%s' deleted successfully\n", name)
	} else {
		printResponse(resp)
	}
}

func handleVaultFill() {
	flag := flag.NewFlagSet("vault fill", flag.ExitOnError)
	sessionID := flag.String("session", "", "Session ID")
	secretName := flag.String("secret", "", "Secret name")
	ref := flag.String("ref", "", "Element reference ID")
	intent := flag.String("intent", "", "Element intent description")
	field := flag.String("field", "password", "Field to inject")
	apiURLFlag := flag.String("api-url", apiURL, "API URL")
	flag.Parse(os.Args[2:])

	// Positional arguments support: axon vault fill <session> <secret>
	if *sessionID == "" && len(flag.Args()) > 0 {
		*sessionID = flag.Args()[0]
	}
	if *secretName == "" && len(flag.Args()) > 1 {
		*secretName = flag.Args()[1]
	}

	if *sessionID == "" || *secretName == "" {
		fmt.Println("Usage: axon vault fill <session> <secret> [--ref <ref> | --intent <intent>] [--field <field>]")
		os.Exit(1)
	}

	if *ref == "" && *intent == "" {
		fmt.Println("Error: Either --ref or --intent is required to identify the target element")
		os.Exit(1)
	}

	*apiURLFlag = getAPIURL(*apiURLFlag)
	vaultRef := fmt.Sprintf("@vault:%s:%s", *secretName, *field)

	var urlPath string
	reqBody := map[string]interface{}{
		"action": "fill",
		"value":  vaultRef,
	}

	if *intent != "" {
		urlPath = fmt.Sprintf("%s/sessions/%s/find_and_act", *apiURLFlag, *sessionID)
		reqBody["intent"] = *intent
	} else {
		urlPath = fmt.Sprintf("%s/sessions/%s/act", *apiURLFlag, *sessionID)
		reqBody["ref"] = *ref
	}

	resp, err := http.Post(urlPath, "application/json", bytes.NewBuffer(mustMarshal(reqBody)))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("Successfully injected vault secret '%s:%s' into session '%s'\n", *secretName, *field, *sessionID)
	} else {
		printResponse(resp)
	}
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		os.Exit(1)
	}
	return data
}
