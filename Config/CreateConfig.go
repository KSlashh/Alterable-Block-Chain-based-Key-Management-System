package main

import (
	"./hash"
	"fmt"
	"os"
)

func main() {
	var p, q, g, hk, tk []byte
	hash.Keygen(128, &p, &q, &g, &hk, &tk)
	f,err := os.Create("./config")
	if err != nil { 
		fmt.Println("Failed to creat config file ")
		return
	}
	defer f.Close()
	f.WriteString("p:"+string(p)+"\nq:"+string(q)+"\ng:"+string(g)+"\nhk:"+string(hk)+"\ntk:"+string(tk))
	fmt.Println("Generate config file succeed")
}