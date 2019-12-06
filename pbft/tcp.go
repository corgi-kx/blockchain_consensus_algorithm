package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
)

//客户端使用的tcp监听
func clientTcpListen() {
	listen, err := net.Listen("tcp", clientAddr)
	if err != nil {
		log.Panic(err)
	}
	defer listen.Close()


	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Panic(err)
		}
		b, err := ioutil.ReadAll(conn)
		if err != nil {
			log.Panic(err)
		}
		fmt.Println(string(b))
	}

}

//节点使用的tcp监听
func (p *pbft) tcpListen(addr string) {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("节点开启监听，地址：%s\n", addr)
	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Panic(err)
		}
		b, err := ioutil.ReadAll(conn)
		if err != nil {
			log.Panic(err)
		}
		go p.handleRequest(b)
	}

}

func tcpDial(context []byte, addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal("connect error", err)
	}
	defer conn.Close()
	_, err = conn.Write(context)
	if err != nil {
		log.Fatal(err)
	}
}
