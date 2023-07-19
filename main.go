package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"strings"

	"github.com/EvilSuperstars/go-cidrman"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/spf13/pflag"
)

func main() {
	var (
		cidrSource []string
		cidrOutput string
		mmdbOutput string
		showHelp   bool
	)

	pflag.StringSliceVarP(&cidrSource, "source", "i", nil, "CIDR source files")
	pflag.StringVarP(&cidrOutput, "cidr-output", "o", "cidr.txt", "CIDR ouput")
	pflag.StringVarP(&mmdbOutput, "mmdb-output", "m", "Country.mmdb", "MMDB ouput")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show usage")
	pflag.CommandLine.SortFlags = false
	pflag.Parse()

	if showHelp {
		pflag.Usage()
		os.Exit(0)
	}

	if len(cidrSource) < 1 {
		log.Fatal("at least one cidr source file is required")
	}

	data, err := merge(cidrSource)
	if err != nil {
		log.Fatal("merge cidr: " + err.Error())
	}

	if err := saveFile(data, cidrOutput); err != nil {
		log.Fatal(err)
	}

	if err := buildMMDB(data, mmdbOutput); err != nil {
		log.Fatal(err)
	}
}

func merge(source []string) ([]*net.IPNet, error) {
	cidrMap := make(map[string]struct{})

	for _, file := range source {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			cidrMap[line] = struct{}{}
		}
		f.Close()
	}

	ips := make([]*net.IPNet, 0, len(cidrMap))
	for cidr := range cidrMap {
		_, c, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Println(err)
			continue
		}
		ips = append(ips, c)
	}
	return cidrman.MergeIPNets(ips)
}

func saveFile(data []*net.IPNet, output string) error {
	outputF, err := os.Create(output)
	if err != nil {
		return err
	}
	defer outputF.Close()

	writer := bufio.NewWriter(outputF)
	for _, v := range data {
		writer.WriteString(v.String() + "\n")
	}
	return writer.Flush()
}

func buildMMDB(data []*net.IPNet, output string) error {
	outputF, err := os.Create(output)
	if err != nil {
		return err
	}
	defer outputF.Close()

	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType: "GeoIP2-Country",
		RecordSize:   24,
	})
	if err != nil {
		return err
	}

	// https://github.com/Hackl0us/GeoIP2-CN/blob/c053afa7ef3d092b1ea84aa229fe035a49fe3603/main.go#L64
	// https://dev.maxmind.com/geoip/docs/databases/city-and-country
	// https://dev.maxmind.com/static/pdf/GeoLite2-and-GeoIP2-Precision-Web-Services-Comparison.pdf
	dataType := mmdbtype.Map{
		"country": mmdbtype.Map{
			"geoname_id":           mmdbtype.Uint32(1814991),
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
			log.Println("fail to insert " + v.String())
		}
	}
	_, err = writer.WriteTo(outputF)
	return err
}
