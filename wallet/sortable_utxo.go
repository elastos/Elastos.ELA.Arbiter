package wallet

import (
	"sort"
)

type SortableUTXOs []*AddressUTXO

func (utxos SortableUTXOs) Len() int      { return len(utxos) }
func (utxos SortableUTXOs) Swap(i, j int) { utxos[i], utxos[j] = utxos[j], utxos[i] }
func (utxos SortableUTXOs) Less(i, j int) bool {
	if *utxos[i].Amount > *utxos[j].Amount {
		return false
	} else {
		return true
	}
}

func SortUTXOs(utxos []*AddressUTXO) []*AddressUTXO {
	sortableUTXOs := SortableUTXOs(utxos)
	sort.Sort(sortableUTXOs)
	return sortableUTXOs
}
