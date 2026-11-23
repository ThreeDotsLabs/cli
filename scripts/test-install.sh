#!/bin/sh
# Checks install.sh's 'tdl' name-conflict handling (omarchy) in Docker.
# Runs offline: a fake curl answers the GitHub requests with a stub binary.
#
# Usage: scripts/test-install.sh   (or: make test-install)
set -eu

cd "$(dirname "$0")/.."
IMAGE=tdl-install-test

docker build -q -t "$IMAGE" -f - . >/dev/null <<'EOF'
FROM debian:bookworm-slim
RUN apt-get update -qq && apt-get install -y -qq --no-install-recommends sudo binutils >/dev/null
# fake curl: the installer only asks for the latest tag and then the binary
RUN printf '%s\n' '#!/bin/sh' \
    'for a; do [ "${prev:-}" = -o ] && out=$a; prev=$a; url=$a; done' \
    'case $url in' \
    '  */releases/latest) echo "{\"tag_name\":\"v0.0.0-test\"}" >"$out" ;;' \
    '  *) printf "#!/bin/sh\necho ThreeDotsLabs\n" >"$out" ;;' \
    'esac' \
    'printf 200' >/usr/local/bin/curl && chmod +x /usr/local/bin/curl
# alice uses omarchy (its 'tdl' is a bash function, invisible to the installer's sh)
RUN useradd -m -s /bin/bash alice && useradd -m -s /bin/bash bob \
 && printf 'alice ALL=(ALL) NOPASSWD:ALL\nbob ALL=(ALL) NOPASSWD:ALL\n' >/etc/sudoers.d/users \
 && mkdir -p /home/alice/.local/share/omarchy \
 && echo 'tdl() { echo omarchy-tdl; }' >>/home/alice/.bashrc \
 && chown -R alice /home/alice
# the exact command the academy shows on Linux, with cli.sh served from /install.sh
RUN printf '%s\n' '#!/bin/sh' 'sudo /bin/sh -c "$(cat /install.sh)" -- -b /usr/local/bin' \
    >/usr/local/bin/academy-install && chmod +x /usr/local/bin/academy-install
COPY install.sh /install.sh
EOF

FAILED=0

# check <name> <user> <command> <expected file in /usr/local/bin> [expected output text]
check() {
  name=$1 user=$2 cmd=$3 want_file=$4 want_text=${5:-}
  out=$(docker run --rm -u "$user" "$IMAGE" sh -c "$cmd; echo EXIT=\$?; ls /usr/local/bin" 2>&1) || true
  if echo "$out" | grep -q '^EXIT=0$' && echo "$out" | grep -qx "$want_file" &&
    { [ -z "$want_text" ] || echo "$out" | grep -qF "$want_text"; }; then
    echo "ok   $name"
  else
    echo "FAIL $name (want /usr/local/bin/$want_file${want_text:+ and \"$want_text\"})"
    echo "$out" | sed 's/^/     | /'
    return 1
  fi
}

# script(1) gives the installer a terminal; its stdin is what the user types at the prompt
tty_answer() {
  printf '%s\n' "printf '%s\\n' '$1' | script -qec academy-install /dev/null"
}

check "omarchy, sudo, no tty: installs as threedots" \
  alice "academy-install </dev/null" threedots "type 'threedots' instead" || FAILED=1
check "omarchy, sudo, tty, default answer: installs as threedots" \
  alice "$(tty_answer '')" threedots "Omarchy detected" || FAILED=1
check "omarchy, sudo, tty, answer 2: installs as tdl" \
  alice "$(tty_answer 2)" tdl "open a new terminal" || FAILED=1
check "omarchy, no sudo, no tty: installs as threedots" \
  alice "sh /install.sh -b /home/alice/bin </dev/null && sudo cp /home/alice/bin/* /usr/local/bin" threedots || FAILED=1
check "no omarchy, sudo: installs as tdl without asking" \
  bob "academy-install </dev/null" tdl || FAILED=1

exit $FAILED
