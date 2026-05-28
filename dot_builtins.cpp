// Builtin plugin registration for the bazel-native graphviz build.
//
// Upstream's cmd/dot/dot_builtins.cpp references plugins that are not
// part of this minimal build (neato_layout, kitty, vt, pango/gd/quartz).
// Statically register only the plugins we actually compile.

#include <gvc/gvplugin.h>

extern "C" {

extern gvplugin_library_t gvplugin_dot_layout_LTX_library;
extern gvplugin_library_t gvplugin_core_LTX_library;

lt_symlist_t lt_preloaded_symbols[] = {
    {"gvplugin_dot_layout_LTX_library", &gvplugin_dot_layout_LTX_library},
    {"gvplugin_core_LTX_library", &gvplugin_core_LTX_library},
    {0, 0},
};

}
