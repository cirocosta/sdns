// lib implements the core functionality of 'sdns'.
package lib

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/armon/go-radix"
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
	Domains   []*Domain
}

// SdnsContext wraps a context that gets passed
// through the methods that are responsible
// for responding to queries.
type SdnsContext struct {
	logger zerolog.Logger
}

// Sdns containers the internal representation of a
// configured set of domains.
type Sdns struct {
	exactDomains    *radix.Tree
	wildcardDomains *radix.Tree
	reverseDomains  *radix.Tree
	address         string
	recursors       []string
	logger          zerolog.Logger
	client          *dns.Client
}

// NewSdns instantiates a Sdns given a configuration.
func NewSdns(cfg SdnsConfig) (s Sdns, err error) {
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

	s.client = &dns.Client{
		SingleInflight: true,
	}
	s.recursors = cfg.Recursors
	s.address = fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
	return
}

// TODO remove the use of 'reverse' in favor
// of an inverted radix lookup
func Reverse(s string) string {
	n := len(s)
	runes := make([]rune, n)
	for _, rune := range s {
		n--
		runes[n] = rune
	}
	return string(runes[n:])
}

// Load loads internal mappings using a configuration.
// This method is fired when the constructor is called
// but can also be used to perform hot reload.
// Note:	the address and port that the server listens
//		to cannot be modified. If so, it'll be ignored.
func (s *Sdns) Load(cfg SdnsConfig) (err error) {
	var (
		lookupDomain string
	)

	s.exactDomains = radix.New()
	s.wildcardDomains = radix.New()

	if len(cfg.Domains) == 0 {
		return
	}

	for _, domain := range cfg.Domains {
		lookupDomain = strings.TrimRight(Reverse(domain.Name), "*")

		if lookupDomain[len(lookupDomain)-1:] == "." {
			s.wildcardDomains.Insert(lookupDomain, domain)
		} else {
			s.exactDomains.Insert(lookupDomain, domain)
		}

		s.logger.Debug().
			Str("domain", domain.Name).
			Str("lookupDomain", lookupDomain).
			Strs("addresses", domain.Addresses).
			Strs("nameservers", domain.Nameservers).
			Msg("loaded")
	}

	return
}

func (s *Sdns) recurse(ctx *SdnsContext, m *dns.Msg, server string) (in *dns.Msg, err error) {
	var rtt time.Duration

	ctx.logger.Info().
		Str("server", server).
		Msg("recursion started")

	var rm = &dns.Msg{}
	rm.Question = m.Question
	rm.RecursionDesired = true

	in, rtt, err = s.client.Exchange(rm, server)
	if err != nil {
		err = errors.Wrapf(err, "errored recursing msg %+v", *rm)
		return
	}

	ctx.logger.Info().
		Str("server", server).
		Dur("duration", rtt).
		Msg("recursion finished")
	return
}

var (
	ErrARecordNotFound = errors.Errorf("no domain for A record")
)

func (s *Sdns) answerQuery(ctx *SdnsContext, m *dns.Msg) (err error) {
	var (
		rr     dns.RR
		domain *Domain
		found  bool
		q      dns.Question
	)

	if len(m.Question) == 0 {
		err = errors.Errorf("no questions provided")
		return
	}

	q = m.Question[0]
	switch q.Qtype {
	case dns.TypeA:
		domain, found = s.ResolveA(strings.TrimRight(q.Name, "."))
		if !found {
			ctx.logger.Info().
				Str("domain", q.Name).
				Msg("not found")
			err = ErrARecordNotFound
			return
		}

		rr, err = dns.NewRR(fmt.Sprintf(
			"%s A %s", q.Name, domain.GetAddress()))
		if err != nil {
			err = errors.Wrapf(err, "Couldn't create RR msg")
			return
		}
		m.Answer = append(m.Answer, rr)
	default:
		err = errors.Errorf("Unsuported query type %d", q.Qtype)
		return
	}

	return
}

func (s *Sdns) handle(w dns.ResponseWriter, r *dns.Msg) {
	var (
		err error
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
		err = s.answerQuery(&ctx, &m)
		s.logger.Error().Err(err).Msg("couldn't answer right away")

		switch err {
		case ErrARecordNotFound:
			var in *dns.Msg

			s.logger.Info().Strs("recursors", s.recursors).Msg("recursing")

			for _, server := range s.recursors {
				in, err = s.recurse(&ctx, &m, server)
				if err != nil {
					ctx.logger.Error().
						Err(err).
						Str("server", server).
						Msg("errored recursing")
					continue
				}

				m.Answer = in.Answer
				break
			}
		default:
			ctx.logger.Error().
				Err(err).
				Msg("couldn't answer query")
		}
	default:
		ctx.logger.Info().
			Int("opcode", r.Opcode).
			Msg("query for unsuported opcode")
	}

	w.WriteMsg(&m)
}

func (s *Sdns) Listen() (err error) {
	dns.HandleFunc(".", s.handle)

	server := &dns.Server{Addr: s.address, Net: "udp"}

	err = server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		err = errors.Wrapf(err,
			"errored listening on address %s",
			s.address)
		return
	}

	return
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
	var (
		strippedDomain string
		domainFound    interface{}
	)

	if name == "" {
		return
	}

	name = Reverse(name)

	s.logger.Info().
		Str("key", name).
		Msg("looking for exact match")

	domainFound, found = s.exactDomains.Get(name)
	if !found {
		lastDomainNdx := strings.LastIndex(name, ".")
		if lastDomainNdx < 0 {
			return
		}

		strippedDomain = name[:lastDomainNdx+1]

		s.logger.Info().
			Str("key", strippedDomain).
			Msg("looking for wildcard match")

		domainFound, found = s.wildcardDomains.Get(strippedDomain)
	}

	if domainFound != nil {
		domain = domainFound.(*Domain)
	}

	return
}

// ResolveNS lists the nameservers responsible
// for a given domain.
// For instance:
//	-	who are the nameservers responsible
//		for the domains of mysite.com?
func (s *Sdns) ResolveNS(domain string) {

}
