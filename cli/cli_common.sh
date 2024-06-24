#!/usr/bin/env bash

set -euo pipefail

declare -r quickstarts_dir="docs/quickstarts"

declare -r type_quickstart="Quick start"
declare -r type_documentation="Documentation"
declare -r type_learning_path="Learning path"
declare -r type_other="Other"

declare -ra available_types=(
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

read_display_name() {
	declare display_name
	IFS="" read -p "Display name (name shown to user): " display_name
	printf "%s" "$display_name"
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

color_for_type() {
	declare -rA available_type_colors=(
		["$type_quickstart"]="green"
		["$type_documentation"]="orange"
		["$type_learning_path"]="cyan"
		["$type_other"]="purple"
	)

	declare -r selected="$1"
	echo "${available_type_colors["$selected"]}"
}

read_duration() {
	declare duration

	while true; do
		IFS="" read -p "Duration (minutes): " duration

		if [[ "$duration" =~ ^[0-9]+$ ]]; then
			break
		fi

		echo "Duration must be a positive integer." >&2
	done

	printf "%s" "$duration"
}

read_description() {
	declare description
	IFS="" read -p "Description (a short, 2-3 sentence summary): " description
	printf "%s" "$description"
}

read_url() {
	declare url
	IFs="" read -rp "URL of resource: " url
	printf "%s" "$url"
}

yaml_escape() {
	declare -r str="$1"
	printf "%s" "'${str//"'"/"''"}'"
}

out_dir_for() {
	declare -r name="$1"
	printf "%s/%s" "$quickstarts_dir" "$name"
}

create_out_dir() {
	declare -r name="$1"
	mkdir -- "$(out_dir_for "$name")" || { echo "Failed to create directory. Perhaps it already exists?" >&2; return 1; }
}

write_metadata() {
	declare -r name="$1"

	declare -r escaped_name="$(yaml_escape "$name")"

	cat > "$(out_dir_for "$name")/metadata.yml" <<EOF
kind: QuickStarts # kind must always be "QuickStarts"
name: $escaped_name
tags: # If you want to use more granular filtering add tags to the quickstart
  - kind: bundle # use bundle tag for a topic to be accessed from a whole bundle eg. console.redhat.com/insights
    value: iam
  - kind: application # use application tag for quickstart used by specific application
    value: my-user-access
EOF
}

show_footer() {
	declare -r kind="$1"
	declare -r name="$2"

	echo "A template $kind has been created in $(out_dir_for "$name"). You should update both metadata.yml and $name.yml to reflect what the item should show." >&2
}
