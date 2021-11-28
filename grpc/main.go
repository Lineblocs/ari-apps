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

var addr = "0.0.0.0:8018"

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    Subprotocols: []string{ "events" },  // <-- add this line
    CheckOrigin: func(r *http.Request) bool {
        return true
    },

} // use default options

func processEvents(c *websocket.Conn, clientId string, wsChan <-chan *ClientEvent, stopChan <-chan bool) {

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
			case _ = <- stopChan:
				fmt.Println("closing event processor..")
				return
			break;
			default:
		
			break;
		}
	}
}
func ws(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	stopChan := make( chan bool )
	clientId := v.Get("clientId")
	wsChan := createWSChan(clientId)
	log.Printf("got connection from: %s\r\n", clientId)
	log.Printf("Req: %s %s\n", r.Host, r.URL.Path) 
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	go processEvents( c, clientId, wsChan, stopChan )
	defer c.Close()
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			log.Print("error: " +  err.Error())
			stopChan <- true
			c.Close()
			break
		}
		//log.Printf("recv: %s", message)
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "text/plain")
  fmt.Fprintf(w, "OK\n")
}

func startWebsocketServer( ) {
	http.HandleFunc("/", ws)
	http.HandleFunc("/healthz", healthz)
	log.Fatal(http.ListenAndServe(addr, nil))
}
func StartListener(cl ari.Client) {
	return
	wsChan := make( chan *ClientEvent )
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", 9000))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go startWebsocketServer()
	fmt.Println("GRPC is running!!");
	s := NewServer(cl, wsChan)

	grpcServer := grpc_engine.NewServer()

	RegisterLineblocsServer(grpcServer, s)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}