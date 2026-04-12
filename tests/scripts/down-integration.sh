#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "${script_dir}/../.." && pwd)
compose_file="${repo_root}/tests/docker-compose.yml"

project_names="$(
  docker compose ls --format json | node -e '
const fs = require("node:fs");

const composeFile = process.argv[1];
const input = fs.readFileSync(0, "utf8").trim();
const projects = input ? JSON.parse(input) : [];
const names = new Set();

for (const project of projects) {
  const configFiles = String(project.ConfigFiles || "")
    .split(",")
    .map((entry) => entry.trim())
    .filter(Boolean);

  if (configFiles.includes(composeFile) && project.Name) {
    names.add(project.Name);
  }
}

process.stdout.write([...names].join("\n"));
' "${compose_file}"
)"

if [[ -z "${project_names}" ]]; then
  echo "No running tests compose projects found."
  exit 0
fi

while IFS= read -r project_name; do
  [[ -n "${project_name}" ]] || continue
  docker compose -f "${compose_file}" -p "${project_name}" down -v --remove-orphans
done <<< "${project_names}"
