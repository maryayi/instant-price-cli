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

const (
	priceURL   = "https://min-api.cryptocompare.com/data/price?fsym=%s&tsyms=%s"
	historyURL = "https://min-api.cryptocompare.com/data/v2/histoday?fsym=%s&tsym=%s&limit=365"
)

var (
	version    = "v0.4.1"
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

type opts struct {
	symbol     string
	currency   string
	showChange bool
}

type changes struct {
	day   float64
	week  float64
	month float64
	year  float64
}

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

	o, ok := parseArgs(os.Args[1:])
	if !ok {
		printUsage(os.Stderr)
		os.Exit(1)
	}

	price, err := fetchPrice(o.symbol, o.currency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s: %s %s\n", o.symbol, formatAmount(price), o.currency)

	if o.showChange {
		ch, err := fetchChanges(o.symbol, o.currency, price)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  24h   %s\n", formatChange(ch.day))
		fmt.Printf("  7d    %s\n", formatChange(ch.week))
		fmt.Printf("  30d   %s\n", formatChange(ch.month))
		fmt.Printf("  1y    %s\n", formatChange(ch.year))
	}
}

// parseArgs handles flags in any position. Returns (opts, ok).
func parseArgs(args []string) (opts, bool) {
	o := opts{currency: "USD"}
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

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
		case "change":
			o.showChange = true
			if hasInline {
				fmt.Fprintf(os.Stderr, "error: --change does not take a value\n")
				return opts{}, false
			}
		case "currency", "c":
			var val string
			if hasInline {
				val = inlineVal
			} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				val = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "error: %s requires a currency code (e.g. EUR)\n", arg)
				return opts{}, false
			}
			if val == "" {
				fmt.Fprintf(os.Stderr, "error: %s requires a non-empty currency code\n", arg)
				return opts{}, false
			}
			o.currency = strings.ToUpper(val)
		default:
			fmt.Fprintf(os.Stderr, "error: unknown flag %q\n", arg)
			return opts{}, false
		}
	}

	if len(positional) != 1 {
		return opts{}, false
	}

	o.symbol = strings.ToUpper(positional[0])
	return o, true
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: price [options] <CRYPTO>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -c, --currency CODE   currency code for the price (default: USD)")
	fmt.Fprintln(w, "      --change          show price change for 24h, 7d, 30d, and 1y")
	fmt.Fprintln(w, "  -v, --version         show version")
	fmt.Fprintln(w, "  -h, --help            show this help message")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  price btc")
	fmt.Fprintln(w, "  price btc --currency EUR")
	fmt.Fprintln(w, "  price btc -c GBP --change")
}

func fetchPrice(symbol, currency string) (float64, error) {
	resp, err := httpClient.Get(fmt.Sprintf(priceURL, symbol, currency))
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

	if priceRaw, ok := raw[currency]; ok {
		var price float64
		if err := json.Unmarshal(priceRaw, &price); err != nil {
			return 0, fmt.Errorf("invalid price data in API response")
		}
		return price, nil
	}

	return 0, fmt.Errorf("no price found for %s/%s — check the symbol and currency code", symbol, currency)
}

func fetchChanges(symbol, currency string, currentPrice float64) (changes, error) {
	resp, err := httpClient.Get(fmt.Sprintf(historyURL, symbol, currency))
	if err != nil {
		return changes{}, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return changes{}, fmt.Errorf("API error (HTTP %d)", resp.StatusCode)
	}

	var result struct {
		Response string `json:"Response"`
		Data     struct {
			Data []struct {
				Close float64 `json:"close"`
			} `json:"Data"`
		} `json:"Data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return changes{}, fmt.Errorf("failed to parse API response: %w", err)
	}

	if result.Response != "Success" {
		return changes{}, fmt.Errorf("no history data available for %s/%s", symbol, currency)
	}

	data := result.Data.Data
	n := len(data)

	pct := func(i int) float64 {
		if i < 0 || i >= n || data[i].Close == 0 {
			return 0
		}
		return (currentPrice - data[i].Close) / data[i].Close * 100
	}

	return changes{
		day:   pct(n - 2), // yesterday's close
		week:  pct(n - 8), // 7 days ago
		month: pct(n - 31), // 30 days ago
		year:  pct(0),      // 365 days ago
	}, nil
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

func formatChange(pct float64) string {
	if pct >= 0 {
		return fmt.Sprintf("+%.2f%%", pct)
	}
	return fmt.Sprintf("%.2f%%", pct)
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
