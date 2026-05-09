# instant-price-cli

A fast, zero-dependency CLI tool to get real-time cryptocurrency prices in any currency.

```
$ price btc
BTC: 80,391.88 USD

$ price eth --currency EUR
ETH: 2,105.43 EUR

$ price doge -c GBP
DOGE: 0.094200 GBP
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
price [-c CODE | --currency CODE] <CRYPTO>
```

The symbol and currency code are both case-insensitive.

**Options:**

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--currency` | `-c` | `USD` | Currency code to display the price in |
| `--version` | `-v` | | Show version |
| `--help` | `-h` | | Show help message |

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
