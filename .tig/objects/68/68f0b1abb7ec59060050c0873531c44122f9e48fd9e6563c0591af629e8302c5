package main

import (
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
)

// version is set when a release is built; otherwise it comes from how Go built
// this, and fallbackVersion is for a build that knows nothing.
var version = ""

// usageOf is what the usage says of a command, one command per entry.
func usageOf(name string) []string {
	var found []string
	in := false
	for _, line := range strings.Split(usage, "\n") {
		switch {
		case line == "  "+name || strings.HasPrefix(line, "  "+name+" "):
			found, in = append(found, strings.TrimSpace(line)), true
		case in && strings.HasPrefix(line, "   "):
			found[len(found)-1] += "\n    " + strings.TrimSpace(line)
		default:
			in = false
		}
	}
	return found
}

// commandHelp prints a command's usage and its flags. This is an answer, so it
// goes to stdout.
func (c *cli) commandHelp(fs *flag.FlagSet) {
	for _, line := range usageOf(fs.Name()) {
		fmt.Fprintf(c.out, "usage: tigwx %s\n", line)
	}
	var defaults strings.Builder
	fs.SetOutput(&defaults)
	fs.PrintDefaults()
	fs.SetOutput(io.Discard)
	if defaults.Len() > 0 {
		fmt.Fprintf(c.out, "\nflags:\n%s", defaults.String())
	}
}

// help answers tigwx help, tigwx help -a and tigwx help <command>.
func (c *cli) help(args []string) error {
	fs := c.flags("help")
	all := fs.Bool("a", false, "every command, and those on PATH")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		name, rest := fs.Arg(0), fs.Args()[1:]
		if cmd, ok := c.commands()[name]; ok {
			c.asJSON = false
			return cmd(append(slices.Clone(rest), "-h"))
		}
		if path, ok := external(name); ok {
			if code := c.runExternal(path, []string{"--help"}); code != exitOK {
				return errNo
			}
			return nil
		}
		fmt.Fprintf(c.errs, "tigwx: no help for '%s'. See 'tigwx help -a'.\n", name)
		return errNo
	}
	if !*all && !c.asJSON {
		fmt.Fprint(c.out, usage)
		return nil
	}
	var builtin []string
	for name := range c.commands() {
		builtin = append(builtin, name)
	}
	slices.Sort(builtin)
	onPath := externals()
	return c.emit(map[string][]string{"commands": builtin, "external": onPath}, func() {
		fmt.Fprintln(c.out, "commands:")
		for _, name := range builtin {
			fmt.Fprintf(c.out, "  %s\n", name)
		}
		if len(onPath) > 0 {
			fmt.Fprintln(c.out, "\ncommands on PATH:")
			for _, name := range onPath {
				fmt.Fprintf(c.out, "  %s\n", name)
			}
		}
	})
}

func (c *cli) version(args []string) error {
	if err := c.parse(c.flags("version"), args); err != nil {
		return err
	}
	v := versionString()
	return c.emit(map[string]string{"version": v}, func() { fmt.Fprintf(c.out, "tigwx version %s\n", v) })
}
