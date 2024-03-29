package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

type stringArrayFlag []string

func (i *stringArrayFlag) String() string {
	return strings.Join(*i, " ")
}

func (i *stringArrayFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var flagRecords stringArrayFlag

func main() {

	zoneId := flag.String("zone", os.Getenv("ZONE_ID"), "Cloudflare zone identifier")
	apiToken := flag.String("token", os.Getenv("CLOUDFLARE_API_TOKEN"), "Cloudflare API token")
	resolverEndpoint := flag.String("resolver", os.Getenv("RESOLVER"), "Public IP resolver endpoint")
	flag.Var(&flagRecords, "flagRecords", "list of identifiers for flagRecords to update")
	loopPeriod := flag.Duration("period", time.Minute*5, "delay between update loops")
	loop := flag.Bool("loop", false, "whether to loop as a service")
	flag.Parse()

	if resolverEndpoint == nil || *resolverEndpoint == "" {
		log.Fatal("resolver endpoint must be specified")
	}

	if zoneId == nil || *zoneId == "" {
		log.Fatal("zone id must be specified")
	}

	if apiToken == nil || *apiToken == "" {
		log.Fatal("api token must be specified")
	}

	envRecords := strings.Split(os.Getenv("RECORDS"), ",")

	var records []string
	// cli flags take precedence
	if len(flagRecords) > 0 {
		records = flagRecords
	} else {
		records = envRecords
	}

	if len(records) < 1 {
		log.Fatal("at least one record must be specified")
	}

	resolver := NewIPResolver(*resolverEndpoint)
	ips := resolver.Resolve()
	cf, err := cloudflare.NewWithAPIToken(*apiToken)
	if err != nil {
		log.Fatal("couldn't create cloudflare client", err)
	}

	// run update at least once
	ctx, cancel := context.WithCancel(context.Background())
	err = updateRecords(ctx, cf, *zoneId, records, ips)
	if err != nil {
		log.Fatal(err)
	}

	// set up interrupt handler to cancel context
	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		select {
		case <-sigs:
			log.Println("received interrupt")
			cancel()
		}
	}()

	// loop for every ticker tick or until context is cancelled
	ticker := time.NewTicker(*loopPeriod)
	for *loop {
		select {
		case <-ticker.C:
			err := updateRecords(ctx, cf, *zoneId, records, ips)
			if err != nil {
				log.Fatal(err)
			}
		case <-ctx.Done():
			log.Println("shutting down cf-ddns")
			return
		}
	}
}

func updateRecords(ctx context.Context, cf *cloudflare.API, zoneId string, records []string, ips ResolutionResult) error {
	for _, record := range records {
		res, err := cf.GetDNSRecord(ctx, cloudflare.ResourceIdentifier(zoneId), record)
		if err != nil {
			log.Printf("couldn't get record %s: %s", record, err)
			continue
		}

		params := cloudflare.UpdateDNSRecordParams{
			Type:    res.Type,
			Name:    res.Name,
			ID:      record,
			Proxied: res.Proxied,
			Comment: fmt.Sprintf("cf-ddns: %s", time.Now().Format(time.RFC3339)),
		}
		if res.Type == "A" {
			if ips.ipv4 == "" {
				log.Println("skipping empty ipv4 content")
				continue
			}
			params.Content = ips.ipv4
		} else if res.Type == "AAAA" {
			if ips.ipv6 == "" {
				log.Println("skipping empty ipv6 content")
				continue
			}
			params.Content = ips.ipv6
		} else {
			log.Printf("record %s not of type A or AAAA", record)
			continue
		}
		res, err = cf.UpdateDNSRecord(ctx, cloudflare.ResourceIdentifier(zoneId), params)
		if err != nil {
			log.Printf("couldn't update record %s: %s", record, err)
			continue
		}
		log.Printf("updated record %s (%s) with %s", res.Name, res.Type, res.Content)
	}
	return nil
}

func NewIPResolver(endpoint string) *IPResolver {
	d4 := &Dialer4{d: &net.Dialer{}}
	c4 := &http.Client{
		Transport: &http.Transport{
			DialContext: d4.DialContext,
		},
		Timeout: time.Second * 3,
	}
	d6 := &Dialer6{d: &net.Dialer{}}
	c6 := &http.Client{
		Transport: &http.Transport{
			DialContext: d6.DialContext,
		},
		Timeout: time.Second * 3,
	}
	return &IPResolver{
		client4:  c4,
		client6:  c6,
		endpoint: endpoint,
	}
}

type IPResolver struct {
	client4  *http.Client
	client6  *http.Client
	endpoint string
}

type ResolutionResult struct {
	ipv4 string
	ipv6 string
}

func (r *IPResolver) Resolve() ResolutionResult {
	result := ResolutionResult{}
	res, err := r.client4.Get(r.endpoint)
	if err != nil {
		log.Printf("couldn't resolve ipv4: %s", err)
	} else {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Printf("couldn't read ipv4 result body: %s", err)
		} else {
			log.Printf("resolved ipv4: %s", string(body))
			result.ipv4 = string(body)
		}
	}
	res, err = r.client6.Get(r.endpoint)
	if err != nil {
		log.Printf("couldn't resolve ipv6: %s", err)
	} else {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Printf("couldn't read ipv6 result body: %s", err)
		} else {
			log.Printf("resolved ipv6: %s", string(body))
			result.ipv6 = string(body)
		}
	}
	return result
}

type Dialer4 struct {
	d *net.Dialer
}

func (d *Dialer4) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.d.DialContext(ctx, "tcp4", address)
}

func (d *Dialer4) Dial(network, address string) (net.Conn, error) {
	return d.d.Dial("tcp4", address)
}

type Dialer6 struct {
	d *net.Dialer
}

func (d *Dialer6) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.d.DialContext(ctx, "tcp6", address)
}

func (d *Dialer6) Dial(network, address string) (net.Conn, error) {
	return d.d.Dial("tcp6", address)
}
