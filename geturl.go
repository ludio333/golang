package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

var worker = runtime.NumCPU()
var respChan = make(chan interface{}, 1)

func gethtml(url string) {
	http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = time.Second * 5
	resp, err := http.Get(url)
	respChan <- url
	if err != nil {
		fmt.Println(err)
		respChan <- err
		return
	} else {
		respChan <- resp.Header["Server"]
		resp.Body.Close()
	}
}

func readfile(filename string) []string {
	list := []string{}
	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	br := bufio.NewReader(f)
	for {
		line, err := br.ReadString('\n')
		if err == io.EOF {
			fmt.Println("读取完毕")
			break
		} else if err != nil {
			fmt.Println("Error:", err)
			break
		}
		list = append(list, line)
	}
	return list
}

func uniq(list []string) []string {
	found := make(map[string]bool)
	j := 0
	for i, val := range list {
		if _, ok := found[val]; !ok {
			found[val] = true
			(list)[j] = (list)[i]
			j++
		}
	}
	list = (list)[:j]
	return list
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	urllist := []string{}
	urllist = readfile("tmp")
	urllist = uniq(urllist)
	for _, url := range urllist {
		go gethtml(url)
	}
	//fmt.Println(runtime.NumGoroutine())
	for i := 0; i < len(urllist)-1; i++ {
			fmt.Println(<-respChan, <-respChan)
	}
}
