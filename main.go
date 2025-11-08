package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	apiEndpoint       = "https://www.repeaterbook.com/api/export.php"
	userAgentTemplate = "RepeaterbookDL CLI (beta), %s"
)

type Config struct {
	Email     string
	Output    string
	Format    string
	OnAir     bool
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
	if err := saveToFile(outputFile, data, config); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully saved data to: %s\n", outputFile)
}

func parseFlags() *Config {
	config := &Config{}
	flag.StringVar(&config.Email, "email", os.Getenv("RBDL_EMAIL"), "Email address (required, or set RBDL_EMAIL)")
	flag.StringVar(&config.Output, "output", "", "Output file path (auto-generated if not specified)")
	flag.StringVar(&config.Format, "format", "", "Output format: json or csv (auto-detected from output filename if not specified)")
	flag.BoolVar(&config.OnAir, "on-air", false, "Only include on-air repeaters")
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
		fmt.Fprintf(os.Stderr, "  rbdl --email user@example.com --country \"United States\" --mode DMR\n")
		fmt.Fprintf(os.Stderr, "  rbdl --email user@example.com --country Canada --mode DMR --format csv\n")
		fmt.Fprintf(os.Stderr, "  rbdl --email user@example.com --country \"United States\" --on-air\n")
		fmt.Fprintf(os.Stderr, "  rbdl --email user@example.com --output repeaters.csv\n")
		fmt.Fprintf(os.Stderr, "  rbdl --email user@example.com --country Mexico --frequency 146.52\n")
		fmt.Fprintf(os.Stderr, "  rbdl --email user@example.com --callsign W%%\n")
		fmt.Fprintf(os.Stderr, "\nNote: Use %% as wildcard for pattern matching\n")
	}
	flag.Parse()
	// If the format isn't explicitly specified, try to detect it from the output file's extension
	if config.Format == "" {
		if config.Output != "" {
			ext := strings.ToLower(filepath.Ext(config.Output))
			if ext == ".csv" {
				config.Format = "csv"
			} else if ext == ".json" {
				config.Format = "json"
			} else {
				// Default to json for unknown or no extension
				config.Format = "json"
			}
		} else {
			// No output file specified, default to json
			config.Format = "json"
		}
	}
	return config
}

func validateConfig(config *Config) error {
	if config.Email == "" {
		return fmt.Errorf("email is required (use --email flag or set a RBDL_EMAIL environment variable)")
	}
	if config.Format != "json" && config.Format != "csv" {
		return fmt.Errorf("format must be either 'json' or 'csv'")
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
	// User-Agent header format required to authenticate with the API
	userAgent := fmt.Sprintf(userAgentTemplate, config.Email)
	req.Header.Set("User-Agent", userAgent)
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
		// The actual rate limits are unpublished, but forum posts suggest it isn't too forgiving
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("rate limit exceeded (429): too many requests. Wait 10-60 seconds before retrying")
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	// Validate the JSON
	// API responses seem fairly standardized
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
	ext := ".json"
	if config.Format == "csv" {
		ext = ".csv"
	}
	filename += "_" + timestamp + ext
	return filename
}

func saveToFile(filepath string, data []byte, config *Config) error {
	if config.Format == "csv" {
		return saveToCSV(filepath, data, config.OnAir)
	}
	return saveToJSON(filepath, data, config.OnAir)
}

func saveToJSON(filepath string, data []byte, onAirOnly bool) error {
	// If filtering is needed, parse and reconstruct
	if onAirOnly {
		records, err := parseJSONToRecords(data, onAirOnly)
		if err != nil {
			return fmt.Errorf("parsing JSON: %w", err)
		}
		// Reconstruct response with filtered results and updated count
		response := map[string]interface{}{
			"count":   len(records),
			"results": records,
		}
		formatted, err := json.MarshalIndent(response, "", "\t")
		if err != nil {
			return fmt.Errorf("formatting JSON: %w", err)
		}
		if err := os.WriteFile(filepath, formatted, 0644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
		return nil
	}

	// No filtering, just pretty print
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

func parseJSONToRecords(data []byte, onAirOnly bool) ([]map[string]interface{}, error) {
	// RepeaterBook API returns: {"count": N, "results": [...]}
	var response struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("unable to parse API response: %w", err)
	}
	if len(response.Results) == 0 {
		return nil, fmt.Errorf("no results in API response")
	}

	// Filter for on-air repeaters if requested
	if onAirOnly {
		filtered := make([]map[string]interface{}, 0, len(response.Results))
		for _, record := range response.Results {
			if status, exists := record["Operational Status"]; exists {
				if statusStr, ok := status.(string); ok && statusStr == "On-air" {
					filtered = append(filtered, record)
				}
			}
		}
		return filtered, nil
	}

	return response.Results, nil
}

func saveToCSV(filepath string, data []byte, onAirOnly bool) error {
	records, err := parseJSONToRecords(data, onAirOnly)
	if err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}
	if len(records) == 0 {
		return fmt.Errorf("no data to write")
	}
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	// Collect all unique headers from all records
	headerSet := make(map[string]bool)
	for _, record := range records {
		for key := range record {
			headerSet[key] = true
		}
	}
	// Sort headers for consistent output
	headers := make([]string, 0, len(headerSet))
	for header := range headerSet {
		headers = append(headers, header)
	}
	sort.Strings(headers)
	// Write headers
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("writing headers: %w", err)
	}
	// Write data rows
	for _, record := range records {
		row := make([]string, len(headers))
		for i, header := range headers {
			if val, exists := record[header]; exists && val != nil {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("writing row: %w", err)
		}
	}
	return nil
}
