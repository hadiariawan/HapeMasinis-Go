package main

import (
    "database/sql"
    "fmt"
    _ "github.com/go-sql-driver/mysql"
    "github.com/kurrik/twittergo"
    "github.com/kurrik/oauth1a"
    "io/ioutil"
    "net/http"
    "net/url"
    "os"
    "strings"
    "log"
    "time"
)

func LoadCredentials() (client *twittergo.Client, err error) {
    credentials, err := ioutil.ReadFile("CREDENTIALS")
    if err != nil {
        return
    }
    lines := strings.Split(string(credentials), "\n")
    config := &oauth1a.ClientConfig{
        ConsumerKey:    lines[0],
        ConsumerSecret: lines[1],
    }
    user := oauth1a.NewAuthorizedConfig(lines[2], lines[3])
    client = twittergo.NewClient(config, user)
    return
}

func main() {
    db, err := sql.Open("mysql", "username:databasename@/password")
    if err != nil {
        panic(err.Error())
    }
    defer db.Close()

    var (
        client  *twittergo.Client
        req     *http.Request
        resp    *twittergo.APIResponse
        results *twittergo.SearchResults
    )

    client, err = LoadCredentials()
    if err != nil {
        fmt.Printf("Could not parse CREDENTIALS file: %v\n", err)
        os.Exit(1)
    }

    query := url.Values{}
    query.Set("q", "#edisikrl")
    url := fmt.Sprintf("/1.1/search/tweets.json?%v", query.Encode())
    req, err = http.NewRequest("GET", url, nil)
    if err != nil {
        fmt.Printf("Could not parse request: %v\n", err)
        os.Exit(1)
    }

    resp, err = client.SendRequest(req)
    if err != nil {
        fmt.Printf("Could not send request: %v\n", err)
        os.Exit(1)
    }

    results = &twittergo.SearchResults{}
    err = resp.Parse(results)
    if err != nil {
        fmt.Printf("Problem parsing response: %v\n", err)
        os.Exit(1)
    }

    // iterate twitter search result
    for i, tweet := range results.Statuses() {

        fmt.Println(i)

        // do not retweet from our own account
        if strings.ToLower(tweet.User().ScreenName()) != "edisikrl" {

            rows, err := db.Query("SELECT * FROM hapemasinis WHERE tweet_id = ?", tweet.Id())
            if err != nil {
                log.Fatal(err)
            }
            defer rows.Close()

            // do not retweet the one that already tweeted
            if !rows.Next() {

                // post RT tweet
                var str_rt_tweet string
                str_rt_tweet = "RT @" + tweet.User().ScreenName() + ": " + tweet.Text()

                query.Set("status", str_rt_tweet)
                body := strings.NewReader(query.Encode())
                req, err = http.NewRequest("POST", "/1.1/statuses/update.json", body)
                if err != nil {
                    fmt.Printf("Could not parse request: %v\n", err)
                    os.Exit(1)
                }
                req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
                resp, err = client.SendRequest(req)
                if err != nil {
                    fmt.Printf("Could not send request: %v\n", err)
                    os.Exit(1)
                }

                // insert the tweet
                stmtIns, err := db.Prepare("INSERT INTO hapemasinis VALUES(?,?,?,?,?)")
                if err != nil {
                    panic(err.Error())
                }
                defer stmtIns.Close()

                _, err = stmtIns.Exec(tweet.Id(),tweet.User().ScreenName(),tweet.Text(),"",tweet.CreatedAt().Format(time.RFC1123))
                if err != nil {
                    panic(err.Error())
                }

            }
        }
    }
}
