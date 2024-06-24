#!/usr/bin/env bash

set -euo pipefail

# Expect to be run in root of the repository
source ./cli/cli_item_common.sh

selected_type="$1"
name="$(read_name)"
display_name="$(read_display_name)"

description="$(read_description)"
url="$(read_url)"

selected_color="$(color_for_type "$selected_type")"

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

  type:
    text: $(yaml_escape "$selected_type")
    color: $(yaml_escape "$selected_color")

  displayName: $escaped_display_name
  icon: ~

  # Optional.
  prerequisites:
    - You are a cool person.

  description: |-
$(printf "%s" "$description" | sed "s/^/    /")

  link:
    href: $(yaml_escape "$url")
    text: View documentation
EOF

show_footer "item" "$out_dir" "$name"
