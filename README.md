# webarchive [![GoDoc](https://godoc.org/github.com/carlmjohnson/webarchive?status.svg)](https://godoc.org/github.com/carlmjohnson/webarchive) [![Go Report Card](https://goreportcard.com/badge/github.com/carlmjohnson/webarchive)](https://goreportcard.com/report/github.com/carlmjohnson/webarchive)

Look up WayBack Machine address for URL.

## Installation

First install [Go](http://golang.org).

If you just want to install the binary to your current directory and don't care about the source code, run

```bash
GOBIN="$(pwd)" go install github.com/carlmjohnson/webarchive@latest
```

## Screenshots

```
$ webarchive -h
webarchive - Look up WayBack Machine address for URL.

Usage:

        webarchive [options]

Options:
  -timeout duration
        connection time out (default 10s)

$ webarchive https://example.com
https://web.archive.org/web/20210504155907/https://example.com/
```
