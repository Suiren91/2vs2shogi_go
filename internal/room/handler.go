package room

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = pongWait * 9 / 10
	maxMsgSize = 4096
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// TODO: 本番ではOriginを制限する
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler はWebSocket接続を確立し、受信したメッセージをそのまま返す。
func Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	//MEMO: Limitで最大サイズを決める(決めないとDoSを喰らう)
	conn.SetReadLimit(maxMsgSize)
	// MEMO: タイムアウト
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	// MEMO: pongが来た時に行う処理
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// 読み込み，書き込みそれぞれ1goroutineのみ
	msgs := make(chan message)
	done := make(chan struct{})
	defer close(done)
	go readLoop(conn, msgs, done)

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case m, ok := <-msgs:
			if !ok {
				return
			}
			// MEMO: 10秒で書き込めなければタイムアウト(コネクション破棄)
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(m.typ, m.data); err != nil {
				log.Printf("ws: write failed: %v", err)
				return
			}
		// MEMO: ここでpingを送ってる
		case <-ticker.C:
			// MEMO: 10秒で書き込めなければタイムアウト(コネクション破棄)
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type message struct {
	typ  int
	data []byte
}

func readLoop(conn *websocket.Conn, out chan<- message, done <-chan struct{}) {
	defer close(out)
	for {
		typ, data, err := conn.ReadMessage()
		if err != nil {
			// TODO: 現時点ではpong未達でもログが出ない?
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("ws: read failed: %v", err)
			}
			return
		}
		select {
		case out <- message{typ, data}:
		case <-done:
			return
		}
	}
}
