package actions

/*
import (
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"

	"log"
	"os"
)

func TorClient(addr string) *http.Client {

	// Dialer
	dialer, err := proxy.SOCKS5("tcp", addr, nil, proxy.Direct)

	if err != nil {
		log.Fatal(err)
	}

	// HTTP transport
	transport := &http.Transport{Dial: dialer.Dial}

	// Client
	client := &http.Client{Transport: transport}

	return client
}
*/
