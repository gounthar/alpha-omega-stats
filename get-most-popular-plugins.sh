#!/bin/bash

jq -r '
  .plugins
  | to_entries
  | map(select(.value | type == "object" and has("popularity")))
  | map({name: .key, popularity: .value.popularity})
  | sort_by(-.popularity)[:250]
  | "name,popularity",
    (.[] | "\(.name),\(.popularity)")
' plugins.json > top-250-plugins.csv