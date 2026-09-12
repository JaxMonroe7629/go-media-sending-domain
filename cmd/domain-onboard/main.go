package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/example/media-sending-domain/domainclient"
)

func main() {
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "usage: domain-onboard [verify|check] sending.example.com")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}

	client, err := domainclient.New(os.Getenv("INFRAI_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	action, domain := flag.Arg(0), strings.ToLower(strings.TrimSpace(flag.Arg(1)))

	var status string
	switch action {
	case "verify":
		result, err := client.VerifyDomain(context.Background(), domain, domain+":domain-onboard")
		if err != nil {
			log.Fatal(err)
		}
		status = result.Verification.Status
	case "check":
		result, err := client.GetDomain(context.Background(), domain)
		if err != nil {
			log.Fatal(err)
		}
		status = result.Verification.Status
	default:
		flag.Usage()
		os.Exit(2)
	}

	fmt.Printf("domain=%s verification.status=%s\n", domain, status)
}
