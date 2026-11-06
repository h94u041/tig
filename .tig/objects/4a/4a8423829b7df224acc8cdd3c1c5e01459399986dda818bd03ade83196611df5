package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/h94u041/tig"
)

// config reads and sets what the config files hold.
func (c *cli) config(args []string) error {
	const use = "usage: tigwx config [-global] <key> [value]\n   or: tigwx config [-global] -list\n   or: tigwx config [-global] -unset <key>"
	fs := c.flags("config")
	global := fs.Bool("global", false, "the file for every project of yours, not this project's")
	list := fs.Bool("list", false, "every value set")
	unset := fs.Bool("unset", false, "take the value away")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	switch {
	case *list && (fs.NArg() != 0 || *unset), *unset && fs.NArg() != 1, !*list && (fs.NArg() < 1 || fs.NArg() > 2):
		return errors.New(use)
	}
	if *list {
		return c.configList(*global)
	}
	key := fs.Arg(0)
	k, ok := configKeyOf(key)
	if !ok {
		return fmt.Errorf("usage: tigwx config: no key '%s'; the keys are %s", key, strings.Join(configKeyNames(), ", "))
	}
	if *global && !k.global {
		return fmt.Errorf("usage: tigwx config: '%s' belongs to a project; it has no global value", key)
	}
	if !*unset && fs.NArg() == 1 {
		return c.configGet(k, *global)
	}
	value := ""
	if !*unset {
		if value = fs.Arg(1); value == "" {
			return fmt.Errorf("usage: tigwx config: an empty value; to take it away use -unset")
		}
		if strings.ContainsAny(value, "\r\n") {
			return errors.New("usage: tigwx config: a config value is one line")
		}
	}
	return c.configSet(k, *global, value)
}

// configKey is one key the config files can hold.
type configKey struct {
	name    string
	global  bool                            // it has a global value too
	project func(tig.Config) string         // what the project's holds
	put     func(*tig.Config, string)       // sets the project's
	ofUser  func(*tig.GlobalConfig) *string // the global value, when it has one
}

var configKeys = []configKey{
	{name: "user.name", global: true,
		project: func(c tig.Config) string { return c.UserName },
		put:     func(c *tig.Config, v string) { c.UserName = v },
		ofUser:  func(g *tig.GlobalConfig) *string { return &g.UserName }},
	{name: "user.by", global: true,
		project: func(c tig.Config) string { return c.UserBy },
		put:     func(c *tig.Config, v string) { c.UserBy = v },
		ofUser:  func(g *tig.GlobalConfig) *string { return &g.UserBy }},
	{name: "root",
		project: func(c tig.Config) string { return c.Root },
		put:     func(c *tig.Config, v string) { c.Root = v }},
	{name: "inferred",
		project: func(c tig.Config) string { return c.Inferred },
		put:     func(c *tig.Config, v string) { c.Inferred = v }},
	{name: "outside",
		project: func(c tig.Config) string { return strings.Join(c.Outside, ",") },
		put: func(c *tig.Config, v string) {
			c.Outside = nil
			for _, p := range strings.Split(v, ",") {
				if p = strings.TrimSpace(p); p != "" {
					c.Outside = append(c.Outside, p)
				}
			}
		}},
}

func configKeyOf(name string) (configKey, bool) {
	for _, k := range configKeys {
		if k.name == name {
			return k, true
		}
	}
	return configKey{}, false
}

func configKeyNames() []string {
	names := make([]string, len(configKeys))
	for i, k := range configKeys {
		names[i] = k.name
	}
	return names
}

// configFiles reads both files. A project is only there if one is found.
func (c *cli) configFiles() (tig.GlobalConfig, *tig.Config, error) {
	global, err := tig.ReadGlobalConfig()
	if err != nil {
		return global, nil, err
	}
	root, err := c.find()
	if err != nil || root == "" {
		return global, nil, err
	}
	project, err := tig.Open(root).Config()
	if err != nil {
		return global, nil, err
	}
	return global, &project, nil
}

// value is what a key holds, the project's ahead of the global one.
func (k configKey) value(global tig.GlobalConfig, project *tig.Config, onlyGlobal bool) string {
	if project != nil && !onlyGlobal {
		if v := k.project(*project); v != "" {
			return v
		}
	}
	if k.global {
		return *k.ofUser(&global)
	}
	return ""
}

func (c *cli) configGet(k configKey, onlyGlobal bool) error {
	global, project, err := c.configFiles()
	if err != nil {
		return err
	}
	v := k.value(global, project, onlyGlobal)
	if err := c.emit(map[string]string{"key": k.name, "value": v}, func() {
		if v != "" {
			fmt.Fprintln(c.out, v)
		}
	}); err != nil {
		return err
	}
	if v == "" {
		return errNo
	}
	return nil
}

func (c *cli) configList(onlyGlobal bool) error {
	global, project, err := c.configFiles()
	if err != nil {
		return err
	}
	values := map[string]string{}
	var lines []string
	for _, k := range configKeys {
		if v := k.value(global, project, onlyGlobal); v != "" {
			values[k.name] = v
			lines = append(lines, k.name+"="+v)
		}
	}
	return c.emit(values, func() {
		for _, line := range lines {
			fmt.Fprintln(c.out, line)
		}
	})
}

func (c *cli) configSet(k configKey, onlyGlobal bool, value string) error {
	if onlyGlobal {
		global, err := tig.ReadGlobalConfig()
		if err != nil {
			return err
		}
		*k.ofUser(&global) = value
		return tig.WriteGlobalConfig(global)
	}
	if err := c.open(); err != nil {
		return err
	}
	cfg, err := c.r.Config()
	if err != nil {
		return err
	}
	k.put(&cfg, value)
	return c.r.SetConfig(cfg)
}
