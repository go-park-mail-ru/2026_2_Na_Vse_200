#!/usr/bin/env bash
set -euo pipefail

if [[ ${1:-} == --help ]]; then
    echo 'Usage: bash db/checks/load.sh [base_url] [compose_project] [clients] [seconds]'
    echo 'Requires Docker Compose, curl and jq.'
    exit 0
fi

base=${1:-http://127.0.0.1:18080}
project=${2:-music-dz3-check}
clients=${3:-12}
duration=${4:-15}
if (( $# > 4 )) || [[ ! $clients =~ ^([1-9]|[12][0-9]|3[0-2])$ || ! $duration =~ ^([1-5][0-9]|60)$ ]]; then
    echo 'Expected 1–32 clients and 10–60 seconds.' >&2
    exit 2
fi

for command in docker curl jq; do
    command -v "$command" >/dev/null || { echo "Missing command: $command" >&2; exit 1; }
done

url=${base%/}/api/v1/home
compose=(docker compose -p "$project")
server=$("${compose[@]}" ps -q server)
db=$("${compose[@]}" ps -q db)
if [[ -z $server || -z $db ]]; then
    echo 'Start server and db first.' >&2
    exit 1
fi

timestamp() {
    jq -n 'now'
}

snapshot() {
    local stats counts
    stats=$(docker stats --no-stream --format '{{json .}}' "$server" "$db") || return 1
    counts=$("${compose[@]}" run --rm --no-deps -T db-monitor -At -c "
        SELECT count(*) FILTER (WHERE state = 'active'), count(*) FILTER (WHERE state = 'idle')
        FROM pg_stat_activity
        WHERE datname = current_database() AND application_name = 'music-api' AND usename = 'music_api';") || return 1
    jq -ces --arg counts "$counts" '
        def mib:
            capture("^(?<size>[0-9.]+)(?<unit>B|KiB|MiB|GiB|TiB)$")
            | (.size | tonumber) * ({B: 1, KiB: 1024, MiB: 1048576, GiB: 1073741824, TiB: 1099511627776}[.unit])
            | (. * 100 / 1048576 | round) / 100;
        ($counts | split("|") | map(tonumber)) as [$active, $idle]
        | map((.NetIO | split(" / ")) as [$rx, $tx] | {
            container: .Name, cpu: .CPUPerc,
            ram_mib: (.MemUsage | split(" / ")[0] | mib), rx: $rx, tx: $tx
        }) as $containers
        | if ($containers | length) != 2 then error("Expected two container samples") else {
            sampled_at: (now | floor | todateiso8601), containers: $containers,
            api_db_active: $active, api_db_idle: $idle
        } end' <<< "$stats"
}

access_log() {
    local from=$1 until=$2
    "${compose[@]}" logs --no-log-prefix --since "$from" --until "$until" server |
        jq -Rsc --argjson from "$from" --argjson until "$until" '
            def round2: (. * 100 | round) / 100;
            split("\n")
            | map(fromjson? | objects | select(.method == "GET" and .url == "/api/v1/home" and .status != null))
            | length as $count
            | (map(.duration_ms) | sort) as $times
            | {
                from: ($from | floor | todateiso8601), until: ($until | floor | todateiso8601),
                requests: $count, rps: ($count / ($until - $from) | round2),
                p95_ms: (if $count > 0 then $times[($count * 0.95 | ceil) - 1] | round2 else null end),
                statuses: (group_by(.status) | map({status: .[0].status, Count: length}))
            }'
}

tracks=$(curl --silent --show-error --fail --max-time 8 "$url" | jq -er '.tracks | length')
catalog=$("${compose[@]}" run --rm --no-deps -T db-check -At -c "SELECT count(*) FROM public.track WHERE status = 'published';")
before=$(snapshot)
idle_start=$(timestamp)
sleep "$duration"
idle_end=$(timestamp)

during_file=$(mktemp)
pids=()
cleanup() {
    if (( ${#pids[@]} )); then kill "${pids[@]}" 2>/dev/null || true; fi
    rm -f -- "$during_file"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

start=$(timestamp)
end=$(jq -n --argjson start "$start" --argjson duration "$duration" '$start + $duration')
deadline=$((SECONDS + duration))
(sleep 3; snapshot) > "$during_file" &
pids+=("$!")
for ((client = 0; client < clients; client++)); do
    (
        while (( SECONDS < deadline )); do
            curl --silent --show-error --max-time 8 --output /dev/null "$url" || exit 1
        done
    ) &
    pids+=("$!")
done
for pid in "${pids[@]}"; do wait "$pid"; done
pids=()

after=$(snapshot)
idle=$(access_log "$idle_start" "$idle_end")
load=$(access_log "$start" "$end")
jq -en --argjson catalog "$catalog" --argjson tracks "$tracks" --argjson clients "$clients" \
    --argjson idle "$idle" --argjson load "$load" --argjson before "$before" \
    --slurpfile during "$during_file" --argjson after "$after" '
    if $load.requests == 0 then error("No GET /api/v1/home in access logs during the load window") else
        {published_tracks: $catalog, home_tracks: $tracks, clients: $clients, idle_http: $idle, load_http: $load,
         before: $before, during: $during[0], after: $after}
    end'
