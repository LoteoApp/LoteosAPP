#!/usr/bin/env bash
# Smoke test for the loteo registration API (issue #145).
#
# Run the backend first, in another terminal:
#   cd apps/backend && doppler run -- go run ./cmd/server
#
# Then, from the repository root:
#   ADMIN_PASSWORD='<password>' doppler run -- bash scripts/smoke-loteos.sh
#
# It signs in as the administrador, registers a loteo with a small plan, loads
# the data of one of its lotes, and checks the permission and validation paths.
# It exits non-zero on the first check that doesn't hold. Everything it creates
# stays in the database: the id is printed at the end.
set -uo pipefail

API="${API:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-administrador@loteosapp.com}"
: "${ADMIN_PASSWORD:?set ADMIN_PASSWORD to the administrador account password}"
: "${SUPABASE_URL:?run through 'doppler run --' so SUPABASE_URL is set}"
: "${SUPABASE_ANON_KEY:?run through 'doppler run --' so SUPABASE_ANON_KEY is set}"

BODY_FILE=$(mktemp)
trap 'rm -f "$BODY_FILE"' EXIT

FAILURES=0

step() { printf '\n\033[1m== %s\033[0m\n' "$1"; }
pass() { printf '\033[32mok\033[0m   %s\n' "$1"; }
fail() {
  printf '\033[31mFAIL\033[0m %s\n' "$1" >&2
  FAILURES=$((FAILURES + 1))
}

# request METHOD PATH [DATA] prints the HTTP status and leaves the response
# body in $BODY_FILE. curl's own exit code is checked so a connection failure
# doesn't read as an HTTP status of 000 being "just another status".
request() {
  local method=$1 path=$2 data=${3:-} status
  local args=(-sS -o "$BODY_FILE" -w '%{http_code}' -X "$method" "$API$path" -H 'Content-Type: application/json')
  if [ -n "${TOKEN:-}" ]; then
    args+=(-H "Authorization: Bearer $TOKEN")
  fi
  if [ -n "$data" ]; then
    args+=(-d "$data")
  fi

  if ! status=$(curl "${args[@]}"); then
    echo "000"
    return 1
  fi
  echo "$status"
}

expect_status() {
  local want=$1 got=$2 label=$3
  if [ "$got" = "$want" ]; then
    pass "$label (HTTP $got)"
  else
    fail "$label: HTTP $got, want $want -- $(tr -d '\n' <"$BODY_FILE")"
  fi
}

step "Signing in as $ADMIN_EMAIL"
TOKEN=$(curl -sS -X POST "$SUPABASE_URL/auth/v1/token?grant_type=password" \
  -H "apikey: $SUPABASE_ANON_KEY" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" \
  | jq -r '.access_token // empty')

if [ -z "$TOKEN" ]; then
  echo "sign-in failed: check ADMIN_PASSWORD" >&2
  exit 1
fi
pass "got an access token"

# A loteo square with two manzanas side by side and one lote in each.
read -r -d '' PLAN <<'JSON' || true
{
  "nombre": "Loteo Smoke Test",
  "ubicacion": "Córdoba",
  "descripcion": "Alta de prueba del endpoint",
  "plano": {
    "loteo": {
      "handle": "1A",
      "vertices": [{"x":0,"y":0},{"x":200,"y":0},{"x":200,"y":100},{"x":0,"y":100}]
    },
    "manzanas": [
      {"ref":"M-1","handle":"2A","vertices":[{"x":10,"y":10},{"x":90,"y":10},{"x":90,"y":90},{"x":10,"y":90}]},
      {"ref":"M-2","handle":"2B","vertices":[{"x":110,"y":10},{"x":190,"y":10},{"x":190,"y":90},{"x":110,"y":90}]}
    ],
    "lotes": [
      {"manzanaRef":"M-2","handle":"3A","vertices":[{"x":120,"y":20},{"x":150,"y":20},{"x":150,"y":80},{"x":120,"y":80}]},
      {"manzanaRef":"M-1","handle":"3B","vertices":[{"x":20,"y":20},{"x":50,"y":20},{"x":50,"y":80},{"x":20,"y":80}]}
    ],
    "calles": [
      {"handle":"4A","vertices":[{"x":90,"y":0},{"x":110,"y":0},{"x":110,"y":100},{"x":90,"y":100}]}
    ]
  }
}
JSON

step "POST /api/v1/loteos"
STATUS=$(request POST /api/v1/loteos "$PLAN")
expect_status 201 "$STATUS" "the loteo is created"
CREATED=$(cat "$BODY_FILE")
jq . <<<"$CREATED"

LOTEO_ID=$(jq -r '.id // empty' <<<"$CREATED")
if [ -z "$LOTEO_ID" ]; then
  fail "the response carries no loteo id, nothing else can be checked"
  exit 1
fi

# The first lote was sent pointing at M-2, the second manzana, so its manzanaId
# must be the second one in the response, not the first.
step "The hierarchy the client sent was respected"
if [ "$(jq -r '.lotes[0].manzanaId' <<<"$CREATED")" = "$(jq -r '.manzanas[1].id' <<<"$CREATED")" ]; then
  pass "lote[0] belongs to manzana[1], the one it named"
else
  fail "lote[0].manzanaId = $(jq -r '.lotes[0].manzanaId' <<<"$CREATED"), want manzanas[1].id = $(jq -r '.manzanas[1].id' <<<"$CREATED")"
fi

LOTE_ID=$(jq -r '.lotes[0].id' <<<"$CREATED")
OTHER_LOTE=$(jq -r '.lotes[1].id' <<<"$CREATED")

step "PATCH the lote data"
STATUS=$(request PATCH "/api/v1/loteos/$LOTEO_ID/lotes/$LOTE_ID" \
  '{"numero":"1","precio":150000,"moneda":"ARS","superficie":1800,"caracteristicas":"esquina"}')
expect_status 200 "$STATUS" "the lote data is stored"
jq -c . "$BODY_FILE"

step "A lote number repeats within the loteo (expect 409)"
STATUS=$(request PATCH "/api/v1/loteos/$LOTEO_ID/lotes/$OTHER_LOTE" '{"numero":"1"}')
expect_status 409 "$STATUS" "a repeated lote number is a conflict"

step "A lote of another loteo (expect 404)"
STATUS=$(request PATCH "/api/v1/loteos/$LOTEO_ID/lotes/11111111-1111-4111-8111-111111111111" '{"numero":"9"}')
expect_status 404 "$STATUS" "a lote outside the loteo reads as missing"

step "A price with more decimals than the column keeps (expect 400)"
STATUS=$(request PATCH "/api/v1/loteos/$LOTEO_ID/lotes/$OTHER_LOTE" '{"numero":"2","precio":1000.005,"moneda":"ARS"}')
expect_status 400 "$STATUS" "an unstorable price is rejected"

step "A lote naming a manzana outside the plan (expect 400)"
STATUS=$(request POST /api/v1/loteos \
  '{"nombre":"Roto","plano":{"loteo":{"vertices":[{"x":0,"y":0},{"x":10,"y":0},{"x":10,"y":10}]},"manzanas":[{"ref":"M-1","vertices":[{"x":0,"y":0},{"x":5,"y":0},{"x":5,"y":5}]}],"lotes":[{"manzanaRef":"NO-EXISTE","vertices":[{"x":0,"y":0},{"x":1,"y":0},{"x":1,"y":1}]}]}}')
expect_status 400 "$STATUS" "an unresolvable manzana reference is rejected"

step "An open ring (expect 400)"
STATUS=$(request POST /api/v1/loteos '{"nombre":"Roto","plano":{"loteo":{"vertices":[{"x":0,"y":0},{"x":10,"y":0}]}}}')
expect_status 400 "$STATUS" "an unusable ring is rejected"

step "A ring that crosses itself (expect 400)"
STATUS=$(request POST /api/v1/loteos \
  '{"nombre":"Roto","plano":{"loteo":{"vertices":[{"x":0,"y":0},{"x":10,"y":0},{"x":5,"y":0},{"x":5,"y":10}]}}}')
expect_status 400 "$STATUS" "a self-intersecting ring is rejected"
if [ "$(jq -r '.code' "$BODY_FILE")" = "self_intersecting_geometry" ]; then
  pass "the crossing is reported as such"
else
  fail "error code = $(jq -r '.code' "$BODY_FILE"), want self_intersecting_geometry"
fi

step "No token (expect 401)"
SAVED_TOKEN=$TOKEN
TOKEN=""
STATUS=$(request POST /api/v1/loteos '{"nombre":"Sin token"}')
expect_status 401 "$STATUS" "an anonymous caller is rejected"
TOKEN=$SAVED_TOKEN

printf '\n\033[1mCreated loteo %s\033[0m\n' "$LOTEO_ID"
echo "To remove it:  psql \"\$DATABASE_URL\" -f scripts/smoke-loteos-cleanup.sql -v loteo_id=\"'$LOTEO_ID'\""

if [ "$FAILURES" -gt 0 ]; then
  printf '\n\033[31m%d check(s) failed\033[0m\n' "$FAILURES" >&2
  exit 1
fi
printf '\n\033[32mall checks passed\033[0m\n'
