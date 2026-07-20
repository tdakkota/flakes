// Command bump reports (and, with -write, applies) version/hash updates for
// the CLI tools pinned in ../../flake.nix, by checking each package's own
// upstream release feed (GitHub releases, an apt Packages index, Proton's
// version.json, ...) instead of hand-tracking releases.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/go-faster/errors"
)

type versionOverrides map[string]string

func (o versionOverrides) String() string { return "" }

func (o versionOverrides) Set(s string) error {
	name, version, ok := strings.Cut(s, "=")
	if !ok {
		return errors.Errorf("expected name=version, got %q", s)
	}
	o[name] = version
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	flakePath := flag.String("flake", "../../flake.nix", "path to flake.nix")
	write := flag.Bool("write", false, "apply updates to flake.nix instead of just reporting them")
	only := flag.String("only", "", "comma-separated list of package names to consider (default: all)")
	overrides := make(versionOverrides)
	flag.Var(overrides, "set", "force a package to a specific version, e.g. -set grok=0.3.0 (repeatable)")
	flag.Parse()

	var wanted map[string]bool
	if *only != "" {
		wanted = make(map[string]bool)
		for name := range strings.SplitSeq(*only, ",") {
			wanted[strings.TrimSpace(name)] = true
		}
	}

	src, err := os.ReadFile(*flakePath)
	if err != nil {
		return errors.Wrap(err, "read flake.nix")
	}
	text := string(src)

	ctx := context.Background()
	anyChanged := false
	hadError := false

	for _, group := range registry() {
		if wanted != nil && !wanted[group.name] {
			continue
		}

		current, err := getVersion(text, group.nixVersionVar)
		if err != nil {
			fmt.Printf("ERROR %-15s %v\n", group.name, err)
			hadError = true
			continue
		}

		target := overrides[group.name]
		if target == "" {
			if group.fetchLatest == nil {
				fmt.Printf("MANUAL %-14s %s (no known version feed; bump with -set %s=X.Y.Z)\n", group.name, current, group.name)
				continue
			}
			target, err = group.fetchLatest(ctx)
			if err != nil {
				fmt.Printf("ERROR %-15s checking latest version: %v\n", group.name, err)
				hadError = true
				continue
			}
		}

		if target == current {
			fmt.Printf("OK    %-15s %s\n", group.name, current)
			continue
		}

		fmt.Printf("BUMP  %-15s %s -> %s\n", group.name, current, target)
		if !*write {
			continue
		}

		newText, err := applyBump(ctx, text, group, target)
		if err != nil {
			fmt.Printf("ERROR %-15s applying update: %v\n", group.name, err)
			hadError = true
			continue
		}
		text = newText
		anyChanged = true
	}

	if anyChanged {
		if err := os.WriteFile(*flakePath, []byte(text), 0o644); err != nil {
			return errors.Wrap(err, "write flake.nix")
		}
		fmt.Println("wrote", *flakePath)
	}

	if hadError {
		return errors.New("one or more packages failed; see above")
	}
	return nil
}

// applyBump fetches every system's artifact for group at the new version and
// rewrites its hash + version fields in text, returning the updated text.
// It touches nothing on error, so a partial fetch failure can't leave a
// package's tables at inconsistent versions.
func applyBump(ctx context.Context, text string, group versionGroup, target string) (string, error) {
	type patch struct {
		tableVar, system, hash string
	}
	var patches []patch

	for _, table := range group.tables {
		for _, system := range sortedSystems(table.systems) {
			fmt.Printf("      %-15s fetching %s (%s)...\n", group.name, system, table.nixVar)
			art, err := table.systems[system](ctx, target)
			if err != nil {
				return "", errors.Wrapf(err, "%s/%s", table.nixVar, system)
			}
			patches = append(patches, patch{table.nixVar, system, art.sri})
		}
	}

	out := text
	for _, p := range patches {
		var err error
		out, err = setHashForSystem(out, p.tableVar, p.system, p.hash)
		if err != nil {
			return "", err
		}
	}
	out, err := setVersion(out, group.nixVersionVar, target)
	if err != nil {
		return "", err
	}
	return out, nil
}
