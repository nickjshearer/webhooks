package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"github.com/objclxt/twitterstream"
	"log"
	"net/http"
	"os"
	"time"
)

type socket struct {
	conn *websocket.Conn
	done chan bool
}

func (s socket) Read(b []byte) (int, error)  { return s.conn.Read(b) }
func (s socket) Write(b []byte) (int, error) { return s.conn.Write(b) }

func (s socket) Close() error {
	println("Closed socket")
	s.done <- true
	return nil
}

func socketHandler(ws *websocket.Conn) {
	s := socket{conn: ws, done: make(chan bool)}
	go stream(s)
	<-s.done
}

func decodeTweet(conn *twitterstream.Connection, s socket) {
	for {
		if tweet, err := conn.Next(); err == nil {
			if tweet.Entities.Media != nil && !tweet.Retweeted {
				log.Printf("%s %s \n\n %s\n\n\n", tweet.IdString, tweet.User.ScreenName, tweet.Text)
				websocket.JSON.Send(s.conn, tweet)
			}
		} else {
			//log.Printf("Failed decoding tweet: %s", err)
			return
		}
	}
}

func stream(s socket) {
	println("Starting stream")
	// Twitter Streaming
	client := twitterstream.NewClient(os.Getenv("TW_USER"), os.Getenv("TW_PASS"))
	for {
		conn, err := client.Track("cat")
		if err != nil {
			log.Println("Tracking failed, sleeping for 1 minute")
			time.Sleep(1 * time.Minute)
			continue
		}
		decodeTweet(conn, s)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
		<meta charset="utf-8" />
		<script>
			  websocket = new WebSocket("ws://localhost:8080/socket");
		    websocket.onmessage = function (e) {
		    	console.log(e);
		    	var obj = JSON.parse(e.data);
		    	document.getElementById("catimg").src = obj.entities.media[0].media_url;
		    };
		    websocket.onclose = function (e) {
					console.log("Closed socket");
		    };
		    window.onbeforeunload = function() {
   		 		websocket.onclose = function () {}; // disable onclose handler first
    			websocket.close()
				};
		 </script>
		</head>
		<body>
			<h1>Cats</h1>
			<h4>Live cat images</h4>
			<img id='catimg' src='' alt='Cats'>
		</body>
		</html>`)
}

func main() {
	http.Handle("/socket", websocket.Handler(socketHandler))
	http.HandleFunc("/", rootHandler)
	http.ListenAndServe(":8080", nil)
}
