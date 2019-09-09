package og

import (
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"github.com/sirupsen/logrus"
	"time"
)

type KafkaProducer struct {
	Url           string
	Topic         string
	NewTxReceiver chan types.Txi
	Producer      sarama.AsyncProducer

	// in seconds
	KafkaTimeOut int

	close chan struct{}
}

func NewKafkaProducer(url, topic string, timeOutSec int) (*KafkaProducer, error) {
	kp := KafkaProducer{}

	kp.Url = url
	kp.Topic = topic
	kp.NewTxReceiver = make(chan types.Txi)

	producer, err := sarama.NewAsyncProducer([]string{kp.Url}, nil)
	if err != nil {
		return nil, fmt.Errorf("create sarama kafka producer error: %v", err)
	}
	kp.Producer = producer

	kp.KafkaTimeOut = timeOutSec
	kp.close = make(chan struct{})

	return &kp, nil
}

func (kp *KafkaProducer) Start() {
	logrus.Trace("KafkaProducer started")
	go kp.loop()
}

func (kp *KafkaProducer) Stop() {
	logrus.Trace("KafkaProducer stopped")
	close(kp.close)
}

func (kp *KafkaProducer) Name() string {
	return "KafkaProducer"
}

func (kp *KafkaProducer) loop() {
	for {
		select {
		case txi := <-kp.NewTxReceiver:
			kp.handleNewTx(txi)
		case <-kp.close:
			kp.Producer.Close()
			return
		}
	}
}

type KafkaMsg struct {
	Type int         `json:"type"`
	Data interface{} `json:"data"`
}

type KafkaMsgSeq struct {
	Type     int      `json:"type"`
	Hash     string   `json:"hash"`
	Parents  []string `json:"parents"`
	From     string   `json:"from"`
	Nonce    uint64   `json:"nonce"`
	Treasure string   `json:"treasure"`
	Height   uint64   `json:"height"`
}

type KafkaMsgTx struct {
	Type      int      `json:"type"`
	Hash      string   `json:"hash"`
	Parents   []string `json:"parents"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Nonce     uint64   `json:"nonce"`
	Guarantee string   `json:"guarantee"`
	Value     string   `json:"value"`
}

func (kp *KafkaProducer) handleNewTx(txi types.Txi) {
	logrus.Tracef("kafka producer handle new tx: %s", txi.GetTxHash().String())

	txMsg := KafkaMsg{}
	txMsg.Type = int(txi.GetType())
	txMsg.Data = txHelper(txi)

	value, _ := json.Marshal(txMsg)

	msg := &sarama.ProducerMessage{}
	msg.Topic = kp.Topic
	msg.Value = sarama.ByteEncoder(value)

	timer := time.NewTimer(time.Second * time.Duration(kp.KafkaTimeOut))
	select {
	case kp.Producer.Input() <- msg:
		logrus.Tracef("produce kafka msg success: %s", txi.GetTxHash().String())
		return
	case <-timer.C:
		logrus.Tracef("produce kafka msg timeout: %s", txi.GetTxHash().String())
		return
	}

}

func txHelper(txi types.Txi) interface{} {
	switch tx := txi.(type) {
	case *tx_types.Tx:
		txMsg := KafkaMsgTx{}
		txMsg.Type = int(tx.Type)
		txMsg.Hash = tx.Hash.Hex()

		for _, p := range tx.ParentsHash {
			txMsg.Parents = append(txMsg.Parents, p.Hex())
		}

		txMsg.From = tx.From.Hex()
		txMsg.To = tx.To.Hex()
		txMsg.Nonce = tx.AccountNonce
		txMsg.Guarantee = tx.Guarantee.String()
		txMsg.Value = tx.Value.String()

		return txMsg

	case *tx_types.Sequencer:
		seqMsg := KafkaMsgSeq{}
		seqMsg.Type = int(tx.Type)
		seqMsg.Hash = tx.Hash.Hex()

		for _, p := range tx.ParentsHash {
			seqMsg.Parents = append(seqMsg.Parents, p.Hex())
		}

		seqMsg.From = tx.Sender().Hex()
		seqMsg.Nonce = tx.AccountNonce
		seqMsg.Treasure = tx.Treasure.String()
		seqMsg.Height = tx.Height

		return seqMsg

	default:
		return nil
	}
}
