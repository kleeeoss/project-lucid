#!/usr/bin/env bash
set -eo pipefail

# Ensure execution starts in the mounted workspace
cd /workspace

# If test command provided, execute it; otherwise default to sh
if [ "$#" -gt 0 ]; then
    exec "$@"
else
    exec /bin/sh
fi
