package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	_ "net/http/pprof"
)

type channelInput struct {
	key    string
	hash   string
	hashes map[string]string
}

type channelOutput struct {
	key    string
	result map[string]int
}

func compareHashes(a string, b string) int {
	diffCount := 0
	l := len(a)
	if len(a) > len(b) {
		diffCount = diffCount + len(a) - len(b)
		l = len(b)
	}
	for i := 0; i < l; i++ {
		if a[i] != b[i] {
			diffCount++
		}
	}
	return diffCount
}

func hexToBin(rawHex string) string {
	i, err := strconv.ParseUint(rawHex, 16, 64)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%024b", i)
}

func getSimilarityComparison(jobs chan channelInput, done chan channelOutput) {
	workers.Add(1)
	defer workers.Done()
	for {
		input, closed := <-jobs
		if closed {
			resultHashes := map[string]int{}
			for compareImg, compareHash := range input.hashes {
				if input.key != compareImg {
					similarity := compareHashes(input.hash, compareHash)
					if similarity <= 20 {
						resultHashes[compareImg] = similarity
					}
				}
			}
			done <- channelOutput{input.key, resultHashes}
		} else {
			log.Printf("all jobs have been processed")
			return
		}
	}
}

func createJobs(jobs chan channelInput, hashes map[string]string) {
	jobCreators.Add(1)
	defer jobCreators.Done()
	// creating jobs
	for img, hash := range hashes {
		jobs <- channelInput{img, hash, hashes}
	}
}

func updateResult(done chan channelOutput, i, l int) {
	mergers.Add(1)
	defer mergers.Done()
	fmt.Printf("{\n")
	for img := range done {
		log.Printf("(%.2f %%)[%d/%d]processing %s", float64(float64(i)*100.0/float64(l)), i, l, img.key)
		i++
		result[img.key] = img.result
		jsonResult, _ := json.MarshalIndent(img.result, "", "    ")
		fmt.Printf(",\n    \"%s\": %s", img.key, jsonResult)
	}
	fmt.Printf("\n}")
}

var result = map[string]map[string]int{}
var workers sync.WaitGroup
var mergers sync.WaitGroup
var jobCreators sync.WaitGroup

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		log.Printf("you must specify one json file containing key->hash to process.")
		return
	}
	go func() { log.Fatal(http.ListenAndServe(":4000", nil)) }()

	file := args[0]
	jsonFile, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	hashes := map[string]string{}
	if err := json.Unmarshal(byteValue, &hashes); err != nil {
		log.Fatal(err)
	}
	for key, value := range hashes {
		hashes[key] = hexToBin(value)
	}

	nWorker := 4
	jobs := make(chan channelInput, nWorker)
	done := make(chan channelOutput, nWorker)

	i := 1
	l := len(hashes)
	go createJobs(jobs, hashes)
	go updateResult(done, i, l)
	for n := 1; n <= nWorker; n++ {
		log.Printf("starting worker %d", n)
		go getSimilarityComparison(jobs, done)
	}

	var sleep time.Duration = 5

	jobCreators.Wait()
	log.Printf("all jobs are created... waiting %d seconds before continuing...", sleep)
	time.Sleep(sleep * time.Second)
	close(jobs)

	workers.Wait()
	log.Printf("all workers are done... waiting %d seconds before continuing...", sleep)
	time.Sleep(sleep * time.Second)
	close(done)

	mergers.Wait()
	log.Printf("all mergers are done... waiting %d seconds before continuing...", sleep)
	time.Sleep(sleep * time.Second)
}
