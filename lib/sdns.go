// lib implements the core functionality of 'sdns'.
package lib

type SdnsConfig struct {
	Port      int      `arg:"-p,env,help:port to listen to"`
	Address   string   `arg:"-a,env,help:address to bind to"`
	Debug     bool     `arg:"-d,env,help:turn debug mode on"`
	Recursors []string `arg:"-r,--recursor,help:list of recursors to honor"`
	Rules     []string `arg:"positional"`
}

// Sdns containers the internal representation of a
// configured set of domains.
type Sdns struct {
	domains        map[string]*Domain
	reverseDomains map[string]*Domain
}

// NewSdns instantiates a Sdns given a configuration.
func NewSdns(cfg SdnsConfig) (s Sdns, err error) {
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
func ResolveA(domain string) {
}

// ResolveNS lists the nameservers responsible
// for a given domain.
// For instance:
//	-	who are the nameservers responsible
//		for the domains of mysite.com?
func ResolveNS(domain string) {

}
