// config.go
package main

import (
	_ "embed"
	"fmt"
	"os"
	"sort"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

//go:generate cue exp gengotypes ./...

// schemaCUE generates cue_types_main_gen.go via `go generate ./...`.
//
//go:embed schema.cue
var schemaCUE []byte

// LoadConfig reads and validates a trayicons.cue file against the embedded schema.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	ctx := cuecontext.New()
	schema := ctx.CompileBytes(schemaCUE)
	if err := schema.Err(); err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	def := schema.LookupPath(cue.ParsePath("#Config"))

	user := ctx.CompileBytes(data)
	if err := user.Err(); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	merged := def.Unify(user)
	if err := merged.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("validate %s: %w", path, err)
	}

	var cfg Config
	if err := merged.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := cfg.check(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &cfg, nil
}

// check enforces cross-field rules the schema cannot express.
func (c *Config) check() error {
	declared := map[string]bool{}
	for _, s := range c.Slots {
		declared[s] = true
	}
	for stateName, state := range c.States {
		for slot := range state.Slots {
			if !declared[slot] {
				return fmt.Errorf("state %q uses undeclared slot %q", stateName, slot)
			}
		}
	}
	for themeName, theme := range c.Themes {
		for stateName, slots := range theme.Colors {
			if _, ok := c.States[stateName]; !ok {
				return fmt.Errorf("theme %q colors reference unknown state %q", themeName, stateName)
			}
			for slot := range slots {
				if !declared[slot] {
					return fmt.Errorf("theme %q colors use undeclared slot %q", themeName, slot)
				}
			}
		}
		for stateName := range theme.StateSources {
			if _, ok := c.States[stateName]; !ok {
				return fmt.Errorf("theme %q stateSources reference unknown state %q", themeName, stateName)
			}
		}
	}
	if len(c.States) == 0 {
		return fmt.Errorf("at least one state is required")
	}
	if len(c.Sizes) == 0 {
		return fmt.Errorf("at least one size is required")
	}
	return nil
}

func (c *Config) StateNames() []string {
	names := make([]string, 0, len(c.States))
	for name := range c.States {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c *Config) ThemeNames() []string {
	names := make([]string, 0, len(c.Themes))
	for name := range c.Themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
