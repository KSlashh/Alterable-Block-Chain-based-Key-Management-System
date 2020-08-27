
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/goraft/raft"
	"../command"
	"../storage"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"sync"
	"time"
	"strconv"
	"strings"
	"os"
)

var Poolpath string = "./TxPool"
var Blockpath string = "./Block"
var Returnpath string = "./Return/"
var tmp string = "./tmp/"
var tag int = 0

// The raftd server is a combination of the Raft server and an HTTP
// server which acts as the transport.
type Server struct {
	name       string
	host       string
	port       int
	path       string
	router     *mux.Router
	raftServer raft.Server
	httpServer *http.Server
	mutex      sync.RWMutex
}

// Creates a new server.
func New(path string, host string, port int) *Server {
	s := &Server{
		host:   host,
		port:   port,
		path:   path,
		router: mux.NewRouter(),
	}

	// Read existing name or generate a new one.
	if b, err := ioutil.ReadFile(filepath.Join(path, "name")); err == nil {
		s.name = string(b)
	} else {
		s.name = fmt.Sprintf("%07x", rand.Int())[0:7]
		if err = ioutil.WriteFile(filepath.Join(path, "name"), []byte(s.name), 0644); err != nil {
			panic(err)
		}
	}

	return s
}

// Returns the connection string.
func (s *Server) connectionString() string {
	return fmt.Sprintf("http://%s:%d", s.host, s.port)
}

// Starts the server.
func (s *Server) ListenAndServe(leader string) error {
	var err error

	log.Printf("Initializing Raft Server: %s", s.path)

	// Initialize and start Raft server.
	transporter := raft.NewHTTPTransporter("/raft", 200*time.Millisecond)
	s.raftServer, err = raft.NewServer(s.name, s.path, transporter, nil, nil, "")
	if err != nil {
		log.Fatal(err)
	}
	transporter.Install(s.raftServer, s)
	s.raftServer.Start()

	if leader != "" {
		// Join to leader if specified.

		log.Println("Attempting to join leader:", leader)

		if !s.raftServer.IsLogEmpty() {
			log.Fatal("Cannot join with an existing log")
		}
		if err := s.Join(leader); err != nil {
			log.Fatal(err)
		}

	} else if s.raftServer.IsLogEmpty() {
		// Initialize the server by joining itself.

		log.Println("Initializing new cluster")

		_, err := s.raftServer.Do(&raft.DefaultJoinCommand{
			Name:             s.raftServer.Name(),
			ConnectionString: s.connectionString(),
		})
		if err != nil {
			log.Fatal(err)
		}

	} else {
		log.Println("Recovered from log")
	}

	log.Println("Initializing HTTP server")

	// Initialize and start HTTP server.
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	s.router.HandleFunc("/{key1}/{key2}", s.searchHandler).Methods("GET")
	s.router.HandleFunc("/new/{key}", s.newtxHandler).Methods("POST")
	s.router.HandleFunc("/revoke", s.revokeHandler).Methods("POST")
	s.router.HandleFunc("/join", s.joinHandler).Methods("POST")
	s.router.HandleFunc("/pack", s.packHandler).Methods("POST")

	log.Println("Listening at:", s.connectionString())

	return s.httpServer.ListenAndServe()
}

// This is a hack around Gorilla mux not providing the correct net/http
// HandleFunc() interface.
func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.router.HandleFunc(pattern, handler)
}

// Joins to the leader of an existing cluster.
func (s *Server) Join(leader string) error {
	command := &raft.DefaultJoinCommand{
		Name:             s.raftServer.Name(),
		ConnectionString: s.connectionString(),
	}

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(command)
	resp, err := http.Post(fmt.Sprintf("http://%s/join", leader), "application/json", &b)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (s *Server) joinHandler(w http.ResponseWriter, req *http.Request) {
	command := &raft.DefaultJoinCommand{}

	if err := json.NewDecoder(req.Body).Decode(&command); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := s.raftServer.Do(command); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) revokeHandler(w http.ResponseWriter, r *http.Request) {
	var tx storage.Tx
	v, _ := ioutil.ReadAll(r.Body)
	str := strings.Split(string(v),",")
	fmt.Printf("Recevie Rovoke Request for: Block %s , Tx %s , Pk: %s\n",str[1],str[2],str[0])
	b,err := strconv.Atoi(str[1])
	t,err := strconv.Atoi(str[2])
	if err != nil {
		w.Write([]byte(err.Error()))
		return 
	}

	tx.Load(Blockpath+"/"+str[1]+"/"+str[2])
	if tx.Pk != str[0] {
		w.Write([]byte("Invalid Revoke Request:Unmatch Key Information!"))
		return
	}

	_, err = s.raftServer.Do(command.NewRevokeCommand(b,t))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		peers := s.raftServer.Peers()
		w.Write([]byte("Current Leader:"+peers[s.raftServer.Leader()].ConnectionString))
		return
	}

	w.Write([]byte("All Right,The Key is Invalid Now :)"))
}

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//key := vars["key"]
	path := Blockpath + "/" + vars["key1"] + "/" + vars["key2"] 
	var t storage.Tx
	err := t.Load(path)
	if err != nil {
		w.Write([]byte("err!@@"+path))
		//w.WriteHeader(http.StatusBadRequest)
		return
	}
	msg :=[] string{"ok!",t.Pk,t.Date,t.Period,t.Status,t.Info}
	w.Write([]byte(strings.Join(msg,"@@")))
}

func (s *Server) newtxHandler(w http.ResponseWriter, r *http.Request) {
	var tx storage.Tx
	vars := mux.Vars(r)
	key := vars["key"]

	v, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Failed to read on POST (%v)\n", err)
		http.Error(w, "Failed on POST", http.StatusBadRequest)
		return
	}
	tmpf := tmp + strconv.Itoa(rand.Intn(100000))
	// f,err := os.Create(tmpf)
	// if err != nil {
	// 	fmt.Println(err)
	// 	w.Write([]byte(err.Error()))
	// 	return
	// }
	// f.Close()
	if err := ioutil.WriteFile(tmpf, v, 0644); err != nil {
		fmt.Println(err)
		w.Write([]byte(err.Error()))
		return
	}
	tx.Load(tmpf)
	defer os.Remove(tmpf)

	var ch storage.CHash
	ch,err = storage.ImportHashParam()
	if err != nil {
		w.Write([]byte("Server failed!"))
		return
	}

	var hashout []byte
	msg := []byte(tx.Status)
	r1 := []byte(tx.Pk)
	s1 := []byte(tx.Date + tx.Period + tx.Info)
	hashout = storage.GetCHash(msg, r1 , s1 , ch)
	tx.Txhash = fmt.Sprintf("%x",hashout)

	_, err = s.raftServer.Do(command.NewAddTxCommand(tx.Pk,tx.Date,tx.Period,tx.Info,tx.Status,tx.Txhash))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		peers := s.raftServer.Peers()
		w.Write([]byte("Current Leader:"+peers[s.raftServer.Leader()].ConnectionString))
		return
	}

	f,err := os.Create(Returnpath + tx.Pk)
	if err != nil {
		fmt.Printf("Fail to return to client")
		w.Write([]byte("Your request has been sent to the transaction pool.But you might not get any more answer for some reason."))
		return
	}
	defer f.Close()
	ss := []string{tx.Pk, tx.Date, tx.Period ,tx.Info}
	f.WriteString(key+"\n"+strings.Join(ss,"@@"))
	
	fmt.Printf("Get New Key Authentication Request\n")
	w.Write([]byte("Your request has been sent to the transaction pool.Wait for packing...."))

}

func (s *Server) packHandler(w http.ResponseWriter, r *http.Request) {
	v, _ := ioutil.ReadAll(r.Body)
	index,err := strconv.Atoi(string(v))
	if err != nil {
		fmt.Println(err)
		w.Write([]byte(err.Error()))
		return
	}
	ch,_ := storage.ImportHashParam()
	storage.Pack(tag,index,ch)
	content,_ := ioutil.ReadFile(tmp+string(v)+".zip")
	defer os.Remove(tmp+string(v)+".zip")
	_, err = s.raftServer.Do(command.NewPackCommand(index, content))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		peers := s.raftServer.Peers()
		w.Write([]byte("Current Leader:"+peers[s.raftServer.Leader()].ConnectionString))
		return
	}
	w.Write([]byte("Ok,packed :)"))
}