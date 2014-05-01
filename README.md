talkateev
=========

Is a small markov chain implementation which supports twitter and libpurple chatlogs.
This means you can have endless hours of fun creating conversations from your pidgin or adium logs or random twitter accounts. Also it doubles as a twitter scraper.

Props to ChimeraCoder for anaconda!

Setup
-----

    go get github.com/cfstras/talkateev

For twitter access, create an app on [apps.twitter.com](https://apps.twitter.com/), create an authentication token and then create a file called `auth.json` in your current directory with these contents:

```
    {
       "ConsumerKey": "<your app API Key here>",
       "ConsumerSecret": "<your app API Secret here>",
      "AccessToken": "<your Access Token here>",
      "AccessSecret": "<your Access Token Secret here>"
    }
```

Usage
-----

### using with libpurple/pidgin/adium logs

    talkateev -purple ~/.purple/logs

### using with twitter user

    talkateev -twitter <yourTwitterUsername>

### using with already downloaded twitter data

    talkateev -json twitter_<username>.json

### some more flags

`-maxLen x`: maximum sentence length in words
`-prefixLen x`: length of prefix to search (low means less sense, but more randomness, too high (>3 for 1k tweets) will result in just the tweets)

License
-------

Beerware!
