package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const apiURL = "https://min-api.cryptocompare.com/data/price?fsym=%s&tsyms=%s"

var (
	version    = "v0.2.0"
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

func main() {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-v", "-version":
			fmt.Println("price", version)
			return
		case "--help", "-h", "-help":
			printUsage(os.Stdout)
			return
		}
	}

	symbol, currency, ok := parseArgs(os.Args[1:])
	if !ok {
		printUsage(os.Stderr)
		os.Exit(1)
	}

	price, err := fetchPrice(symbol, currency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s: %s %s\n", symbol, formatAmount(price), currency)
}

// parseArgs handles flags in any position. Returns (symbol, currency, ok).
func parseArgs(args []string) (string, string, bool) {
	currency := "USD"
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Extract flag name and optional inline value from -flag, --flag, -flag=val, --flag=val
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}

		trimmed := strings.TrimLeft(arg, "-")
		name, inlineVal, hasInline := strings.Cut(trimmed, "=")

		switch name {
		case "version", "v":
			fmt.Println("price", version)
			os.Exit(0)
		case "help", "h":
			printUsage(os.Stdout)
			os.Exit(0)
		case "currency", "c":
			var val string
			if hasInline {
				val = inlineVal
			} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				val = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "error: %s requires a currency code (e.g. EUR)\n", arg)
				return "", "", false
			}
			if val == "" {
				fmt.Fprintf(os.Stderr, "error: %s requires a non-empty currency code\n", arg)
				return "", "", false
			}
			currency = strings.ToUpper(val)
		default:
			fmt.Fprintf(os.Stderr, "error: unknown flag %q\n", arg)
			return "", "", false
		}
	}

	if len(positional) != 1 {
		return "", "", false
	}

	return strings.ToUpper(positional[0]), currency, true
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: price [-c CODE | --currency CODE] <CRYPTO>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -c, --currency CODE   currency code for the price (default: USD)")
	fmt.Fprintln(w, "  -v, --version         show version")
	fmt.Fprintln(w, "  -h, --help            show this help message")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  price btc")
	fmt.Fprintln(w, "  price btc --currency EUR")
	fmt.Fprintln(w, "  price btc -c GBP")
}

func fetchPrice(symbol, currency string) (float64, error) {
	resp, err := httpClient.Get(fmt.Sprintf(apiURL, symbol, currency))
	if err != nil {
		return 0, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API error (HTTP %d)", resp.StatusCode)
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return 0, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Success path: the currency key is present when the request is valid.
	if priceRaw, ok := raw[currency]; ok {
		var price float64
		if err := json.Unmarshal(priceRaw, &price); err != nil {
			return 0, fmt.Errorf("invalid price data in API response")
		}
		return price, nil
	}

	return 0, fmt.Errorf("no price found for %s/%s — check the symbol and currency code", symbol, currency)
}

func formatAmount(price float64) string {
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
