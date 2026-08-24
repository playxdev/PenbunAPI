#!/usr/bin/env sh
set -eu

# Data descriptors and database access must remain usable without an HTTP server.
for dir in internal/repository internal/resources internal/schema; do
  if rg -n 'github\.com/gofiber/fiber' "$dir"; then
    echo "layering violation: $dir must not import Fiber" >&2
    exit 1
  fi
done

# The generic schema package may return application errors, but it must not know
# about concrete domain or CRUD packages.
if rg -n '"penbun/api/internal/(crud|domain|repository|resources)' internal/schema; then
  echo 'layering violation: schema may not depend on higher layers' >&2
  exit 1
fi
