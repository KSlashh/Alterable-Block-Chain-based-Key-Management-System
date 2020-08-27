package command

import (
	"github.com/goraft/raft"
	"../storage"
	"../zip"
	"io/ioutil"
	"strconv"
	"bytes"
	"os"
)

var Blockpath = "./Block/"


// This command revokes a key.
type RevokeCommand struct {
	Block   int `json:"block"`
	Tx int `json:"tx"`
}

// Creates a new revoke command.
func NewRevokeCommand(b int, t int) *RevokeCommand {
	return &RevokeCommand{
		Block:   b,
		Tx: t,
	}
}

// The name of the command in the log.
func (c *RevokeCommand) CommandName() string {
	return "revoke"
}

// Revoke a transaction.
func (c *RevokeCommand) Apply(server raft.Server) (interface{}, error) {

	ch,err := storage.ImportHashParam()
	if err != nil {
		return nil,err
	}
	err = storage.Revoke(c.Block, c.Tx, ch)
	if err != nil {
		return nil,err
	}
	return nil, nil
}




// This command adds a new tx.
type AddTxCommand struct {
	Pk string     `json:"pk"`
	Date string   `json:"date"`
	Period string   `json:"period"` 
	Info string   `json:"info"`
	Status string   `json:"status"`
	Txhash string   `json:"txhash"`
}

// Creates a new tx command.
func NewAddTxCommand(pk,date,period,info,status,txhash string) *AddTxCommand {
	return &AddTxCommand{
		Pk : pk ,
		Date : date ,
		Period : period ,
		Info : info ,
		Status : status ,
		Txhash : txhash ,
	}
}

// The name of the command in the log.
func (c *AddTxCommand) CommandName() string {
	return "New Transaction"
}

// Writes a tx to Txpool.
func (c *AddTxCommand) Apply(server raft.Server) (interface{}, error) {
	var tx storage.Tx
	tx.Pk = c.Pk
	tx.Date = c.Date
	tx.Period = c.Period
	tx.Info = c. Info
	tx.Status = c.Status
	tx.Txhash = c.Txhash
	err := tx.AddTx(0, tx.Pk)
	if err != nil {
		return nil,err
	}
	return nil, nil
}



// This command packs a new block.
type PackCommand struct {
	Index   int `json:"index"`
	Content []byte `json:"content"`
}

// Creates a new block command.
func NewPackCommand(a int, b []byte) *PackCommand {
	return &PackCommand{
		Index:  a,
		Content: b,
	}
}

// The name of the command in the log.
func (c *PackCommand) CommandName() string {
	return "pack"
}

//Pack some tx to a block.
func (c *PackCommand) Apply(server raft.Server) (interface{}, error) {
	s := strconv.Itoa(c.Index)
	r := bytes.NewReader(c.Content)
	v,err := ioutil.ReadAll(r)
	if err != nil {
		return nil,err
	}
	err = ioutil.WriteFile(s+".zip", v, 0644)
	if err != nil {
		return nil,err
	}
	defer os.Remove(s+".zip")
	err = zip.UnzipBlock(s+".zip", Blockpath + s)
	if err != nil {
		return nil,err
	}
	storage.Tellclient(Blockpath+s, c.Index)
	err = storage.Remove(Blockpath+s, c.Index)
	if err != nil {
		return nil,err
	}

	return nil,nil
}
