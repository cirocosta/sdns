// lib implements the core functionality of 'sdns'.
package lib

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
)

type SdnsConfig struct {
	Port      int
	Address   string
	Debug     bool
	Recursors []string
	Domains   map[string]*Domain
}

type SdnsContext struct {
	logger zerolog.Logger
}

// Sdns containers the internal representation of a
// configured set of domains.
type Sdns struct {
	domains        map[string]*Domain
	reverseDomains map[string]*Domain
	address        string
	recursors      []string
	logger         zerolog.Logger
}

// NewSdns instantiates a Sdns given a configuration.
func NewSdns(cfg SdnsConfig) (s Sdns, err error) {
	if cfg.Address == "" {
		err = errors.Errorf("an address must be specified")
		return
	}

	if cfg.Port == 0 {
		err = errors.Errorf("a port must be specified")
		return
	}

	if cfg.Debug {
		s.logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		s.logger = zerolog.New(os.Stderr)
	}

	err = s.Load(cfg)
	if err != nil {
		err = errors.Wrapf(err,
			"couldn't load internal configurations using config supplied.")
		return
	}

	s.address = fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
	return
}

// Load loads internal mappings using a configuration.
// This method is fired when the constructor is called
// but can also be used to perform hot reload.
// Note:	the address and port that the server listens
//		to cannot be modified. If so, it'll be ignored.
func (s *Sdns) Load(cfg SdnsConfig) (err error) {
	if len(cfg.Domains) == 0 {
		err = errors.Errorf("At list an empty domain list must be specified.")
	}

	return
}

func (s *Sdns) answerQuery(ctx *SdnsContext, m *dns.Msg) {
	var (
		rr     dns.RR
		domain *Domain
		found  bool
		err    error
	)

	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			domain, found = s.ResolveA(q.Name)
			if !found {
				continue
			}

			rr, err = dns.NewRR(fmt.Sprintf(
				"%s A %s", q.Name, domain.GetAddress()))
			if err != nil {
				ctx.logger.Error().
					Err(err).
					Msg("couldn't create RR")
				continue
			}
			m.Answer = append(m.Answer, rr)
		default:
			ctx.logger.Info().
				Uint16("query-type", q.Qtype).
				Msg("unsuported query type")
		}
	}
}

func (s *Sdns) handle(w dns.ResponseWriter, r *dns.Msg) {
	var (
		m   = dns.Msg{}
		ctx = SdnsContext{
			logger: s.logger.With().
				Str("id", xid.New().String()).
				Logger(),
		}
	)

	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		s.answerQuery(&ctx, &m)
	default:
		ctx.logger.Info().
			Int("opcode", r.Opcode).
			Msg("query for unsuported opcode")
	}

	w.WriteMsg(&m)
}

func (s *Sdns) Listen() {
	dns.HandleFunc(".", s.handle)
}

// Domain wraps the necessary information about a domain.
type Domain struct {
	// Name of the domain e.g.: mysite.com.
	// This field can also specify wildcards in
	// order to match any intended subdomain.
	// For instance: '*.mysite.com' would match
	//		 'haha.mysite.com'.
	Name string

	// Addresses is a list of IP addresses that
	// are meant to be resolved by the IP.
	Addresses []string

	// Nameservers is a list of nameservers that
	// are capable of resolving domains related
	// to 'Name'.
	Nameservers []string

	nextIdx uint64
	once    sync.Once
}

func (d *Domain) init() {
	d.nextIdx = uint64(time.Now().UnixNano())
}

// GetAddress returns a random address from the pool of
// addresses that it has.
func (d *Domain) GetAddress() string {
	d.once.Do(d.init)
	d.nextIdx++

	return d.Addresses[d.nextIdx%uint64(len(d.Addresses))]
}

// MatchesDomain verifies whether the domain (a) matches
// another.
func DomainMatches(a, b string) bool {
	return a == b
}

// ResolveA performs the job of resolving the
// IP address of a given service from a name.
// For instance:
//	-	what are the IPs of mysite.com ?
func (s *Sdns) ResolveA(name string) (domain *Domain, found bool) {
	return
}

// ResolveNS lists the nameservers responsible
// for a given domain.
// For instance:
//	-	who are the nameservers responsible
//		for the domains of mysite.com?
func (s *Sdns) ResolveNS(domain string) {

}
