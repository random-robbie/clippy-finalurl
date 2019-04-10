package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Crawler struct {
	*http.Client
}

type LookupResult struct {

}





func newCrawler() *Crawler {

	// see http://tleyden.github.io/blog/2016/11/21/tuning-the-go-http-client-library-for-load-testing/

	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		panic(fmt.Sprintf("defaultRoundTripper not an *http.Transport"))
	}
	defaultTransport := *defaultTransportPointer // dereference it to get a copy of the struct that the pointer points to
	defaultTransport.MaxIdleConns = 1000
	defaultTransport.MaxIdleConnsPerHost = 10000
	defaultTransport.CloseIdleConnections()
	defaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{
		Timeout: time.Second * 15,
		Transport: defaultRoundTripper,


	}
	crawler := Crawler{
		Client: client,
	}
	return &crawler
}


func init() {

}


func (c Crawler) getFinalDestination(workStream chan string, resultStream chan map[string]string) (error) {
	url := <-workStream
	resp, err := c.Get(url)
	if err == nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		dest := resp.Request.URL.String()
		if dest != "" {
			resultStream <- map[string]string{url: dest}
		}
		return nil
	}
	return err
}

func (c Crawler) readInfile(fileName string) (map[string]bool, error) {
	log.Printf("reading file: %s", fileName)

	uniqLines := map[string]bool{}

	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if err != nil {
		return nil, err
	}
	i := 0
	for scanner.Scan() {
		i++
		text := scanner.Text()
		if text != "" {
			uniqLines[text] = true
		}
	}
	log.Printf("Read %d lines\n", len(uniqLines))
	return uniqLines, nil
}

func WriteResults(results chan string) {
	for {
		select {

		}
	}
}


func main() {
	crawler := newCrawler()
	inFileFlag := flag.String("f", "", "read file in")
	//urlFlag := flag.String("url", "", "check single url")
	outFileFlag := flag.String("o", "", "output file")
	threadsFlag := flag.Int("t", 10, "number of threads to use")
	//rateLimit := make(chan struct{}, *threadsFlag)
	//workStream := make(chan string)

	flag.Parse()

	if *inFileFlag != "" {

		urls, err := crawler.readInfile(*inFileFlag)
		if err != nil {
			log.Fatal(err)
		}

		wg := &sync.WaitGroup{}
		rateLimit := make(chan int, *threadsFlag)
		workStream := make(chan string, *threadsFlag)
		resultStream := make(chan map[string]string, len(urls))

		i := 0
		for url, _ := range urls{
			i++
			wg.Add(1)
			rateLimit <- 1
			workStream <- url

			go func() {
				defer wg.Done()
				err := crawler.getFinalDestination(workStream, resultStream)
				if err != nil {
					//
				}
				<- rateLimit
			}()
			fmt.Printf("\rchecked: %d/%d", i, len(urls))
		}
		close(rateLimit)
		wg.Wait()
		close(resultStream)
		dedupMap := map[string]string{}
		for r := range resultStream {
			for k, v := range r {
				dedupMap[k] = v
			}
		}

		if *outFileFlag != "" {
			f, err := os.OpenFile(*outFileFlag, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			for k, v := range dedupMap {
				_, err := f.WriteString(fmt.Sprintf("%s -> %s\n", k, v))
				if err != nil {
					log.Fatal(err)
				}
			}
		} else {
			for k, v := range dedupMap {
				log.Printf("% -> %s\n", k, v)
			}
		}

	}
}
