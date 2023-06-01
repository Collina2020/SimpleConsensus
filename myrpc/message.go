package myrpc

// MessageType example
// type MessageType uint8

// const (
// 	PREPREPARE MessageType = iota
// 	PREPARE
// 	COMMIT
// )

// func (mt MessageType) String() string {
// 	switch mt {
// 	case PREPREPARE:
// 		return "PREPREPARE"
// 	case PREPARE:
// 		return "PREPARE"
// 	case COMMIT:
// 		return "COMMIT"
// 	}
// 	return "UNKNOW MESSAGETYPE"
// }

type ConsensusMsg struct {
	// Type MessageType
	From   uint8
	Seq    uint64
	Data   []byte
	Target []byte //用difficulty来构造，要求区块计算的哈希值小于这个目标值
	Nonce  uint64 //每个区块给自己设定的随机值，参与哈希计算
	Hash   []byte //用nonce算出的hash值，通过和target的比较来证明工作量
}

type ConsensusMsgReply struct {
}
