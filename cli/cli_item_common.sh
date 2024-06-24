set -euo pipefail

# Expect to be run in root of repository
source cli/cli_common.sh

read_display_name() {
	declare display_name
	IFS="" read -p "Display name (name shown to user): " display_name
	printf "%s" "$display_name"
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

out_dir_for() {
	declare -r name="$1"
	printf "%s/%s" "$quickstarts_dir" "$name"
}

write_quickstart_metadata() {
	declare -r out_dir="$1"

	cat > "$out_dir/metadata.yml" <<EOF
kind: QuickStarts # kind must always be "QuickStarts"
name: $(yaml_escape "$name")
tags: # If you want to use more granular filtering add tags to the quickstart
  - kind: bundle # use bundle tag for a topic to be accessed from a whole bundle eg. console.redhat.com/insights
    value: iam
  - kind: application # use application tag for quickstart used by specific application
    value: my-user-access
EOF
}
