package topic

import (
	"fmt"
	"sync"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/mainflux/fluxmq"
)

var _ Manager = (*memManager)(nil)

type memManager struct {
	// Sub/unsub mutex
	smu sync.RWMutex
	// Subscription tree
	sroot *snode

	// Retained message mutex
	rmu sync.RWMutex
	// Retained messages topic tree
	rroot *rnode
}

// NewMemManager returns an new instance of the memManager, which is implements the
// Manager interface. memManager is a hidden struct that stores the topic
// subscriptions and retained messages in memory. The content is not persistend so
// when the server goes, everything will be gone. Use with care.
func NewMemManager() *memManager {
	return &memManager{
		sroot: newSNode(),
		rroot: newRNode(),
	}
}

func (m *memManager) Subscribe(sub Subscription) (byte, error) {
	if !fluxmq.ValidateQoS(sub.QoS) {
		return qosFailure, fmt.Errorf("Invalid QoS %d", sub.QoS)
	}

	//if sub == nil {
	//	return qosFailure, fmt.Errorf("Subscriber cannot be nil")
	//}

	m.smu.Lock()
	defer m.smu.Unlock()

	if sub.QoS > fluxmq.QoS1 {
		sub.QoS = fluxmq.QoS1
	}

	//if err := m.sroot.sinsert(sub); err != nil {
	//	return qosFailure, err
	//}

	return sub.QoS, nil
}

func (m *memManager) Unsubscribe(sub Subscription) error {
	m.smu.Lock()
	defer m.smu.Unlock()

	//return m.sroot.sremove(sub)
	return nil
}

// Returned values will be invalidated by the next Subscribers call
func (this *memManager) Subscribers(topic []byte, qos byte, subs *[]interface{}, qoss *[]byte) error {
	if !fluxmq.ValidateQoS(qos) {
		return fmt.Errorf("Invalid QoS %d", qos)
	}

	this.smu.RLock()
	defer this.smu.RUnlock()

	*subs = (*subs)[0:0]
	*qoss = (*qoss)[0:0]

	return this.sroot.smatch(topic, qos, subs, qoss)
}

func (m *memManager) Retain(msg *packets.PublishPacket) error {
	m.rmu.Lock()
	defer m.rmu.Unlock()

	// So apparently, at least according to the MQTT Conformance/Interoperability
	// Testing, that a payload of 0 means delete the retain message.
	// https://eclipse.org/paho/clients/testing/
	if len(msg.Payload) == 0 {
		return m.rroot.rremove([]byte(msg.TopicName))
	}

	return m.rroot.rinsertOrUpdate([]byte(msg.TopicName), msg)
}

func (m *memManager) Retained(topic []byte, msgs *[]*packets.PublishPacket) error {
	m.rmu.RLock()
	defer m.rmu.RUnlock()

	return m.rroot.rmatch(topic, msgs)
}

func (m *memManager) Close() error {
	m.sroot = nil
	m.rroot = nil
	return nil
}
