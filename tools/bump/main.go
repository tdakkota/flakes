// Command bump reports (and, with -write, applies) version/hash updates for
// the CLI tools pinned in ../../versions.nix, by checking each package's own
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
	versionsPath := flag.String("versions", "../../versions.nix", "path to versions.nix")
	write := flag.Bool("write", false, "apply updates to versions.nix instead of just reporting them")
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

	ctx := context.Background()

	v, err := readVersions(ctx, *versionsPath)
	if err != nil {
		return errors.Wrap(err, "read versions.nix")
	}

	anyChanged := false
	hadError := false

	for _, group := range registry() {
		if wanted != nil && !wanted[group.name] {
			continue
		}

		primary, ok := v[group.targets[0].nixKey]
		if !ok {
			fmt.Printf("ERROR %-15s %q not found in versions.nix\n", group.name, group.targets[0].nixKey)
			hadError = true
			continue
		}
		current := primary.Version

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

		if err := applyBump(ctx, v, group, target); err != nil {
			fmt.Printf("ERROR %-15s applying update: %v\n", group.name, err)
			hadError = true
			continue
		}
		anyChanged = true
	}

	if anyChanged {
		if err := writeVersions(ctx, *versionsPath, v); err != nil {
			return errors.Wrap(err, "write versions.nix")
		}
		fmt.Println("wrote", *versionsPath)
	}

	if hadError {
		return errors.New("one or more packages failed; see above")
	}
	return nil
}

// applyBump fetches every system's artifact for every target in group at the
// new version, then writes the new version and hashes into v. It mutates
// nothing on error, so a partial fetch failure can't leave a package's
// entries at inconsistent versions.
func applyBump(ctx context.Context, v versions, group versionGroup, newVersion string) error {
	type patch struct {
		nixKey, system, hash string
	}
	var patches []patch

	for _, t := range group.targets {
		pkg, ok := v[t.nixKey]
		if !ok {
			return errors.Errorf("%q not found in versions.nix", t.nixKey)
		}
		for _, system := range sortedSystems(pkg.Artifacts) {
			fmt.Printf("      %-15s fetching %s (%s)...\n", group.name, system, t.nixKey)
			art, err := t.fetch(ctx, system, pkg.Artifacts[system], newVersion)
			if err != nil {
				return errors.Wrapf(err, "%s/%s", t.nixKey, system)
			}
			patches = append(patches, patch{t.nixKey, system, art.sri})
		}
	}

	for _, p := range patches {
		v[p.nixKey].Artifacts[p.system]["hash"] = p.hash
	}
	for _, t := range group.targets {
		pkg := v[t.nixKey]
		pkg.Version = newVersion
		v[t.nixKey] = pkg
	}
	return nil
}
