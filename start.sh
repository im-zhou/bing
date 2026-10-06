#!/bin/sh
set -eu
cd "$(dirname "$0")"
exec ./bing -config "${BING_CONFIG:-config.yaml}"
