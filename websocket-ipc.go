// Author :		Eric<eehsiao@gmail.com>

package ipc

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

type Client struct {
	Ws  *websocket.Conn
	Msg IpcCmd
}

type IPC struct {
	WsClient chan *Client
	IPCCmd   func(client *Client) (err error)
}
type IpcCmd struct {
	Cmd     string `json:"cmd"`
	CmdFlag string `json:"cmd_flag"`
}

func (c IpcCmd) Serialize() (serialString string) {
	var bytes []byte
	bytes, _ = json.Marshal(c)
	serialString = string(bytes)

	return
}

type IpcMsg struct {
	UnixTime     int64       `json:"unix_time,string"`
	Result       bool        `json:"result,string"`
	Message      string      `json:"msg"`
	ResultObject interface{} `json:"result_object,string"`
}

func (c IpcMsg) Serialize() (serialString string) {
	var bytes []byte
	bytes, _ = json.Marshal(c)
	serialString = string(bytes)

	return
}

var (
	stdlog, errlog *log.Logger
	wsPort         = ":8088"
	wsRoute        = "/ipc"
	wsServer       = "ws://127.0.0.1"
)

func NewIpc(aCmd func(client *Client) (err error), s *log.Logger, e *log.Logger) (i *IPC) {

	if s == nil {
		stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	} else {
		stdlog = s
	}
	if e == nil {
		errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
	} else {
		errlog = e
	}

	i = &IPC{
		WsClient: make(chan *Client, 1),
		IPCCmd:   aCmd,
	}

	return
}

func (i *IPC) ACmd(client *Client) (err error) {
	switch string(client.Msg.Cmd) {
	default:
		client.Ws.WriteJSON("{'echo':'" + string(client.Msg.Cmd) + "'}")
	}

	return
}

func (i *IPC) WsHandel() {
	upgrader := websocket.Upgrader{}

	http.HandleFunc(wsRoute, func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			errlog.Println(err)
			return
		}

		defer conn.Close()
		stdlog.Println("ws connected !!")
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				errlog.Println(err)
				break
			}

			var aCmd IpcCmd
			err = json.Unmarshal([]byte(msg), &aCmd)
			client := &Client{Ws: conn, Msg: aCmd}
			i.WsClient <- client
		}
	})

	http.ListenAndServe(wsPort, nil)
}

func SendCmd(aCmd IpcCmd) (string, error) {
	c, _, err := websocket.DefaultDialer.Dial(wsServer+wsPort+wsRoute, nil)
	if err != nil {
		errlog.Fatal("dial:", err)
	}
	defer c.Close()

	err = c.WriteMessage(websocket.TextMessage, []byte(aCmd.Serialize()))
	if err != nil {
		errlog.Println(err)
		return "", err
	}

	_, msg, err := c.ReadMessage()
	if err != nil {
		errlog.Println(err)
		return "", err
	}
	return string(msg), nil
}
