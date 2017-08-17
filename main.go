package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/alexflint/go-arg"
	"github.com/pkg/errors"

	. "github.com/cirocosta/sdns/lib"
)

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
		Address: ":",
		Debug:   true,
		Recursors: []string{
			"8.8.8.8",
			"8.8.4.4",
		},
	}
	s   Sdns
	err error
)

func main() {
	arg.Parse(args)

	s, err = NewSdns(*args)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"ERROR: Couldn't instantiate sdns - %s",
			errors.Cause(err))
		os.Exit(1)
	}

	// attach request handler func
	dns.HandleFunc(".", handleDnsRequest)

	// start server
	port := 1053
	server := &dns.Server{Addr: ":" + strconv.Itoa(port), Net: "udp"}
	fmt.Printf("Starting at %d\n", port)
	err := server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		fmt.Printf("Failed to start server: %s\n ", err.Error())
	}
}
