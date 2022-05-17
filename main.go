package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Proxying tunnel %s for %s\n", r.Host, r.RemoteAddr)
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func handleHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Printf("Proxying http %s for %s\n", req.Host, req.RemoteAddr)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func main() {
	// Parse proxy -arguments
	var proto = flag.String("proto", "https", "Protocol to listen for: i.e http/https")
	var port = flag.Int("port", 8888, "Port for proxy to bind: i.e 8888")
	var pemPath = flag.String("cert", "./Certs/intercept.crt", "cert pem file for TLS Server")
	var keyPath = flag.String("key", "./Certs/intercept.key", "key file for TLS Server")

	flag.Parse()

	// Create http/s server
	server := &http.Server{
		Addr: fmt.Sprintf("0.0.0.0:%d", *port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				handleHTTP(w, r)
			}
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	// Start server
	switch *proto {
	case "https":
		fmt.Printf("go-proxy https listening on port %d\n", *port)
		log.Fatal(server.ListenAndServeTLS(*pemPath, *keyPath))
	case "http":
		fmt.Printf("go-proxy http listening on port %d\n", *port)
		log.Fatal(server.ListenAndServe())
	default:
		log.Fatal(fmt.Sprintf("Unrecognized protocol %s", *proto))
	}
}
