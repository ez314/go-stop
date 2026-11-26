package main

import (
	"flag"
	"fmt"
	"gostop/internal/server"
	"gostop/web"
	"io/fs"
	"log"
	"net"
	"net/http"
)

func listenIPv4(preferredPort int) (net.Listener, int, error) {
	for p := preferredPort; p < preferredPort+20; p++ {
		ln, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", p))
		if err == nil {
			return ln, p, nil
		}
	}
	ln, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		return nil, 0, err
	}
	addr := ln.Addr().(*net.TCPAddr)
	return ln, addr.Port, nil
}

func main() {
	preferredPort := flag.Int("port", 8080, "HTTP port to listen on")
	noOpen := flag.Bool("no-open", false, "Do not automatically open the browser on startup")
	flag.Parse()

	ln, port, err := listenIPv4(*preferredPort)
	if err != nil {
		log.Fatalf("failed to bind IPv4 port: %v", err)
	}

	ips := server.GetLANIPs()
	var lanURLs []string
	for _, ip := range ips {
		lanURLs = append(lanURLs, fmt.Sprintf("http://%s:%d", ip, port))
	}
	lanURL := fmt.Sprintf("http://localhost:%d", port)
	if len(lanURLs) > 0 {
		lanURL = lanURLs[0]
	}

	room := server.NewRoom(lanURL, lanURLs)

	staticFS, err := fs.Sub(web.Assets, ".")
	if err != nil {
		log.Fatalf("failed to mount embedded web assets: %v", err)
	}

	fileServer := http.FileServer(http.FS(staticFS))
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		server.ServeWS(room, w, r)
	})
	mux.HandleFunc("/qr", server.ServeQR(lanURL))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	})

	server.PrintBanner(port, ips)

	if !*noOpen {
		go server.OpenBrowser(fmt.Sprintf("http://localhost:%d", port))
	}

	if err := http.Serve(ln, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
