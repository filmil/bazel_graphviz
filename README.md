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
