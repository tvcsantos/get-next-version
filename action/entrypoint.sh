#!/usr/bin/env sh

set -e

/action/get-next-version \
  --repository /github/workspace \
  --target github-action \
  --prefix "$INPUT_PREFIX" \
  --feature-prefixes "$INPUT_FEATURE_PREFIXES" \
  --fix-prefixes "$INPUT_FIX_PREFIXES" \
  --chore-prefixes "$INPUT_CHORE_PREFIXES" \
  --tags-filter-regex "$INPUT_TAGS_FILTER_REGEX" \
  --commits-filter-path-regex "$INPUT_COMMITS_FILTER_PATH_REGEX" \
  --version-regex "$INPUT_VERSION_REGEX"
