#!/usr/bin/env bash

set -euo pipefail

# Expect to be run in root of the repository
source ./cli/cli_common.sh

name="$(read_name)"
out_dir="docs/help-topics/$name"

create_out_dir "$out_dir"

cat > "$out_dir/metadata.yml" <<EOF
kind: HelpTopic
name: $(yaml_escape "$name")
EOF

cat > "$out_dir/$name.yml" <<EOF
# Name is an internal name. Title shows up in the UI as a side panel title.
# Tags to be kept empty for now. Tags will specify where in the app descriptions will be available.
# Links to be external only. We don't know yet whether referencing to other side panels will be supported but referencing to in-depth docs is expected to be supported.

- name: $(yaml_escape "$name-whatever")
  tags:
  title: Solve the problem
  content: |-
    Solve your problem using the following method:

    1. Write down the problem.
    2. Think really hard.
    3. Write down the solution.

    Then, your problem will be solved.
EOF

show_footer "help topic" "$out_dir" "$name"
