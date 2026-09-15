// Command verify smoke-tests the build artifacts before they are deployed:
// every line of cidr.txt must parse, the line count must exceed a floor,
// and Country.mmdb must resolve known CN / non-CN addresses correctly.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/netip"
	"os"

	"github.com/oschwald/maxminddb-golang"
)

var (
	mustCN = []string{
		"223.5.5.5",       // AliDNS
		"114.114.114.114", // 114DNS
		"2400:3200::1",    // AliDNS IPv6
	}
	mustNotCN = []string{
		"8.8.8.8",              // Google
		"1.1.1.1",              // Cloudflare
		"2001:4860:4860::8888", // Google IPv6
	}
)

func main() {
	cidrPath := flag.String("cidr", "dist/cidr.txt", "path to cidr.txt")
	mmdbPath := flag.String("mmdb", "dist/Country.mmdb", "path to Country.mmdb")
	minLines := flag.Int("min-lines", 5000, "minimum number of CIDR lines")
	flag.Parse()

	if err := run(*cidrPath, *mmdbPath, *minLines); err != nil {
		fmt.Fprintln(os.Stderr, "verify:", err)
		os.Exit(1)
	}
	fmt.Println("verify: OK")
}

func run(cidrPath, mmdbPath string, minLines int) error {
	count, err := checkCIDRFile(cidrPath, minLines)
	if err != nil {
		return err
	}

	db, err := maxminddb.Open(mmdbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, ip := range mustCN {
		if err := checkLookup(db, ip, true); err != nil {
			return err
		}
	}
	for _, ip := range mustNotCN {
		if err := checkLookup(db, ip, false); err != nil {
			return err
		}
	}
	fmt.Printf("verify: %d CIDR lines, %d spot checks passed\n", count, len(mustCN)+len(mustNotCN))
	return nil
}

func checkCIDRFile(path string, minLines int) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if _, err := netip.ParsePrefix(line); err != nil {
			return 0, fmt.Errorf("%s: invalid CIDR %q: %w", path, line, err)
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	if count < minLines {
		return 0, fmt.Errorf("%s: only %d lines, want at least %d", path, count, minLines)
	}
	return count, nil
}

func checkLookup(db *maxminddb.Reader, ipStr string, wantCN bool) error {
	var record struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	err := db.Lookup(net.ParseIP(ipStr), &record)
	if err != nil {
		return err
	}
	isCN := record.Country.ISOCode == "CN"
	if isCN != wantCN {
		return errors.New(ipStr + ": resolved iso_code=" + record.Country.ISOCode + " (want CN=" + fmt.Sprint(wantCN) + ")")
	}
	return nil
}
