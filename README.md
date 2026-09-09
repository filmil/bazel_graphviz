# Graphviz, built with Bazel

[![Test](https://github.com/filmil/bazel_graphviz/actions/workflows/test.yml/badge.svg)](https://github.com/filmil/bazel_graphviz/actions/workflows/test.yml)
[![Publish the registry entry](https://github.com/filmil/bazel_graphviz/actions/workflows/publish.yml/badge.svg)](https://github.com/filmil/bazel_graphviz/actions/workflows/publish.yml)

A Bazel module for [graphviz][gv], in the shape a Bazel registry uses for a
third-party library whose upstream ships no Bazel files: an **overlay**.
The registry serves upstream's own release archive, and lays the build files
in `overlay/` on top of it.
This is how the Bazel Central Registry packages such projects; see, for
example, [nasm][nasm].
It is built with `rules_cc` alone, no `rules_foreign_cc`, which is what
[bazel-central-registry#4599][bcr4599] asks for.

The module is named `graphviz`.
Its version is `<upstream version>.bcr.<edition>`, so `14.0.0.bcr.2` is
graphviz 14.0.0 with the second edition of these build files.
A change to the build files alone bumps the edition; a new upstream release
resets it to 1.
The module declares `bazel_compatibility = [">=8.0.0"]`; this repository
itself builds with the Bazel in `.bazelversion`.

[gv]: https://gitlab.com/graphviz/graphviz
[nasm]: https://registry.bazel.build/modules/nasm
[bcr4599]: https://github.com/bazelbuild/bazel-central-registry/issues/4599

## Using it

```starlark
bazel_dep(name = "graphviz", version = "14.0.0.bcr.2")
```

Targets: `@graphviz//:graphviz` (the library, with the dot and neato layout
plugins linked in), `@graphviz//:dot`, and one binary per layout engine
(`neato`, `fdp`, `sfdp`, `circo`, `twopi`, `osage`, `patchwork`), each a
wrapper that passes `-K` to `dot`; `@graphviz//:all_layout_bins` is a
filegroup of all of them.
The module depends on `rules_cc`, `rules_shell` and `zlib`.
It registers no C++ toolchain; the consumer's applies.

## Layout of this repository

* `overlay/`: the build files, laid over the upstream archive by the
  registry. `MODULE.bazel` here is the module's; `test_module/` is the
  registry's test module for the entry, and consumes the overlay through
  `local_path_override`.
* `tests/`: the tests. The root module runs them against `@graphviz`, and
  the generator copies them into `test_module/`.
* `hosttest/`: a second root module that builds `@graphviz` with the
  platform's compiler and linker, which is what a consumer that registers no
  toolchain gets, and what the Bazel Central Registry's presubmit uses. The
  root module here builds with `hermetic_cc_toolchain`, whose linker resolves
  symbol cycles between graphviz's static libraries that GNU ld does not, so
  without this a link the registry rejects passes here. Run it with the
  registry order the comment in `hosttest/.bazelrc` gives.
* `upstream.json`: which upstream archive, and which edition of the overlay.
  `repository` lists the project and the prefix of its release archives:
  the BCR requires the archive URL to start with one of these.
* `tools/make_entry/`: the generator, a Go program, `//tools:make_entry`.
  `tools/presubmit.yml` is the presubmit that goes into the entry.
* `registry/`: the generated registry entry, committed.
  `.bazelrc` puts it first among the registries, so `bazel test //...`
  builds `@graphviz` exactly as a registry user would get it.
  Every version ever generated stays here, as in a registry.
* The root module is a development harness only: it registers the
  `hermetic_cc_toolchain` (zig) toolchain, builds the generator with
  `rules_go` (the Go toolchain is downloaded; `go.mod` names its version),
  and runs `buildifier`. `.bazelignore` hides `overlay/` and `registry/`
  from it, so the only way it reaches the overlay is through the registry.
* `.cache/`: where the generator keeps the downloaded upstream archive.
  Not committed.

## Maintenance

### A new upstream release, or a change to the build files

1. Edit `upstream.json`: the URL, `upstream_version` and `strip_prefix` for
   a new release, or `overlay_edition` for a change to the build files.
2. Put the same version in `overlay/MODULE.bazel` (the generator refuses to
   run otherwise), in `overlay/test_module/MODULE.bazel`, and in the root
   `MODULE.bazel`. The generator checks only the first.
3. `bazel run //tools:buildifier`, so that what goes into the entry is
   formatted. Formatting after generating leaves `source.json` with the
   hashes of the unformatted files.
4. `bazel run //tools:make_entry`, which downloads the archive once to
   compute its integrity and writes `registry/`.
5. `bazel test //...`.

CI runs the same, and more:

* `bazel run //tools:make_entry -- check` fails, naming the files, when
  `registry/` does not match `overlay/`, `tests/`, `tools/presubmit.yml`
  and `upstream.json`.
* `bazel test //...`.
* `hosttest`, the same module built with the platform's toolchain.
* The Bazel Central Registry's own `tools/bcr_validation.py`, from a
  checkout of the registry, against the entry: the archive's integrity,
  every overlay file's hash, `MODULE.bazel` against the overlay,
  `metadata.json`, and the shape of `presubmit.yml`. The tool's exit
  status 42 means every check passes and a BCR maintainer has to review
  the new `presubmit.yml`; every new version gets that, and CI accepts it.

### Publishing

`Publish the registry entry` in `.github/workflows/publish.yml`, run by hand
with the version.
It confirms `registry/` is current, copies the entry into a clone of
filmil/bazel-registry, merges our `metadata.json` into the registry's
(`bazel run //tools:make_entry -- merge-metadata OURS THEIRS`), adds the
overlay directory to the registry's `.bazelignore` (that registry builds
itself with Bazel, and must not load the overlay's BUILD files as its own
packages), and opens a pull request.
When asked with `bcr: true`, it does the same against the Bazel Central
Registry, through the fork filmil/bazel-central-registry; that pull request
is public, reviewed by the BCR maintainers, and cites
[bazel-central-registry#4599][bcr4599].
Both need the `BCR_PUBLISH_TOKEN` secret, a token that can push to the
fork and the registry and open pull requests.

Because the source archive is upstream's, there is no release of this
repository and no attestation of the archive.

### Formatting

`bazel run //tools:buildifier`.
