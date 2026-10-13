#!/usr/bin/env bash
set -euo pipefail
test -s README.md
git diff --check
