package core

import (
	"bytes"
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
	diff = 1
)

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

func (c *Consensus) handleMsgExample(msg *myrpc.ConsensusMsg, bestblock *Block, besttime int64) *Block {
	block := &Block{
		Seq:    msg.Seq,
		Data:   msg.Data,
		Target: msg.Target,
		Nonce:  msg.Nonce,
		Hash:   msg.Hash,
	}

	if bestblock == nil {
		return block
	}

	valid := checkWork(msg.Hash, msg.Target)
	if valid && block.Seq == c.seq {
		if msg.Time < besttime {
			return block
		}
	}

	return bestblock
}

func (c *Consensus) propose(block *Block, time int64) {

	msg := &myrpc.ConsensusMsg{
		From:   c.id,
		Seq:    block.Seq,
		Data:   block.Data,
		Target: block.Target,
		Nonce:  block.Nonce,
		Hash:   block.Hash,
		Time: time,
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
	Time := int64(0)
	besttime := int64(0)
	bestblock := &Block{}
	msgcnt := 0
	for {
		seq := c.seq
		block := c.blockChain.getBlock(seq)
		c.GetTarget(block)
		c.GetNonce(block)
		valid := checkWork(block.Hash, block.Target)
		if valid {
			Time = time.Now().UnixNano()
			c.propose(block, Time)
			bestblock = block
			besttime = Time
		}
		for {
			if msgcnt < 6{
			select{
		case msg := <-c.msgChan:
			if msg.Seq == c.seq {
				msgcnt++
				optionalbestblock := c.handleMsgExample(msg, bestblock, besttime) //比较两者哪个更好，更好的做bestblock
				if optionalbestblock != bestblock {
					bestblock = optionalbestblock
					besttime = msg.Time
				}
			}
		default:
			time.Sleep(time.Duration(20) * time.Millisecond)
		}
	}else{
		break
	}
}
		c.blockChain.commitBlock(bestblock)
		c.seq++
		bestblock = nil
		msgcnt = 0


	}

}

// 以下为添加的代码，包括给consensus的区块链中的每个区块确定target， 计算哈希值，寻找Nonce，etc.
func (c *Consensus) GetTarget(block *Block) {
	if block == nil {
		fmt.Printf("nil error")
		return
	}

	for i:= 0; i<diff; i++{
		block.Target[i] = byte(0)
	}

	for i := diff; i < 10; i++ {
		block.Target[i] = byte(rand.Intn(256))
	}

	return
}

// 对于给定的nonce和区块的data，通过nonce计算区块的哈希值
func Nonce2Hash(nonce uint8, data []byte) []byte {
	hash := make([]byte, 10)

	for i := uint8(0); i < 10; i++ {
		hash[i] = nonce + data[i]
	}

	return hash

}

func (c *Consensus) GetNonce(block *Block) {
	var nonce uint8
	if block == nil {
		fmt.Printf("nil error")
		return
	}

	nonce = 0

	Target := block.Target
	data := block.Data

	Hash := Nonce2Hash(nonce, data)
	for {
		if bytes.Compare(Hash, Target) == 1 {
			block.Hash = Hash
			break
		} else {
			nonce = nonce + 1
		}

	}

	block.Nonce = nonce

	return

}
