package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

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

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		log.Printf("you must specify one json file containing key->hash to process.")
		return
	}
	file := args[0]
	result := map[string]map[string]int{}
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
	i := 1
	l := len(hashes)
	for img, hash := range hashes {
		log.Printf("[%d/%d]processing %s", i, l, img)
		i++
		resultHashes := map[string]int{}
		oHash := hexToBin(hash)
		for compareImg, compareHash := range hashes {
			if img != compareImg {
				similarity := compareHashes(oHash, hexToBin(compareHash))
				if similarity <= 30 {
					resultHashes[compareImg] = similarity
				}
			}
		}
		result[img] = resultHashes
	}
	jsonResult, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", jsonResult)
}
