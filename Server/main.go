package main

import (
	"flag"
	"fmt"
	"github.com/goraft/raft"
	"./command"
	"./server"
	"log"
	"math/rand"
	"os"
	"time"
)

var verbose bool
var trace bool
var debug bool
var host string
var port int
var join string

func init() {
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.BoolVar(&trace, "trace", false, "Raft trace debugging")
	flag.BoolVar(&debug, "debug", false, "Raft debugging")
	flag.StringVar(&host, "h", "localhost", "hostname")
	flag.IntVar(&port, "p", 4001, "port")
	flag.StringVar(&join, "join", "", "host:port of leader to join")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [arguments] <data-path> \n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	log.SetFlags(0)
	flag.Parse()
	if verbose {
		log.Print("Verbose logging enabled.")
	}
	if trace {
		raft.SetLogLevel(raft.Trace)
		log.Print("Raft trace debugging enabled.")
	} else if debug {
		raft.SetLogLevel(raft.Debug)
		log.Print("Raft debugging enabled.")
	}

	rand.Seed(time.Now().UnixNano())

	// Setup commands.
	raft.RegisterCommand(&command.RevokeCommand{})
	raft.RegisterCommand(&command.AddTxCommand{})
	raft.RegisterCommand(&command.PackCommand{})

	// Set up blockchain related dirs
	if !PathExists("./Block") {
		os.Mkdir("./Block",os.ModePerm)
	}
	if !PathExists("./TxPool") {
		os.Mkdir("./TxPool",os.ModePerm)
	}
	if !PathExists("./TxPool/a") {
		os.Mkdir("./TxPool/a",os.ModePerm)
	}
	if !PathExists("./TxPool/b") {
		os.Mkdir("./TxPool/b",os.ModePerm)
	}
	if !PathExists("./Return") {
		os.Mkdir("./Return",os.ModePerm)
	}
	if !PathExists("./tmp") {
		os.Mkdir("./tmp",os.ModePerm)
	}
	if !PathExists("./Block/0") {
		os.Mkdir("./Block/0",os.ModePerm)
		f,err := os.Create("./Block/0/Block0")
		if err != nil {
			log.Println(err)
		}
		f.WriteString("BlcokHash:0\nNumOfTx:0\nCreateDate:"+time.Now().Format("2006-01-02 15:04"))
		f.Close()
	}

	// Set the data directory.
	if flag.NArg() == 0 {
		flag.Usage()
		log.Fatal("Data path argument required")
	}
	path := flag.Arg(0)
	if err := os.MkdirAll(path, 0744); err != nil {
		log.Fatalf("Unable to create path: %v", err)
	}

	log.SetFlags(log.LstdFlags)
	s := server.New(path, host, port)
	log.Fatal(s.ListenAndServe(join))
}

func PathExists(path string) (bool) {
    _, err := os.Stat(path)
    if err == nil {
        return true
    }
    if os.IsNotExist(err) {
        return false
    }
    return false
}