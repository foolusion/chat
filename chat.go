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

type chat struct {
	sync.RWMutex
	peers []*websocket.Conn
}

func (c *chat) add(ws *websocket.Conn) {
	c.Lock()
	c.peers = append(c.peers, ws)
	c.Unlock()
}

func (c *chat) remove(ws *websocket.Conn) {
	c.Lock()
	for i := 0; i < len(c.peers); i++ {
		if c.peers[i] == ws {
			c.peers[i], c.peers[len(c.peers)-1], c.peers = c.peers[len(c.peers)-1], nil, c.peers[:len(c.peers)-1]
		}
	}
	c.Unlock()
}

func (c *chat) broadcast(m message) {
	c.RLock()
	for _, v := range c.peers {
		enc := json.NewEncoder(v)
		if err := enc.Encode(m); err != nil {
			log.Fatal(err)
		}
	}
	c.RUnlock()
}

var mainChat chat

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
	mainChat.add(ws)
	defer mainChat.remove(ws)
	dec := json.NewDecoder(ws)
	for {
		var m message
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		m.Name = ws.Request().URL.Path[len(wsAddr):]
		mainChat.broadcast(m)
	}
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
<textarea id="msg" onkeydown="textKeyDown(event)"></textarea>
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
var textKeyDown = function(e) {
	if (e.keyCode == 13 && !e.shiftKey) {
		e.preventDefault();
		sendMessage();
	}
}

</script>
</body>
</html>
`
