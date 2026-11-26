#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C

if [[ ${1:-} == --help ]]; then
    echo 'Usage: bash db/checks/http.sh [base_url] [compose_project] [samples]'
    echo 'Requires Docker Compose, curl and jq.'
    exit 0
fi

base=${1:-http://127.0.0.1:8080}
project=${2:-music-dz3-check}
samples=${3:-20}
if (( $# > 3 )) || [[ ! $samples =~ ^([1-9][0-9]?|100)$ ]]; then
    echo 'Expected up to three arguments and 1–100 samples.' >&2
    exit 2
fi

for command in docker curl jq; do
    command -v "$command" >/dev/null || { echo "Missing command: $command" >&2; exit 1; }
done

api=${base%/}/api/v1
compose=(docker compose -p "$project")
tmp=$(mktemp -d)
touch "$tmp/cookies"
response=$tmp/response.json
http=(curl --silent --show-error --fail --max-time 10 --cookie "$tmp/cookies" --cookie-jar "$tmp/cookies")
nonce=$(od -An -N16 -tx1 /dev/urandom | tr -d '[:space:]')
email="dz3-$nonce@example.invalid"
password="Dz3$nonce"
user_path=

request() {
    local method=$1 path=$2 token
    if [[ $method == GET ]]; then
        "${http[@]}" --output "$response" --write-out '%{time_total}\n' "$api$path"
        return
    fi

    token=$("${http[@]}" "$api/auth/csrf" | jq -er '.csrf_token | strings') || return 1
    "${http[@]}" --output "$response" --write-out '%{time_total}\n' --request "$method" \
        --header "X-CSRF-Token: $token" --header 'Content-Type: application/json' \
        --data-binary @- "$api$path" <<< "$3"
}

cleanup() {
    local status=$? body
    if [[ -n $user_path ]]; then
        if ! body=$(jq -nc --arg password "$password" '{current_password: $password}') ||
           ! request DELETE "$user_path" "$body" >/dev/null; then
            echo 'Could not delete the test account.' >&2
            if (( status == 0 )); then status=1; fi
        fi
    fi
    rm -f -- "$tmp/cookies" "$response"
    rmdir -- "$tmp"
    exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

read_db() {
    local service=$1 query=$2
    "${compose[@]}" run --rm --no-deps -T "$service" -At -c "$query"
}

profile() {
    jq -nc --arg email "$email" --arg name "$1" '{email: $email, display_name: $name}'
}

measure() {
    local label=$1 method=$2 path=$3 body=${4-} count rank
    rank=$(( (samples * 95 + 99) / 100 ))
    for ((count = 0; count < samples; count++)); do
        request "$method" "$path" "$body" || return 1
    done | sort -n | awk -v label="$label" -v rank="$rank" -v samples="$samples" \
        'NR == rank { printf "%s: p95=%.1f ms (%d requests, 1 client)\n", label, $1 * 1000, samples }'
}

signup=$(jq -nc --arg email "$email" --arg password "$password" \
    '{email: $email, password: $password, display_name: "Andrey"}')
request POST /auth/signup "$signup" >/dev/null
id=$(jq -er '.id | strings' "$response")
if [[ ! $id =~ ^[0-9a-fA-F]{8}(-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}$ ]]; then
    echo 'Signup returned an invalid account UUID.' >&2
    exit 1
fi
user_path="/users/$id"

role=$(read_db db-monitor "SELECT DISTINCT usename FROM pg_stat_activity WHERE datname = current_database() AND application_name = 'music-api';")
if [[ $role != music_api ]]; then
    printf 'Unexpected API database role: %s\n' "$role" >&2
    exit 1
fi
echo 'API database role: music_api (application_name=music-api)'

printf 'Input\tAPI\tDatabase\n'
for name in Andrey "O'Connor" 'select 1'; do
    body=$(profile "$name")
    request PUT "$user_path" "$body" >/dev/null
    returned=$(jq -er '.display_name | strings' "$response")
    expected=$(jq -nr --arg name "$name" '$name | gsub("\u0027"; "&#39;")')
    stored=$(read_db db-check "SELECT display_name FROM public.account WHERE id = '$id'::uuid;")
    if [[ $returned != "$expected" || $stored != "$name" ]]; then
        printf 'Display name did not round-trip: %s\n' "$name" >&2
        exit 1
    fi
    printf '%s\t%s\t%s\n' "$name" "$returned" "$stored"
done

catalog=$(read_db db-check "SELECT count(*) FROM public.track WHERE status = 'published';")
printf 'Published tracks: %s\n' "$catalog"
login=$(jq -nc --arg email "$email" --arg password "$password" '{email: $email, password: $password}')
body=$(profile Andrey)
measure Home GET /home
measure Login POST /auth/login "$login"
measure 'Update profile' PUT "$user_path" "$body"
