package grpc;

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"encoding/json"
	"github.com/CyCoreSystems/ari/v5"
	grpc_engine "google.golang.org/grpc"
	"github.com/gorilla/websocket"
)

type ClientEvent struct {
	ClientId string `json:"client_id"`
	Type string `json:"type"`
	Data map[string]string `json:"data"`
}
var addr = "0.0.0.0:8018"

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    Subprotocols: []string{ "events" },  // <-- add this line
    CheckOrigin: func(r *http.Request) bool {
        return true
    },

} // use default options

func processEvents(c *websocket.Conn, clientId string, wsChan <-chan *ClientEvent) {

	for {
		select {
			case evt := <- wsChan:
				fmt.Println(evt.ClientId);
				if clientId != evt.ClientId {
					continue
				}
				fmt.Println("received client event...");
				mt := websocket.TextMessage
				b, err := json.MarshalIndent(&evt, "", "\t")
				if err != nil {
					fmt.Println("error:", err)
				}
				//message := "hello"
				err = c.WriteMessage(mt, b)
				if err != nil {
					fmt.Println("error: " + err.Error())	
				}
			
			break;
			default:
		
			break;
		}
	}
}
func ws( wsChan <-chan *ClientEvent ) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request)  {
		v := r.URL.Query()
       	clientId := v.Get("clientId")
		   log.Printf("got connection from: %s\r\n", clientId)
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		go processEvents( c, clientId, wsChan )
		defer c.Close()
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}
			//log.Printf("recv: %s", message)
		}
	}
}

func startWebsocketServer( wsChan <-chan *ClientEvent ) {
	wsHandler := ws( wsChan )
	http.HandleFunc("/", wsHandler)
	log.Fatal(http.ListenAndServe(addr, nil))
}
func StartListener(cl ari.Client) {
	wsChan := make( chan *ClientEvent )
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", 9000))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go startWebsocketServer(wsChan)
	fmt.Println("GRPC is running!!");
	s := NewServer(cl, wsChan)

	grpcServer := grpc_engine.NewServer()

	RegisterLineblocsServer(grpcServer, s)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}