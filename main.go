package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/spf13/pflag"
	"go4.org/netipx"
)

func main() {
	var (
		cidrSource []string
		cidrOutput string
		mmdbOutput string
	)

	pflag.StringSliceVarP(&cidrSource, "source", "i", nil, "CIDR source files")
	pflag.StringVarP(&cidrOutput, "cidr-output", "o", "cidr.txt", "CIDR output")
	pflag.StringVarP(&mmdbOutput, "mmdb-output", "m", "Country.mmdb", "MMDB output")
	pflag.CommandLine.SortFlags = false
	pflag.Parse()

	if len(cidrSource) < 1 {
		log.Fatal("at least one cidr source file is required")
	}

	data, err := merge(cidrSource)
	if err != nil {
		log.Fatalf("merge cidr: %v", err)
	}

	if err := saveFile(data, cidrOutput); err != nil {
		log.Fatal(err)
	}

	if err := buildMMDB(data, mmdbOutput); err != nil {
		log.Fatal(err)
	}
}

func merge(source []string) ([]*net.IPNet, error) {
	builder := &netipx.IPSetBuilder{}

	for _, file := range source {
		if err := readSource(file, builder); err != nil {
			return nil, err
		}
	}

	set, err := builder.IPSet()
	if err != nil {
		return nil, err
	}

	prefixes := set.Prefixes()
	if len(prefixes) == 0 {
		return nil, errors.New("no valid CIDR in sources")
	}
	ips := make([]*net.IPNet, 0, len(prefixes))
	for _, p := range prefixes {
		ips = append(ips, &net.IPNet{
			IP:   p.Addr().AsSlice(),
			Mask: net.CIDRMask(p.Bits(), p.Addr().BitLen()),
		})
	}
	return ips, nil
}

func readSource(file string, builder *netipx.IPSetBuilder) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for lineno := 1; scanner.Scan(); lineno++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prefix, err := netip.ParsePrefix(line)
		if err != nil {
			return fmt.Errorf("%s:%d: %w", file, lineno, err)
		}
		if prefix.Addr().Is4In6() {
			return fmt.Errorf("%s:%d: 4-in-6 address %q is not supported", file, lineno, line)
		}
		builder.AddPrefix(prefix.Masked())
	}
	return scanner.Err()
}

func saveFile(data []*net.IPNet, output string) error {
	return atomicWrite(output, func(f *os.File) error {
		writer := bufio.NewWriter(f)
		for _, v := range data {
			writer.WriteString(v.String())
			writer.WriteByte('\n')
		}
		return writer.Flush()
	})
}

// atomicWrite writes output via a temp file in the same directory and
// renames it into place, so a failure never leaves a truncated file behind.
func atomicWrite(output string, write func(*os.File) error) error {
	dir := filepath.Dir(output)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(output)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename

	if err := write(tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), output)
}

func buildMMDB(data []*net.IPNet, output string) error {
	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType: "GeoIP2-Country",
		Description: map[string]string{
			"en":    "China IP database (merged from public sources)",
			"zh-CN": "中国大陆 IP 库（多源合并）",
		},
		Languages:  []string{"de", "en", "es", "fr", "ja", "pt-BR", "ru", "zh-CN"},
		RecordSize: 24,
	})
	if err != nil {
		return err
	}

	// https://github.com/Hackl0us/GeoIP2-CN/blob/c053afa7ef3d092b1ea84aa229fe035a49fe3603/main.go#L64
	// https://dev.maxmind.com/geoip/docs/databases/city-and-country
	// https://dev.maxmind.com/static/pdf/GeoLite2-and-GeoIP2-Precision-Web-Services-Comparison.pdf
	const chinaGeoNameID = 1814991
	dataType := mmdbtype.Map{
		"country": mmdbtype.Map{
			"geoname_id":           mmdbtype.Uint32(chinaGeoNameID),
			"is_in_european_union": mmdbtype.Bool(false),
			"iso_code":             mmdbtype.String("CN"),
			"names": mmdbtype.Map{
				"de":    mmdbtype.String("China"),
				"en":    mmdbtype.String("China"),
				"es":    mmdbtype.String("China"),
				"fr":    mmdbtype.String("Chine"),
				"ja":    mmdbtype.String("中国"),
				"pt-BR": mmdbtype.String("China"),
				"ru":    mmdbtype.String("Китай"),
				"zh-CN": mmdbtype.String("中国"),
			},
		},
	}
	for _, v := range data {
		if err := writer.Insert(v, dataType); err != nil {
			return fmt.Errorf("insert %s: %w", v, err)
		}
	}
	return atomicWrite(output, func(f *os.File) error {
		_, err := writer.WriteTo(f)
		return err
	})
}
