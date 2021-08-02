package main

import (
	"fmt"
	"net"
	"strings"
)

// Person 用户结构
type Person struct {
	C chan string
	Name string
	Addr string
}

//map存储用户
var onlinePerson map[string]Person

//全局channel传递消息
var massage  = make(chan string)

func MakeMsg(person Person,msg string)(buf string){
	buf = "> ["+person.Name+"]: "+msg
	return
}

func SendMsgToPerson(person Person, conn net.Conn){
	for msg := range person.C{
		_, err := conn.Write([]byte(msg))
		if err != nil {
			return
		}
	}
}

// UserName 修改name
func UserName(conn net.Conn,person *Person)  {
	BufName :=make([]byte,32)
	m ,err := conn.Read(BufName)
	if m == 0 {
		_, err := conn.Write([]byte("未修改成功...\n"))
		if err != nil {
			return
		}
	}
	if err != nil {
		fmt.Println("read Name error",err)
	}
	name := string(BufName[:m])
	name = strings.Replace(name,"\n","",-1)
	name = strings.Replace(name,"\r","",-1)
	name = strings.Trim(name," ")
	person.Name = name      //不能只修改当前用户的名字
	onlinePerson[person.Addr] = *person    //要把用户表一并修改
}

// ExecCommend 执行命令
func ExecCommend(cmd string,conn net.Conn,person *Person){
	switch cmd {
	case "ls":
		_, err := conn.Write([]byte("**************当前用户列表**************\n"))
		if err != nil {
			return
		}
		for _, ps := range onlinePerson {
			_, err := conn.Write([]byte(ps.Addr + " : " + ps.Name + "\n"))
			if err != nil {
				return
			}
		}
		_, err = conn.Write([]byte("****************************************\n"))
		if err != nil {
			return
		}
	case "rename":
		_, err := conn.Write([]byte("请输入您的新名字(不超过32个字)："))
		if err != nil {
			return
		}
		//修改名字
		UserName(conn,person)
		_, err = conn.Write([]byte("修改成功~~\n"))
		if err != nil {
			return
		}
	case "exit":
		_, err := conn.Write([]byte("您已退出，欢迎下次光临！\n"))
		if err != nil {
			return
		}
		err = conn.Close()
		if err != nil {
			return
		}
		delete(onlinePerson, person.Addr)
	default:
		//conn.Write(buf[1:n])
		_, err := conn.Write([]byte("[Error]没有此命令\n"))
		if err != nil {
			return
		}
	}
}

func cmdTrim(cmd string) string{
	cmd = strings.Replace(cmd,"\n","",-1)
	cmd = strings.Replace(cmd,"\r","",-1)
	cmd = strings.Replace(cmd," ","",-1)
	return cmd
}
func HandlerConnect(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("conn Close error",err)
			return
		}
	}(conn)
	//获取用户IP
	ipAddr := conn.RemoteAddr().String()
	//创建用户信息   默认用户名为IP+Port
	person := Person{
		C:    make(chan string),
		Name: ipAddr,
		Addr: ipAddr,
	}
	//往map里添加用户
	onlinePerson[ipAddr] = person

	//用来给当前用户发送消息
	go SendMsgToPerson(person,conn)
	//发送用户上线消息
	_, err := conn.Write([]byte("请输入您的名字:"))
	if err != nil {
		return
	}
	UserName(conn,&person)
	massage <- MakeMsg(person,"进入房间～\n")
	//获取用户消息并群发
	go func() {
		buf := make([]byte,4096)
		for {
			//_, err := conn.Write([]byte("> "))
			//if err != nil {
			//	return
			//}
			n,err := conn.Read(buf)
			if n == 0{
				//删除在线信息
				delete(onlinePerson, person.Addr)
				fmt.Println(ipAddr,"用户退出")
				return
			}
			if err != nil {
				fmt.Println("conn.Read error",err)
				return
			}
			//用户命令
			if buf[0] == '#'{
				cmd := string(buf[1:n])
				cmd = cmdTrim(cmd)
				ExecCommend(cmd,conn,&person)
			}else{
				//用户消息
				massage <- MakeMsg(person,string(buf[:n]))
			}

		}
	}()
	for {
		;
	}
}

func Manager(){
	//初始化
	onlinePerson = make(map[string]Person)
	for {
		msg := <- massage
		for _,person := range onlinePerson{
			person.C <- msg
		}
	}
}

func main(){
	//创建监听套接字
	listener,err := net.Listen("tcp","0.0.0.0:9527")
	if err != nil{
		fmt.Println("Listen error",err)
		return
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("listener Close error",err)
			return
		}
	}(listener)

	go Manager()

	//循环监听连接请求
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error",err)
			return
		}
		//启动go程处理客户端请求
		go HandlerConnect(conn)
	}
}
