package topic

import (
	"fmt"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

// Retained message node
type rnode struct {
	// If this is the end of the topic string, then add retained messages here
	msg *packets.PublishPacket
	// Otherwise add the next topic level here
	rnodes map[string]*rnode
}

func newRNode() *rnode {
	return &rnode{
		rnodes: make(map[string]*rnode),
	}
}

func (this *rnode) rinsertOrUpdate(topic []byte, msg *packets.PublishPacket) error {
	// If there's no more topic levels, that means we are at the matching rnode.
	if len(topic) == 0 {
		// Reuse the message if possible
		this.msg = msg

		return nil
	}

	// Not the last level, so let's find or create the next level snode, and
	// recursively call it's insert().

	// ntl = next topic level
	ntl, rem, err := nextTopicLevel(topic)
	if err != nil {
		return err
	}

	level := string(ntl)

	// Add snode if it doesn't already exist
	n, ok := this.rnodes[level]
	if !ok {
		n = newRNode()
		this.rnodes[level] = n
	}

	return n.rinsertOrUpdate(rem, msg)
}

// Remove the retained message for the supplied topic
func (this *rnode) rremove(topic []byte) error {
	// If the topic is empty, it means we are at the final matching rnode. If so,
	// let's remove the buffer and message.
	if len(topic) == 0 {
		this.msg = nil
		return nil
	}

	// Not the last level, so let's find the next level rnode, and recursively
	// call it's remove().

	// ntl = next topic level
	ntl, rem, err := nextTopicLevel(topic)
	if err != nil {
		return err
	}

	level := string(ntl)

	// Find the rnode that matches the topic level
	n, ok := this.rnodes[level]
	if !ok {
		return fmt.Errorf("No topic found")
	}

	// Remove the subscriber from the next level rnode
	if err := n.rremove(rem); err != nil {
		return err
	}

	// If there are no more rnodes to the next level we just visited let's remove it
	if len(n.rnodes) == 0 {
		delete(this.rnodes, level)
	}

	return nil
}

// rmatch() finds the retained messages for the topic and qos provided. It's somewhat
// of a reverse match compare to match() since the supplied topic can contain
// wildcards, whereas the retained message topic is a full (no wildcard) topic.
func (this *rnode) rmatch(topic []byte, msgs *[]*packets.PublishPacket) error {
	// If the topic is empty, it means we are at the final matching rnode. If so,
	// add the retained msg to the list.
	if len(topic) == 0 {
		if this.msg != nil {
			*msgs = append(*msgs, this.msg)
		}
		return nil
	}

	// ntl = next topic level
	ntl, rem, err := nextTopicLevel(topic)
	if err != nil {
		return err
	}

	level := string(ntl)

	if level == "#" {
		// If '#', add all retained messages starting this node
		this.allRetained(msgs)
	} else if level == "+" {
		// If '+', check all nodes at this level. Next levels must be matched.
		for _, n := range this.rnodes {
			if err := n.rmatch(rem, msgs); err != nil {
				return err
			}
		}
	} else {
		// Otherwise, find the matching node, go to the next level
		if n, ok := this.rnodes[level]; ok {
			if err := n.rmatch(rem, msgs); err != nil {
				return err
			}
		}
	}

	return nil
}

func (this *rnode) allRetained(msgs *[]*packets.PublishPacket) {
	if this.msg != nil {
		*msgs = append(*msgs, this.msg)
	}

	for _, n := range this.rnodes {
		n.allRetained(msgs)
	}
}
