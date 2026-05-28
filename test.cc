// Compile/link smoke test for the bazel-native graphviz library.
// Pulls in two of the major public headers so a missing header or a
// mis-wired include path turns into a build failure here.

#include <cgraph/cgraph.h>
#include <gvc/gvc.h>

int main() {
    return 0;
}
