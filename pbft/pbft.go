package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
)

//本地消息池模拟持久化层，只有确认提交成功后才会存入此池
var localMessagePool = []Message{}

type node struct {
	nodeID string
	addr   string
}
type pbft struct {
	//节点信息
	node node
	//每笔请求自增序号
	sequenceID int
	//锁
	lock sync.Mutex
	//临时消息池，消息摘要对应消息本体
	messagePool map[string]Request
	//确认收到的prepare数量(至少需要收到并确认2f个)，根据摘要来对应
	prePareConfirmCount map[string]map[string]bool
	//确认收到的prepare数量（至少需要收到并确认2f+1个），根据摘要来对应
	commitConfirmCount map[string]map[string]bool
}

func NewPBFT(nodeID, addr string) *pbft {
	p := new(pbft)
	p.node.nodeID = nodeID
	p.node.addr = addr
	p.sequenceID = 0
	p.messagePool = make(map[string]Request)
	p.prePareConfirmCount = make(map[string]map[string]bool)
	p.commitConfirmCount = make(map[string]map[string]bool)
	return p
}

func (p *pbft) handleRequest(data []byte) {
	cmd, content := splitMessage(data)
	switch command(cmd) {
	case cRequest:
		go p.handleClientRequest(content)
	case cPrePrepare:
		go p.handlePrePrepare(content)
	case cPrepare:
		go p.handlePrepare(content)
	case cCommit:
		go p.handleCommit(content)
	}
}

func (p *pbft) handleClientRequest(content []byte) {
	fmt.Println("主节点已接收到客户端发来的request ...")
	//使用json解析出Request结构体
	r := new(Request)
	err := json.Unmarshal(content, r)
	if err != nil {
		log.Panic(err)
	}
	//添加信息序号
	p.sequenceIDAdd()
	//获取消息摘要
	digest := getDigest(*r)
	fmt.Println("已将request存入临时消息池")
	//存入临时消息池
	p.messagePool[digest] = *r
	//拼接成PrePrepare，准备发往follower节点
	pp := PrePrepare{*r, digest, p.sequenceID}
	b, err := json.Marshal(pp)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println("正在向其他节点进行进行PrePrepare广播 ...")
	//进行PrePrepare广播
	p.broadcast(cPrePrepare, b)
	fmt.Println("PrePrepare广播完成")
}

func (p *pbft) handlePrePrepare(content []byte) {
	fmt.Println("本节点已接收到主节点发来的PrePrepare ...")
	//	//使用json解析出PrePrepare结构体
	pp := new(PrePrepare)
	err := json.Unmarshal(content, pp)
	if err != nil {
		log.Panic(err)
	}

	if digest := getDigest(pp.RequestMessage); digest != pp.Digest {
		fmt.Println("信息摘要对不上，拒绝进行prepare广播")
	} else if p.sequenceID+1 != pp.SequenceID {
		fmt.Println("消息序号对不上，拒绝进行prepare广播")
	} else {
		p.sequenceID = pp.SequenceID
		//将信息存入临时消息池
		fmt.Println("已将消息存入临时节点池")
		p.messagePool[pp.Digest] = pp.RequestMessage
		pre := Prepare{pp.Digest, pp.SequenceID, p.node.nodeID}
		bPre, err := json.Marshal(pre)
		if err != nil {
			log.Panic(err)
		}
		//进行准备阶段的广播
		fmt.Println("正在进行Prepare广播 ...")
		p.broadcast(cPrepare, bPre)
		fmt.Println("Prepare广播完成")
	}
}

func (p *pbft) handlePrepare(content []byte) {

	//使用json解析出Prepare结构体
	pre := new(Prepare)
	err := json.Unmarshal(content, pre)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("本节点已接收到%s节点发来的Prepare ... \n", pre.NodeID)
	if _, ok := p.messagePool[pre.Digest]; !ok {
		fmt.Println("当前临时消息池无此摘要，拒绝执行commit广播")
	} else if p.sequenceID != pre.SequenceID {
		fmt.Println("消息序号对不上，拒绝执行commit广播")
	} else {
		p.setPrePareConfirmMap(pre.Digest,pre.NodeID,true)
		count := 0
		for _, _ = range p.prePareConfirmCount[pre.Digest] {
			count++
		}
		//如果节点至少收到了2f个prepare的消息，则进行commit广播
		if count >= (nodeCount / 3 * 2) {
			fmt.Println("本节点已收到至少2f个节点发来的Prepare信 ...")
			c := Commit{pre.Digest, pre.SequenceID, p.node.nodeID}
			bc, err := json.Marshal(c)
			if err != nil {
				log.Panic(err)
			}
			//进行提交信息的广播
			fmt.Println("正在进行commit广播")
			p.broadcast(cCommit, bc)
			fmt.Println("commit广播完成")
		}
	}
}

func (p *pbft) handleCommit(content []byte) {

	//使用json解析出Commit结构体
	c := new(Commit)
	err := json.Unmarshal(content, c)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("本节点已接收到%s节点发来的Commit ... \n", c.NodeID)
	if _, ok := p.prePareConfirmCount[c.Digest]; !ok {
		fmt.Println("当前prepare池无此摘要，拒绝将信息持久化到本地消息池")
	} else if p.sequenceID != c.SequenceID {
		fmt.Println("消息序号对不上，拒绝将信息持久化到本地消息池")
	} else {
		p.setCommitConfirmMap(c.Digest,c.NodeID,true)
		count := 0
		for _, _ = range p.prePareConfirmCount[c.Digest] {
			count++
		}
		//如果节点至少收到了2f + 1个commit消息，则提交信息至本地消息池，并reply成功标志至客户端！
		if count >= (nodeCount/3*2)+1 {
			fmt.Println("本节点已收到至少2f + 1 个节点发来的Commit信息 ...")
			//将消息信息，提交到本地消息池中！
			localMessagePool = append(localMessagePool, p.messagePool[c.Digest].Message)
			info := p.node.nodeID + "节点已将msgid:" + strconv.Itoa(p.messagePool[c.Digest].ID) + "存入本地消息池中！"
			fmt.Println(info)
			fmt.Println("消息为：", p.messagePool[c.Digest].Content)
			fmt.Println("正在reply客户端 ...")
			tcpDial([]byte(info), p.messagePool[c.Digest].ClientAddr)
			fmt.Println("reply完毕")
		}
	}
}

func (p *pbft) sequenceIDAdd() {
	p.lock.Lock()
	p.sequenceID++
	p.lock.Unlock()
}

//向除自己外的其他节点进行广播
func (p *pbft) broadcast(cmd command, content []byte) {
	for i, v := range nodeTable {
		if i == p.node.nodeID {
			continue
		}
		message := jointMessage(cmd, content)
		tcpDial(message, v)
	}
}

//为多重映射开辟赋值
func (p *pbft) setPrePareConfirmMap(val, val2 string, b bool) {
	if _, ok := p.prePareConfirmCount[val]; !ok {
		p.prePareConfirmCount[val] = make(map[string]bool)
	}
	p.prePareConfirmCount[val][val2] = b
}

//为多重映射开辟赋值
func (p *pbft) setCommitConfirmMap(val, val2 string, b bool) {
	if _, ok := p.commitConfirmCount[val]; !ok {
		p.commitConfirmCount[val] = make(map[string]bool)
	}
	p.commitConfirmCount[val][val2] = b
}