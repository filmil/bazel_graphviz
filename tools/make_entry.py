#!/usr/bin/env python3
"""Writes the registry entry for this module from overlay/ and upstream.json.

overlay/test_module/ holds only its MODULE.bazel and BUILD.bazel; the test
sources are copied in from tests/, where the root module also runs them.

The entry is what a Bazel registry holds for one version of an overlay
module: MODULE.bazel, source.json, presubmit.yml and the overlay files, under
modules/<name>/<version>/, plus modules/<name>/metadata.json. It is written
into registry/ in this repository, where the root module consumes it through
`--registry=file://...`, and from where a publish copies it into a registry.

The version is <upstream version>.bcr.<overlay edition>: the registry's
convention for a module whose build files are not upstream's.

Usage: make_entry.py [--check]
  --check   exit 1 if registry/ differs from what would be written.
"""
import base64
import hashlib
import json
import os
import shutil
import sys
import urllib.request

# Under `bazel run` this file lives in a runfiles tree; the workspace is where
# Bazel says it is.
ROOT = os.environ.get("BUILD_WORKSPACE_DIRECTORY") or os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
HERE = os.path.join(ROOT, "tools")


def sri(data):
    return "sha256-" + base64.b64encode(hashlib.sha256(data).digest()).decode()


def archive_integrity(url, cache):
    """SRI of the upstream archive, downloaded once and kept in cache."""
    if not os.path.exists(cache):
        os.makedirs(os.path.dirname(cache), exist_ok=True)
        with urllib.request.urlopen(url) as r, open(cache, "wb") as f:
            shutil.copyfileobj(r, f)
    with open(cache, "rb") as f:
        return sri(f.read())


def overlay_files(overlay_dir):
    out = {}
    for dirpath, _, files in os.walk(overlay_dir):
        for fn in files:
            full = os.path.join(dirpath, fn)
            with open(full, "rb") as f:
                out[os.path.relpath(full, overlay_dir)] = sri(f.read())
    return dict(sorted(out.items()))


def main():
    check = "--check" in sys.argv[1:]
    up = json.load(open(os.path.join(ROOT, "upstream.json")))
    version = "{}.bcr.{}".format(up["upstream_version"], up["overlay_edition"])
    overlay_dir = os.path.join(ROOT, "overlay")
    module_bazel = open(os.path.join(overlay_dir, "MODULE.bazel")).read()
    declared = 'version = "{}"'.format(version)
    if declared not in module_bazel:
        sys.exit("overlay/MODULE.bazel must declare {} (from upstream.json)".format(declared))

    entry = os.path.join(ROOT, "registry", "modules", up["name"], version)
    staged = entry + ".new"
    shutil.rmtree(staged, ignore_errors=True)
    shutil.copytree(overlay_dir, os.path.join(staged, "overlay"))
    # The test module's sources are the same tests the root module runs; they
    # live once, in tests/, and are copied in here.
    for fn in ("test.cc", "run_dot.sh"):
        shutil.copy(os.path.join(ROOT, "tests", fn), os.path.join(staged, "overlay", "test_module", fn))
    shutil.copy(os.path.join(overlay_dir, "MODULE.bazel"), os.path.join(staged, "MODULE.bazel"))
    shutil.copy(os.path.join(HERE, "presubmit.yml"), os.path.join(staged, "presubmit.yml"))
    cache = os.path.join(ROOT, ".cache", os.path.basename(up["url"]))
    source = {
        "url": up["url"],
        "integrity": archive_integrity(up["url"], cache),
        "strip_prefix": up["strip_prefix"],
        "overlay": overlay_files(os.path.join(staged, "overlay")),
    }
    with open(os.path.join(staged, "source.json"), "w") as f:
        json.dump(source, f, indent=4)
        f.write("\n")

    meta_path = os.path.join(ROOT, "registry", "modules", up["name"], "metadata.json")
    versions = []
    if os.path.exists(meta_path):
        versions = json.load(open(meta_path)).get("versions", [])
    if version not in versions:
        versions.append(version)
    metadata = {
        "homepage": up["homepage"],
        "maintainers": up["maintainers"],
        "repository": up["repository"],
        "versions": versions,
        "yanked_versions": {},
    }

    if check:
        problems = _differences(entry, staged)
        if not os.path.exists(meta_path) or json.load(open(meta_path)) != metadata:
            problems.append("metadata.json differs")
        shutil.rmtree(staged)
        if problems:
            for p in problems:
                print("  " + p, file=sys.stderr)
            sys.exit("registry/ is out of date; run: bazel run //tools:make_entry")
        print("registry/ is up to date")
        return
    shutil.rmtree(entry, ignore_errors=True)
    os.rename(staged, entry)
    os.makedirs(os.path.dirname(meta_path), exist_ok=True)
    with open(meta_path, "w") as f:
        json.dump(metadata, f, indent=4)
        f.write("\n")
    print("wrote", os.path.relpath(entry, ROOT))


def _differences(committed, staged):
    """Names every file that differs between the committed entry and a fresh one."""
    def walk(d):
        out = {}
        if not os.path.isdir(d):
            return out
        for dirpath, _, files in os.walk(d):
            for fn in files:
                full = os.path.join(dirpath, fn)
                with open(full, "rb") as f:
                    out[os.path.relpath(full, d)] = f.read()
        return out
    a, b = walk(committed), walk(staged)
    out = []
    for name in sorted(set(a) | set(b)):
        if name not in a:
            out.append("missing from registry/: " + name)
        elif name not in b:
            out.append("no longer generated: " + name)
        elif a[name] != b[name]:
            out.append("differs: " + name)
    return out


if __name__ == "__main__":
    main()
