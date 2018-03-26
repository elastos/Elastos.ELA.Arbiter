package cs

type SignMessage struct {
	Command string
	Content []byte
}

func (msg *SignMessage) CMD() string {
	return msg.Command
}

func (msg *SignMessage) Serialize() ([]byte, error) {
	return msg.Content, nil
}

func (msg *SignMessage) Deserialize(content []byte) error {
	msg.Content = content
	return nil
}
