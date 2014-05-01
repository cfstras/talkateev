package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ChimeraCoder/anaconda"
	"io/ioutil"
	"math"
	"math/rand"
	"net/url"
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
	spaceReplace  = regexp.MustCompile(`[\s]+|[^a-zA-Zäöü@ß0-9 ]+`)
)

type Chain map[string][]string

type TwitterData struct {
	User   string
	Tweets []Tweet
}
type Tweet struct {
	Time time.Time
	Text string
}

type TwitterAuth struct {
	ConsumerKey    string
	ConsumerSecret string
	AccessToken    string
	AccessSecret   string
}

var stuff struct {
	// gets every line
	rawInput chan string
	// gets useful lines
	input chan string
	// gets words, nil means end of sentence
	words chan *string

	usingTwitter string
	twitter      *TwitterData

	prefixLen int
	maxLen    int

	Chain Chain

	numSentences int
	numWords     int
}

func main() {
	prefixLen := flag.Int("prefixLen", 2, `sets length of prefix to use`)
	maxLen := flag.Int("maxLen", 20, `maximal sentence length in words`)
	json := flag.String("json", "", `set a file path to load in data`)
	twitter := flag.String("twitter", "", `use a twitter account for data`)
	flag.Parse()

	stuff.prefixLen = *prefixLen
	stuff.maxLen = *maxLen

	rand.Seed(time.Now().UnixNano())
	stuff.Chain = make(map[string][]string)
	stuff.input = make(chan string, 32)
	stuff.rawInput = make(chan string, 32)
	stuff.words = make(chan *string, 32)

	go lineFilter(stuff.rawInput, stuff.input)
	go wordFilter(stuff.input, stuff.words)

	if json == nil || *json == "" {
		if twitter == nil || *twitter == "" {
			fmt.Println("using data/...")
			readFromData()
		} else {
			stuff.usingTwitter = *twitter
			fmt.Println("Loading twitter of", *twitter, "...")
			auth := TwitterAuth{}
			readFromJSON("auth.json", &auth)
			authempt := TwitterAuth{}
			if auth == authempt {
				fmt.Println("error: auth is empty")
				return
			}
			readFromTwitter(auth, *twitter)
			saveData("twitter_"+*twitter+".json", stuff.twitter)
		}
	} else {
		stuff.twitter = &TwitterData{}
		readFromJSON(*json, stuff.twitter)
	}

	if stuff.twitter != nil {
		go func(output chan string) {
			for _, tw := range stuff.twitter.Tweets {
				output <- tw.Text
			}
			close(output)
		}(stuff.rawInput)
	}

	finish := make(chan bool)
	go train(stuff.words, stuff.Chain, finish, stuff.prefixLen)
	<-finish

	fmt.Println("Sentences:", stuff.numSentences)
	fmt.Println("Words:", stuff.numWords)
	fmt.Println("----")
	generateSome()
	saveData("chain.json", stuff.Chain)
}

func readFromTwitter(auth TwitterAuth, username string) {
	anaconda.SetConsumerKey(auth.ConsumerKey)
	anaconda.SetConsumerSecret(auth.ConsumerSecret)
	api := anaconda.NewTwitterApi(auth.AccessToken, auth.AccessSecret)
	var earliestID int64
	earliestID = math.MaxInt64

	tw := TwitterData{User: username}
	for num := 10; num > 0; {
		v := url.Values{}
		v.Set("screen_name", username)
		v.Set("count", "200")
		if earliestID != math.MaxInt64 {
			v.Set("max_id", fmt.Sprint(earliestID-1))
		}
		fmt.Println("next...")
		tweets, err := api.GetUserTimeline(v)
		if err != nil {
			fmt.Println("Twitter error:", err)
			return
		}
		num = len(tweets)
		fmt.Println("data get", num, "toots")

		for _, rawTweet := range tweets {
			t, err := rawTweet.CreatedAtTime()
			if err != nil {
				fmt.Println("Twitter error:", err)
				return
			}
			if rawTweet.Id < earliestID {
				earliestID = rawTweet.Id
			} else if rawTweet.Id == earliestID {
				fmt.Println("stopping now, id", earliestID, " already got")
				num = 0
			}
			tweet := Tweet{Time: t, Text: rawTweet.Text}
			tw.Tweets = append(tw.Tweets, tweet)
		}
	}
	stuff.twitter = &tw
}

func readFromData() {
	_, err := os.Stat("data/")
	if os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "data folder not found")
		return
	}
	go func(output chan string) {
		filepath.Walk("data/", visit)
		close(output)
	}(stuff.rawInput)
}

func readFromJSON(path string, target interface{}) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, path, "not found")
		return
	}
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer file.Close()
	r := bufio.NewReader(file)
	contents, _ := ioutil.ReadAll(r)
	err = json.Unmarshal(contents, target)
	if err != nil {
		fmt.Println("JSON:", err)
	}
}

func generateSome() {
	for i := 0; i < 50; i++ {
		fmt.Println(generateContent(stuff.Chain, stuff.prefixLen, stuff.maxLen))
	}
}

func saveData(path string, data interface{}) {
	if out, err := os.Create(path); err == nil {
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
			stuff.numSentences++
			words := wordSplit.Split(sentence, -1)
			for _, word := range words {
				if word != "" {
					w := word
					output <- &w
					stuff.numWords++
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
	for i := 0; i < maxLen && !decide(0.001); i++ {
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
