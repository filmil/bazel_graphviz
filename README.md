[![Test](https://github.com/filmil/bazel_graphviz_foreign/actions/workflows/test.yml/badge.svg)](https://github.com/filmil/bazel_graphviz_foreign/actions/workflows/test.yml)
[![Tag and Release](https://github.com/filmil/bazel_graphviz_foreign/actions/workflows/tag-and-release.yml/badge.svg)](https://github.com/filmil/bazel_graphviz_foreign/actions/workflows/tag-and-release.yml)
[![Publish to my Bazel registry](https://github.com/filmil/bazel_graphviz_foreign/actions/workflows/publish.yml/badge.svg)](https://github.com/filmil/bazel_graphviz_foreign/actions/workflows/publish.yml)
[![Publish on Bazel Central Registry](https://github.com/filmil/bazel_graphviz_foreign/actions/workflows/publish-bcr.yml/badge.svg)](https://github.com/filmil/bazel_graphviz_foreign/actions/workflows/publish-bcr.yml)

# Summary

This project is a [Bazel] module for [Graphviz]. It builds Graphviz 14.0.0
from upstream sources using native Bazel `cc_library` / `cc_binary` rules
(no `rules_foreign_cc`, no `./configure`, no `make`). The goal is a hermetic
build of Graphviz that participates in Bazel's incremental build graph and
remote-cache the same way the rest of your C/C++ code does.

## Scope

The build covers the core libraries (`cdt`, `cgraph`, `util`, `pathplan`,
`xdot`, `label`, `pack`, `common`, `gvc`, `dotgen`), the statically-linked
`plugin/core` (renders to `dot`, `json`, `svg`, `ps`, `fig`, `map`, `tk`,
`pov`, `pic`, `xdot`) and `plugin/dot_layout` (the dot layout engine), plus
the `dot` CLI. Layout engines other than `dot` (`neato`, `fdp`, `circo`,
`twopi`, `sfdp`, `osage`, `patchwork`) and graphical-output plugins (pango,
gd, xlib, kitty, etc.) are not built. Plugin loading via `libltdl` is
disabled — all plugins are statically registered via `lt_preloaded_symbols`.

[Bazel]: https://bazel.build/
[Graphviz]: https://graphviz.org/


# Bill-of-Material notices

You may notice that I have a few similar projects. This is my effort to provide
hermetic libraries for the upcoming Bazel modules world. Refer to
[standardization notes][stdn] for details.

[stdn]: https://hdlfactory.com/post/2025/09/29/getting-ready-for-the-brave-new-bazel-modules-world/

Other modules for the same library may be available. It is not my intention to
check for duplicated effort.

See [LICENSE](./LICENSE) for licensing information.

## Public targets

All targets below live in the root package of the `graphviz` module
(reference them as `@graphviz//:<name>` from a consuming module).

### Common entry points

| Target                 | Kind        | Use for                                  |
| ---------------------- | ----------- | ---------------------------------------- |
| `@graphviz//:graphviz` | `cc_library`| The library most downstream users want — every core lib plus `plugin/core` and `plugin/dot_layout` aggregated, and the static plugin registration table (`lt_preloaded_symbols`) for `gvContextPlugins`. |
| `@graphviz//:lib`      | `alias`     | Backwards-compatible alias for `:graphviz`. |
| `@graphviz//:dot`      | `cc_binary` | The `dot` CLI. Run-only consumers (e.g. `genrule` data deps) should prefer the filegroup below. |
| `@graphviz//:dot_bin`  | `filegroup` | Wraps `:dot` for use as `data = [...]` or in `$(location ...)` expansions. |

### Granular libraries

Useful if you only need part of the dependency graph — e.g. you want to
parse a `.dot` file but do your own layout. They build cleanly in
isolation and only pull in the deps they actually need.

| Target                          | What it contains                                  |
| ------------------------------- | ------------------------------------------------- |
| `@graphviz//:cdt`               | `lib/cdt` — container data types (dict, tree).    |
| `@graphviz//:util`              | `lib/util` — string, alloc, list utilities.       |
| `@graphviz//:cgraph`            | `lib/cgraph` — graph data structure + dot parser. |
| `@graphviz//:pathplan`          | `lib/pathplan` — shortest-path routing.           |
| `@graphviz//:xdot`              | `lib/xdot` — extended-dot drawing operations.     |
| `@graphviz//:label`             | `lib/label` — label placement.                    |
| `@graphviz//:pack`              | `lib/pack` — graph packing / connected-component split. |
| `@graphviz//:common`            | `lib/common` — layout-engine shared code.         |
| `@graphviz//:gvc`               | `lib/gvc` — graphviz context, the main API.       |
| `@graphviz//:dotgen`            | `lib/dotgen` — dot layout algorithm.              |
| `@graphviz//:plugin_core`       | `plugin/core` — output drivers (dot, json, svg, ps, fig, map, tk, pov, pic). |
| `@graphviz//:plugin_dot_layout` | `plugin/dot_layout` — the dot layout plugin wrapper. |

### Minimal usage example

```python
# MODULE.bazel
bazel_dep(name = "graphviz", version = "<latest>")
```

```python
# BUILD.bazel
cc_binary(
    name = "my_tool",
    srcs = ["my_tool.cc"],
    deps = ["@graphviz//:graphviz"],
)
```

```cpp
// my_tool.cc — see integration/test.cc for the full example.
#include <cgraph/cgraph.h>
#include <gvc/gvc.h>
#include <gvc/gvcext.h>  // declares lt_preloaded_symbols

int main() {
    GVC_t *gvc = gvContextPlugins(lt_preloaded_symbols, 0);
    // ... build a graph with agopen/agnode/agedge, then gvLayout +
    //     gvRenderData with "dot".
    gvFreeContext(gvc);
    return 0;
}
```

> Use `gvContextPlugins(lt_preloaded_symbols, 0)` rather than
> `gvContext()`. `gvContext()` passes `NULL` for builtins, so the
> statically-registered `dot_layout` and `core` plugins never get
> installed and layout/rendering both fail.

## Verify the build

```
bazel test //... && cd integration && bazel test //...
```

## Hermeticity

This build is entirely hermetic.


## Formatting

```
bazel run //tools:buildifier
```

## Release Registry

Refer to the [BCR][bcr] for the latest release.

[bcr]: https://registry.bazel.build/
