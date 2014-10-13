package main

import (
	"html/template"
	"io"
	"log"
	"net/http"

	"code.google.com/p/go.net/websocket"
)

func main() {
	http.Handle("/sock", websocket.Handler(sockHandler))
	http.HandleFunc("/", rootHandler)
	if err := http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", nil); err != nil {
		log.Fatal(err)
	}
}

func sockHandler(ws *websocket.Conn) {
	io.Copy(ws, ws)
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
	msgEl.textContent = m.data;
	chat.insertBefore(msgEl, chat.firstChild);
};
var sendMessage = function() {
	var msg = document.getElementById("msg");
	sock.send(msg.value);
	msg.value = "";
};
</script>
</body>
</html>
`
