package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/minio/sha256-simd"
	"log"
	"math/big"
	"time"
)

var blockchain []block

const  diffNum  = 15

type block struct {
	Lasthash  string
	Hash      string
	Data      string
	Timestamp string
	Height    int
	Nonce     int64
}

func mine(data string) block{
	if len(blockchain) < 1 {
		log.Panic("还未生成创世区块！")
	}
	lastBlock := blockchain[len(blockchain) - 1 ]

	newBlock := new(block)
	newBlock.Lasthash = lastBlock.Hash
	newBlock.Timestamp = time.Now().String()
	newBlock.Height =lastBlock.Height +1
	newBlock.Data = data
	var nonce int64 = 0
	//根据难度值获得一个大数
	newBigint :=big.NewInt(1)
	newBigint.Lsh(newBigint,256 - diffNum)
	for {
		newBlock.Nonce = nonce
		newBlock.getHash()

		hashInt := big.Int{}
		hashBytes,_:=hex.DecodeString(newBlock.Hash)
		hashInt.SetBytes(hashBytes) //把hash值转换为大数
		//如果hash小于规定的难度值大数，则代表挖矿成功
		if hashInt.Cmp(newBigint) == -1 {
			break
		}else  {
			nonce ++
		}
	}
	return *newBlock
}


func (b *block)serialize() []byte{
	bytes,err:=json.Marshal(b)
	if err != nil {
		log.Panic(err)
	}
	return bytes
}

func (b *block) getHash() {
	result:=sha256.Sum256(b.serialize())
	b.Hash =  hex.EncodeToString(result[:])
}

func main() {
	//制造一个创世区块
	genesisBlock:=new(block)
	genesisBlock.Timestamp = time.Now().String()
	genesisBlock.Data = "我是创世区块！"
	genesisBlock.Lasthash = "0"
	genesisBlock.Height = 1
	genesisBlock.Nonce = 0
	genesisBlock.getHash()
	blockchain = append(blockchain, *genesisBlock)
	for i:=0;i<=10;i++ {
		blockchain = append(blockchain,mine("天气不错"))
	}
	for _,v:=range blockchain {
		fmt.Println(v)
	}
}