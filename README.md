# talkateev

Is a small markov chain implementation which supports twitter and libpurple chatlogs.
This means you can have endless hours of fun creating conversations from your pidgin or adium logs or random twitter accounts. Also it doubles as a twitter scraper.

Have fun with it!

Also, props to ChimeraCoder for building anaconda!


## Setup

    go get github.com/cfstras/talkateev

## Usage

    talkateev -help # helps sometimes.

### With libpurple/pidgin/adium logs

    talkateev -purple ~/.purple/logs

This will load logs from your libpurple (or pidgin, adium, gajim, etc), to train a markov chain and generate some output.
The log data is stripped from the most useless stuff (such as logon/logoff messages, partychat ramble etc) while loading. Your logs will not be edited or written in any way.

### With twitter user

For twitter access, create an app on [apps.twitter.com](https://apps.twitter.com/), create an authentication token and then create a file called `auth.json` in your current directory with these contents:

    {
      "ConsumerKey": "<your app API Key here>",
      "ConsumerSecret": "<your app API Secret here>",
      "AccessToken": "<your Access Token here>",
      "AccessSecret": "<your Access Token Secret here>"
    }

Then, run

    talkateev -twitter <your twitter handle>

This will load up _all_ tweets from that twitter user, save them in `twitter_<handle>.json` and then train a markov chain and some output.  
You can later use this downloaded data with the `-json` flag, see below.

### With already downloaded twitter data

    talkateev -json twitter_<username>.json

This will load tweets from a json file, train a markov chain and generate some output.

### some more flags

`-maxLen x`: maximum sentence length in words
`-prefixLen x`: length of prefix to search (low means less sense, but more randomness, too high (>3 for 1k tweets) will result in just the tweets)


## Development

If you want to hack on talkateev, go ahead!  
I have provided a Makefile, so you can jump right in:

    make deps
    make start

Pull requests and comments welcome!

## License

Beerware!
