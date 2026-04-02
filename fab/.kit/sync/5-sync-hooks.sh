#!/usr/bin/env bash
set -euo pipefail
fab hook sync 2>/dev/null || echo "WARN: fab not found on PATH — skipping hook sync (install: brew install fab-kit)"
