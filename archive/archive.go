package archive

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/carlmjohnson/flagx"
	"github.com/carlmjohnson/flagx/lazyio"
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
		fmt.Fprintf(os.Stderr, "\n\nError: %+v\n", err)
	}
	return err
}

func (app *appEnv) ParseArgs(args []string) error {
	fl := flag.NewFlagSet(AppName, flag.ContinueOnError)
	app.src = lazyio.FileOrURL(lazyio.StdIO, nil)
	fl.Var(app.src, "src", "source `file or URL` for things to replace")
	fl.DurationVar(&http.DefaultClient.Timeout, "timeout", 10*time.Second, "connection time out")
	fl.DurationVar(&app.retryTime, "retry-time", 5*time.Second, "duration to wait before retrying")
	fl.UintVar(&app.retryCount, "retries", 3, "number of times to retry")
	fl.StringVar(&app.from, "from", "", "date to search from in YYYYMMDD format")
	silent := fl.Bool("silent", false, "don't log lookups")
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
	if err := flagx.ParseEnv(fl, AppName); err != nil {
		return err
	}
	log.SetPrefix("webarchive: ")
	log.SetFlags(log.Lmsgprefix | log.LstdFlags)
	if *silent {
		log.SetOutput(io.Discard)
	}
	return nil
}

type appEnv struct {
	src        lazyio.Reader
	from       string
	retryCount uint
	retryTime  time.Duration
}

func (app *appEnv) Exec() (err error) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	b, err := io.ReadAll(app.src)
	if err != nil {
		return err
	}
	_ = app.src.Close()
	body := string(b)

	originalURLs := getURLs(body)
	replacements, err := app.getReplacements(ctx, originalURLs)
	// Always substitute what we can
	output := substituteReplacements(body, replacements)
	fmt.Print(output)
	if term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Println()
	}
	// Now return error
	if err != nil {
		return err
	}
	return nil
}

var urlPattern = regexp.MustCompile(`https?://[^\s()[\]'"]+`)

func getURLs(body string) []string {
	return urlPattern.FindAllString(body, -1)
}

func (app *appEnv) getReplacements(ctx context.Context, originalURLs []string) (map[string]string, error) {
	m := make(map[string]string, len(originalURLs))
	var errs []error
outer:
	for _, u := range originalURLs {
		if _, ok := m[u]; ok {
			continue
		}

		if strings.HasSuffix(u, ".js") {
			log.Println("skip", u)
			continue
		}
		uu, err := url.Parse(u)
		if err != nil {
			log.Println("skip", u)
			continue
		}
		if slices.Contains([]string{
			"web.archive.org",
			"www.spotlightpa.org", "spotlightpa.org", // TODO: Don't just harcode this
		}, uu.Hostname()) {
			log.Println("skip", u)
			continue
		}
		log.Println("lookup", u)
		var lookupErr error
		for i := range app.retryCount {
			if i > 0 {
				log.Println("retry", i, "in", app.retryTime)
				time.Sleep(app.retryTime)
			}
			s, err := app.lookup(ctx, u)
			if err == nil {
				log.Println("found", s)
				m[u] = s
				continue outer
			}
			log.Println("error", s)
			lookupErr = err
		}
		log.Println("failed")
		errs = append(errs, lookupErr)
	}

	return m, errors.Join(errs...)
}

func (app *appEnv) lookup(ctx context.Context, u string) (s string, err error) {
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
			return u, fmt.Errorf("could not find %q in WayBack machine", u)
		}
		return u, fmt.Errorf("problem connecting to WayBack machine: %w", err)
	}
	if len(rows) < 2 || len(rows[0]) != len(rows[1]) {
		return u, fmt.Errorf("bad response from WayBack machine: %q", rows)
	}
	tsIndex := -1
	for i, v := range rows[0] {
		if v == "timestamp" {
			tsIndex = i
			break
		}
	}
	if tsIndex == -1 {
		return u, fmt.Errorf("bad response from WayBack machine: %q", rows)
	}
	s = fmt.Sprintf("https://web.archive.org/%s/%s",
		rows[1][tsIndex], u)
	return s, nil
}

func substituteReplacements(body string, replacements map[string]string) string {
	return urlPattern.ReplaceAllStringFunc(body, func(s string) string {
		if r, ok := replacements[s]; ok {
			return r
		}
		return s
	})
}
