#!/bin/sh
# Compare copied version claims with go.mod; historical review notes are excluded.
set -eu

cd "$(dirname "$0")/.."
go_version=$(awk '$1 == "go" { print $2; exit }' go.mod)
httpserve_version=$(awk '
    $1 == "github.com/vroomy/httpserve" { print $2; exit }
    $1 == "require" && $2 == "github.com/vroomy/httpserve" { print $3; exit }
' go.mod)
if [ -z "$go_version" ] || [ -z "$httpserve_version" ]; then
    echo "Missing Go or httpserve version in go.mod" >&2
    exit 1
fi

status=0
for file in README.md AGENTS.md docs/development.md docs/configuration.md docs/plugins.md; do
    if ! awk -v go_version="$go_version" -v httpserve_version="$httpserve_version" '
        {
            line = $0
            while (match(line, /Go[[:space:]*`]+[0-9]+\.[0-9]+(\.[0-9]+)?/)) {
                version = substr(line, RSTART, RLENGTH)
                sub(/^Go[[:space:]*`]+/, "", version)
                go_claims++
                if (version != go_version) {
                    print FILENAME ":" FNR ": Go " version " disagrees with go.mod (" go_version ")" > "/dev/stderr"
                    failed = 1
                }
                line = substr(line, RSTART + RLENGTH)
            }
            line = $0
            while (match(line, /httpserve[[:space:]*`]+v[0-9]+\.[0-9]+\.[0-9]+/)) {
                version = substr(line, RSTART, RLENGTH)
                sub(/^httpserve[[:space:]*`]+/, "", version)
                if (version != httpserve_version) {
                    print FILENAME ":" FNR ": httpserve " version " disagrees with go.mod (" httpserve_version ")" > "/dev/stderr"
                    failed = 1
                }
                line = substr(line, RSTART + RLENGTH)
            }
        }
        END {
            if (FILENAME == "README.md" && !go_claims) {
                print "README.md: missing explicit Go minimum" > "/dev/stderr"
                failed = 1
            }
            exit failed
        }
    ' "$file"; then
        status=1
    fi
done
exit "$status"
