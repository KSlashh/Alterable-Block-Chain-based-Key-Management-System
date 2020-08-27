package main

import (
	"fmt"
	"net/http"
	"os"
	"bufio"
	"io/ioutil"
	"bytes"
	"strconv"
	"time"
	"math/big"
	"crypto/rand"
	"strings"
)

var q []byte = []byte("dfabf534e990af8350f09cf6b500d44f")
var Keys string = "./Key/"
var IP string = "127.0.0.1"
var port string = "32003"
var Address string = "127.0.0.1:32003"

type Tx struct {
	Pk string     // 用户公钥
	Date string   // 生效日期
	Period string    // 有效期
	Info string   // 附加信息
	Status string // 有效状态,valid表示有效，invalid表示无效
	Txhash string // 交易哈希，Pk为r，234项为s，Status为message
	Hash string   // （区块中）与前一个交易哈希的总md5校验和，若是首个交易，则取自身Txhash的校验和
}

type RevokeReq struct {
	Pk string     // 用户公钥
	block string  // 所在区块高度(区块号)
	tx string     // 所在交易编号
}

func main() {
	if !PathExists("./Key") {
		os.Mkdir("./Key",os.ModePerm)
	}
	if !PathExists("./tmp") {
		os.Mkdir("./tmp",os.ModePerm)
	}

	fmt.Printf("Choose a ip addr for server to reply (host:port) : ")
	reader := bufio.NewReader(os.Stdin)
	adr, _ := reader.ReadString('\n')
	ipp := strings.Split(string(adr[:len(adr)-2]),":")
	if len(ipp) == 2 {
		IP = ipp[0]
		port = ipp[1]
		Address = IP+":"+port
	}

	srv := http.Server{
		Addr: ":"+port ,
		Handler: &httpCVAPI{
		},
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			fmt.Printf("\n*********\n%s\n*********\n", err)
		}
	}()

	time.Sleep(1)

	for true{
		fmt.Printf("功能选择(1:密钥认证 , 2:密钥撤销 , 3:密钥查询 , 4:打包 , 0:退出): ")
		reader := bufio.NewReader(os.Stdin)
		str, _ := reader.ReadString('\n')
		switch string(str[0]) {
			case "0": 
			    os.Exit(0)
			case "1": 
				NewTx()
			case "2": 
				Revoke()
			case "3": 
				Get()
			case "4":
				fmt.Printf("Raft leader's addr(host:port): ")
				reader := bufio.NewReader(os.Stdin)
				url, _ := reader.ReadString('\n')
				url = url[:len(url)-2]
				fmt.Printf("New block index: ")
				index, _ := reader.ReadString('\n')
				index = index[:len(index)-2]
				msg := strings.NewReader(index)
				rsp,err := http.Post("http://"+url+"/pack","text/plain",msg)
				if err != nil {
					fmt.Printf("请求上传失败！请检查服务器地址及网络状况！:%s\n\n",err)
					return
				}
				defer rsp.Body.Close()
				v,_ := ioutil.ReadAll(rsp.Body)
				fmt.Printf("服务器应答：%s\n\n", string(v))
			default: 
			    fmt.Println("无效功能！")
			    break
	    }
	}
}

func NewTx() {
	var t Tx
	var url string
	t.Date = time.Now().Format("2006-01-02")
	t.Info = "Default Msg"
	t.Status = "valid"
	t.Txhash = "0"
	t.Hash = "0"

	fmt.Printf("\n密钥来源(1:系统生成 0:用户输入): ")
	reader := bufio.NewReader(os.Stdin)
	s, _ := reader.ReadString('\n')
	switch string(s[0]) {
		case "0": 
			s, _ = reader.ReadString('\n')
			t.Pk = s[:len(s)-2]
		default: 
			t.Pk = string(Randgen(&q)[:])
			fmt.Printf("自动生成128位密钥字符串: %s\n",t.Pk)
	}

	fmt.Printf("密钥有效期/天(默认365)：")
	s, _ = reader.ReadString('\n')
	_,err := strconv.Atoi(s[:len(s)-2])
	switch err {
		case nil :
			t.Period = s[:len(s)-2]
		default:
			t.Period = "365"
	}

	fmt.Printf("输入附加信息: ")
	s, _ = reader.ReadString('\n')
	t.Info = s[:len(s)-2]

	fmt.Printf("需要送达的服务器地址(Format:http://127.0.0.1:8080): ")
	s, _ = reader.ReadString('\n')
	url = s[:len(s)-2]

	path := "./tmp/"+t.Pk
	file,err := os.Create(path)
	if err != nil { 
		fmt.Println("Failed to creat Tx file ", path)
		return
	}
	defer os.Remove(path)

	//fmt.Printf(t.Pk+"\n"+t.Date+"\n"+t.Period+"\n"+t.Info+"\n"+t.Status+"\n"+t.Txhash+"\n")
	_,err = file.WriteString(t.Pk+"\n"+t.Date+"\n"+t.Period+"\n"+t.Info+"\n"+t.Status+"\n"+t.Txhash)
	if err != nil { 
		fmt.Println(err)
		file.Close()
		return
	}
	file.Close()
	
	content,err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	r := bytes.NewReader(content)
	resp,err := http.Post(url+"/new/"+Address,"application/octet-stream",r)
	if err != nil {
		fmt.Printf("密钥上传失败！请检查服务器地址及网络状况！\n\n")
		return
	}
	defer resp.Body.Close()
	v,err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("服务器应答：%s\n\n", string(v))
}

func Revoke() {
	var r RevokeReq
	var url string

	fmt.Printf("\n输入需要撤销的公钥: ")
	reader := bufio.NewReader(os.Stdin)
	s, _ := reader.ReadString('\n')
	r.Pk = s[:len(s)-2]

	fmt.Printf("输入公钥所在区块号: ")
	s, _ = reader.ReadString('\n')
	r.block = s[:len(s)-2]

	fmt.Printf("输入公钥所在交易编号: ")
	s, _ = reader.ReadString('\n')
	r.tx = s[:len(s)-2]

	fmt.Printf("需要送达的服务器地址(Format:http://127.0.0.1:8080): ")
	s, _ = reader.ReadString('\n')
	url = s[:len(s)-2]

	rs := []string{r.Pk , r.block , r.tx}
	msg := strings.Join(rs,",")
	message := strings.NewReader(msg)
	request,_ := http.NewRequest("POST",url+"/revoke",message)
	client := &http.Client{}
	resp,err := client.Do(request)
	if err != nil {
		fmt.Printf("请求上传失败！请检查服务器地址及网络状况！\n\n")
		return
	}
	defer resp.Body.Close()
	v,err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("服务器应答：%s\n\n", string(v))
}

func Get() {
	var url,b,tx string

	fmt.Printf("\n需要查询的密钥所在区块号: ")
	reader := bufio.NewReader(os.Stdin)
	s, _ := reader.ReadString('\n')
	b = s[:len(s)-2]

	fmt.Printf("需要查询的密钥所在交易号: ")
	s, _ = reader.ReadString('\n')
	tx = s[:len(s)-2]

	fmt.Printf("需要送达的服务器地址(Format:http://127.0.0.1:8080): ")
	s, _ = reader.ReadString('\n')
	url = s[:len(s)-2]

	rsp,err := http.Get(url+"/"+b+"/"+tx)
	if err != nil {
		fmt.Printf("请求上传失败！请检查服务器地址及网络状况！\n\n")
		return
	}
	defer rsp.Body.Close()
	v,_ := ioutil.ReadAll(rsp.Body)
	m := strings.Split(string(v),"@@")
	if m[0] != "ok!" {
		fmt.Println("验证失败！未找到相应交易！")
		fmt.Println(url+"/"+b+"/"+tx)
		return 
	}
	fmt.Printf("查询成功！查询到密钥信息如下：\n密钥：%s\n生效日期：%s\n有效期：%s 天\n是否有效：%s\n附加信息：%s\n\n", m[1], m[2], m[3], m[4], m[5])

	return
}


func Randgen(upperBoundHex *[]byte) []byte {
	upperBoundBig := new(big.Int)
	upperBoundBig, success := upperBoundBig.SetString(string(*upperBoundHex), 16)
	if success != true {
		fmt.Printf("Conversion from hex: %s to bigInt failed.", upperBoundHex)
	}

	randomBig, err := rand.Int(rand.Reader, upperBoundBig)
	if err != nil {
		fmt.Printf("Generation of random bigInt in bounds [0...%v] failed.", upperBoundBig)
	}

	return []byte(fmt.Sprintf("%x", randomBig))
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


type httpCVAPI struct {
}

func (h *httpCVAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//key := r.RequestURI
	defer r.Body.Close()
	switch {
	case r.Method == "PUT":
		w.WriteHeader(http.StatusBadRequest)
	case r.Method == "GET":
		w.WriteHeader(http.StatusBadRequest)
	case r.Method == "POST":
		v, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("\n*********\n%s\n*********\n", err)
			http.Error(w, "Failed on POST", http.StatusBadRequest)
			return
		}

		ss := strings.Split(string(v),"@@")
		if len(ss) <= 5 {
			fmt.Printf("\n*********\nStrange msg:%s\n*********\n",string(v))
			http.Error(w, "Invalid message form", http.StatusBadRequest)
			return
		} 
		w.Write([]byte("Ok."))

		f,err := os.Create(Keys + ss[2])
		if err != nil {
			fmt.Printf("\n*********\nFail to Create key file:%s\n*********\n",err)
			return
		}
		defer f.Close()
		f.WriteString("BlockHeight:"+ss[0])
		f.WriteString("\nTxIndex:"+ss[1])
		f.WriteString("\nKey:"+ss[2])
		f.WriteString("\nEffectiveDate:"+ss[3])
		f.WriteString("\nDuration:"+ss[4])
		f.WriteString("\nAttachedInfo:"+ss[5])
		fmt.Printf("\n*********\nKey:%s has been packed at block %s,tx %s\n*********\n",ss[2],ss[0],ss[1])
		return
		
	case r.Method == "DELETE":
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.Header().Set("Allow", "PUT")
		w.Header().Add("Allow", "GET")
		w.Header().Add("Allow", "POST")
		w.Header().Add("Allow", "DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
