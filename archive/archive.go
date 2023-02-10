package archive

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/carlmjohnson/flagext"
	"github.com/carlmjohnson/requests"
	"golang.org/x/term"
)

const AppName = "webarchive"

func CLI(args []string) error {
	var app appEnv
	err := app.ParseArgs(args)
	if err != nil {
		return err
	}
	if err = app.Exec(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	return err
}

func (app *appEnv) ParseArgs(args []string) error {
	fl := flag.NewFlagSet(AppName, flag.ContinueOnError)
	fl.DurationVar(&http.DefaultClient.Timeout, "timeout", 10*time.Second, "connection time out")
	fl.StringVar(&app.from, "from", "", "date to search from in YYYYMMDD format")
	fl.Usage = func() {
		fmt.Fprintf(fl.Output(), `webarchive - Look up WayBack Machine address for URL.

Usage:

	webarchive [options]

Options:
`)
		fl.PrintDefaults()
		fmt.Fprintln(fl.Output(), "")
	}
	if err := fl.Parse(args); err != nil {
		return err
	}
	if err := flagext.ParseEnv(fl, AppName); err != nil {
		return err
	}
	if err := flagext.MustHaveArgs(fl, 1, -1); err != nil {
		return err
	}
	app.urls = fl.Args()
	return nil
}

type appEnv struct {
	urls []string
	from string
}

func (app *appEnv) Exec() (err error) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	for i, url := range app.urls {
		if i > 0 {
			fmt.Println()
		}
		if err := app.lookup(ctx, url); err != nil {
			return err
		}
	}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Println()
	}
	return nil
}

func (app *appEnv) lookup(ctx context.Context, u string) (err error) {
	rb := requests.
		URL("https://web.archive.org/cdx/search/cdx?output=json&limit=1").
		Param("url", u)
	if app.from != "" {
		rb.Param("from", app.from)
	}
	var rows [][]string
	if err = rb.
		ToJSON(&rows).
		Fetch(ctx); err != nil {
		if requests.HasStatusErr(err, http.StatusNotFound) {
			return fmt.Errorf("could not find %q in WayBack machine", u)
		}
		return fmt.Errorf("problem connecting to WayBack machine: %w", err)
	}
	if len(rows) < 2 || len(rows[0]) != len(rows[1]) {
		return fmt.Errorf("bad response from WayBack machine: %q", rows)
	}
	tsIndex := -1
	for i, v := range rows[0] {
		if v == "timestamp" {
			tsIndex = i
			break
		}
	}
	if tsIndex == -1 {
		return fmt.Errorf("bad response from WayBack machine: %q", rows)
	}
	fmt.Printf("https://web.archive.org/%s/%s\n",
		rows[1][tsIndex], u)
	return nil
}
