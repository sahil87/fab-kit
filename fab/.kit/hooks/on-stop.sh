#!/usr/bin/env bash
ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
fab hook stop 2>/dev/null; exit 0
