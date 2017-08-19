package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/pkg/errors"

	. "github.com/cirocosta/sdns/lib"
	util "github.com/cirocosta/sdns/util"
)

// config contains the structure for retrieval of
// the SDNS configuration from the command line.
type config struct {
	Port      int      `arg:"-p,env,help:port to listen to"`
	Address   string   `arg:"-a,env,help:address to bind to"`
	Debug     bool     `arg:"-d,env,help:turn debug mode on"`
	Recursors []string `arg:"-r,--recursor,help:list of recursors to honor"`
	Domains   []string `arg:"positional,help:list of domains"`
}

var (
	args = &config{
		Port:    1053,
		Debug:   true,
		Recursors: []string{
			"8.8.8.8",
			"8.8.4.4",
		},
	}
	sdnsConfig = SdnsConfig{}
	s          Sdns
	err        error
)

func main() {
	arg.MustParse(args)

	if len(args.Domains) > 0 {
		sdnsConfig.Domains = make([]*Domain, len(args.Domains))
		for idx, domainString := range args.Domains {
			domain := &Domain{}
			mapping, err := util.CsvStringToMap(domainString)
			if err != nil {
				fmt.Fprintf(os.Stderr,
					"ERROR: Malformed domain configuration - %s",
					errors.Cause(err))
				os.Exit(1)
			}

			name, present := mapping["domain"]
			if !present {
				fmt.Fprintf(os.Stderr,
					"ERROR: Malformed domain configuration. "+
						"A domain name must be present")
				os.Exit(1)
			}

			if present {
				domain.Name = name[0]
			}

			ips, present := mapping["ip"]
			if present {
				domain.Addresses = ips
			}

			nameservers, present := mapping["ns"]
			if present {
				domain.Nameservers = nameservers
			}

			sdnsConfig.Domains[idx] = domain
		}
	}

	sdnsConfig.Recursors = args.Recursors
	sdnsConfig.Debug = args.Debug
	sdnsConfig.Address = args.Address
	sdnsConfig.Port = args.Port

	s, err = NewSdns(sdnsConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"ERROR: Couldn't instantiate sdns - %s",
			errors.Cause(err))
		os.Exit(1)
	}

	err = s.Listen()
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"ERROR: Errored listening - %s",
			errors.Cause(err))
		os.Exit(1)
	}
}
