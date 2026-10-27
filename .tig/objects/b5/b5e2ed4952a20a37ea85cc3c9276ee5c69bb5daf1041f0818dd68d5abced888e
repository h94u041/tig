// Command tighub is a hub: it keeps the documents and chains projects push to
// it with tigprint push, and shows them.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/h94u041/tig/hub"
)

const usage = `tighub keeps the documents and chains projects push to it, and shows them.

Usage: tighub [-addr :8080] [-data dir] [-tokens file]

The tokens file holds one "owner token" per line; a push needs its owner's
token. Without one, the hub takes no pushes and only shows what it has.
`

func main() {
	fs := flag.NewFlagSet("tighub", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	addr := fs.String("addr", ":8080", "the address to listen on")
	data := fs.String("data", "tighub-data", "the folder the projects are kept in")
	tokensFile := fs.String("tokens", "", "the tokens file")
	fs.Parse(os.Args[1:])

	tokens := map[string]string{}
	if *tokensFile != "" {
		var err error
		if tokens, err = hub.LoadTokens(*tokensFile); err != nil {
			log.Fatal(err)
		}
	}
	if err := os.MkdirAll(*data, 0o755); err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Addr: *addr, Handler: hub.NewServer(*data, tokens).Handler(),
		ReadHeaderTimeout: 10 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	log.Printf("tighub on %s, keeping projects in %s, %d owners can push", *addr, *data, owners(tokens))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func owners(tokens map[string]string) int {
	seen := map[string]bool{}
	for _, o := range tokens {
		seen[o] = true
	}
	return len(seen)
}
