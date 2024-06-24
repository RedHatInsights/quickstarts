#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname -- "$(realpath -- "$BASH_SOURCE")")"
source ./cli/cli_common.sh

update_directory

selected_type="$(read_type)"

echo "Selected type: $selected_type"

if [[ "$selected_type" = "$type_help_topic" ]]; then
	exec ./cli/make_help_topic.sh
elif [[ "$selected_type" = "$type_quickstart" ]]; then
	exec ./cli/make_item_quickstart.sh
else
	exec ./cli/make_item_generic.sh "$selected_type"
fi
