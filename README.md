# Graphviz, built with Bazel

[![Test](https://github.com/filmil/bazel_graphviz/actions/workflows/test.yml/badge.svg)](https://github.com/filmil/bazel_graphviz/actions/workflows/test.yml)
[![Publish the registry entry](https://github.com/filmil/bazel_graphviz/actions/workflows/publish.yml/badge.svg)](https://github.com/filmil/bazel_graphviz/actions/workflows/publish.yml)

A Bazel module for [graphviz][gv], in the shape a Bazel registry uses for a
third-party library whose upstream ships no Bazel files: an **overlay**.
The registry serves upstream's own release archive, and lays the build files
in `overlay/` on top of it.
This is how the Bazel Central Registry packages such projects; see, for
example, [nasm][nasm].

The module is named `graphviz`.
Its version is `<upstream version>.bcr.<edition>`, so `14.0.0.bcr.1` is
graphviz 14.0.0 with the first edition of these build files.
A change to the build files alone bumps the edition; a new upstream release
resets it to 1.

[gv]: https://gitlab.com/graphviz/graphviz
[nasm]: https://registry.bazel.build/modules/nasm

## Using it

```starlark
bazel_dep(name = "graphviz", version = "14.0.0.bcr.1")
```

Targets: `@graphviz//:graphviz` (the library, with the dot and neato layout
plugins linked in), `@graphviz//:dot`, and one binary per layout engine
(`neato`, `fdp`, `sfdp`, `circo`, `twopi`, `osage`, `patchwork`), each a
wrapper that passes `-K` to `dot`.
The module registers no C++ toolchain; the consumer's applies.

## Layout of this repository

* `overlay/`: the build files, laid over the upstream archive by the
  registry. `MODULE.bazel` here is the module's; `test_module/` is the
  registry's test module for the entry.
* `tests/`: the tests. The root module runs them against `@graphviz`, and
  the generator copies them into `test_module/`.
* `upstream.json`: which upstream archive, and which edition of the overlay.
* `tools/make_entry.py`, `tools/presubmit.yml`: the generator and the
  presubmit that goes into the entry.
* `registry/`: the generated registry entry, committed.
  `.bazelrc` puts it first among the registries, so `bazel test //...`
  builds `@graphviz` exactly as a registry user would get it.

## Maintenance

### A new upstream release, or a change to the build files

1. Edit `upstream.json`: the URL, `upstream_version` and `strip_prefix` for
   a new release, or `overlay_edition` for a change to the build files.
2. Put the same version in `overlay/MODULE.bazel`.
3. `bazel run //tools:buildifier`, so that what goes into the entry is
   formatted. Formatting after generating leaves `source.json` with the
   hashes of the unformatted files.
4. `bazel run //tools:make_entry`, which downloads the archive once to
   compute its integrity and writes `registry/`.
5. `bazel test //...`.

CI runs `bazel run //tools:make_entry -- --check` and fails when `registry/`
does not match its sources.

### Publishing

`Publish the registry entry` in `.github/workflows/publish.yml`, run by hand
with the version.
It opens a pull request against filmil/bazel-registry, and against the Bazel
Central Registry as well when asked; that one is public and reviewed by the
BCR maintainers.
Because the source archive is upstream's, there is no release of this
repository and no attestation of the archive.

### Formatting

`bazel run //tools:buildifier`.
