package storage

import (
	"fmt"
	"../hash"
	//"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"bufio"
	"crypto/md5"
	"strconv"
	//"math/big"
	"path/filepath"
	"time"
	"../zip"
	"net/http"
	"strings"
)

var Configpath string = "./config"
var Poolpath string = "./TxPool"
var Blockpath string = "./Block"
var Returnpath string = "./Return/"
var tmp string = "./tmp/"

type CHash struct {
	p []byte
	q []byte
	g []byte
	hk []byte
	tk []byte
}

type Tx struct {
	Pk string     // 用户公钥
	Date string   // 生效日期
	Period string    // 有效期
	Info string   // 附加信息
	Status string // 有效状态,valid表示有效，invalid表示无效
	Txhash string // 交易哈希，Pk为r，234项为s，Status为message
	Hash string   // （区块中）与前一个交易哈希的总md5校验和，若是首个交易，则取自身Txhash的校验和
}

// 比较两个交易信息（只比较前6项，不比较Hash项）
func (x Tx) Equal(y Tx) (bool) {
	flag := true
	if x.Pk != y.Pk {
		flag = false
	}
	if x.Date != y.Date {
		flag = false
	}
	if x.Period != y.Period {
		flag = false
	}
	if x.Info != y.Info {
		flag = false
	}
	if x.Status != y.Status {
		flag = false
	}
	if x.Txhash != y.Txhash {
		flag = false
	}
	return flag
}

// 从文件读取交易
func (t *Tx) Load(path string) (error) {
	file, err := os.Open(path)
	if err != nil {
		return err // 打开失败
	}
	defer file.Close()

	// 读取交易信息
	if 1 == 1 {	
		br := bufio.NewReader(file)
		line, _ ,err3 := br.ReadLine()
		if err3 != nil {
			return err3 //读取出错			
		}
		t.Pk = string(line)
		line, _ ,err3 = br.ReadLine()
		if err3 != nil {
			return err3 //读取出错			
		}
		t.Date = string(line)		
		line, _ ,err3 = br.ReadLine()
		if err3 != nil {
			return err3 //读取出错			
		}
		t.Period = string(line)
		line, _ ,err3 = br.ReadLine()
		if err3 != nil {
			return err3 //读取出错			
		}
		t.Info = string(line)
		line, _ ,err3 = br.ReadLine()
		if err3 != nil {
			return err3 //读取出错			
		}
		t.Status = string(line)
		line, _ ,err3 = br.ReadLine()
		if err3 != nil {
			return err3 //读取出错			
		}
		t.Txhash = string(line)
		line, _ ,err3 = br.ReadLine()
		if err3 != nil {
			if err3 != io.EOF {
				return err3 //读取出错
			}
			//读取到文件最后
			t.Hash = ""
			return nil
		}
		t.Hash = string(line)
	}
	return nil
}

// 检查交易哈希正确性
func (t Tx) Check(ch CHash) (flag bool) {
	r := []byte(t.Pk)
	msg := []byte(t.Status)
	var hashout []byte
	s := []byte(t.Date+t.Period+t.Info)
	hash.ChameleonHash(&ch.hk, &ch.p, &ch.q, &ch.g, &msg, &r, &s, &hashout)
	if t.Txhash == fmt.Sprintf("%x",hashout) {
		return true
	}
	return false
}

// 节点收到或生成一个区块后，相应得将交易池中的已被打包的交易删除
// path为区块的路径
func Remove(path string,index int) (error) {
	file,err := os.Open(path+"\\Block"+strconv.Itoa(index))
	if err != nil { 
		fmt.Println("Failed to open file "+path+"\\Block"+strconv.Itoa(index))
	}
	defer file.Close()
	br := bufio.NewReader(file)
	line, _ ,_ := br.ReadLine()
	line, _ ,_ = br.ReadLine()
	cnt,_ := strconv.Atoi(string(line)[8:])
	for i := 0;i <= cnt; i++ {
		var x,y Tx
		flag := false
		x.Load(path+"\\"+strconv.Itoa(i))
		fs,_ := ioutil.ReadDir(Poolpath+"\\a")
		for _,file := range fs{
			y.Load(Poolpath+"\\a\\"+file.Name())
			if x.Equal(y) {
				//fmt.Println(Poolpath+"\\a\\"+file.Name())
				err := os.Remove(Poolpath+"\\a\\"+file.Name())
				if err != nil{
					fmt.Println(err)
				}
				flag = true
				break
			}
		}
		if flag {
			continue
		}
		fs,_ = ioutil.ReadDir(Poolpath+"\\b")
		for _,file := range fs{
			y.Load(Poolpath+"\\b\\"+file.Name())
			if x.Equal(y) {
				//fmt.Println(Poolpath+"\\b\\"+file.Name())
				err := os.Remove(Poolpath+"\\b\\"+file.Name())
				if err != nil{
					fmt.Println(err)
				}
				flag = true
				break
			}
		}
	}
	return nil
}

// 按照输入数字区间自动产生若干交易并打包进交易池
func Auto_gen(tag,start,end int,ch CHash) (err error){
	var newt Tx
	newt.Pk = "0"
	newt.Date = "2020-7-23"
	newt.Period = "365"
	newt.Info = "just a test"
	newt.Status = "valid"
	newt.Txhash = "0"
	newt.Hash = "0"

	var msg = []byte(newt.Status)
	var hashout []byte
   
	for i := start; i <= end; i++ {
		newt.Pk = string(hash.Randgen(&ch.q)[:])
		r := []byte(newt.Pk)
		newt.Info = "Test Tx No." + strconv.Itoa(i)	
		s := []byte(newt.Date+newt.Period+newt.Info)
		hash.ChameleonHash(&ch.hk, &ch.p, &ch.q, &ch.g, &msg, &r, &s, &hashout)
		newt.Txhash = fmt.Sprintf("%x",hashout)
		// fmt.Printf("\n%d:%x\n",i,hashout)
		err = newt.AddTx(tag,strconv.Itoa(i))
		if err != nil{
			return
		}
	}
	return nil
}

// 导入变色龙哈希参数,默认在./config中
func ImportHashParam()(ch CHash,err error){
    var cnt int = 0 // cnt用于统计配置文件完整性

	file,err := os.Open(Configpath)
	if err != nil { 
		fmt.Println("Failed to open the config file ",Configpath)
		return
	}
	defer file.Close()

	br := bufio.NewReader(file)
	for {
		line, isPrefix ,err1 := br.ReadLine()

		if err1 != nil {
			if err1 != io.EOF {
				err = err1 //读取出错
			}
			//读取到文件最后
			break
		}

		if isPrefix { //一行数据字节太长
			fmt.Println("A too long line, seems unexpected.")
			return
		}

		if string(line)[:1] == "p"{
			cnt += 1
			ch.p = []byte(string(line)[2:])
		}

		if string(line)[:1] == "q"{
			cnt += 1
			ch.q = []byte(string(line)[2:])
		}
		
        if string(line)[:1] == "g"{
			cnt += 1
			ch.g = []byte(string(line)[2:])
		}

		if string(line)[:2] == "hk"{
			cnt += 1
			ch.hk = []byte(string(line)[3:])
		}

		if string(line)[:2] == "tk"{
			cnt += 1
			ch.tk = []byte(string(line)[3:])
		}
	}
	if cnt == 5 {
		// fmt.Println("\nimport config success")
	}else{
		// fmt.Println("\nincomplete config file")
		return ch,fmt.Errorf("imcomplete config file")
	}
	return ch,nil
}

// AddTx将交易写入交易池，等待打包
// tag 0 写入a文件夹，1写入b文件夹；index为交易文件索引名
func (t Tx) AddTx(tag int,index string) (err error) {
	var path string 
	path = Poolpath + "\\b\\" + index
	if tag == 0 {
		path = Poolpath + "\\a\\" + index
	}
	file,err := os.Create(path)
	if err != nil { 
		fmt.Println("Failed to open the output file ", path)
		return
	}
	defer file.Close()

	_,err = file.WriteString(t.Pk+"\n"+t.Date+"\n"+t.Period+"\n"+t.Info+"\n"+t.Status+"\n"+t.Txhash)
	if err != nil { 
		fmt.Println(err)
		return
	}

	return
}

// Pack将交易打包成块，并且生成区块hash，压缩后写入tmp文件夹
// tag 0 打包文件夹a内交易， tag 1 打包文件夹b内交易
// index为生成的区块序号
func Pack(tag,index int,ch CHash) (err error) {
	var Txpath string 
	Txpath = Poolpath + "\\b"
	if tag == 0 {
		Txpath = Poolpath + "\\a"
	}
	newblock := Blockpath + "\\" + strconv.Itoa(index)
	e := os.Mkdir(newblock, os.ModePerm)
	if e != nil {
		fmt.Printf("Cant create new block dir")
		err = e
		return
	}
	var tp string
	tmphash := ""
	cnt := 0

	filepath.Walk(Txpath, func (path string, info os.FileInfo, err1 error) error {
		if info.IsDir() {
			return nil
		}
		file, err2 := os.Open(path)
		if err2 != nil {
			return err2// 打开失败
		}
		defer file.Close()

		var t Tx
		// 读取交易信息
		if 1 == 1 {	
			br := bufio.NewReader(file)
			line, _ ,err3 := br.ReadLine()
			if err3 != nil {
				return err3 //读取出错			
			}
			t.Pk = string(line)
			line, _ ,err3 = br.ReadLine()
			if err3 != nil {
				return err3 //读取出错			
			}
			t.Date = string(line)		
			line, _ ,err3 = br.ReadLine()
			if err3 != nil {
				return err3 //读取出错			
			}
			t.Period = string(line)
			line, _ ,err3 = br.ReadLine()
			if err3 != nil {
				return err3 //读取出错			
			}
			t.Info = string(line)
			line, _ ,err3 = br.ReadLine()
			if err3 != nil {
				return err3 //读取出错			
			}
			t.Status = string(line)
			line, _ ,err3 = br.ReadLine()
			if err3 != nil {
				return err3 //读取出错			
			}
			t.Txhash = string(line)
		}
		
		// 交易在块内打包成交易链
		if t.Check(ch) {
			h := md5.New()
			io.WriteString(h,tmphash)
			io.WriteString(h,t.Txhash)
			t.Hash = fmt.Sprintf("%x",h.Sum(nil))
			tmphash = t.Hash
			tp = newblock + "\\" + strconv.Itoa(cnt)
			cnt += 1
			file2,er := os.Create(tp)
			if er != nil { 
				fmt.Println("Failed to create file ", tp)
				return er
			}
			defer file2.Close()
			_,er = file2.WriteString(t.Pk+"\n"+t.Date+"\n"+t.Period+"\n"+t.Info+"\n"+t.Status+"\n"+t.Txhash+"\n"+t.Hash)
			if er != nil { 
				fmt.Println(er)
				return er
			}
		} else {
			fmt.Println("Incorrect Tx!"+path)
			return fmt.Errorf("Incorrect Tx!")
		}
		
		return nil
	})

	// 区块信息，包括区块hash(由上一区块哈希和本区块最后一个交易的哈希求md5校验和)，打包的交易数量，生成时间
    file,err1 := os.Create(newblock + "\\Block" + strconv.Itoa(index))
	if err1 != nil { 
		fmt.Println("Failed to create file ", Blockpath + "\\Block" + strconv.Itoa(index))
		err = err1
		return
	}
	defer file.Close()

	// 读取上个区块信息，创世区块的哈希值为“0”，创世区块不包含任何交易
	file1, err2 := os.Open(Blockpath + "\\" + strconv.Itoa(index-1) + "\\Block" + strconv.Itoa(index-1))
	if err2 != nil { 
		fmt.Println("Failed to open file "+ Blockpath + "\\" + strconv.Itoa(index-1) + "\\Block" + strconv.Itoa(index-1))
		err = err2
		return
	}
	defer file1.Close()
	br := bufio.NewReader(file1)
	line, _ ,_ := br.ReadLine()
	prevhash := string(line)[10:]
	h := md5.New()
	io.WriteString(h,prevhash)
	io.WriteString(h,tmphash)
	_,err1 = file.WriteString("BlockHash:"+fmt.Sprintf("%x",h.Sum(nil)))
	_,err1 = file.WriteString("\nNumOfTx:"+strconv.Itoa(cnt))
	_,err1 = file.WriteString("\nCreateDate:"+time.Now().Format("2006-01-02 15:04"))
	if err1 != nil { 
		fmt.Println(err1)
		err = err1
		return 
	}
	fmt.Println("New Block Created!Block Index:"+strconv.Itoa(index))

	err = zip.ZipBlock(Blockpath+"/"+strconv.Itoa(index)+"/", tmp+strconv.Itoa(index)+".zip", index)
	if err != nil {
		return
	}

	return nil
}

// 撤销指定区块b中指定交易t中的密钥，即把Status位设置为invalid后生成碰撞
func Revoke(b,t int,ch CHash) (error) {
	var tx Tx
	err := tx.Load(Blockpath+"\\"+strconv.Itoa(b)+"\\"+strconv.Itoa(t))
	if err != nil {
		fmt.Println("No such Tx!")
		return err
	}
	if string(tx.Status) != "valid" {
		fmt.Println("Invalid Tx!")
		return fmt.Errorf("Invalid Tx!")
	}
	if !tx.Check(ch) {
		fmt.Println("Incorrect TX!")
		return fmt.Errorf("Incorrect TX!")
	} 
	msg1 := []byte("valid")
	msg2 := []byte("invalid")
	var r1,s1,r2,s2 []byte
	r1 = []byte(tx.Pk)
	s1 = []byte(tx.Date+tx.Period+tx.Info)
	hash.GenerateCollision(&ch.hk, &ch.tk, &ch.p, &ch.q, &ch.g, &msg1, &msg2, &r1, &s1, &r2, &s2)
	tx.Pk = string(r2)
	tx.Info = string(s2)
	tx.Date = ""
	tx.Period = ""
	tx.Status = "invalid"
	// var hash1,hash2,hash3 []byte
	// r3 := []byte(tx.Pk)
	// s3 := []byte(tx.Date+tx.Period+tx.Info)
	// hash.ChameleonHash(&hk, &p, &q, &g, &msg1, &r1, &s1, &hash1)
	// hash.ChameleonHash(&hk, &p, &q, &g, &msg2, &r2, &s2, &hash2)
	// hash.ChameleonHash(&hk, &p, &q, &g, &msg2, &r3, &s3, &hash3)
	// fmt.Printf("\n%x\n%x\n%x\n",hash1,hash2,hash3)
	if tx.Check(ch) {
		os.Remove(Blockpath+"\\"+strconv.Itoa(b)+"\\"+strconv.Itoa(t))
		file,_ := os.Create(Blockpath+"\\"+strconv.Itoa(b)+"\\"+strconv.Itoa(t))
		defer file.Close()
		file.WriteString(tx.Pk+"\n"+tx.Date+"\n"+tx.Period+"\n"+tx.Info+"\n"+tx.Status+"\n"+tx.Txhash+"\n"+tx.Hash)
		fmt.Printf("Revoke success!(Block: %d , Tx: %d) has been revoked",b,t)
		return nil
	} else {
		fmt.Println(tx)
		return fmt.Errorf("Generate Collsiom failed!")
	}
}

// 计算某笔的交易的变色龙哈希值
func GetCHash(msg,r,s []byte,ch CHash) (hashout []byte) {
	hash.ChameleonHash(&ch.hk, &ch.p, &ch.q, &ch.g, &msg, &r, &s, &hashout)
	return hashout
}

// path为区块路径
// 交易被打包后，各节点根据Returnpath中储存的客户端的网络信息
// 将交易的所在区块位置返回给之前发起申请的客户
func Tellclient(path string,index int) (error) {
	file,err := os.Open(path+"\\Block"+strconv.Itoa(index))
	if err != nil { 
		fmt.Println("Failed to open file "+path+"\\Block"+strconv.Itoa(index))
	}
	defer file.Close()
	br := bufio.NewReader(file)
	line, _ ,_ := br.ReadLine()
	line, _ ,_ = br.ReadLine()
	cnt,_ := strconv.Atoi(string(line)[8:])
	fs,_ := ioutil.ReadDir(Returnpath)
	for i:=0; i<=cnt; i++ {
		var x Tx
		x.Load(path+"/"+strconv.Itoa(i))
		for _,file := range fs{
			if file.Name()==x.Pk {
				f,err := os.Open(Returnpath + file.Name())
				if err != nil {
					return err
				}
				rd := bufio.NewReader(f)
				rd1,_,_ := rd.ReadLine()
				url := string(rd1)
				rd2,_,_ := rd.ReadLine()
				f.Close()
				defer os.Remove(Returnpath + file.Name())
				ctnt := strconv.Itoa(index)+"@@"+strconv.Itoa(i)+"@@"+string(rd2)
				// ss := []string{strconv.Itoa(index), strconv.Itoa(cnt), pk, date, period, info}
				// msg := strings.NewReader("Your request has been packed!Located at Block: "+strconv.Itoa(index)+"  Tx: "+strconv.Itoa(cnt)+"@@"+strconv.Itoa(index)+"@@"+strconv.Itoa(cnt))
				msg := strings.NewReader(ctnt)
				_,err = http.Post("http://"+url, "text/plain", msg)
				if err != nil {
					fmt.Printf("%s\n",err)
				}
				break
			}
		}
	}
	return nil
}

