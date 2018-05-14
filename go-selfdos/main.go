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
	routeToSelf   string
	port          int
	maxConns      int
	instanceIndex int
	appID         string
)

const (
	mbPerConnection = 0.2
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

func connection(id int) {
	fmt.Println("starting connection", id, "for instance", instanceIndex)
	ms := time.Duration(rand.Intn(10000)+250) * time.Millisecond
	time.Sleep(ms)
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
	url := fmt.Sprintf("%s/%s/%d/%d", routeToSelf, fakeEndpoint, fakeSessionID, id)
	var err error
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	if appID != "" {
		req.Header.Set("X-CF-APP-INSTANCE", fmt.Sprintf("%s:%d", appID, instanceIndex))
	}
	client := newHTTPClient()
	for {
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
	if req.URL.Path == "/" {
		fmt.Fprintf(res, "Hello, World")
		return
	}
	fmt.Fprintf(res, "Hello, \n")
	if f, ok := res.(http.Flusher); ok {
		f.Flush()
	}
	n := time.Duration(rand.Intn(3)+6) * time.Second
	time.Sleep(n)
	fmt.Fprintf(res, "World")
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
		maxConns = int(float64(app.Limits.Mem) / mbPerConnection)
		port = app.Port
		instanceIndex = app.Index
		appID = app.AppID
	} else {
		maxConns = 500
		routeToSelf = fmt.Sprintf("http://127.0.0.1:8080")
		port = 8080
	}
	go func() {
		time.Sleep(10 * time.Second)
		for i := 0; i < maxConns; i++ {
			go connection(i)
		}
	}()
	addr := fmt.Sprintf(":%d", port)
	fmt.Println("listen:", addr, "maxconns:", maxConns, "route:", routeToSelf, "instanceIndex:", instanceIndex, "appID", appID)
	log.Fatal(http.ListenAndServe(addr, nil))
}
