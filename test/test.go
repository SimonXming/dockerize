package main

import (
	// "flag"
	// "fmt"
	"log"
	// "net"
	"net/http"
	// "net/url"
	// "os"
	// "strings"
	"sync"
	// "time"
	// "golang.org/x/net/context"
)

var (
	buildVersion string
	version      bool
	poll         bool
	wg           sync.WaitGroup
)

func main() {
	log.Println("Start waitForDependencies")
	http.Get("http://www.baidu.com/")
	// waitForDependencies()
	log.Println("End waitForDependencies")
}

func waitForDependencies() {
	urls := []string{
		"http://www.golang.org/",
		"http://www.google.com/",
		"http://www.somestupidname.com/",
	}
	for _, url := range urls {
		// Increment the WaitGroup counter.
		wg.Add(1)
		// Launch a goroutine to fetch the URL.
		go func(url string) {
			// Decrement the counter when the goroutine completes.
			// defer log.Println("After wg.Done", url)
			defer wg.Done()
			// defer log.Println("Before wg.Done", url)
			log.Println("Fetch the URL:", url)
			// Fetch the URL.
			http.Get(url)
		}(url)
	}
	// Wait for all HTTP fetches to complete.
	wg.Wait()
}
