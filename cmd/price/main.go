package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const apiURL = "https://min-api.cryptocompare.com/data/price?fsym=%s&tsyms=USD"

type priceResponse struct {
	USD      *float64 `json:"USD"`
	Response string   `json:"Response"`
	Message  string   `json:"Message"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: price <CRYPTO>")
		os.Exit(1)
	}

	symbol := strings.ToUpper(strings.TrimSpace(os.Args[1]))
	price, err := fetchPrice(symbol)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s: $%s\n", symbol, formatUSD(price))
}

func fetchPrice(symbol string) (float64, error) {
	resp, err := httpClient.Get(fmt.Sprintf(apiURL, symbol))
	if err != nil {
		return 0, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API error (HTTP %d)", resp.StatusCode)
	}

	var result priceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to parse API response: %w", err)
	}

	if result.Response == "Error" {
		return 0, fmt.Errorf("%q not found — check the symbol and try again", symbol)
	}

	if result.USD == nil {
		return 0, fmt.Errorf("no price data returned for %q", symbol)
	}

	return *result.USD, nil
}

func formatUSD(price float64) string {
	switch {
	case price >= 1:
		return withCommas(fmt.Sprintf("%.2f", price))
	case price >= 0.0001:
		return fmt.Sprintf("%.6f", price)
	default:
		return fmt.Sprintf("%.10f", price)
	}
}

func withCommas(s string) string {
	parts := strings.SplitN(s, ".", 2)
	n := parts[0]
	var b strings.Builder
	for i, ch := range n {
		if i > 0 && (len(n)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}
	if len(parts) > 1 {
		b.WriteByte('.')
		b.WriteString(parts[1])
	}
	return b.String()
}
