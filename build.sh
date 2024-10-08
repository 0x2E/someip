#!/bin/sh

set -e

source="source.txt"

rm $source || true

download() {
  wget --no-verbose -O - $1 >>$source
  echo "\n" >>$source
}

# IPv4
download https://raw.githubusercontent.com/metowolf/iplist/master/data/special/china.txt
download https://raw.githubusercontent.com/17mon/china_ip_list/master/china_ip_list.txt
# IPv6
download https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/cernet6.txt
download https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/china6.txt
download https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/chinanet6.txt
download https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/cmcc6.txt
download https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/cstnet6.txt
download https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/drpeng6.txt
download https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/googlecn6.txt
download https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/tietong6.txt
download https://raw.githubusercontent.com/gaoyifan/china-operator-ip/refs/heads/ip-lists/unicom6.txt

mkdir -p dist

go run main.go -i $source -o dist/cidr.txt -m dist/Country.mmdb >/dev/null
