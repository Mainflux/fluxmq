package client

import (
	"errors"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/mainflux/fluxmq/auth"
	"github.com/mainflux/fluxmq/topic"
	"go.uber.org/zap"
)

const (
	qos0 byte = iota
	qos1
	qos2
	qosFail = 0x80
)

var groupRegexp = regexp.MustCompile(`^\$share/([0-9a-zA-Z_-]+)/(.*)$`)

// ClientInfo stores basic MQTT client data.
// Separated in the struct so that it can be
// easily used as a param of Auth handler
type ClientInfo struct {
	ID       string
	Username string
	Password []byte
}

// Client stores complete MQTT client data
type Client struct {
	info      ClientInfo
	conn      net.Conn
	keepalive uint16
	lwt       *packets.PublishPacket
	handler   auth.Handler
	subs      map[string]topic.Subsciption
	logger    *zap.Logger
}

func New(info ClientInfo, keepalive uint16, lwt *packets.PublishPacket, handler auth.Handler, logger *zap.Logger) Client {
	return Client{
		info:      info,
		keepalive: keepalive,
		lwt:       lwt,
		handler:   handler,
		subs:      make(map[string]topic.Subsciption),
		logger:    logger,
	}
}

func NewInfo(clientID, username string, password []byte) ClientInfo {
	return ClientInfo{
		ID:       clientID,
		Username: username,
		Password: password,
	}
}

func (c Client) ReadLoop() {
	keepAlive := time.Second * time.Duration(c.info.keepalive)
	timeOut := keepAlive + (keepAlive / 2)
	dpkt := packets.NewControlPacket(packets.Disconnect).(*packets.DisconnectPacket)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if keepAlive > 0 {
				if err := c.conn.SetReadDeadline(time.Now().Add(timeOut)); err != nil {
					c.logger.Error("Set read timeout error: ", zap.Error(err), zap.String("client_id", c.info.ID))
					c.ProcessMessage(dpkt)
					return
				}
			}

			packet, err := packets.ReadPacket(c.conn)
			if err != nil {
				c.logger.Error("Read packet error: ", zap.Error(err), zap.String("client_id", c.info.ID))
				c.ProcessMessage(dpkt)
				return
			}

			c.ProcessMessage(packet)
		}
	}

}

func (c Client) ProcessMessage(p packets.ControlPacket) {
	switch p.(type) {
	case *packets.ConnackPacket:
	case *packets.ConnectPacket:
	case *packets.PublishPacket:
		packet := p.(*packets.PublishPacket)
		c.ProcessPublish(packet)
	case *packets.PubackPacket:
	case *packets.PubrecPacket:
	case *packets.PubrelPacket:
	case *packets.PubcompPacket:
	case *packets.SubscribePacket:
		packet := p.(*packets.SubscribePacket)
		c.ProcessSubscribe(packet)
	case *packets.SubackPacket:
	case *packets.UnsubscribePacket:
		packet := p.(*packets.UnsubscribePacket)
		c.ProcessUnSubscribe(packet)
	case *packets.UnsubackPacket:
	case *packets.PingreqPacket:
		c.ProcessPing()
	case *packets.PingrespPacket:
	case *packets.DisconnectPacket:
		c.Close()
	default:
		c.logger.Info("Unknown packet", zap.String("client_id", c.info.ID))
	}
}

// Publish
func (c Client) ProcessPublish(packet *packets.PublishPacket) {
	switch packet.Qos {
	case qos0:
		c.ProcessPublishMessage(packet)
	case qos1:
		puback := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
		puback.MessageID = packet.MessageID
		if err := c.WritePacket(puback); err != nil {
			c.logger.Error("Send puback error, ", zap.Error(err), zap.String("client_id", c.info.ID))
			return
		}
		c.ProcessPublishMessage(packet)
	case qos2:
		// TODO
		return
	default:
		c.logger.Error("Publish with unknown QoS", zap.String("client_id", c.info.ID))
		return
	}
}

func (c *client) ProcessPublishMessage(packet *packets.PublishPacket) {
	if packet.Retain {
		if err := c.topicsMgr.Retain(packet); err != nil {
			c.logger.Error("Error retaining message: ", zap.Error(err), zap.String("client_id", c.info.ID))
		}
	}

	err := c.topicsMgr.Subscribers([]byte(packet.TopicName), packet.Qos, &c.subs, &c.qoss)
	if err != nil {
		c.logger.Error("Error retrieving subscribers list: ", zap.String("client_id", c.info.ID))
		return
	}

	if len(c.subs) == 0 {
		return
	}

	var qsub []int
	for i, sub := range c.subs {
		s, ok := sub.(*subscription)
		if ok {
			if s.client.typ == ROUTER {
				if typ != CLIENT {
					continue
				}
			}
			if s.share {
				qsub = append(qsub, i)
			} else {
				publish(s, packet)
			}

		}

	}

	if len(qsub) > 0 {
		idx := r.Intn(len(qsub))
		sub := c.subs[qsub[idx]].(*subscription)
		publish(sub, packet)
	}

}

// Subscribe
func (c *client) processSubscribe(packet *packets.SubscribePacket) {
	if c.status == Disconnected {
		return
	}

	b := c.broker
	if b == nil {
		return
	}
	topics := packet.Topics
	qoss := packet.Qoss

	suback := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	suback.MessageID = packet.MessageID
	var retcodes []byte

	for i, topic := range topics {

		// Handle Auth Subscribe
		if err := c.handler.AuthSubscribe(&c.info); err != nil {
			s.logger.Error("Auth Subscribe error, ", zap.Error(err), zap.String("client_id", ci.ID))
			retcodes = append(retcodes, qosFail)
			continue
		}

		// Shared subscriptions
		group := ""
		shared := false
		if strings.HasPrefix(topic, "$share/") {
			substr := groupRegexp.FindStringSubmatch(topic)
			if len(substr) != 3 {
				retcodes = append(retcodes, qosFail)
				continue
			}
			shared = true
			group = substr[1]
			topic = substr[2]
		}

		// Add subscription to map
		if oldSub, exist := c.subMap[topic]; exist {
			c.topicsMgr.Unsubscribe([]byte(oldSub.topic), oldSub)
			delete(c.subMap, topic)
		}

		sub := topic.NewSubscribtion(c.info.ID, topic, qos, shared, group)

		rqos, err := c.topicsMgr.Subscribe([]byte(topic), qoss[i], sub)
		if err != nil {
			c.logger.Error("subscribe error, ", zap.Error(err), zap.String("client_id", c.info.ID))
			retcodes = append(retcodes, qosFail)
			continue
		}

		c.subMap[topic] = sub

		c.session.AddTopic(topic, qoss[i])
		retcodes = append(retcodes, rqos)
		c.topicsMgr.Retained([]byte(topic), &c.rmsgs)
	}

	suback.ReturnCodes = retcodes

	err := c.WritePacket(suback)
	if err != nil {
		c.logger.Error("Send suback error, ", zap.Error(err), zap.String("client_id", c.info.ID))
		return
	}

	// Process retain message
	for _, rm := range c.rmsgs {
		if err := c.WritePacket(rm); err != nil {
			c.logger.Error("Error publishing retained message:", zap.Any("err", err), zap.String("client_id", c.info.ID))
		} else {
			c.logger.Info("process retain  message: ", zap.Any("packet", packet), zap.String("client_id", c.info.ID))
		}
	}
}

// Unsubscribe
func (c *client) processUnsubscribe(packet *packets.UnsubscribePacket) {
	if c.status == Disconnected {
		return
	}
	b := c.broker
	if b == nil {
		return
	}
	topics := packet.Topics

	for _, topic := range topics {
		sub, exist := c.subMap[topic]
		if exist {
			c.topicsMgr.Unsubscribe([]byte(sub.topic), sub)
			c.session.RemoveTopic(topic)
			delete(c.subMap, topic)
		}

	}

	unsuback := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	unsuback.MessageID = packet.MessageID

	err := c.WritePacket(unsuback)
	if err != nil {
		c.logger.Error("send unsuback error, ", zap.Error(err), zap.String("client_id", c.info.ID))
		return
	}
	// //process ubsubscribe message
	b.BroadcastSubOrUnsubMessage(packet)
}

// Ping
func (c *client) ProcessPing() {
	if c.status == Disconnected {
		return
	}

	resp := packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	if err := c.WritePacket(resp); err != nil {
		c.logger.Error("Send PingResponse error, ", zap.Error(err), zap.String("client_id", c.info.ID))
		return
	}
}

func (c *client) Close() {
	if c.status == Disconnected {
		return
	}

	c.cancelFunc()

	c.status = Disconnected
	//wait for message complete
	// time.Sleep(1 * time.Second)
	// c.status = Disconnected

	b := c.broker
	b.Publish(&bridge.Elements{
		ClientID:  c.info.ID,
		Username:  c.info.username,
		Action:    bridge.Disconnect,
		Timestamp: time.Now().Unix(),
	})

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	subs := c.subMap

	if b != nil {
		b.removeClient(c)
		for _, sub := range subs {
			err := b.topicsMgr.Unsubscribe([]byte(sub.topic), sub)
			if err != nil {
				c.logger.Error("unsubscribe error, ", zap.Error(err), zap.String("client_id", c.info.ID))
			}
		}

		if c.typ == CLIENT {
			b.BroadcastUnSubscribe(subs)
			//offline notification
			b.OnlineOfflineNotification(c.info.ID, false)
		}

		if c.info.willMsg != nil {
			b.PublishMessage(c.info.willMsg)
		}

		if c.typ == CLUSTER {
			b.ConnectToDiscovery()
		}

		//do reconnect
		if c.typ == REMOTE {
			go b.connectRouter(c.route.remoteID, c.route.remoteUrl)
		}
	}
}

func (c *client) WritePacket(packet packets.ControlPacket) error {
	if c.status == Disconnected {
		return nil
	}

	if packet == nil {
		return nil
	}
	if c.conn == nil {
		c.Close()
		return errors.New("Connect lost ....")
	}

	c.mu.Lock()
	err := packet.Write(c.conn)
	c.mu.Unlock()

	return err
}
