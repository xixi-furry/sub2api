#!/usr/bin/env bash
set -euo pipefail
: "${IMAGE:?}" "${VERSION:?}" "${ARCH:?}" "${GITHUB_RUN_ID:?}"
image_ref="$IMAGE:build-$GITHUB_RUN_ID-$ARCH"
docker run --rm "$image_ref" /app/sub2api -version 2>&1 | tee /tmp/fork-version.txt
grep -F "Sub2API $VERSION " /tmp/fork-version.txt

cleanup() {
  docker logs fork-smoke 2>&1 || true
  docker rm -f -v fork-smoke fork-postgres fork-redis >/dev/null 2>&1 || true
  docker network rm fork-smoke-network >/dev/null 2>&1 || true
}
trap cleanup EXIT
db_password=$(openssl rand -hex 24)
admin_password=$(openssl rand -hex 24)
jwt_secret=$(openssl rand -hex 32)
echo "::add-mask::$db_password"
echo "::add-mask::$admin_password"
echo "::add-mask::$jwt_secret"
docker network create fork-smoke-network
docker run -d --name fork-postgres --network fork-smoke-network \
  -e POSTGRES_USER=sub2api -e POSTGRES_PASSWORD="$db_password" -e POSTGRES_DB=sub2api postgres:18-alpine
docker run -d --name fork-redis --network fork-smoke-network redis:8-alpine
for attempt in $(seq 1 60); do
  if docker exec fork-postgres pg_isready -U sub2api -d sub2api; then break; fi
  sleep 2
done
docker exec fork-postgres pg_isready -U sub2api -d sub2api
docker run -d --name fork-smoke --network fork-smoke-network -p 127.0.0.1:18080:8080 \
  -e AUTO_SETUP=true -e SERVER_HOST=0.0.0.0 -e SERVER_PORT=8080 \
  -e DATABASE_HOST=fork-postgres -e DATABASE_USER=sub2api -e DATABASE_DBNAME=sub2api \
  -e DATABASE_PASSWORD="$db_password" -e DATABASE_SSLMODE=disable -e REDIS_HOST=fork-redis \
  -e ADMIN_EMAIL=smoke@example.test -e ADMIN_PASSWORD="$admin_password" -e JWT_SECRET="$jwt_secret" \
  "$image_ref"
for attempt in $(seq 1 90); do
  if curl --fail --silent http://127.0.0.1:18080/health | jq -e '.status == "ok"'; then
    curl --fail --silent http://127.0.0.1:18080/ | grep -q '<html'
    echo "Image version, database initialization, API health and embedded frontend passed."
    exit 0
  fi
  sleep 2
done
exit 1
