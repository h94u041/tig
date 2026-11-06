// Command tighub is a hub: it keeps the documents and chains projects push to
// it with tigwx push, and shows them.
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
	"runtime"
	"time"

	"github.com/h94u041/tig/hub"
)

const usage = `tighub keeps the documents and chains projects push to it, and shows them.

Usage: tighub [-addr :8080] [-data dir] [-tokens file]

The tokens file holds one "owner token" per line; a push needs its owner's
token, of at least 16 characters. Without one, the hub takes no pushes and only
shows what it has. tighub speaks plain http: on a network, put it behind a
reverse proxy that does https and limits how fast anyone may ask.
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
		if runtime.GOOS != "windows" {
			if fi, err := os.Stat(*tokensFile); err == nil && fi.Mode().Perm()&0o077 != 0 {
				log.Printf("warning: others on this machine can read %s; chmod 600 it", *tokensFile)
			}
		}
	}
	if err := os.MkdirAll(*data, 0o755); err != nil {
		log.Fatal(err)
	}
	// The limits are for a slow or stuck client: a push of the largest size has two
	// minutes to arrive.
	srv := &http.Server{Addr: *addr, Handler: hub.NewServer(*data, tokens).Handler(),
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 2 * time.Minute, WriteTimeout: 2 * time.Minute,
		IdleTimeout: 2 * time.Minute, MaxHeaderBytes: 64 << 10}

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
