package core

import (
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"time"
	"encoding/binary"
	"bytes"

	"nis3607/mylogger"
	"nis3607/myrpc"
)
const(	diff = 4)	//PoW的difficulty, 代表难度系数，如果赋值为1，则需要判断生成区块时所产生的Hash前缀至少包含1个0

type Consensus struct {
	id   uint8		//节点的编号
	n    uint8		//节点总数
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

	//以下为自行添加的共识属性，包括每个区块的target, nonce
	targetsMap  map[*Block][]byte
	nonceMap map[*Block]uint64
}

func InitConsensus(config *Configuration) *Consensus {
	rand.Seed(time.Now().UnixNano())
	c := &Consensus{
		id:         config.Id,
		n:          config.N,
		port:       config.Port,
		seq:        0,
		blockChain: InitBlockChain(config.Id, config.BlockSize),
		targetsMap: make(map[*Block][]byte),
		nonceMap: make(map[*Block]uint64),
		logger:     mylogger.InitLogger("node", config.Id),
		peers:      make([]*myrpc.ClientEnd, 0),

		msgChan: make(chan *myrpc.ConsensusMsg, 1024),
	} 
	for _, peer := range config.Committee {
		clientEnd := &myrpc.ClientEnd{Port: uint64(peer)}
		c.peers = append(c.peers, clientEnd)
	}

	//在这里添加pow共识机制
	c.GetTarget()
	c.GetNonce()


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
		Seq:  msg.Seq,
		Data: msg.Data,
	}
	c.blockChain.commitBlock(block)
}

func (c *Consensus) proposeLoop() {
	for {
		if c.id == 0 {
			block := c.blockChain.getBlock(c.seq)
			msg := &myrpc.ConsensusMsg{
				From: c.id,
				Seq:  block.Seq,
				Data: block.Data,
			}
			c.broadcastMessage(msg)
			c.seq++
		}
	}
}

func (c *Consensus) Run() {
	// wait for other node to start
	time.Sleep(time.Duration(1) * time.Second)
	//init rpc client
	for id := range c.peers {
		c.peers[id].Connect()
	}

	go c.proposeLoop()
	//handle received message
	for {

		msg := <-c.msgChan
		c.handleMsgExample(msg)
	}
}



//以下为添加的代码，包括给consensus的区块链中的每个区块确定target， 计算哈希值，寻找Nonce，etc.
func (c *Consensus) GetTarget() {
	for _, block := range c.blockChain.Blocks {
		target := make([]byte, 1024)
		for i := uint64(0); i < 1024 - diff; i++{
			target[i] = byte(0)
		}
		for i := 1024 - diff; i < 1024; i++ {
			target[i] = byte(rand.Intn(256))
		}

		c.targetsMap[block] = target
		block.target = target
		
	}	
	
	return 
}

//对于给定的nonce和区块的data，通过nonce计算区块的哈希值
func Nonce2Hash(nonce uint64,data []byte) []byte{
	nonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceBytes, nonce)

	combinedData := append(nonceBytes, data...)

	hash,_ := ComputeHash(combinedData)

	return hash

}

func (c *Consensus) GetNonce(){
	var nonce uint64
	for _, block := range c.blockChain.Blocks {
		nonce = 0
		for{
			data := block.Data
			hash := Nonce2Hash(nonce,data)
			target := c.targetsMap[block]
			if bytes.Compare(hash, target)==-1{
				block.hash = hash
				break
			} else{
				nonce++
			}

		}
		c.nonceMap[block] = nonce
		block.nonce = nonce
		
	}	
	
	return 
	
}

