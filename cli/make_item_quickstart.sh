#!/usr/bin/env bash

set -euo pipefail

# Expect to be run in the root of the repository
source ./cli/cli_item_common.sh

selected_type="$type_quickstart"
selected_color="$(color_for_type "$selected_type")"

name="$(read_name)"
display_name="$(read_display_name)"

duration="$(read_duration)"
description="$(read_description)"

out_dir="$(out_dir_for "$name")"

create_out_dir "$out_dir"
write_quickstart_metadata "$out_dir"

escaped_name="$(yaml_escape "$name")"
escaped_display_name="$(yaml_escape "$display_name")"

cat > "$out_dir/$name.yml" <<EOF
# Additional info: https://docs.openshift.com/container-platform/4.9/web_console/creating-quick-start-tutorials.html
# Template from https://github.com/patternfly/patternfly-quickstarts/blob/main/packages/dev/src/quickstarts-data/yaml/template.yaml
# See quick start instructions here https://github.com/RedHatInsights/quickstarts/tree/main/docs/quickstarts
metadata:
  name: $escaped_name
  # you can add additional metadata here
  # instructional: true
spec:
  version: 0.1

  displayName: $escaped_display_name
  durationMinutes: $duration
  icon: ~

  # Display the quickstart tag on the tile.
  type:
    text: $(yaml_escape "$selected_type")
    color: $(yaml_escape "$selected_color")

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

show_footer "quick start" "$out_dir" "$name"
