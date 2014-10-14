package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"sync"

	"code.google.com/p/go.net/websocket"
)

var chat = struct {
	sync.RWMutex
	peers []*websocket.Conn
}{}

type message struct {
	Name string `json:"name,"`
	Body string `json:"body,"`
}

func main() {
	http.Handle("/sock", websocket.Handler(sockHandler))
	http.HandleFunc("/", rootHandler)
	if err := http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", nil); err != nil {
		log.Fatal(err)
	}
}

func sockHandler(ws *websocket.Conn) {
	chat.Lock()
	chat.peers = append(chat.peers, ws)
	// TODO(andrew): need to handle disconnects
	chat.Unlock()
	dec := json.NewDecoder(ws)
	for {
		var m message
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		broadcast(m)
	}
}

func broadcast(m message) {
	chat.RLock()
	for _, v := range chat.peers {
		enc := json.NewEncoder(v)
		if err := enc.Encode(m); err != nil {
			log.Fatal(err)
		}
	}
	chat.RUnlock()
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	rootTemplate.Execute(w, nil)
}

var rootTemplate = template.Must(template.New("root").Parse(rootTmpl))

const rootTmpl = `
<!doctype html>
<html>
<head>
<title>Chat</title>
</head>
<body>
<input id="msg" type="text"><br>
<input value="send" type="button" onclick="sendMessage()">
<div id="chat">
</div>
<script>
var sock = new WebSocket("wss://127.0.0.1:8080/sock");
sock.onmessage = function(m) {
	var chat = document.getElementById("chat");
	var msgEl = document.createElement("p");
	msgEl.textContent = JSON.parse(m.data).body;
	chat.insertBefore(msgEl, chat.firstChild);
};
var sendMessage = function() {
	var msg = document.getElementById("msg");
	msgJson = JSON.stringify({name: "bob", body: msg.value});
	msg.value = "";
	sock.send(msgJson);
};
</script>
</body>
</html>
`