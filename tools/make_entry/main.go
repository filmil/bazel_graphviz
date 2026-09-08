// make_entry writes the registry entry for this module from overlay/,
// tests/ and upstream.json, checks that the committed entry is current, and
// merges the module's metadata.json into a registry's.
//
// The entry is what a Bazel registry holds for one version of an overlay
// module: MODULE.bazel, source.json, presubmit.yml and the overlay files,
// under modules/<name>/<version>/, plus modules/<name>/metadata.json. It is
// written into registry/ in this repository, where the root module consumes
// it through --registry=file://..., and from where a publish copies it into a
// registry.
//
// The version is <upstream version>.bcr.<overlay edition>: the registry's
// convention for a module whose build files are not upstream's.
//
//	bazel run //tools:make_entry                       write registry/
//	bazel run //tools:make_entry -- check              exit 1 if registry/ is stale
//	bazel run //tools:make_entry -- merge-metadata OURS THEIRS
//	                                                   add our versions to a
//	                                                   registry's metadata.json
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// upstream mirrors upstream.json.
type upstream struct {
	Name            string            `json:"name"`
	UpstreamVersion string            `json:"upstream_version"`
	OverlayEdition  int               `json:"overlay_edition"`
	URL             string            `json:"url"`
	StripPrefix     string            `json:"strip_prefix"`
	Homepage        string            `json:"homepage"`
	Repository      []string          `json:"repository"`
	Maintainers     []json.RawMessage `json:"maintainers"`
}

// metadata mirrors a registry's modules/<name>/metadata.json.
type metadata struct {
	Homepage       string            `json:"homepage"`
	Maintainers    []json.RawMessage `json:"maintainers"`
	Repository     []string          `json:"repository"`
	Versions       []string          `json:"versions"`
	YankedVersions map[string]string `json:"yanked_versions"`
}

// The test sources live once, in tests/, and are copied into the entry's
// test module.
var testSources = []string{"test.cc", "run_dot.sh"}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	root := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	if root == "" {
		return fmt.Errorf("run this through bazel run, which sets BUILD_WORKSPACE_DIRECTORY")
	}
	cmd := "generate"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "generate":
		return generate(root, false)
	case "check", "--check":
		return generate(root, true)
	case "merge-metadata":
		if len(args) != 3 {
			return fmt.Errorf("usage: merge-metadata OURS THEIRS")
		}
		return mergeMetadata(args[1], args[2])
	}
	return fmt.Errorf("unknown command %q", cmd)
}

func generate(root string, check bool) error {
	var up upstream
	if err := readJSON(filepath.Join(root, "upstream.json"), &up); err != nil {
		return err
	}
	version := fmt.Sprintf("%s.bcr.%d", up.UpstreamVersion, up.OverlayEdition)
	overlayDir := filepath.Join(root, "overlay")
	moduleBazel, err := os.ReadFile(filepath.Join(overlayDir, "MODULE.bazel"))
	if err != nil {
		return err
	}
	if declared := fmt.Sprintf("version = %q", version); !bytes.Contains(moduleBazel, []byte(declared)) {
		return fmt.Errorf("overlay/MODULE.bazel must declare %s (from upstream.json)", declared)
	}

	// Assemble the entry in memory: path relative to the entry -> content.
	entry := map[string][]byte{}
	overlay, err := readTree(overlayDir)
	if err != nil {
		return err
	}
	for name, data := range overlay {
		entry[filepath.Join("overlay", name)] = data
	}
	for _, name := range testSources {
		data, err := os.ReadFile(filepath.Join(root, "tests", name))
		if err != nil {
			return err
		}
		entry[filepath.Join("overlay", "test_module", name)] = data
	}
	entry["MODULE.bazel"] = moduleBazel
	presubmit, err := os.ReadFile(filepath.Join(root, "tools", "presubmit.yml"))
	if err != nil {
		return err
	}
	entry["presubmit.yml"] = presubmit

	integrity, err := archiveIntegrity(up.URL, filepath.Join(root, ".cache", filepath.Base(up.URL)))
	if err != nil {
		return err
	}
	overlayHashes := map[string]string{}
	for name, data := range entry {
		if strings.HasPrefix(name, "overlay/") {
			overlayHashes[strings.TrimPrefix(name, "overlay/")] = sri(data)
		}
	}
	source, err := marshal(map[string]any{
		"url":          up.URL,
		"integrity":    integrity,
		"strip_prefix": up.StripPrefix,
		"overlay":      overlayHashes,
	})
	if err != nil {
		return err
	}
	entry["source.json"] = source

	moduleDir := filepath.Join(root, "registry", "modules", up.Name)
	entryDir := filepath.Join(moduleDir, version)
	metaPath := filepath.Join(moduleDir, "metadata.json")
	meta := metadata{YankedVersions: map[string]string{}}
	if _, err := os.Stat(metaPath); err == nil {
		if err := readJSON(metaPath, &meta); err != nil {
			return err
		}
	}
	if !contains(meta.Versions, version) {
		meta.Versions = append(meta.Versions, version)
	}
	meta.Homepage, meta.Maintainers, meta.Repository = up.Homepage, up.Maintainers, up.Repository
	if meta.YankedVersions == nil {
		meta.YankedVersions = map[string]string{}
	}
	metaJSON, err := marshal(meta)
	if err != nil {
		return err
	}

	if check {
		problems := differences(entryDir, entry)
		if committed, err := os.ReadFile(metaPath); err != nil || !bytes.Equal(committed, metaJSON) {
			problems = append(problems, "metadata.json differs")
		}
		if len(problems) > 0 {
			for _, p := range problems {
				fmt.Fprintln(os.Stderr, "  "+p)
			}
			return fmt.Errorf("registry/ is out of date; run: bazel run //tools:make_entry")
		}
		fmt.Println("registry/ is up to date")
		return nil
	}
	if err := os.RemoveAll(entryDir); err != nil {
		return err
	}
	for name, data := range entry {
		path := filepath.Join(entryDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(metaPath, metaJSON, 0o644); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, entryDir)
	fmt.Println("wrote", rel)
	return nil
}

// mergeMetadata adds our versions, and our homepage, maintainers and
// repository, to a registry's metadata.json, creating it when absent.
func mergeMetadata(oursPath, theirsPath string) error {
	var ours, theirs metadata
	if err := readJSON(oursPath, &ours); err != nil {
		return err
	}
	if _, err := os.Stat(theirsPath); err == nil {
		if err := readJSON(theirsPath, &theirs); err != nil {
			return err
		}
	}
	for _, v := range ours.Versions {
		if !contains(theirs.Versions, v) {
			theirs.Versions = append(theirs.Versions, v)
		}
	}
	theirs.Homepage, theirs.Maintainers, theirs.Repository = ours.Homepage, ours.Maintainers, ours.Repository
	if theirs.YankedVersions == nil {
		theirs.YankedVersions = map[string]string{}
	}
	data, err := marshal(theirs)
	if err != nil {
		return err
	}
	return os.WriteFile(theirsPath, data, 0o644)
}

// archiveIntegrity is the SRI of the upstream archive, downloaded once and
// kept at cache.
func archiveIntegrity(url, cache string) (string, error) {
	if _, err := os.Stat(cache); err != nil {
		if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
			return "", err
		}
		resp, err := http.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("GET %s: %s", url, resp.Status)
		}
		f, err := os.Create(cache)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(f, resp.Body); err != nil {
			f.Close()
			return "", err
		}
		if err := f.Close(); err != nil {
			return "", err
		}
	}
	data, err := os.ReadFile(cache)
	if err != nil {
		return "", err
	}
	return sri(data), nil
}

func sri(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256-" + base64.StdEncoding.EncodeToString(sum[:])
}

// readTree maps every file under dir, by path relative to dir, to its content.
func readTree(dir string) (map[string][]byte, error) {
	out := map[string][]byte{}
	if _, err := os.Stat(dir); err != nil {
		return out, nil
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		out[rel] = data
		return nil
	})
	return out, err
}

// differences names every file that differs between the committed entry
// and a fresh one.
func differences(committedDir string, fresh map[string][]byte) []string {
	committed, err := readTree(committedDir)
	if err != nil {
		return []string{err.Error()}
	}
	names := map[string]bool{}
	for n := range committed {
		names[n] = true
	}
	for n := range fresh {
		names[n] = true
	}
	var sorted []string
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	var out []string
	for _, n := range sorted {
		a, inA := committed[n]
		b, inB := fresh[n]
		switch {
		case !inA:
			out = append(out, "missing from registry/: "+n)
		case !inB:
			out = append(out, "no longer generated: "+n)
		case !bytes.Equal(a, b):
			out = append(out, "differs: "+n)
		}
	}
	return out
}

// marshal writes JSON the way the registries' tooling does: four-space
// indent, sorted keys, a trailing newline.
func marshal(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
