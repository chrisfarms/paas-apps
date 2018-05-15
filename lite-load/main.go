package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/cloudfoundry-community/go-cfenv"
)

var (
	routeToSelf string
	port        int
)

func newHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   1000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     true,
	}
	return &http.Client{Transport: transport}
}

func request(client *http.Client, req *http.Request) error {
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	io.Copy(ioutil.Discard, res.Body)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		b, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("responded with code=%d body=%s", res.StatusCode, string(b))
	}
	return nil
}

func connection() {
	for {
		ms := time.Duration(rand.Intn(2000)) * time.Millisecond
		time.Sleep(ms)
		client := newHTTPClient()
		fakeSessionID := rand.Intn(100000)
		fakeEndpoints := []string{
			"api/account",
			"api/preferences",
			"api/transactions",
			"queue/notifications",
			"queue/metrics",
			"metrics",
		}
		fakeEndpoint := fakeEndpoints[rand.Intn(len(fakeEndpoints))]
		id := rand.Intn(1000)
		url := fmt.Sprintf("%s/%s/%d/%d", routeToSelf, fakeEndpoint, fakeSessionID, id)
		var err error
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}
		if err = request(client, req); err != nil {
			fmt.Fprintln(os.Stderr, "GET", url, err, "...will try again in 3s")
			time.Sleep(3 * time.Second)
		} else {
			fmt.Println("GET", url, "ok")
		}
		runtime.GC()
		debug.FreeOSMemory()
		time.Sleep(250 * time.Millisecond)
	}
}

func handler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if req.Method == "OPTIONS" {
		return
	}
	fmt.Fprintf(res, "Hello, World")
}

func main() {
	http.HandleFunc("/", handler)
	if cfenv.IsRunningOnCF() {
		app, err := cfenv.Current()
		if err != nil {
			log.Fatal(err)
		}
		for _, appURI := range app.ApplicationURIs {
			routeToSelf = fmt.Sprintf("https://%s", appURI)
		}
		port = app.Port
	} else {
		routeToSelf = fmt.Sprintf("http://127.0.0.1:8080")
		port = 8080
	}
	go connection()
	addr := fmt.Sprintf(":%d", port)
	fmt.Println("listen:", addr, "route:", routeToSelf)
	log.Fatal(http.ListenAndServe(addr, nil))
}
