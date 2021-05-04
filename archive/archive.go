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
	u = "https://web.archive.org/" + u
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status for %q: %s", u, res.Status)
	}
	fmt.Print(res.Request.URL)
	return nil
}
