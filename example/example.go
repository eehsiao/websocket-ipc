// Author :		Eric<eehsiao@gmail.com>

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	ipc "github.com/eehsiao/websocket-ipc"
	"github.com/gorilla/websocket"
	"github.com/takama/daemon"
)

type Service struct {
	daemon.Daemon
}

var (
	dependencies   = []string{}
	stdlog, errlog *log.Logger
	service        *Service
	wsIpc          *ipc.IPC

	verNo = "v0.0.1"
)

func init() {
	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
}

func main() {
	args := os.Args
	if len(args) == 1 {
		errlog.Println("need a command ex: example start")
		return
	}

	srv, err := daemon.New("deamon-example", "deamon-example", dependencies...)
	if err != nil {
		errlog.Println("Error: ", err)
		os.Exit(1)
	}
	service = &Service{srv}
	if service == nil {
		errlog.Println("Error: no service")
		os.Exit(1)
	}

	switch args[1] {
	case "install":
		if status, err := service.Install(); err != nil {
			errlog.Println(status, "\nError: ", err)
			os.Exit(1)
		}
	case "remove":
		if status, err := service.Remove(); err != nil {
			errlog.Println(status, "\nError: ", err)
			os.Exit(1)
		}
	case "start":
		if status, err := service.Start(); err != nil {
			errlog.Println(status, "\nError: ", err)
			os.Exit(1)
		}

		wsIpc = ipc.NewIpc(ipcCmd, stdlog, errlog)
		go wsIpc.WsHandel()
		status := bgLoop()
		stdlog.Println(status)
	default:
		var aCmd ipc.IpcCmd
		aCmd.Cmd = args[1]
		aCmd.CmdFlag = "test"
		if msg, err := ipc.SendCmd(aCmd); err == nil {
			fmt.Printf("%v\n%s => %s\n", msg, msg.ReqCmd.Cmd, msg.Rsp.Message)
		} else {
			fmt.Println(err)
		}
	}
}

func bgLoop() string {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	for {
		select {
		case sysInterrupt := <-interrupt:
			stdlog.Println("Got signal:", sysInterrupt)
			if status, err := service.Stop(); err != nil {
				errlog.Println(status, "\nError: ", err)
				os.Exit(1)
			}
			if sysInterrupt == os.Interrupt {
				return "Daemon was interruped by system signal"
			}
			return "Daemon was killed"
		case client := <-wsIpc.WsClient:
			go ipcCmd(client)
		}
	}
}

func ipcCmd(client *ipc.Client) (err error) {
	stdlog.Println(string(client.Msg.Cmd))

	var nowT = time.Now()
	retMsg := ipc.IpcRsp{
		UnixTime: nowT.Add(time.Since(nowT)).UnixNano(),
		Result:   true,
		Message:  client.Msg.Cmd,
	}

	switch string(client.Msg.Cmd) {
	case "stop":
		if status, err := service.Stop(); err != nil {
			errlog.Println(status, "\nError: ", err)
		}
		stdlog.Println(retMsg.Serialize())
		client.Ws.WriteMessage(websocket.TextMessage, []byte(retMsg.Serialize()))
		os.Exit(1)
	case "version":
		retMsg.Message = verNo
		stdlog.Println(retMsg.Serialize())
		client.Ws.WriteMessage(websocket.TextMessage, []byte(retMsg.Serialize()))
	default:
		stdlog.Println(retMsg.Serialize())
		client.Ws.WriteMessage(websocket.TextMessage, []byte(retMsg.Serialize()))
	}

	return
}
