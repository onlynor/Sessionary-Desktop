#!/usr/bin/env bash
# smoke-deb.sh DEB — installs the .deb on a clean Ubuntu (no Node.js, npm or bun) in a container and runs the
# smoke test there, under Xvfb: the package must carry everything the app needs.
set -euo pipefail
deb=$(realpath "$1"); here=$(dirname "$(realpath "$0")")
ctx=$(mktemp -d); trap 'rm -rf "$ctx"' EXIT
cp "$deb" "$ctx/app.deb"; cp "$here/smoke.sh" "$ctx/"
cat > "$ctx/Dockerfile" <<'DOCKER'
FROM ubuntu:${UBUNTU:-26.04}
COPY app.deb smoke.sh /tmp/
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends /tmp/app.deb xvfb xauth dbus procps \
 && rm -rf /var/lib/apt/lists/* && useradd -m tester
USER tester
DOCKER
sed -i "s/\${UBUNTU:-26.04}/${UBUNTU:-26.04}/" "$ctx/Dockerfile"
docker build --network host -q -t sessionary-smoke "$ctx" >/dev/null
docker run --rm sessionary-smoke bash -c '
  if command -v node >/dev/null; then echo "smoke: Node.js is on PATH in the clean image" >&2; exit 1; fi
  /tmp/smoke.sh /usr/bin/sessionary-desktop /tmp/out'
