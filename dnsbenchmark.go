package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/miekg/dns"
)

// Runs concurrent dns A request to specified server every 6 seconds

type DNSQuery struct {
	Name string
	Type uint16
}

type queryResult struct {
	rtt        float64
	numQueries int
	numError   int
}

func (q DNSQuery) ExecDNSQuery(client *dns.Client, server string) (time.Duration, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(q.Name), q.Type)
	_, rtt, err := client.Exchange(m, server)
	return rtt, err
}

func main() {
	flag.Parse()
	if flag.NArg() != 3 {
		log.Printf("Usage: ./dnsbenchmark SERVER QPS DOMAIN_NAME")
		os.Exit(1)
	}

	args := flag.Args()

	server := args[0]
	qps, _ := strconv.Atoi(args[1])
	name := args[2]
	durationSeconds := 5.0
	query := DNSQuery{
		Name: name,
		Type: dns.TypeA,
	}

	var iteration int

	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case <-cancelChan:
			log.Printf("Caught exit signal")
			os.Exit(137)
		default:
			offset := 0.5
			sec := time.Duration(durationSeconds+offset) * time.Second
			end := time.Now().Add(sec)

			resultChan := make(chan queryResult)

			sleep := 1000 / qps
			for i := 0; i < qps; i++ {
				go func(i int) {
					var rttTotal time.Duration
					var numQueries int
					var numError int

					client := new(dns.Client)
					for time.Now().Before(end) {
						rtt, err := query.ExecDNSQuery(client, server)
						if err != nil {
							numError++
							log.Printf("%s\n", err)
						}
						rttTotal += rtt
						numQueries++
						time.Sleep(time.Second)
					}
					resultChan <- queryResult{float64(rttTotal) / float64(numQueries), numQueries, numError}
				}(i)

				time.Sleep(time.Duration(sleep) * time.Millisecond)
			}

			var rttAvg float64
			var numQueries int
			var numError int
			for i := 0; i < qps; i++ {
				res := <-resultChan
				rttAvg += res.rtt / float64(qps)
				numQueries += res.numQueries
				numError += res.numError
			}
			log.Printf("\n\nIteration %d:\n", iteration)
			log.Printf("number of queries: %d; avg latency: %.4f milliseconds; number of errors: %d; qps: %d\n\n", numQueries, float64(rttAvg)/float64(time.Millisecond), numError, numQueries/int(durationSeconds))
			iteration++
			log.Printf("Sleeping for 1 second. . .\n")
			time.Sleep(time.Second)
		}
	}
}
