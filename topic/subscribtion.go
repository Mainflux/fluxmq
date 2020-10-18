package topic

type Subscription struct {
	ClientID string
	Topic    string
	Qos      byte
	Shared   bool
	Group    string
}

func NewSubscription(clientID string, topic string, qos byte, shared bool, group string) Subscription {
	return Subscription{
		ClientID: clientID,
		Topic:    topic,
		Qos:      qos,
		Shared:   shared,
		Group:    group,
	}
}
