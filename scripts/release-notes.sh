#!/usr/bin/env bash
#
# Emit the release notes for the latest CHANGELOG.md entry on stdout.
#
# The changelog lists releases as level-2 headings ("## 2.18.1 (...)") with the
# newest first, under a "# Changelog" title. We select every version block
# (headings that start with a digit, which excludes the title), keep only the
# first one's body, and drop its own version heading so the notes start at the
# content. The result is suitable for `goreleaser release --release-notes`.
set -euo pipefail

# mdq's heading selector is depth-agnostic and matches on text: "# /^[0-9]/"
# picks every version heading (title "Changelog" is excluded) and renders it at
# its original level (## for versions, ### for sections). awk then keeps only
# the first block's body: it counts the rendered "## <digit>" version headings
# (so a "## foo" section or code line is not mistaken for a boundary) and prints
# lines while inside the first one. sed trims the leading blank line left by the
# stripped heading.
mdq --no-br '# /^[0-9]/' CHANGELOG.md \
  | awk '/^## [0-9]/{ c++; next } c==1' \
  | sed -e '/./,$!d'
