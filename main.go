package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ignoreRegex = regexp.MustCompile(`^Conversation with|.* (hat sich a[nb]gemeldet` +
		`|ist( wieder)? a[nb]wesend)\.$|^Listing members of .*|^\* .* \(.*\).*|.*\?OTR(:[A-Za-z0-9=.]+|\?v2\?)`)
	chatRegex = regexp.MustCompile(`^\((\d{2}\.\d{2}\.\d{4} )?\d{2}:\d{2}:\d{2}\)\s[^:]+:\s*`)
	cutRegex  = regexp.MustCompile(`https?://[^ ]+|\[[^\[\]]*\]\s*|^: |[^a-zA-Zäöü0-9 ]+`)

	sentenceSplit = regexp.MustCompile(`[\n\r.,]`)
	wordSplit     = regexp.MustCompile(`[\s]`)
	spaceReplace  = regexp.MustCompile(`[\s]+`)
)

const (
	endSentence   = "\u0000"
	startSentence = "\u0002"
)

type Node struct {
	Count float32
	IsEnd float32
}

type ngram map[string]map[string]*Node

var stuff struct {
	// gets every line
	rawInput chan string
	// gets useful lines
	input chan string

	/*
		index 0: 1grams -> key is one word
		index 1: 2grams -> key is two words
		...
	*/
	ngrams []ngram
}

func main() {
	_, err := os.Stat("data/")
	if os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "data folder not found")
		return
	}

	stuff.input = make(chan string, 32)
	stuff.rawInput = make(chan string, 32)

	stuff.ngrams = make([]ngram, 2)
	for i := range stuff.ngrams {
		stuff.ngrams[i] = make(ngram)
	}

	go func(output chan string) {
		filepath.Walk("data/", visit)
		close(output)
	}(stuff.rawInput)

	go lineFilter(stuff.rawInput, stuff.input)

	finish := make(chan bool)
	go train(stuff.input, stuff.ngrams, finish)
	<-finish
	//fmt.Println("root:", stuff.root)
	//fmt.Println("all Nodes:", stuff.allNodes)

	for _, gram := range stuff.ngrams {
		summaries(gram)
	}

	if out, err := os.Create("ngrams.json"); err == nil {
		defer out.Close()
		b, err := json.MarshalIndent(stuff.ngrams, "", "\t")
		if err != nil {
			fmt.Println("error saving ngrams:", err)
		} else {
			out.WriteString(string(b))
		}
	} else {
		fmt.Println("error saving ngrams:", err)
	}

	for i := 0; i < 10; i++ {
		generateContent(stuff.ngrams)
	}
}

func visit(path string, f os.FileInfo, err error) error {
	if !f.IsDir() {
		readFile(path)
	}
	return nil
}

func readFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer file.Close()
	r := bufio.NewReader(file)
	contents, _ := ioutil.ReadAll(r)
	lines := strings.Split(string(contents), "\n")
	for _, l := range lines {
		stuff.rawInput <- l
	}
}

func lineFilter(input, output chan string) {
	for line := range input {
		if ignoreRegex.MatchString(line) {
			continue
		}
		line = strings.Trim(line, " \t\n\r")
		if line == "" {
			continue
		}
		line = chatRegex.ReplaceAllString(line, startSentence)
		line = cutRegex.ReplaceAllString(line, "")
		line = strings.ToLower(line)
		line = spaceReplace.ReplaceAllString(line, " ")
		line += endSentence
		output <- line
	}
	close(output)
}

func train(input chan string, ngrams []ngram, finish chan bool) {
	for line := range input {
		line = strings.Trim(line, startSentence+endSentence+" \t\r\n.")
		sentences := sentenceSplit.Split(line, -1)
		for _, sentence := range sentences {
			sentence = strings.Trim(sentence, " ")
			if sentence == "" {
				continue
			}
			words := wordSplit.Split(sentence, -1)

			var lastPossible *Node
			for i, word := range words {
				for n, gram := range ngrams {
					if i < n {
						continue
					}
					prefix := strings.Join(words[len(words)-n:], " ")
					if prefix == word {
						continue
					}
					possibles := gram[prefix]
					if possibles == nil {
						possibles = make(map[string]*Node)
						gram[prefix] = possibles
					}
					possible, ok := possibles[word]
					if !ok {
						possible = &Node{}
						possibles[word] = possible
					}
					possible.Count++
					lastPossible = possible
				}
			}
			if lastPossible != nil {
				lastPossible.IsEnd++
			}
		}
	}
	close(finish)
}

func summaries(gram ngram) {
	for _, nodes := range gram {
		var endings float32
		var count float32

		for _, node := range nodes {
			endings += node.IsEnd
			count += node.Count
		}
		//count = saneNumber(log(count))
		//endings = saneNumber(log(endings))
		if endings == 0 {
			endings = 1
		}
		for _, node := range nodes {
			node.Count /= count
			node.IsEnd /= endings
			//node.Count = saneNumber(log(node.Count) / count)
			//node.IsEnd = saneNumber(log(node.IsEnd) / endings)
		}
	}
}
func log(a float32) float32 {
	return float32(math.Log10(float64(a)))
}
func saneNumber(v float32) float32 {
	if math.IsInf(float64(v), -1) {
		return 0
	}
	if math.IsNaN(float64(v)) {
		return 0
	}
	return v
}

func generateContent(ngrams []ngram) {
	str := make([]string, 0)
	var n int
	for i := 0; i < 30; i++ {
		if n >= len(ngrams) {
			n = len(ngrams) - 1
		}
		prefix := strings.Join(str[len(str)-n:], " ")
		//fmt.Println("string", str, "n=", n, "- prefix:", prefix)
		words := ngrams[n][prefix]
		word, ending := next(words)
		str = append(str, word)

		if decide(ending * 0.001) {
			break
		}
		n++
	}
	fmt.Println(strings.Join(str, " "))
	fmt.Println("---")
}

func decide(prob float32) bool {
	return rand.Float32()*prob > 0.5
}

func next(words map[string]*Node) (word string, ending float32) {
	nodes := make([]*Node, 0, len(words))
	strings := make([]string, 0, len(words))
	for word, node := range words {
		nodes = append(nodes, node)
		strings = append(strings, word)
	}
	if len(nodes) == 0 {
		return "", float32(math.Inf(1))
	}
	scores := make([]float32, len(nodes))
	for i, n := range nodes {
		scores[i] = rand.Float32() * n.Count
	}
	var max float32
	var maxI int
	for i, v := range scores {
		if v > max {
			max = v
			maxI = i
		}
	}
	word = strings[maxI]
	ending = nodes[maxI].IsEnd
	return
}
