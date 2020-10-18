package session

import (
	"errors"
	"net"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/mainflux/fluxmq/auth"
	"github.com/mainflux/fluxmq/client"
	"go.uber.org/zap"
)

// Session represents MQTT Proxy session between client and broker.
type Session struct {
	ID        string
	client    client.Client
	conn      net.Conn
	keepalive uint16
	lwt       *packets.PublishPacket
	logger    *zap.Logger
	repo      *Repository
}

// New creates a new Session.
func New(client client.Client, conn net.Conn, keepalive uint16, lwt *packets.PublishPacket, repo *Repository, logger *zap.Logger) Session {
	return Session{
		client:    client,
		conn:      conn,
		keepalive: keepalive,
		lwt:       lwt,
		repo:      repo,
		logger:    logger,
	}
}

func (s *Session) ReadLoop() {
	keepAlive := time.Second * time.Duration(s.keepalive)
	timeOut := keepAlive + (keepAlive / 2)
	dpkt := packets.NewControlPacket(packets.Disconnect).(*packets.DisconnectPacket)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			if keepAlive > 0 {
				if err := s.conn.SetReadDeadline(time.Now().Add(timeOut)); err != nil {
					s.logger.Error("Set read timeout error: ", zap.Error(err), zap.String("client_id", s.client.Info.ID))
					s.ProcessMessage(dpkt)
					return
				}
			}

			packet, err := packets.ReadPacket(s.conn)
			if err != nil {
				s.logger.Error("Read packet error: ", zap.Error(err), zap.String("client_id", s.client.Info.ID))
				s.ProcessMessage(dpkt)
				return
			}

			s.ProcessMessage(packet)
		}
	}

}

func (s Session) ProcessMessage(p packets.ControlPacket) {
	switch p.(type) {
	case *packets.ConnackPacket:
	case *packets.ConnectPacket:
	case *packets.PublishPacket:
		packet := p.(*packets.PublishPacket)
		s.ProcessPublish(packet)
	case *packets.PubackPacket:
	case *packets.PubrecPacket:
	case *packets.PubrelPacket:
	case *packets.PubcompPacket:
	case *packets.SubscribePacket:
		packet := p.(*packets.SubscribePacket)
		s.ProcessSubscribe(packet)
	case *packets.SubackPacket:
	case *packets.UnsubscribePacket:
		packet := p.(*packets.UnsubscribePacket)
		s.ProcessUnSubscribe(packet)
	case *packets.UnsubackPacket:
	case *packets.PingreqPacket:
		s.ProcessPing()
	case *packets.PingrespPacket:
	case *packets.DisconnectPacket:
		s.Close()
	default:
		s.logger.Info("Unknown packet", zap.String("client_id", s.client.Info.ID))
	}
}

// Publish
func (s Session) ProcessPublish(packet *packets.PublishPacket) {
	switch packet.Qos {
	case qos0:
		s.ProcessPublishMessage(packet)
	case qos1:
		puback := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
		puback.MessageID = packet.MessageID
		if err := s.WritePacket(puback); err != nil {
			s.logger.Error("Send puback error, ", zap.Error(err), zap.String("client_id", s.client.Info.ID))
			return
		}
		s.ProcessPublishMessage(packet)
	case qos2:
		// TODO
		return
	default:
		s.logger.Error("Publish with unknown QoS", zap.String("client_id", s.client.Info.ID))
		return
	}
}

func (s *Session) ProcessPublishMessage(packet *packets.PublishPacket) {
	if packet.Retain {
		if err := s.topicsMgr.Retain(packet); err != nil {
			s.logger.Error("Error retaining message: ", zap.Error(err), zap.String("client_id", s.client.Info.ID))
		}
	}

	err := s.topicsMgr.Subscribers([]byte(packet.TopicName), packet.Qos, &s.subs, &s.qoss)
	if err != nil {
		s.logger.Error("Error retrieving subscribers list: ", zap.String("client_id", s.info.ID))
		return
	}

	if len(s.subs) == 0 {
		return
	}

	var qsub []int
	for i, sub := range s.subs {
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
		sub := s.subs[qsub[idx]].(*subscription)
		publish(sub, packet)
	}

}

// Subscribe
func (s *Session) processSubscribe(packet *packets.SubscribePacket) {
	if s.status == Disconnected {
		return
	}

	b := s.broker
	if b == nil {
		return
	}

	qoss := packet.Qoss

	suback := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	suback.MessageID = packet.MessageID
	var retcodes []byte

	for i, _ := range packet.Topics {

		// Handle Auth Subscribe
		if err := auth.Handler.AuthSubscribe(&(s.client.Info), &(packet.Topics[i])); err != nil {
			s.logger.Error("Auth Subscribe error, ", zap.Error(err), zap.String("client_id", s.client.Info.ID))
			retcodes = append(retcodes, qosFail)
			continue
		}

		/*
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
		*/

		// Add subscription to map
		if oldSub, exist := s.subMap[topic]; exist {
			s.topicsMgr.Unsubscribe([]byte(oldSub.topic), oldSub)
			delete(s.subMap, topic)
		}

		sub := topis.NewSubscribtion(s.info.ID, topic, qos, shared, group)

		rqos, err := s.topicsMgr.Subscribe([]byte(topic), qoss[i], sub)
		if err != nil {
			s.logger.Error("subscribe error, ", zap.Error(err), zap.String("client_id", s.info.ID))
			retcodes = append(retcodes, qosFail)
			continue
		}

		s.subMap[topic] = sub

		s.session.AddTopic(topic, qoss[i])
		retcodes = append(retcodes, rqos)
		s.topicsMgr.Retained([]byte(topic), &s.rmsgs)
	}

	suback.ReturnCodes = retcodes

	err := s.WritePacket(suback)
	if err != nil {
		s.logger.Error("Send suback error, ", zap.Error(err), zap.String("client_id", s.info.ID))
		return
	}

	// Process retain message
	for _, rm := range s.rmsgs {
		if err := s.WritePacket(rm); err != nil {
			s.logger.Error("Error publishing retained message:", zap.Any("err", err), zap.String("client_id", s.info.ID))
		} else {
			s.logger.Info("process retain  message: ", zap.Any("packet", packet), zap.String("client_id", s.info.ID))
		}
	}
}

// Unsubscribe
func (s *Session) processUnsubscribe(packet *packets.UnsubscribePacket) {
	if s.status == Disconnected {
		return
	}
	b := s.broker
	if b == nil {
		return
	}
	topics := packet.Topics

	for _, topic := range topics {
		sub, exist := s.subMap[topic]
		if exist {
			s.topicsMgr.Unsubscribe([]byte(sub.topic), sub)
			s.session.RemoveTopic(topic)
			delete(s.subMap, topic)
		}

	}

	unsuback := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	unsuback.MessageID = packet.MessageID

	err := s.WritePacket(unsuback)
	if err != nil {
		s.logger.Error("send unsuback error, ", zap.Error(err), zap.String("client_id", s.info.ID))
		return
	}
	// //process ubsubscribe message
	b.BroadcastSubOrUnsubMessage(packet)
}

// Ping
func (s *Session) ProcessPing() {
	if s.status == Disconnected {
		return
	}

	resp := packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	if err := s.WritePacket(resp); err != nil {
		s.logger.Error("Send PingResponse error, ", zap.Error(err), zap.String("client_id", s.info.ID))
		return
	}
}

func (s *Session) Close() {
	if s.status == Disconnected {
		return
	}

	s.cancelFunc()

	s.status = Disconnected
	//wait for message complete
	// time.Sleep(1 * time.Second)
	// s.status = Disconnected

	b := s.broker
	b.Publish(&bridge.Elements{
		ClientID:  s.info.ID,
		Username:  s.info.username,
		Action:    bridge.Disconnect,
		Timestamp: time.Now().Unix(),
	})

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}

	subs := s.subMap

	if b != nil {
		b.removeClient(c)
		for _, sub := range subs {
			err := b.topicsMgr.Unsubscribe([]byte(sub.topic), sub)
			if err != nil {
				s.logger.Error("unsubscribe error, ", zap.Error(err), zap.String("client_id", s.info.ID))
			}
		}

		if s.typ == CLIENT {
			b.BroadcastUnSubscribe(subs)
			//offline notification
			b.OnlineOfflineNotification(s.info.ID, false)
		}

		if s.info.willMsg != nil {
			b.PublishMessage(s.info.willMsg)
		}

		if s.typ == CLUSTER {
			b.ConnectToDiscovery()
		}

		//do reconnect
		if s.typ == REMOTE {
			go b.connectRouter(s.route.remoteID, s.route.remoteUrl)
		}
	}
}

func (s *Session) WritePacket(packet packets.ControlPacket) error {
	if s.status == Disconnected {
		return nil
	}

	if packet == nil {
		return nil
	}
	if s.conn == nil {
		s.Close()
		return errors.New("Connect lost ....")
	}

	s.mu.Lock()
	err := packet.Write(s.conn)
	s.mu.Unlock()

	return err
}
