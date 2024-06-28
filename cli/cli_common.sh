#!/usr/bin/env bash

set -euo pipefail

declare -r quickstarts_dir="docs/quickstarts"

declare -r type_help_topic="Help topic"
declare -r type_quickstart="Quick start"
declare -r type_documentation="Documentation"
declare -r type_learning_path="Learning path"
declare -r type_other="Other"

declare -ra available_types=(
	"$type_help_topic"
	"$type_quickstart"
	"$type_documentation"
	"$type_learning_path"
	"$type_other"
)

update_directory() {
	# BASH_SOURCE is this file, in the cli directory. We want to be in the root of the repository.
	cd -- "$(dirname -- "$(realpath -- "$BASH_SOURCE")")"/..

	if [[ ! -d "$quickstarts_dir" ]]; then
		echo "The directory docs/quickstarts does not already exist. This script must be run from the quickstarts repo." >&2
	fi
}

read_type() {
	declare selected_type

	select option in "${available_types[@]}"; do
		if [[ -n "$option" && -n "$REPLY" ]]; then
			declare index="$(($REPLY-1))"
			selected_type="${available_types["$index"]}"
			break
		else
			echo "Input must be a number corresponding to an option." >&2
		fi
	done

	printf "%s" "$selected_type"
}

read_name() {
	declare name

	while true; do
		IFS="" read -p "Name of item (internal only; alphanumeric with hyphens): " name

		if [[ "$name" =~ ^[a-z0-9-]+$ ]]; then
			break
		fi

		echo "Quickstart name must be non-empty and composed of alphanumeric characters, dashes, and underscores." >&2
	done
	
	printf "%s" "$name"
}

create_out_dir() {
	declare -r out_dir="$1"
	mkdir -- "$out_dir" || { echo "Failed to create directory. Perhaps it already exists?" >&2; return 1; }
}

yaml_escape() {
	declare -r str="$1"
	printf "%s" "'${str//"'"/"''"}'"
}

show_footer() {
	declare -r kind="$1"
	declare -r out_dir="$2"
	declare -r name="$3"

	echo "A template $kind has been created in $out_dir. You should update both metadata.yml and $name.yml to reflect what the item should show." >&2
}
