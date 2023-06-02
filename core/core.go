package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"time"

	"nis3607/mylogger"
	"nis3607/myrpc"
)

const (
	diff = 2
) //PoW的difficulty, 代表难度系数，如果赋值为1，则需要判断生成区块时所产生的Hash前缀至少包含1个0

type Consensus struct {
	id   uint8 //节点的编号
	n    uint8 //节点总数
	port uint64
	seq  uint64
	//BlockChain
	blockChain *BlockChain
	//logger
	logger *mylogger.MyLogger
	//rpc network
	peers []*myrpc.ClientEnd
	//message channel exapmle
	msgChan chan *myrpc.ConsensusMsg
}

func InitConsensus(config *Configuration) *Consensus {
	rand.Seed(time.Now().UnixNano())
	c := &Consensus{
		id:         config.Id,
		n:          config.N,
		port:       config.Port,
		seq:        0,
		blockChain: InitBlockChain(config.Id, config.BlockSize),
		logger:     mylogger.InitLogger("node", config.Id),
		peers:      make([]*myrpc.ClientEnd, 0),

		msgChan: make(chan *myrpc.ConsensusMsg, 1024),
	}

	for _, peer := range config.Committee {
		clientEnd := &myrpc.ClientEnd{Port: uint64(peer)}
		c.peers = append(c.peers, clientEnd)
	}

	//在这里添加pow共识机制

	go c.serve()
	return c
}

func (c *Consensus) serve() {
	rpc.Register(c)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":"+strconv.Itoa(int(c.port)))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

func (c *Consensus) OnReceiveMessage(args *myrpc.ConsensusMsg, reply *myrpc.ConsensusMsgReply) error {

	c.logger.DPrintf("Invoke RpcExample: receive message from %v at %v", args.From, time.Now().Nanosecond())
	c.msgChan <- args
	return nil
}

func (c *Consensus) broadcastMessage(msg *myrpc.ConsensusMsg) {
	reply := &myrpc.ConsensusMsgReply{}
	for id := range c.peers {
		c.peers[id].Call("Consensus.OnReceiveMessage", msg, reply)
	}
}

func (c *Consensus) handleMsgExample(msg *myrpc.ConsensusMsg) {
	block := &Block{
		Seq:    msg.Seq,
		Data:   msg.Data,
		Target: msg.Target,
		Nonce:  msg.Nonce,
		Hash:   msg.Hash,
	}

	valid := checkWork(msg.Hash, msg.Target)
	if valid {
		c.blockChain.commitBlock(block) //如果有效的话，提交这个block
		//seq更新，以备节点重新生成一个区块并开始计算
		c.seq++
	}
}

func (c *Consensus) propose(block *Block) {

	msg := &myrpc.ConsensusMsg{
		From:   c.id,
		Seq:    block.Seq,
		Data:   block.Data,
		Target: block.Target,
		Nonce:  block.Nonce,
		Hash:   block.Hash,
	}
	c.broadcastMessage(msg)

}

func (c *Consensus) Run() {
	// wait for other node to start
	time.Sleep(time.Duration(1) * time.Second)
	//init rpc client
	for id := range c.peers {
		c.peers[id].Connect()
	}
	//handle received message
	for {
		select {
		case msg := <-c.msgChan:
			c.handleMsgExample(msg)
		default:
			seq := c.seq
			block := c.blockChain.getBlock(seq)
			c.GetTarget(block)
			c.GetNonce(block)
			valid := checkWork(block.Hash, block.Target)
			if valid {
				c.blockChain.commitBlock(block) //如果有效的话，提交这个block
				//seq更新，以备节点重新生成一个区块并开始计算
				c.seq++
			}
			c.propose(block)
		}
	}
}

// 以下为添加的代码，包括给consensus的区块链中的每个区块确定target， 计算哈希值，寻找Nonce，etc.
func (c *Consensus) GetTarget(block *Block) {
	if block == nil {
		fmt.Printf("nil error")
		return
	}

	l := c.blockChain.BlockSize
	for i := uint64(0); i < l-diff; i++ {
		block.Target[i] = byte(0)
	}
	for i := l - diff; i < l; i++ {
		block.Target[i] = byte(rand.Intn(256))
	}

	return
}

// 对于给定的nonce和区块的data，通过nonce计算区块的哈希值
func Nonce2Hash(nonce uint64, data []byte) []byte {
	nonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceBytes, nonce)

	combinedData := append(nonceBytes, data...)

	hash, _ := ComputeHash(combinedData)

	return hash

}

func (c *Consensus) GetNonce(block *Block) {
	var nonce uint64
	if block == nil {
		fmt.Printf("nil error")
		return
	}

	nonce = 0
	for {
		data := block.Data
		Hash := Nonce2Hash(nonce, data)
		Target := block.Target
		if bytes.Compare(Hash, Target) == -1 {
			block.Hash = Hash
			break
		} else {
			nonce++
		}

	}
	block.Nonce = nonce

	return

}
