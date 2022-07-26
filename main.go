package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// This handles HTTPS requests using http CONNECT method
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

// This handles HTTP requests (usually OCSP)
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

func authenticateProxyUser(w http.ResponseWriter, r *http.Request, proxyUsername string, proxyPassword string) error {
	username, password := parseBasicAuth(r.Header.Get("Proxy-Authorization"))
	if username == proxyUsername && password == proxyPassword {
		return nil
	}
	// If the Authentication header is not present or is invalid, ask browser for creds
	w.Header().Set("Proxy-Authenticate", "Basic")
	return errors.New("Unauthorized")
}

func main() {
	// Parse proxy -arguments
	var proto = flag.String("proto", "https", "Protocol to listen for: i.e http/https")
	var port = flag.Int("port", 8888, "Port for proxy to bind: i.e 8888")
	var pemPath = flag.String("cert", "./Certs/intercept.crt", "cert pem file for TLS Server")
	var keyPath = flag.String("key", "./Certs/intercept.key", "key file for TLS Server")
	var proxyUsername = flag.String("username", "test", "Users username to authenticate to proxy")
	var proxyPassword = flag.String("password", "testPassword", "Users password to authenticate to proxy")

	flag.Parse()

	// Create http/s server
	server := &http.Server{
		Addr: fmt.Sprintf("0.0.0.0:%d", *port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Authenticate User to Proxy
			err := authenticateProxyUser(w, r, *proxyUsername, *proxyPassword)
			if err != nil {
				fmt.Printf("Proxy-Authorization required. rejected with status %d\n", http.StatusProxyAuthRequired)
				http.Error(w, "Unauthorized", http.StatusProxyAuthRequired)
				return
			}
			// Proxy the request
			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				handleHTTP(w, r)
			}
		}),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
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
