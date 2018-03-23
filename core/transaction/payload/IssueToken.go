package payload

import "io"

type IssueToken struct {
}

func (t *IssueToken) Data(version byte) []byte {
	//TODO: implement IssueToken.Data()
	return []byte{0}
}

func (t *IssueToken) Serialize(w io.Writer, version byte) error {
	return nil
}

func (t *IssueToken) Deserialize(r io.Reader, version byte) error {
	return nil
}
