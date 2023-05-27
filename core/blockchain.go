package core

import (
	"math/rand"
	"nis3607/mylogger"
	"time"
	"bytes"
)

type Block struct {		//seq是区块的编号，data的值不重要
	Seq  uint64
	Data []byte
	target	[]byte		//用difficulty来构造，要求区块计算的哈希值小于这个目标值
	nonce	uint64		//每个区块给自己设定的随机值，参与哈希计算
	hash  []byte		//用nonce算出的hash值，通过和target的比较来证明工作量
}

type BlockChain struct {
	Id        uint8
	BlockSize uint64
	Blocks    []*Block
	BlocksMap map[string]*Block
	KeysMap   map[*Block]string
	logger    *mylogger.MyLogger
}

func InitBlockChain(id uint8, blocksize uint64) *BlockChain {
	blocks := make([]*Block, 1024)
	blocksMap := make(map[string]*Block)
	keysMap := make(map[*Block]string)
	//Generate gensis block
	blockChain := &BlockChain{
		Id:        id,
		BlockSize: blocksize,
		Blocks:    blocks,
		BlocksMap: blocksMap,
		KeysMap:   keysMap,
		logger:    mylogger.InitLogger("blockchain", id),
	}
	return blockChain
}

func Block2Hash(block *Block) []byte {
	hash, _ := ComputeHash(block.Data)
	return hash
}

func Hash2Key(hash []byte) string {		//将哈希值转成密钥
	var key []byte
	for i := 0; i < 20; i++ {
		key = append(key, uint8(97)+uint8(hash[i]%(26)))
	}
	return string(key)
}
func Block2Key(block *Block) string {
	return Hash2Key(Block2Hash(block))
}

func (bc *BlockChain) AddBlockToChain(block *Block) {
	//在这里添加检查validity的代码，prove了工作量的区块才能加进来
	h := block.hash
	t := block.target
	if bytes.Compare(h, t) !=-1{
		return
	}


	bc.Blocks = append(bc.Blocks, block)
	bc.KeysMap[block] = Block2Key(block)
	bc.BlocksMap[Block2Key(block)] = block
}

// Generate a Block: max rate is 20 blocks/s
func (bc *BlockChain) getBlock(seq uint64) *Block {//generate a block
	//slow down
	time.Sleep(time.Duration(50) * time.Millisecond)
	data := make([]byte, bc.BlockSize)
	for i := uint64(0); i < bc.BlockSize; i++ {
		data[i] = byte(rand.Intn(256))
	}
	block := &Block{
		Seq:  seq,
		Data: data,
		//初始化
		target: data,
		nonce: uint64(0),
		hash: data,
	}
	bc.logger.DPrintf("generate Block[%v] in seq %v at %v", Block2Key(block), block.Seq, time.Now().UnixNano())
	return block
}

func (bc *BlockChain) commitBlock(block *Block) {	//上传到logger
	bc.AddBlockToChain(block)
	bc.logger.DPrintf("commit Block[%v] in seq %v at %v", Block2Key(block), block.Seq, time.Now().UnixNano())
}
