package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
)

type Chunk struct {
	seq  int
	data []byte
}

func parseOptions() map[string]string {
	pn := flag.String("n", "1", "Number of Download Channels")
	px := flag.String("X", "GET", "HTTP Request Method")
	po := flag.String("o", "", "Output File Name")
	flag.Parse()

	result := map[string]string{
		"number":  *pn,
		"method":  *px,
		"outfile": *po,
	}
	return result
}

func hasAcceptRangesBytes(url string) (http.Header, string) {
	res, err := http.Head(url)
	if err != nil {
		panic(err)
	}

	acceptRanges := res.Header.Get("Accept-Ranges")

	if acceptRanges == "bytes" {
		return res.Header, "true"
	}

	return res.Header, "false"
}

func makeRanges(num int, length int) []string {
	var result []string

	// 整数同士の除算の結果は整数
	div := length / num
	s := 0
	e := div
	for length > 0 {
		str := fmt.Sprintf("bytes=%d-%d", s, e)
		s = e + 1
		length -= div
		if length < 0 {
			e = s + div + length
		} else {
			e = s + div
		}
		result = append(result, str)
	}

	return result
}

func getHTTP(url string, hdr http.Header, opts map[string]string) {
	n, err := strconv.Atoi(opts["number"])
	if err != nil {
		panic(err)
	}
	lStr := hdr.Get("ContentLength")
	var l = 0
	if len(lStr) > 0 {
		l, err = strconv.Atoi(lStr)
		if err != nil {
			panic(err)
		}
	}
	ranges := makeRanges(n, l)

	var fileName = ""
	if opts["outfile"] != "" {
		fileName = opts["outfile"]
	} else {
		fileName = hdr.Get("Contents-Disposition")
	}

	if len(fileName) == 0 {
		fileName = "index.html"
	}

	if n < 2 {
		res, err := http.Get(url)
		if err != nil {
			panic(err)
		}

		data, err := ioutil.ReadAll(res.Body)
		err = res.Body.Close()
		if err != nil {
			panic(err)
		}

		err = ioutil.WriteFile(fileName, data, 0644)
		if err != nil {
			panic(err)
		}
	} else {
		// WIP
		waitGroup := new(sync.WaitGroup)
		queue := make(chan Chunk, n)

		for i := 0; i < n; i++ {
			waitGroup.Add(1)
			go func(seq int, waitGroup *sync.WaitGroup) {
				defer waitGroup.Done()

				req, _ := http.NewRequest("GET", url, nil)
				req.Header.Set("Range", ranges[seq])
				client := new(http.Client)
				res, err := client.Do(req)
				if err != nil {
					panic(err)
				}

				data, err := ioutil.ReadAll(res.Body)
				res.Body.Close()

				chunk := Chunk{seq: seq, data: data}
				queue <- chunk
			}(i, waitGroup)
		}
		waitGroup.Wait()
		close(queue)

	}
}

func postHTTP(url string, hdr http.Header, opts map[string]string) {
	// WIP
}

func main() {
	// 引数取得
	opts := parseOptions()
	// ターゲットURL
	url := flag.Args()[0]

	var hdr http.Header
	hdr, opts["AcceptRangesBytes"] = hasAcceptRangesBytes(url)

	switch opts["method"] {
	case "GET":
		getHTTP(url, hdr, opts)

	case "POST":
		postHTTP(url, hdr, opts)
	}
}
