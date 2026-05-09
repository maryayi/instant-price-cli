# instant-price-cli

A fast, zero-dependency CLI tool to get real-time cryptocurrency prices in any currency.

```
$ price btc
BTC: 80,259.82 USD

$ price btc --change
BTC: 80,259.82 USD
  24h   +0.08%
  7d    +2.01%
  30d  +11.78%
  1y   -22.06%

$ price btc --graph w
BTC: 80,259.82 USD
BTC/USD — Last 7d
81,722.22 ┤                                                            
          ┤                              ╱╱╱╱╱╲                        
80,954.78 ┤                        ╱╱╱╱╱╱      ╲╲                      
          ┤                      ╱╱              ╲╲╲                   
80,187.33 ┤                   ╱╱╱                   ╲╲─────────────────
          ┤                ╱╱╱                                         
79,419.89 ┤             ╱╱╱                                            
          ┤          ╱╱╱                                               
          ┤─────────╱                                                  
78,268.72 └                                                            
           ────────────────────────────────────────────────────────────
          Sat     Sun              Tue              Thu             Sat

$ price btc --json
{
  "symbol": "BTC",
  "currency": "USD",
  "price": 80259.82
}

$ price btc --change --json
{
  "symbol": "BTC",
  "currency": "USD",
  "price": 80259.82,
  "change": {
    "24h": 0.08,
    "7d": 2.01,
    "30d": 11.78,
    "1y": -22.06
  }
}
```

## Requirements

- Go 1.21 or later

## Install

```sh
go install github.com/maryayi/instant-price-cli/cmd/price@latest
```

This installs the `price` binary into `$GOPATH/bin` (usually `~/go/bin`).
Make sure that directory is in your `$PATH`:

```sh
# Add to ~/.bashrc or ~/.zshrc if not already there
export PATH="$HOME/go/bin:$PATH"
```

## Usage

```
price [options] <CRYPTO>
```

The symbol and currency code are both case-insensitive.

**Options:**

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--currency` | `-c` | `USD` | Currency code to display the price in |
| `--change` | | | Show price change for 24h, 7d, 30d, and 1y |
| `--graph PERIOD` | `-g` | | ASCII price chart: `d`=day, `w`=week, `m`=month, `y`=year |
| `--json` | | | Output results as JSON |
| `--version` | `-v` | | Show version |
| `--help` | `-h` | | Show help message |

> **Note:** `--change` has no shorthand because `-c` is reserved for `--currency`.

**Examples:**

```sh
# Default (USD)
price btc
price ETH

# Other currencies — flag can go before or after the symbol
price btc --currency EUR
price btc -c GBP
price --currency JPY eth
price -c=CAD sol

# Show price change over multiple timeframes
price btc --change
price btc -c EUR --change

# ASCII price chart
price btc --graph d        # last 24 hours (hourly)
price btc --graph w        # last 7 days
price btc -g m             # last 30 days
price btc -g y -c EUR      # last year in EUR

# Output as JSON (combinable with other flags)
price btc --json
price btc --change --json
price btc -c EUR --json

# Help and version
price --help
price --version
```

## Build from source

```sh
git clone https://github.com/maryayi/instant-price-cli.git
cd instant-price-cli
go build -o price ./cmd/price
```

Then move the binary to a directory in your `$PATH`:

```sh
# Linux / macOS
mv price /usr/local/bin/
```

## Data source

Prices are fetched live from [CryptoCompare](https://www.cryptocompare.com/), a free public API — no account or API key required.

## License

[MIT](LICENSE)
