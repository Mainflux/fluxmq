package topic

type Subscription struct {
	Client string
	Topic  string
	Qos    byte
	Shared bool
	Group  string
}

func NewSubscription(Client, Topic string, Qos byte, Shared bool, Group string) Subscription {
	return s := Subscription {
		Client: client,
		Topic: topic,
		Qos: qos,
		Shared: shared,
		Group: group
	}
}
