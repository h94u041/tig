package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/h94u041/tig/hub"
	"github.com/h94u041/tig/mcpserver"
)

func (c *cli) mcp(args []string) error {
	fs := c.flags("mcp")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if err := c.open(); err != nil {
		return err
	}
	s := mcpserver.New(c.r, mcpserver.Options{Version: version})
	return s.Run(c.ctx, &mcp.StdioTransport{})
}

func (c *cli) push(args []string) error {
	fs := c.flags("push")
	token := fs.String("token", os.Getenv("TIG_TOKEN"), "the owner's token on the hub (default $TIG_TOKEN)")
	note := fs.String("m", "", "a note for this publication")
	insecure := fs.Bool("insecure", false, "send the token over plain http to another machine, where it can be read on the way")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	project := os.Getenv("TIG_HUB")
	if fs.NArg() > 0 {
		project = fs.Arg(0)
	}
	if project == "" {
		return &flagError{fs.Name(), errors.New("push needs the project's address on a hub: tigwx push https://host/owner/name (or $TIG_HUB)")}
	}
	if err := c.open(); err != nil {
		return err
	}
	res, err := hub.PushWith(c.ctx, c.r, project, hub.PushOptions{Token: *token, Actor: c.actor, Note: *note, Insecure: *insecure})
	if len(res.Unsnapped) > 0 {
		fmt.Fprintf(c.errs, "these documents changed since the latest snapshot and are pushed as it has them (run tigwx snap): %s\n",
			strings.Join(res.Unsnapped, ", "))
	}
	if err != nil {
		return err
	}
	if c.asJSON {
		return printJSON(c.out, res)
	}
	if res.UpToDate {
		fmt.Fprintf(c.out, "up to date: %s has these documents and this chain (publication %s)\n", project, short(res.Publication.ID))
		return nil
	}
	fmt.Fprintf(c.out, "pushed publication %s to %s (%d objects sent)\n", short(res.Publication.ID), project, res.Sent)
	return nil
}
