package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

// peer is a peer
type peer struct {
	ws   *websocket.Conn
	name string
}

type chat struct {
	sync.RWMutex
	peers []*peer
}

func (c *chat) add(p *peer) {
	c.Lock()
	c.peers = append(c.peers, p)
	c.Unlock()
	systemBroadcast(c, p.name+" joined the chat.")
}

func (c *chat) remove(p *peer) {
	c.Lock()
	for i := 0; i < len(c.peers); i++ {
		if c.peers[i] == p {
			c.peers[i], c.peers[len(c.peers)-1], c.peers = c.peers[len(c.peers)-1], nil, c.peers[:len(c.peers)-1]
		}
	}
	c.Unlock()
	systemBroadcast(c, p.name+" left the chat.")
}

func (c *chat) broadcast(m chatMessage) {
	c.RLock()
	for _, v := range c.peers {
		enc := json.NewEncoder(v.ws)
		if err := enc.Encode(m); err != nil {
			log.Fatal(err)
		}
	}
	c.RUnlock()
}

func systemBroadcast(c *chat, s string) {
	c.RLock()
	peers := make([]string, 0, len(c.peers)+1)
	for _, v := range c.peers {
		peers = append(peers, v.name)
	}
	c.RUnlock()
	m := chatMessage{
		Msg: &message{
			Name: "SYSTEM",
			Body: s},
		Status: &status{Peers: peers},
	}
	c.broadcast(m)
}

var mainChat chat

type chatMessage struct {
	Msg    *message `json:"msg,omitempty"`
	Status *status  `json:"status,omitempty"`
}

type message struct {
	Name string `json:"name,"`
	Body string `json:"body,"`
}

type status struct {
	Peers []string `json:"peers,omitempty"`
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
	flag.Parse()
	http.Handle(wsAddr, websocket.Handler(sockHandler))
	http.HandleFunc("/", rootHandler)
	if err := http.ListenAndServeTLS(port, "cert.pem", "key.pem", nil); err != nil {
		log.Fatal(err)
	}
}

func sockHandler(ws *websocket.Conn) {
	p := &peer{ws: ws, name: ws.Request().URL.Path[len(wsAddr):]}
	mainChat.add(p)
	defer mainChat.remove(p)
	dec := json.NewDecoder(p.ws)
	for {
		var m message
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		m.Name = p.name
		mainChat.broadcast(chatMessage{Msg: &m})
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
<div class="container">
<div class="col">
<textarea id="msg" onkeydown="textKeyDown(event)"></textarea>
<div id="chat">
</div>
</div>
<div class="col">
<div id="peers">
</div>
<script>
var sock = new WebSocket("wss://{{.Host}}{{.Port}}/ws/{{.Name}}");
sock.onmessage = function(m) {
	var md = JSON.parse(m.data)
	if (md.msg) {
		var chat = document.getElementById("chat");
		var msgEl = document.createElement("p");
		msgEl.textContent = md.msg.name + ": " + md.msg.body;
		chat.insertBefore(msgEl, chat.firstChild);
	}
	if (md.status) {
		var ps = document.querySelectorAll('#peers p');
		var peers = document.getElementById("peers");
		for (var i = 0; i < ps.length; i++) {
			peers.removeChild(ps[i]);
		}
		for (var i = 0; i < md.status.peers.length; i++) {
			var p = document.createElement("p");
			p.textContent = md.status.peers[i];
			peers.appendChild(p);
		}
	}
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
