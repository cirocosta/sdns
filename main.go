package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/alexflint/go-arg"
	"github.com/miekg/dns"
)

type SdnsConfig struct {
	Port      int      `arg:"-p,env,help:port to listen to"`
	Address   string   `arg:"-a,env,help:address to bind to"`
	Debug     bool     `arg:"-d,env,help:turn debug mode on"`
	Recursors []string `arg:"-r,--recursor,help:list of recursors to honor"`
	Rules     []string `arg:"positional"`
}

var (
	args = &SdnsConfig{Port: 53}
)

type Sdns struct {
	rules map[string]string
}

func NewSdns(cfg SdnsConfig) (s Sdns, err error) {
	return
}

var records = map[string]string{
	"test.service.": "192.168.0.2",
}

func parseQuery(m *dns.Msg) {
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			log.Printf("Query for %s\n", q.Name)
			ip := records[q.Name]
			if ip != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m)
	}

	w.WriteMsg(m)
}

func main() {
	arg.Parse(args)

	// attach request handler func
	dns.HandleFunc("service.", handleDnsRequest)

	// start server
	port := 1053
	server := &dns.Server{Addr: ":" + strconv.Itoa(port), Net: "udp"}
	log.Printf("Starting at %d\n", port)
	err := server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		log.Fatalf("Failed to start server: %s\n ", err.Error())
	}
}
