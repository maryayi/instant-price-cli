# instant-price-cli

A fast, zero-dependency CLI tool to get real-time cryptocurrency prices in USD.

```
$ price btc
BTC: $80,476.29

$ price eth
ETH: $2,318.42

$ price doge
DOGE: $0.110600
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
price <CRYPTO>
```

The symbol is case-insensitive:

```sh
price btc
price BTC
price ETH
price SOL
price DOGE
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
