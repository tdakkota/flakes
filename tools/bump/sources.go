package main

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"hash"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/go-faster/errors"
)

var httpClient = &http.Client{}

func httpGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "build request")
	}
	req.Header.Set("User-Agent", "flakes-bump/1")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, errors.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}
	return resp, nil
}

func httpGetString(ctx context.Context, url string) (string, error) {
	resp, err := httpGet(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "read body")
	}
	return strings.TrimSpace(string(b)), nil
}

// newHash returns a hash.Hash for the given Nix hash algorithm name.
func newHash(algo string) (hash.Hash, error) {
	switch algo {
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, errors.Errorf("unsupported hash algo %q", algo)
	}
}

// sriFromHex converts a hex digest into a Nix SRI hash string, e.g. "sha256-...=".
func sriFromHex(algo, hexDigest string) (string, error) {
	raw, err := hex.DecodeString(hexDigest)
	if err != nil {
		return "", errors.Wrap(err, "decode hex digest")
	}
	return algo + "-" + base64.StdEncoding.EncodeToString(raw), nil
}

// downloadAndHash streams url through algo, returning its hex digest. Used
// for upstreams (GitHub releases, x.ai, Google storage) that don't publish a
// checksum manifest, so the artifact must be hashed after the fact.
func downloadAndHash(ctx context.Context, url, algo string) (string, error) {
	resp, err := httpGet(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	h, err := newHash(algo)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", errors.Wrapf(err, "download %s", url)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// githubLatestTag returns the tag of the latest release of owner/repo, with
// trimPrefix removed (e.g. "v" or "rust-v").
func githubLatestTag(ctx context.Context, repo, trimPrefix string) (string, error) {
	url := "https://api.github.com/repos/" + repo + "/releases/latest"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", errors.Wrap(err, "build request")
	}
	req.Header.Set("User-Agent", "flakes-bump/1")
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", url)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}

	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", errors.Wrap(err, "decode response")
	}
	return strings.TrimPrefix(body.TagName, trimPrefix), nil
}

// compareVersions compares dot-separated numeric versions (e.g. "1.22209.3"),
// returning -1, 0, or 1. Non-numeric components compare as 0.
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var av, bv int
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}

// aptStanza is one paragraph of a Debian Packages index.
type aptStanza map[string]string

func fetchAptPackages(ctx context.Context, indexURL string) ([]aptStanza, error) {
	body, err := httpGetString(ctx, indexURL)
	if err != nil {
		return nil, err
	}

	var stanzas []aptStanza
	cur := aptStanza{}
	for line := range strings.SplitSeq(body, "\n") {
		if strings.TrimSpace(line) == "" {
			if len(cur) > 0 {
				stanzas = append(stanzas, cur)
				cur = aptStanza{}
			}
			continue
		}
		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			continue // continuation line (e.g. multi-line Description); irrelevant fields we read are single-line
		}
		cur[key] = value
	}
	if len(cur) > 0 {
		stanzas = append(stanzas, cur)
	}
	return stanzas, nil
}

// aptLatest returns the highest-versioned stanza for pkgName in an apt Packages index.
func aptLatest(ctx context.Context, indexURL, pkgName string) (aptStanza, error) {
	stanzas, err := fetchAptPackages(ctx, indexURL)
	if err != nil {
		return nil, err
	}

	var best aptStanza
	for _, s := range stanzas {
		if s["Package"] != pkgName {
			continue
		}
		if best == nil || compareVersions(s["Version"], best["Version"]) > 0 {
			best = s
		}
	}
	if best == nil {
		return nil, errors.Errorf("package %q not found in %s", pkgName, indexURL)
	}
	return best, nil
}

// protonRelease is one entry of Proton's PassDesktop version.json feed.
type protonRelease struct {
	CategoryName string `json:"CategoryName"`
	Version      string `json:"Version"`
	File         []struct {
		URL            string `json:"Url"`
		Sha512CheckSum string `json:"Sha512CheckSum"`
	} `json:"File"`
}

// protonLatestStable returns the highest-versioned "Stable" release from a
// Proton desktop app's version.json feed.
func protonLatestStable(ctx context.Context, versionJSONURL string) (protonRelease, error) {
	body, err := httpGet(ctx, versionJSONURL)
	if err != nil {
		return protonRelease{}, err
	}
	defer body.Body.Close()

	var feed struct {
		Releases []protonRelease `json:"Releases"`
	}
	if err := json.NewDecoder(body.Body).Decode(&feed); err != nil {
		return protonRelease{}, errors.Wrap(err, "decode version.json")
	}

	var best protonRelease
	for _, r := range feed.Releases {
		if r.CategoryName != "Stable" {
			continue
		}
		if best.Version == "" || compareVersions(r.Version, best.Version) > 0 {
			best = r
		}
	}
	if best.Version == "" {
		return protonRelease{}, errors.Errorf("no Stable release found in %s", versionJSONURL)
	}
	return best, nil
}

// fetchZipNarHash shells out to `nix store prefetch-file --unpack` to get the
// NAR hash of a zip's unpacked contents - see zipArtifact in registry.go for
// why this can't be done in pure Go.
func fetchZipNarHash(ctx context.Context, url string) (string, error) {
	cmd := exec.CommandContext(ctx, "nix", "store", "prefetch-file", "--unpack", "--json", url)
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "nix store prefetch-file --unpack %s", url)
	}

	var res struct {
		Hash string `json:"hash"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return "", errors.Wrap(err, "parse nix store prefetch-file output")
	}
	return res.Hash, nil
}

// sortedSystems returns m's keys sorted for deterministic iteration/output.
func sortedSystems[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
