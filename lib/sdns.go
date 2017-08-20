// lib implements the core functionality of 'sdns'.
package lib

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// SdnsConfig configures SDNS.
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
	exactDomains    map[string]*Domain
	wildcardDomains map[string]*Domain
	reverseDomains  map[string]*Domain
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

	s.client = &dns.Client{SingleInflight: true}
	s.recursors = cfg.Recursors
	s.address = fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)

	return
}

// Load loads internal mappings using a configuration.
// This method is fired when the constructor is called
// but can also be used to perform hot reload.
// Note:	the address and port that the server listens
//		to cannot be modified. If so, it'll be ignored.
func (s *Sdns) Load(cfg SdnsConfig) (err error) {
	s.exactDomains = make(map[string]*Domain)
	s.wildcardDomains = make(map[string]*Domain)

	if len(cfg.Domains) == 0 {
		return
	}

	for _, domain := range cfg.Domains {
		if domain.Name[0] == '*' {
			if domain.Name[1] != '.' {
				err = errors.Errorf("malformed domain name. " +
					"'*' must be followed by '.'")
				return
			}
			s.wildcardDomains[domain.Name[1:]] = domain
		} else {
			s.exactDomains[domain.Name] = domain
		}

		s.logger.Debug().
			Str("domain", domain.Name).
			Strs("addresses", domain.Addresses).
			Strs("nameservers", domain.Nameservers).
			Msg("loaded")
	}

	return
}

func (s *Sdns) recurse(ctx *SdnsContext, m *dns.Msg, server string) (in *dns.Msg, err error) {
	var (
		rtt time.Duration
		rm  = &dns.Msg{Question: m.Question}
	)

	rm.RecursionDesired = true

	ctx.logger.Info().
		Str("server", server).
		Msg("recursing question")

	in, rtt, err = s.client.Exchange(rm, server)
	if err != nil {
		err = errors.Wrapf(err,
			"errored forwarding msg %+v",
			*rm)
		return
	}

	ctx.logger.Info().
		Str("server", server).
		Dur("duration", rtt).
		Msg("recursion finished")

	return
}

var (
	ErrDomainNotFound       = errors.Errorf("Domain not found")
	ErrNoQuestions          = errors.Errorf("No questions provided")
	ErrUnsupportedQueryType = errors.Errorf("Query type not support")
)

func (s *Sdns) answerNS(ctx *SdnsContext, m *dns.Msg) (err error) {
	var (
		name string = m.Question[0].Name
		rr   dns.RR
	)

	s.logger.Info().
		Str("name", name).
		Str("query", "NS").
		Msg("looking for domain")

	domain, found := s.FindDomainFromName(strings.TrimRight(name, "."))
	if !found {
		err = ErrDomainNotFound
		return
	}

	for _, ns := range domain.Nameservers {
		rr, err = dns.NewRR(fmt.Sprintf("%s NS %s", name, ns))
		if err != nil {
			err = errors.Wrapf(err, "Couldn't create RR msg")
			return
		}
		m.Answer = append(m.Answer, rr)
	}
	return
}

func (s *Sdns) answerA(ctx *SdnsContext, m *dns.Msg) (err error) {
	var (
		name string = m.Question[0].Name
		rr   dns.RR
	)

	s.logger.Info().
		Str("name", name).
		Str("query", "A").
		Msg("looking for domain")

	domain, found := s.FindDomainFromName(strings.TrimRight(name, "."))
	if !found {
		err = ErrDomainNotFound
		return
	}

	rr, err = dns.NewRR(fmt.Sprintf(
		"%s A %s", name, domain.GetAddress()))
	if err != nil {
		err = errors.Wrapf(err, "Couldn't create RR msg")
		return
	}
	m.Answer = append(m.Answer, rr)
	return
}

func (s *Sdns) answerQuery(ctx *SdnsContext, m *dns.Msg) (err error) {
	if len(m.Question) == 0 {
		err = ErrNoQuestions
		return
	}

	switch m.Question[0].Qtype {
	case dns.TypeA:
		err = s.answerA(ctx, m)
	case dns.TypeNS:
		err = s.answerNS(ctx, m)
	default:
		err = ErrUnsupportedQueryType
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
				Uint16("id", r.Id).
				Logger(),
		}
	)

	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		err = s.answerQuery(&ctx, &m)
		if err != nil {
			s.logger.Warn().
				Err(err).
				Msg("couldn't answer right away")
		}

		switch err {
		case ErrUnsupportedQueryType:
			fallthrough
		case ErrDomainNotFound:
			var in *dns.Msg

			s.logger.Info().
				Strs("recursors", s.recursors).
				Msg("starting to recurse")

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

// FindDomainFromName performs the job of resolving the
// IP address of a given service from a name.
// For instance:
//	-	what are the IPs of mysite.com ?
func (s *Sdns) FindDomainFromName(name string) (domain *Domain, found bool) {
	var (
		strippedDomain string
		domainFound    interface{}
	)

	if name == "" {
		return
	}

	domainFound, found = s.exactDomains[name]
	if !found {
		lastDomainNdx := strings.IndexByte(name, '.')
		if lastDomainNdx < 0 {
			return
		}

		strippedDomain = name[lastDomainNdx:]
		domainFound, found = s.wildcardDomains[strippedDomain]
	}

	if domainFound != nil {
		domain = domainFound.(*Domain)
	}

	return
}
