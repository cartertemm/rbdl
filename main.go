package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	apiEndpoint = "https://www.repeaterbook.com/api/export.php"
	userAgentTemplate = "RepeaterbookDL CLI (beta), %s"
)

type Config struct {
	Email     string
	Output    string
	Callsign  string
	City      string
	Country   string
	Frequency string
	Mode      string
	Landmark  string
	StateID   string
	Region    string
	SType     string
}

func main() {
	config := parseFlags()
	if err := validateConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	data, err := fetchRepeaterData(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching data: %v\n", err)
		os.Exit(1)
	}
	outputFile := config.Output
	if outputFile == "" {
		outputFile = generateFilename(config)
	}
	if err := saveToFile(outputFile, data); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully saved data to: %s\n", outputFile)
}

func parseFlags() *Config {
	config := &Config{}
	flag.StringVar(&config.Email, "email", os.Getenv("RBDL_EMAIL"), "Email address (required, or set RBDL_EMAIL)")
	flag.StringVar(&config.Output, "output", "", "Output file path (auto-generated if not specified)")
	flag.StringVar(&config.Callsign, "callsign", "", "Repeater callsign (supports % wildcard)")
	flag.StringVar(&config.City, "city", "", "Repeater city (supports % wildcard)")
	flag.StringVar(&config.Country, "country", "", "Repeater country (supports % wildcard)")
	flag.StringVar(&config.Frequency, "frequency", "", "Repeater frequency")
	flag.StringVar(&config.Mode, "mode", "", "Operating mode (analog, DMR, NXDN, P25, tetra)")
	flag.StringVar(&config.Landmark, "landmark", "", "Landmark (supports % wildcard)")
	flag.StringVar(&config.StateID, "state", "", "State/Province FIPS code")
	flag.StringVar(&config.Region, "region", "", "Region (for international repeaters)")
	flag.StringVar(&config.SType, "stype", "", "Service type (e.g., GMRS)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: rbdl [options]\n\n")
		fmt.Fprintf(os.Stderr, "RepeaterbookDL - Download repeater data from RepeaterBook API\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  rbdl --email user@example.com --state CA --mode DMR\n")
		fmt.Fprintf(os.Stderr, "  rbdl --email user@example.com --country USA --frequency 146.52\n")
		fmt.Fprintf(os.Stderr, "  rbdl --email user@example.com --callsign W%%\n")
		fmt.Fprintf(os.Stderr, "\nNote: Use %% as wildcard for pattern matching\n")
	}
	flag.Parse()
	return config
}

func validateConfig(config *Config) error {
	if config.Email == "" {
		return fmt.Errorf("email is required (use --email flag or set a RBDL_EMAIL environment variable)")
	}
	return nil
}

func fetchRepeaterData(config *Config) ([]byte, error) {
	// Build query parameters
	params := url.Values{}
	if config.Callsign != "" {
		params.Add("callsign", config.Callsign)
	}
	if config.City != "" {
		params.Add("city", config.City)
	}
	if config.Country != "" {
		params.Add("country", config.Country)
	}
	if config.Frequency != "" {
		params.Add("frequency", config.Frequency)
	}
	if config.Mode != "" {
		params.Add("mode", config.Mode)
	}
	if config.Landmark != "" {
		params.Add("landmark", config.Landmark)
	}
	if config.StateID != "" {
		params.Add("state_id", config.StateID)
	}
	if config.Region != "" {
		params.Add("region", config.Region)
	}
	if config.SType != "" {
		params.Add("stype", config.SType)
	}
	fullURL := apiEndpoint
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	// Set User-Agent header
	userAgent := fmt.Sprintf(userAgentTemplate, config.Email)
	req.Header.Set("User-Agent", userAgent)
	// Make the request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()
	// Check for error status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("rate limit exceeded (429): too many requests. Wait 10-60 seconds before retrying")
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	// Validate JSON
	var js json.RawMessage
	if err := json.Unmarshal(data, &js); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}
	return data, nil
}

func generateFilename(config *Config) string {
	timestamp := time.Now().Format("20060102_150405")
	// Build filename based on search parameters
	parts := []string{"repeaterbook"}
	if config.StateID != "" {
		parts = append(parts, "state_"+config.StateID)
	}
	if config.Country != "" {
		parts = append(parts, "country_"+config.Country)
	}
	if config.Mode != "" {
		parts = append(parts, "mode_"+config.Mode)
	}
	if config.Frequency != "" {
		parts = append(parts, "freq_"+config.Frequency)
	}
	filename := parts[0]
	if len(parts) > 1 {
		for i := 1; i < len(parts); i++ {
			filename += "_" + parts[i]
		}
	}
	filename += "_" + timestamp + ".json"
	return filename
}

func saveToFile(filepath string, data []byte) error {
	// Pretty print the JSON slightly
	var prettyJSON interface{}
	if err := json.Unmarshal(data, &prettyJSON); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}
	formatted, err := json.MarshalIndent(prettyJSON, "", "\t")
	if err != nil {
		return fmt.Errorf("formatting JSON: %w", err)
	}
	if err := os.WriteFile(filepath, formatted, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	return nil
}
