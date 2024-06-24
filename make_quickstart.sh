#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname -- "$(realpath -- "$BASH_SOURCE")")"

quickstarts_dir="docs/quickstarts"

if [[ ! -d "$quickstarts_dir" ]]; then
	echo "The directory docs/quickstarts does not already exist. This script must be run from the quickstarts repo." >&2
fi

while true; do
	read -p "Name of quickstart: " name

	if [[ "$name" =~ [a-z0-9_-]+ ]]; then
		break
	fi

	echo "Quickstart name must be non-empty and composed of alphanumeric characters, dashes, and underscores." >&2
done

type_quickstart="Quick start"

available_types=(
	"$type_quickstart"
	"Documentation"
	"Learning path"
	"Other"
)

available_type_colors=(
	"green"
	"orange"
	"cyan"
	"purple"
)

read -p "Display name: " display_name

selected_type=""
selected_color=""

select option in "${available_types[@]}"; do
	if [[ -n "$option" && -n "$REPLY" ]]; then
		index="$(($REPLY-1))"
		selected_type="${available_types["$index"]}"
		selected_color="${available_type_colors["$index"]}"
		break
	else
		echo "Input must be a number corresponding to an option." >&2
	fi
done

while true; do
	read -p "Duration (minutes): " duration

	if [[ "$duration" =~ [0-9]+ ]]; then
		break
	fi

	echo "Duration must be a positive integer." >&2
done

read -p "Description (a short, 2-3 sentence summary): " description

out_dir="$quickstarts_dir/$name"

mkdir -- "$out_dir" || { echo "Failed to create directory. Perhaps it already exists?" >&2; exit 1; }

escaped_name="'${name//"'"/"''"}'"
escaped_display_name="'${display_name//"'"/"''"}'"

cat > "$out_dir/metadata.yml" <<EOF
kind: QuickStarts # kind must always be "QuickStarts"
name: $escaped_name
tags: # If you want to use more granular filtering add tags to the quickstart
  - kind: bundle # use bundle tag for a topic to be accessed from a whole bundle eg. console.redhat.com/insights
    value: iam
  - kind: application # use application tag for quickstart used by specific application
    value: my-user-access
EOF

cat > "$out_dir/$name.yml" <<EOF
# Additional info: https://docs.openshift.com/container-platform/4.9/web_console/creating-quick-start-tutorials.html
# Template from https://github.com/patternfly/patternfly-quickstarts/blob/main/packages/dev/src/quickstarts-data/yaml/template.yaml
# See quick start instructions here https://github.com/RedHatInsights/quickstarts/tree/main/docs/quickstarts
metadata:
  name: $escaped_name
  # you can add additional metadata here
  # instructional: true
spec:
  displayName: $escaped_display_name
  durationMinutes: $duration

  # Display the quickstart tag on the tile.
  type:
    text: Quick start
    color: green

  # Optional.
  prerequisites:
    - You are a cool person.

  description: |-
$(printf "%s" "$description" | sed "s/^/    /")

  introduction: |-
    This is a longer description of the quickstart, generally multiple paragraphs. You can also use Markdown here (and in all later fields).

  tasks:
    - title: The *title* of the first task
      description: |-
        What the user will be told to do in the task.

      # Optional. This will display as the "Check Your Work" portion.
      review:
        instructions: |-
          - Tell the user how to verify that they performed the steps correctly.

        failedTaskHelp: Try completing the steps again.

    # You can add more tasks, as below.
    - title: Solve your problem
      description: |-
        Solve your problem using the following method:
        1. Write down the problem.
        2. Think really hard.
        3. Write down the solution.

      # Optional. The task's success and failure messages
      summary:
        success: Shows a success message in the task header
        failed: Shows a failed message in the task header
  conclusion: |-
    Summarize the task.  
EOF

echo "A template quickstart has been created in $out_dir" >&2
