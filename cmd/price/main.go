package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

const apiKeyEnv = "APP_CONFIG_CRYPTO_COMPARE_API_KEY"

const (
	priceURL   = "https://min-api.cryptocompare.com/data/price?fsym=%s&tsyms=%s"
	historyURL = "https://min-api.cryptocompare.com/data/v2/histoday?fsym=%s&tsym=%s&limit=365"
	graphW     = 60
	graphH     = 10
)

var (
	version    = "v0.7.0"
	httpClient = &http.Client{Timeout: 10 * time.Second}
	apiKey     string
	lastCall   time.Time
)

// minInterval spaces out API calls to respect CryptoCompare's free-tier limit
// of one request per second. Commands like --change and --graph make two calls.
const minInterval = 1100 * time.Millisecond

type opts struct {
	symbol      string
	currency    string
	showChange  bool
	jsonOut     bool
	graphPeriod string
}

type changes struct {
	day   float64
	week  float64
	month float64
	year  float64
}

type jsonResult struct {
	Symbol   string      `json:"symbol"`
	Currency string      `json:"currency"`
	Price    float64     `json:"price"`
	Change   *jsonChange `json:"change,omitempty"`
}

type jsonChange struct {
	Day   float64 `json:"24h"`
	Week  float64 `json:"7d"`
	Month float64 `json:"30d"`
	Year  float64 `json:"1y"`
}

type histoPoint struct {
	Time  int64   `json:"time"`
	Close float64 `json:"close"`
}

func main() {
	loadAPIKey()

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

	if o.graphPeriod != "" {
		pts, err := fetchHistory(o.symbol, o.currency, o.graphPeriod)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s: %s %s\n", o.symbol, formatAmount(price), o.currency)
		fmt.Print(renderGraph(pts, o.symbol, o.currency, o.graphPeriod))
		return
	}

	var ch *changes
	if o.showChange {
		c, err := fetchChanges(o.symbol, o.currency, price)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		ch = &c
	}

	if o.jsonOut {
		result := jsonResult{
			Symbol:   o.symbol,
			Currency: o.currency,
			Price:    price,
		}
		if ch != nil {
			r2 := func(f float64) float64 { return math.Round(f*100) / 100 }
			result.Change = &jsonChange{
				Day:   r2(ch.day),
				Week:  r2(ch.week),
				Month: r2(ch.month),
				Year:  r2(ch.year),
			}
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
		return
	}

	fmt.Printf("%s: %s %s\n", o.symbol, formatAmount(price), o.currency)
	if ch != nil {
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
		case "json":
			o.jsonOut = true
			if hasInline {
				fmt.Fprintf(os.Stderr, "error: --json does not take a value\n")
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
		case "graph", "g":
			var val string
			if hasInline {
				val = inlineVal
			} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				val = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "error: --graph requires a period: d (day), w (week), m (month), or y (year)\n")
				return opts{}, false
			}
			switch val {
			case "d", "w", "m", "y":
				o.graphPeriod = val
			default:
				fmt.Fprintf(os.Stderr, "error: --graph value must be d, w, m, or y\n")
				return opts{}, false
			}
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
	fmt.Fprintln(w, "  -g, --graph PERIOD    ASCII price chart: d=day, w=week, m=month, y=year")
	fmt.Fprintln(w, "      --json            output results as JSON")
	fmt.Fprintln(w, "  -v, --version         show version")
	fmt.Fprintln(w, "  -h, --help            show this help message")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  price btc")
	fmt.Fprintln(w, "  price btc --currency EUR")
	fmt.Fprintln(w, "  price btc -c GBP --change")
	fmt.Fprintln(w, "  price btc --graph d")
	fmt.Fprintln(w, "  price btc --graph w")
	fmt.Fprintln(w, "  price btc -g m")
	fmt.Fprintln(w, "  price btc -g y -c EUR")
	fmt.Fprintln(w, "  price btc --json")
	fmt.Fprintln(w, "  price btc --change --json")
}

// loadAPIKey reads the CryptoCompare API key from the environment, falling
// back to a .env file in the current directory. An already-exported
// environment variable takes precedence over the .env file.
func loadAPIKey() {
	if v := os.Getenv(apiKeyEnv); v != "" {
		apiKey = v
		return
	}

	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(key) == apiKeyEnv {
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			apiKey = val
			return
		}
	}
}

// apiGet performs a GET request against the CryptoCompare API, attaching the
// API key as an Authorization header when one is configured.
func apiGet(url string) (*http.Response, error) {
	if wait := minInterval - time.Since(lastCall); !lastCall.IsZero() && wait > 0 {
		time.Sleep(wait)
	}
	lastCall = time.Now()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Apikey "+apiKey)
	}
	return httpClient.Do(req)
}

// statusError converts a non-200 response into a descriptive error.
func statusError(code int) error {
	if code == http.StatusUnauthorized {
		if apiKey == "" {
			return fmt.Errorf("API error (HTTP 401): CryptoCompare requires an API key. Set %s in your environment or a .env file", apiKeyEnv)
		}
		return fmt.Errorf("API error (HTTP 401): CryptoCompare rejected the API key. Check that %s is valid", apiKeyEnv)
	}
	return fmt.Errorf("API error (HTTP %d)", code)
}

// apiBodyError detects CryptoCompare error payloads returned with HTTP 200
// (e.g. rate-limit messages) in the flat price response.
func apiBodyError(raw map[string]json.RawMessage) error {
	respRaw, ok := raw["Response"]
	if !ok {
		return nil
	}
	var response string
	if json.Unmarshal(respRaw, &response) == nil && response == "Error" {
		var msg string
		json.Unmarshal(raw["Message"], &msg)
		if msg == "" {
			msg = "unknown API error"
		}
		return fmt.Errorf("API error: %s", msg)
	}
	return nil
}

func fetchPrice(symbol, currency string) (float64, error) {
	resp, err := apiGet(fmt.Sprintf(priceURL, symbol, currency))
	if err != nil {
		return 0, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, statusError(resp.StatusCode)
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return 0, fmt.Errorf("failed to parse API response: %w", err)
	}

	if err := apiBodyError(raw); err != nil {
		return 0, err
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
	resp, err := apiGet(fmt.Sprintf(historyURL, symbol, currency))
	if err != nil {
		return changes{}, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return changes{}, statusError(resp.StatusCode)
	}

	var result struct {
		Response string `json:"Response"`
		Message  string `json:"Message"`
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
		if result.Message != "" {
			return changes{}, fmt.Errorf("API error: %s", result.Message)
		}
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
		day:   pct(n - 2),  // yesterday's close
		week:  pct(n - 8),  // 7 days ago
		month: pct(n - 31), // 30 days ago
		year:  pct(0),      // 365 days ago
	}, nil
}

func fetchHistory(symbol, currency, period string) ([]histoPoint, error) {
	var url string
	switch period {
	case "d":
		url = fmt.Sprintf("https://min-api.cryptocompare.com/data/v2/histohour?fsym=%s&tsym=%s&limit=24", symbol, currency)
	case "w":
		url = fmt.Sprintf("https://min-api.cryptocompare.com/data/v2/histoday?fsym=%s&tsym=%s&limit=7", symbol, currency)
	case "m":
		url = fmt.Sprintf("https://min-api.cryptocompare.com/data/v2/histoday?fsym=%s&tsym=%s&limit=30", symbol, currency)
	case "y":
		url = fmt.Sprintf("https://min-api.cryptocompare.com/data/v2/histoday?fsym=%s&tsym=%s&limit=365", symbol, currency)
	}

	resp, err := apiGet(url)
	if err != nil {
		return nil, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, statusError(resp.StatusCode)
	}

	var result struct {
		Response string `json:"Response"`
		Message  string `json:"Message"`
		Data     struct {
			Data []histoPoint `json:"Data"`
		} `json:"Data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if result.Response != "Success" {
		if result.Message != "" {
			return nil, fmt.Errorf("API error: %s", result.Message)
		}
		return nil, fmt.Errorf("no history data available for %s/%s", symbol, currency)
	}

	// Filter out zero-price entries (API gap fill)
	pts := result.Data.Data
	clean := pts[:0]
	for _, p := range pts {
		if p.Close > 0 {
			clean = append(clean, p)
		}
	}
	return clean, nil
}

func renderGraph(pts []histoPoint, symbol, currency, period string) string {
	if len(pts) < 2 {
		return "not enough data to render graph\n"
	}

	prices := make([]float64, len(pts))
	for i, p := range pts {
		prices[i] = p.Close
	}

	minP, maxP := prices[0], prices[0]
	for _, p := range prices {
		if p < minP {
			minP = p
		}
		if p > maxP {
			maxP = p
		}
	}

	rng := maxP - minP
	if rng == 0 {
		rng = maxP * 0.02
		if rng == 0 {
			rng = 1
		}
	}
	pad := rng * 0.1
	lo := minP - pad
	hi := maxP + pad
	span := hi - lo

	toCol := func(i int) int {
		return int(math.Round(float64(i) * float64(graphW-1) / float64(len(pts)-1)))
	}
	toRow := func(p float64) int {
		r := int(math.Round((hi - p) / span * float64(graphH-1)))
		if r < 0 {
			return 0
		}
		if r >= graphH {
			return graphH - 1
		}
		return r
	}

	grid := make([][]rune, graphH)
	for i := range grid {
		row := make([]rune, graphW)
		for j := range row {
			row[j] = ' '
		}
		grid[i] = row
	}

	for i := 0; i < len(pts)-1; i++ {
		x0, y0 := toCol(i), toRow(prices[i])
		x1, y1 := toCol(i+1), toRow(prices[i+1])

		if x0 == x1 {
			lo2, hi2 := y0, y1
			if lo2 > hi2 {
				lo2, hi2 = hi2, lo2
			}
			for y := lo2; y <= hi2; y++ {
				if grid[y][x0] == ' ' {
					grid[y][x0] = '│'
				}
			}
			continue
		}

		for x := x0; x <= x1; x++ {
			t := float64(x-x0) / float64(x1-x0)
			y := int(math.Round(float64(y0) + t*float64(y1-y0)))
			if y < 0 {
				y = 0
			}
			if y >= graphH {
				y = graphH - 1
			}
			dy := y1 - y0
			var ch rune
			switch {
			case dy == 0:
				ch = '─'
			case dy > 0:
				ch = '╲'
			default:
				ch = '╱'
			}
			if grid[y][x] == ' ' {
				grid[y][x] = ch
			}
		}
	}

	// Compute Y-axis label width from all label rows
	lw := 0
	for row := 0; row < graphH; row++ {
		p := hi - float64(row)/float64(graphH-1)*span
		s := formatAmount(p)
		if len(s) > lw {
			lw = len(s)
		}
	}

	var sb strings.Builder

	periodNames := map[string]string{"d": "24h", "w": "7d", "m": "30d", "y": "1y"}
	fmt.Fprintf(&sb, "%s/%s — Last %s\n", symbol, currency, periodNames[period])

	labelRows := map[int]bool{
		0:            true,
		(graphH-1)/4: true,
		(graphH-1)/2: true,
		3*(graphH-1)/4: true,
		graphH - 1:  true,
	}

	for row := 0; row < graphH; row++ {
		p := hi - float64(row)/float64(graphH-1)*span
		var priceStr string
		if labelRows[row] {
			priceStr = formatAmount(p)
		}
		axis := " ┤"
		if row == graphH-1 {
			axis = " └"
		}
		fmt.Fprintf(&sb, "%*s%s%s\n", lw, priceStr, axis, string(grid[row]))
	}

	fmt.Fprintf(&sb, "%s%s\n", strings.Repeat(" ", lw+2), strings.Repeat("─", graphW))
	sb.WriteString(makeXLabels(pts, period, lw+2))
	sb.WriteByte('\n')

	return sb.String()
}

func makeXLabels(pts []histoPoint, period string, offset int) string {
	n := len(pts)
	total := offset + graphW
	line := []rune(strings.Repeat(" ", total))

	var formatTime func(t int64) string
	switch period {
	case "d":
		formatTime = func(t int64) string { return time.Unix(t, 0).Local().Format("15:04") }
	case "w":
		formatTime = func(t int64) string { return time.Unix(t, 0).Local().Format("Mon") }
	case "m":
		formatTime = func(t int64) string { return time.Unix(t, 0).Local().Format("Jan 2") }
	case "y":
		formatTime = func(t int64) string { return time.Unix(t, 0).Local().Format("Jan 06") }
	}

	toCol := func(i int) int {
		return offset + int(math.Round(float64(i)*float64(graphW-1)/float64(n-1)))
	}

	const nLabels = 5
	for j := 0; j < nLabels; j++ {
		idx := j * (n - 1) / (nLabels - 1)
		label := []rune(formatTime(pts[idx].Time))
		col := toCol(idx)
		start := col - len(label)/2
		if start+len(label) > total {
			start = total - len(label)
		}
		if start < 0 {
			start = 0
		}
		for k, ch := range label {
			if pos := start + k; pos < total {
				line[pos] = ch
			}
		}
	}

	return strings.TrimRight(string(line), " ")
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
