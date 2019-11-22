package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strconv"
	"time"
)

const (
	voteNodeNum  = 100
	superNodeNum = 10
	mineSuperNodeNum = 3
	)

type block struct {
	//上一个块的hash
	prehash string
	//本块hash
	hash string
	//时间戳
	timestamp string
	//区块内容
	data string
	//区块高度
	height int
	//挖出本块的地址
	address string
}
//用于存储区块链
var blockchain []block
//代表挖矿节点
type node struct{
	//代币数量
	votes int
	//节点地址
	address string
}

type superNode struct {
	 node
}
//投票节点池
var voteNodesPool []node
//超级节点池
var superNodesPool []superNode
//获胜，可以挖矿的超级节点池
var mineSuperNodesPool []superNode
//随机节点池
var randNodesPool []superNode
//生成新的区块
func generateNewBlock(oldBlock block,data string,address string) block {
	newBlock:=block{}
	newBlock.prehash = oldBlock.hash
	newBlock.data = data
	newBlock.timestamp = time.Now().Format("2006-01-02 15:04:05")
	newBlock.height = oldBlock.height + 1
	newBlock.address = getMineNodeAddress()
	newBlock.getHash()
	return newBlock
}
//对自身进行散列
func ( b *block) getHash () {
	sumString:= b.prehash + b.timestamp + b.data + b.address + strconv.Itoa(b.height)
	hash:=sha256.Sum256([]byte(sumString))
	b.hash = hex.EncodeToString(hash[:])
}
//随机挖矿节点
func getMineNodeAddress() string{
	bInt:=big.NewInt(int64(len(randNodesPool)))
	rInt,err:=rand.Int(rand.Reader,bInt)
	if err != nil {
		log.Panic(err)
	}
	return randNodesPool[int(rInt.Int64())].address
}

func voting() {
	for _,v:=range voteNodesPool {
		rInt,err:=rand.Int(rand.Reader,big.NewInt(superNodeNum))
		if err != nil {
			log.Panic(err)
		}
		superNodesPool[int(rInt.Int64())].votes += v.votes
	}
}
//对挖矿节点进行排序
func sortMineNodes() {
	sort.Slice(superNodesPool, func(i, j int) bool {
		return superNodesPool[i].votes > superNodesPool[j].votes
	})
	mineSuperNodesPool = superNodesPool[:mineSuperNodeNum]
}


//初始化
func init() {
	//初始化投票节点
	for i:=0;i<=voteNodeNum;i++ {
		rInt,err:=rand.Int(rand.Reader,big.NewInt(10000))
		if err != nil {
			log.Panic(err)
		}
		voteNodesPool = append(voteNodesPool,node{int(rInt.Int64()),"投票节点"+strconv.Itoa(i)})
	}
	//初始化超级节点
	for i:=0;i<=superNodeNum;i++ {
		superNodesPool = append(superNodesPool,superNode{node{0,"超级节点"+strconv.Itoa(i)}})
	}
}

func main() {
	genesisBlock := block{"0000000000000000000000000000000000000000000000000000000000000000","",time.Now().Format("2006-01-02 15:04:05"),"我是创世区块",1,"0000000000"}
	genesisBlock.getHash()
	blockchain = append(blockchain,genesisBlock)
	fmt.Println(blockchain[0])
	i:=0
	j:=0
	for  {
		time.Sleep(time.Second)
		newBlock:=generateNewBlock(blockchain[i],"我是区块内容",mineSuperNodesPool[j].address)
		blockchain = append(blockchain,newBlock)
		fmt.Println(blockchain[i + 1])
		i++
		j++
		j = j % len(mineSuperNodesPool)
	}
}
