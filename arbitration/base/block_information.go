package base

import (
	. "Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/common/serialization"
	"Elastos.ELA.Arbiter/core/asset"
	. "Elastos.ELA.Arbiter/core/transaction"
	"bytes"
	"errors"
	"io"
)

type PayloadInfo interface {
	Data(version byte) string
	Serialize(w io.Writer, version byte) error
	Deserialize(r io.Reader, version byte) error
}

type RegisterAssetInfo struct {
	Asset      *asset.Asset
	Amount     string
	Controller string
}

type TransferAssetInfo struct {
}

type TxAttributeInfo struct {
	Usage TransactionAttributeUsage
	Data  string
}

type UTXOTxInputInfo struct {
	ReferTxID          string
	ReferTxOutputIndex uint16
	Sequence           uint32
	Address            string
	Value              string
}

type BalanceTxInputInfo struct {
	AssetID     string
	Value       Fixed64
	ProgramHash string
}

type TxoutputInfo struct {
	AssetID    string
	Value      string
	Address    string
	OutputLock uint32
}

type ProgramInfo struct {
	Code      string
	Parameter string
}

type TxoutputMap struct {
	Key   Uint256
	Txout []TxoutputInfo
}

type AmountMap struct {
	Key   Uint256
	Value Fixed64
}

type BlockHead struct {
	Version          uint32
	PrevBlockHash    string
	TransactionsRoot string
	Timestamp        uint32
	Bits             uint32
	Height           uint32
	Nonce            uint32

	Hash string
}

type TransactionInfo struct {
	TxType         TransactionType
	PayloadVersion byte
	Payload        PayloadInfo
	Attributes     []TxAttributeInfo
	UTXOInputs     []UTXOTxInputInfo
	BalanceInputs  []BalanceTxInputInfo
	Outputs        []TxoutputInfo
	LockTime       uint32
	Programs       []ProgramInfo

	AssetOutputs      []TxoutputMap
	AssetInputAmount  []AmountMap
	AssetOutputAmount []AmountMap
	Timestamp         uint32 `json:",omitempty"`
	Confirminations   uint32 `json:",omitempty"`
	TxSize            uint32 `json:",omitempty"`
	Hash              string
}

type BlockInfo struct {
	Hash            string
	BlockData       *BlockHead
	Transactions    []*TransactionInfo
	Confirminations uint32
	MinerInfo       string
}

func (i RegisterAssetInfo) Data(version byte) string {
	return ""
}

func (i RegisterAssetInfo) Serialize(w io.Writer, version byte) error {
	return nil
}

func (i RegisterAssetInfo) Deserialize(r io.Reader, version byte) error {
	return nil
}

func (i TransferAssetInfo) Data(version byte) string {
	return ""
}

func (i TransferAssetInfo) Serialize(w io.Writer, version byte) error {
	return nil
}

func (i TransferAssetInfo) Deserialize(r io.Reader, version byte) error {
	return nil
}

func (a *TxAttributeInfo) Serialize(w io.Writer) error {
	if err := serialization.WriteUint8(w, byte(a.Usage)); err != nil {
		return errors.New("Transaction attribute Usage serialization failed.")
	}
	if !IsValidAttributeType(a.Usage) {
		return errors.New("[TxAttribute] error: Unsupported attribute Description.")
	}
	if err := serialization.WriteVarString(w, a.Data); err != nil {
		return errors.New("Transaction attribute Data serialization failed.")
	}
	return nil
}

func (a *TxAttributeInfo) Deserialize(r io.Reader) error {
	usage, err := serialization.ReadBytes(r, 1)
	if err != nil {
		return errors.New("Transaction attribute Usage deserialization failed.")
	}
	a.Usage = TransactionAttributeUsage(usage[0])
	if !IsValidAttributeType(a.Usage) {
		return errors.New("[TxAttribute] error: Unsupported attribute Description.")
	}

	data, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Transaction attribute Data deserialization failed.")
	}
	a.Data = data

	return nil
}

func (u *UTXOTxInputInfo) Serialize(w io.Writer) error {
	if err := serialization.WriteVarString(w, u.ReferTxID); err != nil {
		return errors.New("Transaction UTXOTxInputInfo ReferTxID serialization failed.")
	}
	if err := serialization.WriteUint16(w, u.ReferTxOutputIndex); err != nil {
		return errors.New("Transaction UTXOTxInputInfo ReferTxOutputIndex serialization failed.")
	}
	if err := serialization.WriteUint32(w, u.Sequence); err != nil {
		return errors.New("Transaction UTXOTxInputInfo Sequence serialization failed.")
	}
	if err := serialization.WriteVarString(w, u.Address); err != nil {
		return errors.New("Transaction UTXOTxInputInfo Address serialization failed.")
	}
	if err := serialization.WriteVarString(w, u.Value); err != nil {
		return errors.New("Transaction UTXOTxInputInfo Value serialization failed.")
	}
	return nil
}

func (u *UTXOTxInputInfo) Deserialize(r io.Reader) error {
	refer, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Transaction UTXOTxInputInfo ReferTxID deserialization failed.")
	}
	u.ReferTxID = refer

	index, err := serialization.ReadUint16(r)
	if err != nil {
		return errors.New("Transaction UTXOTxInputInfo ReferTxOutputIndex deserialization failed.")
	}
	u.ReferTxOutputIndex = index

	sequence, err := serialization.ReadUint32(r)
	if err != nil {
		return errors.New("Transaction UTXOTxInputInfo Sequence deserialization failed.")
	}
	u.Sequence = sequence

	addr, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Transaction UTXOTxInputInfo Address deserialization failed.")
	}
	u.Address = addr

	value, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Transaction UTXOTxInputInfo Value deserialization failed.")
	}
	u.Value = value

	return nil
}

func (b *BalanceTxInputInfo) Serialize(w io.Writer) error {
	if err := serialization.WriteVarString(w, b.AssetID); err != nil {
		return errors.New("Transaction BalanceTxInputInfo AssetID serialization failed.")
	}
	if err := serialization.WriteVarUint(w, uint64(b.Value)); err != nil {
		return errors.New("Transaction BalanceTxInputInfo Value serialization failed.")
	}
	if err := serialization.WriteVarString(w, b.ProgramHash); err != nil {
		return errors.New("Transaction BalanceTxInputInfo ProgramHash serialization failed.")
	}
	return nil
}

func (b *BalanceTxInputInfo) Deserialize(r io.Reader) error {
	assetid, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Transaction BalanceTxInputInfo AssetID deserialization failed.")
	}
	b.AssetID = assetid

	value, err := serialization.ReadVarUint(r, 0)
	if err != nil {
		return errors.New("Transaction BalanceTxInputInfo Value deserialization failed.")
	}
	b.Value = Fixed64(value)

	programHash, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Transaction BalanceTxInputInfo ProgramHash deserialization failed.")
	}
	b.ProgramHash = programHash

	return nil
}

func (o *TxoutputInfo) Serialize(w io.Writer) error {
	if err := serialization.WriteVarString(w, o.AssetID); err != nil {
		return errors.New("Transaction TxoutputInfo AssetID serialization failed.")
	}
	if err := serialization.WriteVarString(w, o.Value); err != nil {
		return errors.New("Transaction TxoutputInfo Value serialization failed.")
	}
	if err := serialization.WriteVarString(w, o.Address); err != nil {
		return errors.New("Transaction TxoutputInfo Address serialization failed.")
	}
	if err := serialization.WriteUint32(w, o.OutputLock); err != nil {
		return errors.New("Transaction TxoutputInfo OutputLock serialization failed.")
	}

	return nil
}

func (b *TxoutputInfo) Deserialize(r io.Reader) error {
	assetid, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Transaction TxoutputInfo AssetID deserialization failed.")
	}
	b.AssetID = assetid

	value, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Transaction TxoutputInfo Value deserialization failed.")
	}
	b.Value = value

	addr, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Transaction TxoutputInfo Address deserialization failed.")
	}
	b.Address = addr

	lock, err := serialization.ReadUint32(r)
	if err != nil {
		return errors.New("Transaction TxoutputInfo OutputLock deserialization failed.")
	}
	b.OutputLock = lock

	return nil
}

func (p *ProgramInfo) Serialize(w io.Writer) error {
	if err := serialization.WriteVarString(w, p.Code); err != nil {
		return errors.New("Transaction ProgramInfo Code serialization failed.")
	}
	if err := serialization.WriteVarString(w, p.Parameter); err != nil {
		return errors.New("Transaction ProgramInfo Parameter serialization failed.")
	}
	return nil
}

func (p *ProgramInfo) Deserialize(r io.Reader) error {
	code, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Transaction ProgramInfo Code deserialization failed.")
	}
	p.Code = code

	param, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Transaction ProgramInfo Parameter deserialization failed.")
	}
	p.Parameter = param

	return nil
}

func (m *TxoutputMap) Serialize(w io.Writer) error {
	if _, err := m.Key.Serialize(w); err != nil {
		return errors.New("Transaction TxoutputMap Key serialization failed.")
	}
	if err := serialization.WriteVarUint(w, uint64(len(m.Txout))); err != nil {
		return errors.New("Transaction TxoutputMap Txout length serialization failed.")
	}
	for _, txout := range m.Txout {
		if err := txout.Serialize(w); err != nil {
			return err
		}
	}

	return nil
}

func (m *TxoutputMap) Deserialize(r io.Reader) error {
	if err := m.Key.Deserialize(r); err != nil {
		return errors.New("Transaction TxoutputMap Key deserialization failed.")
	}
	length, err := serialization.ReadVarUint(r, 0)
	if err != nil {
		return errors.New("Transaction TxoutputMap Txout length deserialization failed.")
	}
	for i := uint64(0); i < length; i++ {
		txout := TxoutputInfo{}
		if err = txout.Deserialize(r); err != nil {
			return err
		}
		m.Txout = append(m.Txout, txout)
	}
	return nil
}

func (m *AmountMap) Serialize(w io.Writer) error {
	if _, err := m.Key.Serialize(w); err != nil {
		return errors.New("Transaction AmountMap Key serialization failed.")
	}
	if err := serialization.WriteVarUint(w, uint64(m.Value)); err != nil {
		return errors.New("Transaction AmountMap Value serialization failed.")
	}
	return nil
}

func (m *AmountMap) Deserialize(r io.Reader) error {
	if err := m.Key.Deserialize(r); err != nil {
		return errors.New("Transaction AmountMap Key deserialization failed.")
	}
	val, err := serialization.ReadVarUint(r, 0)
	if err != nil {
		return errors.New("Transaction AmountMap Value deserialization failed.")
	}
	m.Value = Fixed64(val)
	return nil
}

func (t *TransactionInfo) Serialize() ([]byte, error) {
	var err error
	//SerializeUnsigned
	buf := new(bytes.Buffer)
	//txType
	buf.Write([]byte{byte(t.TxType)})
	//PayloadVersion
	buf.Write([]byte{t.PayloadVersion})
	if t.Payload == nil {
		return nil, errors.New("Transaction Payload is nil.")
	}
	//Serialize Payload
	t.Payload.Serialize(buf, t.PayloadVersion)
	//[]TxAttributeInfo
	if err = serialization.WriteVarUint(buf, uint64(len(t.Attributes))); err != nil {
		return nil, errors.New("Transaction item txAttribute length serialization failed.")
	}
	for _, attr := range t.Attributes {
		if err = attr.Serialize(buf); err != nil {
			return nil, err
		}
	}
	//[]UTXOTxInputInfo
	if err = serialization.WriteVarUint(buf, uint64(len(t.UTXOInputs))); err != nil {
		return nil, errors.New("Transaction item UTXOInputs length serialization failed.")
	}
	for _, utxo := range t.UTXOInputs {
		if err = utxo.Serialize(buf); err != nil {
			return nil, err
		}
	}
	//[]BalanceTxInputInfo
	if err = serialization.WriteVarUint(buf, uint64(len(t.BalanceInputs))); err != nil {
		return nil, errors.New("Transaction item BalanceInputs length serialization failed.")
	}
	for _, balance := range t.BalanceInputs {
		if err = balance.Serialize(buf); err != nil {
			return nil, err
		}
	}
	//[]TxoutputInfo
	if err = serialization.WriteVarUint(buf, uint64(len(t.Outputs))); err != nil {
		return nil, errors.New("Transaction item BalanceInputs length serialization failed.")
	}
	for _, output := range t.Outputs {
		if err = output.Serialize(buf); err != nil {
			return nil, err
		}
	}
	//LockTime
	if err = serialization.WriteUint32(buf, t.LockTime); err != nil {
		return nil, errors.New("Transaction item LockTime length serialization failed.")
	}
	//[]ProgramInfo
	if err = serialization.WriteVarUint(buf, uint64(len(t.Outputs))); err != nil {
		return nil, errors.New("Transaction item ProgramInfo length serialization failed.")
	}
	for _, program := range t.Programs {
		if err = program.Serialize(buf); err != nil {
			return nil, err
		}
	}
	//[]TxoutputMap
	if err = serialization.WriteVarUint(buf, uint64(len(t.AssetOutputs))); err != nil {
		return nil, errors.New("Transaction item TxoutputMap length serialization failed.")
	}
	for _, m := range t.AssetOutputs {
		if err = m.Serialize(buf); err != nil {
			return nil, err
		}
	}
	//[]AmountMap
	if err = serialization.WriteVarUint(buf, uint64(len(t.AssetInputAmount))); err != nil {
		return nil, errors.New("Transaction item AssetInputAmount length serialization failed.")
	}
	for _, m := range t.AssetInputAmount {
		if err = m.Serialize(buf); err != nil {
			return nil, err
		}
	}
	//[]AmountMap
	if err = serialization.WriteVarUint(buf, uint64(len(t.AssetOutputAmount))); err != nil {
		return nil, errors.New("Transaction item AssetOutputAmount length serialization failed.")
	}
	for _, m := range t.AssetOutputAmount {
		if err = m.Serialize(buf); err != nil {
			return nil, err
		}
	}
	//Timestamp uint32 `json:",omitempty"`
	if err = serialization.WriteUint32(buf, t.Timestamp); err != nil {
		return nil, errors.New("Transaction item Timestamp serialization failed.")
	}
	//Confirminations uint32 `json:",omitempty"`
	if err = serialization.WriteUint32(buf, t.Confirminations); err != nil {
		return nil, errors.New("Transaction item Confirminations serialization failed.")
	}
	//TxSize uint32 `json:",omitempty"`
	if err = serialization.WriteUint32(buf, t.TxSize); err != nil {
		return nil, errors.New("Transaction item TxSize serialization failed.")
	}
	//Hash string
	if err = serialization.WriteVarString(buf, t.Hash); err != nil {
		return nil, errors.New("Transaction item Hash serialization failed.")
	}

	return buf.Bytes(), nil
}

func (t *TransactionInfo) Deserialize(data []byte) error {
	var err error
	tmpByte := make([]byte, 1)
	r := bytes.NewReader(data)
	//txType
	if _, err = io.ReadFull(r, tmpByte); err != nil {
		return errors.New("Transaction type deserialize failed.")
	}
	t.TxType = TransactionType(tmpByte[0])
	switch t.TxType {
	case RegisterAsset:
		t.Payload = new(RegisterAssetInfo)
	case TransferAsset:
		t.Payload = new(TransferAssetInfo)
	case Deploy:
	default:
		return errors.New("Invalid transaction type.")
	}
	//PayloadVersion
	if _, err = io.ReadFull(r, tmpByte); err != nil {
		return err
	}
	t.PayloadVersion = tmpByte[0]
	//Payload
	if err = t.Payload.Deserialize(r, t.PayloadVersion); err != nil {
		return err
	}
	//Attributes     []TxAttributeInfo
	length, err := serialization.ReadVarUint(r, 0)
	if err != nil {
		return errors.New("Attributes length deserialize failed.")
	}
	for i := uint64(0); i < length; i++ {
		attr := TxAttributeInfo{}
		if err = attr.Deserialize(r); err != nil {
			return err
		}
		t.Attributes = append(t.Attributes, attr)
	}
	//UTXOInputs     []UTXOTxInputInfo
	if length, err = serialization.ReadVarUint(r, 0); err != nil {
		return errors.New("UTXOInputs length deserialize failed.")
	}
	for i := uint64(0); i < length; i++ {
		utxo := UTXOTxInputInfo{}
		if err = utxo.Deserialize(r); err != nil {
			return err
		}
		t.UTXOInputs = append(t.UTXOInputs, utxo)
	}
	//BalanceInputs  []BalanceTxInputInfo
	if length, err = serialization.ReadVarUint(r, 0); err != nil {
		return errors.New("BalanceInputs length deserialize failed.")
	}
	for i := uint64(0); i < length; i++ {
		balance := BalanceTxInputInfo{}
		if err = balance.Deserialize(r); err != nil {
			return err
		}
		t.BalanceInputs = append(t.BalanceInputs, balance)
	}
	//Outputs        []TxoutputInfo
	if length, err = serialization.ReadVarUint(r, 0); err != nil {
		return errors.New("TxoutputInfo length deserialize failed.")
	}
	for i := uint64(0); i < length; i++ {
		output := TxoutputInfo{}
		if err = output.Deserialize(r); err != nil {
			return err
		}
		t.Outputs = append(t.Outputs, output)
	}
	//LockTime       uint32
	temp, err := serialization.ReadUint32(r)
	if err != nil {
		return errors.New("LockTime deserialize failed.")
	}
	t.LockTime = temp
	//Programs       []ProgramInfo
	if length, err = serialization.ReadVarUint(r, 0); err != nil {
		return errors.New("Programinfo length deserialize failed.")
	}
	for i := uint64(0); i < length; i++ {
		program := ProgramInfo{}
		if err = program.Deserialize(r); err != nil {
			return err
		}
		t.Programs = append(t.Programs, program)
	}
	//AssetOutputs      []TxoutputMap
	if length, err = serialization.ReadVarUint(r, 0); err != nil {
		return errors.New("AssetOutputs length deserialize failed.")
	}
	for i := uint64(0); i < length; i++ {
		output := TxoutputMap{}
		if err = output.Deserialize(r); err != nil {
			return err
		}
		t.AssetOutputs = append(t.AssetOutputs, output)
	}
	//AssetInputAmount  []AmountMap
	if length, err = serialization.ReadVarUint(r, 0); err != nil {
		return errors.New("AssetInputAmount length deserialize failed.")
	}
	for i := uint64(0); i < length; i++ {
		amount := AmountMap{}
		if err = amount.Deserialize(r); err != nil {
			return err
		}
		t.AssetInputAmount = append(t.AssetInputAmount, amount)
	}
	//AssetOutputAmount []AmountMap
	if length, err = serialization.ReadVarUint(r, 0); err != nil {
		return errors.New("AssetOutputAmount length deserialize failed.")
	}
	for i := uint64(0); i < length; i++ {
		amount := AmountMap{}
		if err = amount.Deserialize(r); err != nil {
			return err
		}
		t.AssetOutputAmount = append(t.AssetOutputAmount, amount)
	}
	//Timestamp uint32 `json:",omitempty"`
	timestamp, err := serialization.ReadUint32(r)
	if err != nil {
		return errors.New("Timestamp deserialize failed.")
	}
	t.Timestamp = timestamp
	//Confirminations uint32 `json:",omitempty"`
	confirm, err := serialization.ReadUint32(r)
	if err != nil {
		return errors.New("Confirminations deserialize failed.")
	}
	t.Confirminations = confirm
	//TxSize uint32 `json:",omitempty"`
	txSize, err := serialization.ReadUint32(r)
	if err != nil {
		return errors.New("TxSize deserialize failed.")
	}
	t.TxSize = txSize
	//Hash string
	hash, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("Hash deserialize failed.")
	}
	t.Hash = hash

	return nil
}

func (trans *TransactionInfo) ConvertFrom(tx *Transaction) error {
	return nil
}

func (trans *TransactionInfo) ConvertTo() (*Transaction, error) {
	return nil, nil
}
