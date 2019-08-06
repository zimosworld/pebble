package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jmhodges/clock"
	"github.com/zimosworld/pebble/ca"
	"github.com/zimosworld/pebble/cmd"
	"github.com/zimosworld/pebble/db"
	"github.com/zimosworld/pebble/va"
	"github.com/zimosworld/pebble/wfe"
)

const (
    tlsDisabled = "PEBBLE_TLS_DISABLED"
)

type config struct {
	Pebble struct {
		ListenAddress           string
		ManagementListenAddress string
		HTTPPort                int
		TLSPort                 int
		Certificate             string
		PrivateKey              string
		OCSPResponderURL        string
	}
}

func main() {
	configFile := flag.String(
		"config",
		"test/config/pebble-config.json",
		"File path to the Pebble configuration file")
	strictMode := flag.Bool(
		"strict",
		false,
		"Enable strict mode to test upcoming API breaking changes")
	resolverAddress := flag.String(
		"dnsserver",
		"",
		"Define a custom DNS server address (ex: 192.168.0.56:5053 or 8.8.8.8:53).")
	flag.Parse()
	if *configFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Log to stdout
	logger := log.New(os.Stdout, "Pebble ", log.LstdFlags)
	logger.Printf("Starting Pebble ACME server")

	var c config
	err := cmd.ReadConfigFile(*configFile, &c)
	cmd.FailOnError(err, "Reading JSON config file into config structure")

	alternateRoots := 0
	alternateRootsVal := os.Getenv("PEBBLE_ALTERNATE_ROOTS")
	if val, err := strconv.ParseInt(alternateRootsVal, 10, 0); err == nil && val >= 0 {
		alternateRoots = int(val)
	}

	db := db.NewMemoryStore()
	ca := ca.New(logger, db, c.Pebble.OCSPResponderURL, alternateRoots)
	va := va.New(logger, c.Pebble.HTTPPort, c.Pebble.TLSPort, *strictMode, *resolverAddress)

	wfeImpl := wfe.New(logger, db, va, ca, *strictMode)
	muxHandler := wfeImpl.Handler()

	tlsDisabled := os.Getenv(tlsDisabled)

    logger.Printf("Pebble running, listening on: %s\n", c.Pebble.ListenAddress)

    switch tlsDisabled {
	case "1", "true", "True", "TRUE":
	    err = http.ListenAndServe(
                c.Pebble.ListenAddress,
                muxHandler)
	default:
	    err = http.ListenAndServeTLS(
        		c.Pebble.ListenAddress,
        		c.Pebble.Certificate,
        		c.Pebble.PrivateKey,
        		muxHandler)
	}

	cmd.FailOnError(err, "Calling ListenAndServeTLS()")
}
