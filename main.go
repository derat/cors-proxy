// Copyright 2021 Daniel Erat <dan@erat.org>.
// All rights reserved.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/fcgi"
	"net/url"
	"os"
	"strings"
)

// forwardHeaders lists headers to preserve from the original request.
var forwardHeaders = []string{
	"Accept",
	"Accept-Encoding",
	"Accept-Language",
	"Content-Length",
	"Content-Type",
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flag]...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Forwards HTTP requests and adds 'Access-Control-Allow-Origin: *'.\n\n")
		flag.PrintDefaults()
	}
	addr := flag.String("addr", "localhost:8000", "host:port to listen on")
	fastcgi := flag.Bool("fastcgi", false, "Use FastCGI instead of listening on -addr")
	hosts := flag.String("hosts", "", "Comma-separated list of allowed forwarding hosts")
	refs := flag.String("referrers", "", "Comma-separated list of allowed referrer hosts")
	flag.Parse()

	hostList := strings.Split(*hosts, ",")
	refList := strings.Split(*refs, ",")

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		us := req.URL.Query().Get("url")
		log.Printf("%v %v", req.Method, us)

		status, err := func() (int, error) {
			if ref, err := url.Parse(req.Referer()); err != nil {
				return http.StatusBadRequest, fmt.Errorf("bad referrer %q: %v", req.Referer(), err)
			} else if !contains(refList, ref.Host) {
				return http.StatusBadRequest, fmt.Errorf("invalid referrer %q", req.Referer())
			}

			url, err := url.Parse(us)
			if err != nil {
				return http.StatusBadRequest, fmt.Errorf("bad URL %q: %v", us, err)
			} else if !contains(hostList, url.Host) {
				return http.StatusBadRequest, fmt.Errorf("invalid host in URL %q", us)
			}

			// I don't understand why, but if I just pass req.Body directly to http.NewRequest,
			// it looks like the body isn't getting sent.
			b, err := ioutil.ReadAll(req.Body)
			if err != nil {
				return http.StatusInternalServerError, err
			}
			fwd, err := http.NewRequest(req.Method, url.String(), bytes.NewReader(b))
			if err != nil {
				return http.StatusInternalServerError, err
			}
			for _, h := range forwardHeaders {
				fwd.Header.Set(h, req.Header.Get(h))
			}

			resp, err := http.DefaultClient.Do(fwd)
			if err != nil {
				return http.StatusInternalServerError, err
			}
			defer resp.Body.Close()

			// Copy all response headers and add the CORS header.
			for k, vs := range resp.Header {
				for _, v := range vs {
					w.Header().Add(k, v)
				}
			}
			w.Header().Set("Access-Control-Allow-Origin", "*")

			if resp.StatusCode != http.StatusOK {
				return http.StatusInternalServerError, fmt.Errorf("server returned %q", resp.Status)
			}
			if _, err := io.Copy(w, resp.Body); err != nil {
				return http.StatusInternalServerError, fmt.Errorf("failed copying response: %v", err)
			}
			return http.StatusOK, nil
		}()

		if status != http.StatusOK {
			http.Error(w, "Failed", status)
		}
		if err != nil {
			log.Printf("%v %v failed: %v", req.Method, us, err)
		}
	})

	if *fastcgi {
		log.Print("Listening for FastCGI requests")
		fcgi.Serve(nil, handler)
	} else {
		log.Print("Listening on ", *addr)
		srv := http.Server{Addr: *addr, Handler: handler}
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal("Serving failed: ", err)
		}
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
