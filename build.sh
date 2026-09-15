#!/bin/sh

set -e

source="source.txt"
parts=".parts"

# IPv4
urls="https://raw.githubusercontent.com/metowolf/iplist/master/data/special/china.txt
https://raw.githubusercontent.com/17mon/china_ip_list/master/china_ip_list.txt"
# IPv6
urls="$urls
https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/china6.txt
https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/googlecn6.txt"

rm -f "$source"
rm -rf "$parts"
mkdir -p "$parts"

download() {
  # $1: index, $2: url
  wget --no-verbose --tries=3 --timeout=30 -O "$parts/part-$1" "$2"
}

# download in parallel, fail if any source fails
pids=""
i=0
for url in $urls; do
  i=$((i + 1))
  download "$i" "$url" &
  pids="$pids $!"
done
for pid in $pids; do
  wait "$pid" || exit 1
done

# sanity check: every source must be non-empty and not an HTML error page
i=0
for url in $urls; do
  i=$((i + 1))
  part="$parts/part-$i"
  if [ ! -s "$part" ]; then
    echo "download is empty: $url" >&2
    exit 1
  fi
  if head -c 256 "$part" | grep -Eq '<(html|!doctype)'; then
    echo "download is not a CIDR list (HTML?): $url" >&2
    exit 1
  fi
done

# concatenate in order; some sources lack a trailing newline, so add one after each
i=0
for url in $urls; do
  i=$((i + 1))
  cat "$parts/part-$i" >>"$source"
  printf '\n' >>"$source"
done
rm -rf "$parts"

mkdir -p dist

go run main.go -i "$source" -o dist/cidr.txt -m dist/Country.mmdb
