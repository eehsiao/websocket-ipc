// Author :		Eric<eehsiao@gmail.com>

package ipc

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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

type IpcResult struct {
	ReqCmd IpcCmd
	Rsp    IpcRsp
}

type IpcRsp struct {
	UnixTime int64  `json:"unix_time,string"`
	Result   bool   `json:"result,string"`
	Message  string `json:"msg"`
}

func (c IpcRsp) Serialize() (serialString string) {
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

func SetWsPort(p int) {
	wsPort = ":" + strconv.Itoa(p)
}

func SetWsRoute(s string) {
	wsRoute = s
}

func SetWsServer(s string) {
	wsServer = s
}

func GetWsPort() string {
	return wsPort
}

func GetWsRoute() string {
	return wsRoute
}

func GetWsServer() string {
	return wsServer
}

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

// ACmd : an example Cmd process function
func (i *IPC) ACmd(client *Client) (err error) {

	var nowT = time.Now()
	switch string(client.Msg.Cmd) {
	default:
		retMsg := IpcRsp{
			UnixTime: nowT.Add(time.Since(nowT)).UnixNano(),
			Result:   true,
			Message:  "client.Msg.Cmd",
		}
		client.Ws.WriteMessage(websocket.TextMessage, []byte(retMsg.Serialize()))
	}

	return
}

func (i *IPC) WsHandel() {
	upgrader := websocket.Upgrader{}

	http.HandleFunc(wsRoute, func(w http.ResponseWriter, r *http.Request) {
		if conn, err := upgrader.Upgrade(w, r, nil); err == nil {
			defer conn.Close()
			// stdlog.Println("ipc connected !!")
			for {
				if _, msgCmd, err := conn.ReadMessage(); err == nil {
					var aCmd IpcCmd
					err = json.Unmarshal([]byte(msgCmd), &aCmd)
					client := &Client{Ws: conn, Msg: aCmd}
					i.WsClient <- client
				} else {
					errlog.Println("ipc read error", err)
					break
				}
			}
		} else {
			errlog.Println("ipc conn error", err)
		}
	})

	http.ListenAndServe(wsPort, nil)
}

func SendCmd(aCmd IpcCmd) (res *IpcResult, err error) {
	var (
		rsp []byte
		c   *websocket.Conn
	)

	if c, _, err = websocket.DefaultDialer.Dial(wsServer+wsPort+wsRoute, nil); err == nil {
		defer c.Close()
		if err = c.WriteMessage(websocket.TextMessage, []byte(aCmd.Serialize())); err == nil {
			if _, rsp, err = c.ReadMessage(); err == nil {
				var aMsg IpcRsp
				err = json.Unmarshal([]byte(rsp), &aMsg)
				res = &IpcResult{
					ReqCmd: aCmd,
					Rsp:    aMsg,
				}
			}
		}
	}

	return
}
