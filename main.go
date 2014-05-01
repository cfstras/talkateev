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
	"time"
	"utils"
)

var (
	ignoreRegex = regexp.MustCompile(`^Conversation with|.* (hat sich a[nb]gemeldet` +
		`|ist( wieder)? a[nb]wesend)\.$|^Listing members of .*|^\* .* \(.*\).*` +
		`|.*\?OTR(:[A-Za-z0-9=.]+` +
		`|\?v2\?)|\(\d{2}:\d{2}:\d{2}\) _[a-zA-Z0-9]+: .* invited .*@.*\.[a-zA-Z]+_`)
	chatRegex = regexp.MustCompile(`^\((\d{2}\.\d{2}\.\d{4} )?\d{2}:\d{2}:\d{2}\)\s[^:]+:\s*`)
	cutRegex  = regexp.MustCompile(`https?://[^ ]+|\[[^\[\]]*\]\s*|^: `)

	sentenceSplit = regexp.MustCompile(`[\n\r.,]`)
	wordSplit     = regexp.MustCompile(`[\s]`)
	spaceReplace  = regexp.MustCompile(`[\s]+|[^a-zA-Zäöüß0-9 ]+`)
)

type Chain map[string][]string

var stuff struct {
	// gets every line
	rawInput chan string
	// gets useful lines
	input chan string
	// gets words, nil means end of sentence
	words chan *string

	prefixLen int
	maxLen    int

	Chain Chain
}

func main() {
	_, err := os.Stat("data/")
	if os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "data folder not found")
		return
	}

	stuff.prefixLen = 2
	stuff.maxLen = 20

	rand.Seed(time.Now().UnixNano())
	stuff.Chain = make(map[string][]string)
	stuff.input = make(chan string, 32)
	stuff.rawInput = make(chan string, 32)
	stuff.words = make(chan *string, 32)

	go func(output chan string) {
		filepath.Walk("data/", visit)
		close(output)
	}(stuff.rawInput)

	go lineFilter(stuff.rawInput, stuff.input)
	go wordFilter(stuff.input, stuff.words)

	finish := make(chan bool)
	go train(stuff.words, stuff.Chain, finish, stuff.prefixLen)
	<-finish

	fmt.Println("----")
	generateSome()
	saveData(stuff.Chain)
}

func generateSome() {
	for i := 0; i < 50; i++ {
		fmt.Println(generateContent(stuff.Chain, stuff.prefixLen, stuff.maxLen))
	}
}

func saveData(data Chain) {
	if out, err := os.Create("ngrams.json"); err == nil {
		defer out.Close()
		b, err := json.MarshalIndent(data, "", "\t")
		if err != nil {
			fmt.Println("error saving ngrams:", err)
		} else {
			out.WriteString(string(b))
		}
	} else {
		fmt.Println("error saving ngrams:", err)
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
		line = chatRegex.ReplaceAllString(line, "")
		line = cutRegex.ReplaceAllString(line, "")
		line = strings.ToLower(line)
		line = spaceReplace.ReplaceAllString(line, " ")
		output <- line
	}
	close(output)
}

func wordFilter(input chan string, output chan *string) {
	for line := range input {
		line = strings.Trim(line, " \t\r\n.")
		sentences := sentenceSplit.Split(line, -1)
		for _, sentence := range sentences {
			sentence = strings.Trim(sentence, " ")
			if sentence == "" {
				continue
			}
			words := wordSplit.Split(sentence, -1)
			for _, word := range words {
				if word != "" {
					w := word
					output <- &w
				}
			}
			output <- nil
		}
	}
	close(output)
}

func train(input chan *string, chain Chain, finish chan bool, prefixLen int) {
	circle := utils.NewCircle(prefixLen)
	for word := range input {
		if word == nil {
			circle = utils.NewCircle(prefixLen)
			//fmt.Println()
			continue
		}
		//fmt.Print(*word, " ")
		prefix := circle.String()
		chain[prefix] = append(chain[prefix], *word)
		circle.Shift(*word)
	}
	close(finish)
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

func generateContent(chain Chain, prefixLen, maxLen int) string {
	str := make([]string, 0)
	circle := utils.NewCircle(prefixLen)
	for i := 0; i < maxLen && !decide(0.01); i++ {
		prefix := circle.String()
		words := chain[prefix]
		if words == nil || len(words) == 0 {
			break
		}
		word := words[rand.Intn(len(words))]
		circle.Shift(word)
		//fmt.Println("string", str, "i=", i, "- prefix:", prefix)
		str = append(str, word)
	}
	return strings.Join(str, " ") + "."
}

func decide(prob float32) bool {
	return rand.Float32()*prob > 0.5
}
