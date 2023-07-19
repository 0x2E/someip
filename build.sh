#!/bin/sh

wget -O cidr1.txt https://raw.githubusercontent.com/metowolf/iplist/master/data/special/china.txt
wget -O cidr2.txt https://raw.githubusercontent.com/17mon/china_ip_list/master/china_ip_list.txt

mkdir dist

go run main.go -i cidr1.txt,cidr2.txt -o dist/cidr.txt -m dist/Country.mmdb
