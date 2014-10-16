package main

import (
	"encoding/json"
	"flag"
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

const (
	wsAddr      = "/ws/"
	defaultHost = "192.168.1.2"
	defaultPort = ":8080"
)

var (
	host string
	port string
)

func init() {
	const (
		hostHelp = "The address of the host system."
		portHelp = "The port of the host system."
	)

	flag.StringVar(&host, "host", defaultHost, hostHelp)
	flag.StringVar(&port, "port", defaultPort, portHelp)
}

func main() {
	http.Handle(wsAddr, websocket.Handler(sockHandler))
	http.HandleFunc("/", rootHandler)
	if err := http.ListenAndServeTLS(port, "cert.pem", "key.pem", nil); err != nil {
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
		m.Name = ws.Request().URL.Path[len(wsAddr):]
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
	data := struct {
		Host, Port, Name string
	}{
		Host: host,
		Port: port,
		Name: r.URL.Path[1:],
	}
	rootTemplate.Execute(w, data)
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
var sock = new WebSocket("wss://{{.Host}}{{.Port}}/ws/{{.Name}}");
sock.onmessage = function(m) {
	var chat = document.getElementById("chat");
	var msgEl = document.createElement("p");
	var msg = JSON.parse(m.data)
	msgEl.textContent = msg.name + ": " + msg.body;
	chat.insertBefore(msgEl, chat.firstChild);
};
var sendMessage = function() {
	var msg = document.getElementById("msg");
	msgJson = JSON.stringify({body: msg.value});
	msg.value = "";
	sock.send(msgJson);
};
</script>
</body>
</html>
`
