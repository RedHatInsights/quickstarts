#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname -- "$(realpath -- "$BASH_SOURCE")")"
source ./cli_common.sh

update_directory

name="$(read_name)"
display_name="$(read_display_name)"
selected_type="$(read_type)"

echo "Selected type: $selected_type"

if [[ "$selected_type" = "$type_quickstart" ]]; then
	exec ./make_quickstart.sh "$name" "$display_name"
else
	exec ./make_generic.sh "$name" "$display_name" "$selected_type"
fi
