package sideauxpow

import (
	"bytes"
	"crypto/sha256"
	"errors"

	. "github.com/elastos/Elastos.ELA.Utility/common"
)

type MerkleTree struct {
	Depth uint
	Root  *MerkleTreeNode
}

type MerkleTreeNode struct {
	Hash  Uint256
	Left  *MerkleTreeNode
	Right *MerkleTreeNode
}

func DoubleSHA256(s []Uint256) Uint256 {
	b := new(bytes.Buffer)
	for _, d := range s {
		d.Serialize(b)
	}
	temp := sha256.Sum256(b.Bytes())
	f := sha256.Sum256(temp[:])
	return Uint256(f)
}

func (t *MerkleTreeNode) IsLeaf() bool {
	return t.Left == nil && t.Right == nil
}

//use []Uint256 to create a new MerkleTree
func NewMerkleTree(hashes []Uint256) (*MerkleTree, error) {
	if len(hashes) == 0 {
		return nil, errors.New("NewMerkleTree input no item error.")
	}

	var height uint = 1
	nodes := generateLeaves(hashes)
	for len(nodes) > 1 {
		nodes = levelUp(nodes)
		height += 1
	}
	mt := &MerkleTree{
		Root:  nodes[0],
		Depth: height,
	}
	return mt, nil
}

//Generate the leaves nodes
func generateLeaves(hashes []Uint256) []*MerkleTreeNode {
	var leaves []*MerkleTreeNode
	for _, d := range hashes {
		node := &MerkleTreeNode{
			Hash: d,
		}
		leaves = append(leaves, node)
	}
	return leaves
}

//calc the next level's hash use double sha256
func levelUp(nodes []*MerkleTreeNode) []*MerkleTreeNode {
	var nextLevel []*MerkleTreeNode
	for i := 0; i < len(nodes)/2; i++ {
		var data []Uint256
		data = append(data, nodes[i*2].Hash)
		data = append(data, nodes[i*2+1].Hash)
		hash := DoubleSHA256(data)
		node := &MerkleTreeNode{
			Hash:  hash,
			Left:  nodes[i*2],
			Right: nodes[i*2+1],
		}
		nextLevel = append(nextLevel, node)
	}
	if len(nodes)%2 == 1 {
		var data []Uint256
		data = append(data, nodes[len(nodes)-1].Hash)
		data = append(data, nodes[len(nodes)-1].Hash)
		hash := DoubleSHA256(data)
		node := &MerkleTreeNode{
			Hash:  hash,
			Left:  nodes[len(nodes)-1],
			Right: nodes[len(nodes)-1],
		}
		nextLevel = append(nextLevel, node)
	}
	return nextLevel
}

//input a []Uint256, create a MerkleTree & calc the root hash
func ComputeRoot(hashes []Uint256) (Uint256, error) {
	if len(hashes) == 0 {
		return Uint256{}, errors.New("NewMerkleTree input no item error.")
	}
	if len(hashes) == 1 {
		return hashes[0], nil
	}
	tree, _ := NewMerkleTree(hashes)
	return tree.Root.Hash, nil
}

func GetMerkleHashtree(hashes []Uint256) ([]Uint256, error) {
	if len(hashes) == 0 {
		return nil, errors.New("GetTreeArray input no item error.")
	}

	nodesmap := make(map[int][]*MerkleTreeNode, 0)
	var height int = 1
	nodes := generateLeaves(hashes)
	if len(nodes)%2 == 1 {
		nodes = append(nodes, nodes[len(nodes)-1])
	}
	nodesmap[height] = nodes

	for len(nodes) > 1 {
		nodes = levelUp(nodes)
		height += 1
		nodesmap[height] = nodes
	}

	var nodetree []*MerkleTreeNode
	for i := height; i > 0; i-- {
		nodetree = append(nodetree, nodesmap[i]...)
	}

	var hashtree []Uint256
	for _, node := range nodetree {
		hashtree = append(hashtree, node.Hash)
	}

	return hashtree, nil
}

func storeNode(hashtree []Uint256, branch []Uint256, pos int) ([]Uint256, int) {
	var brother Uint256
	var mask int
	side := pos % 2
	if side == 0 {
		// node on right
		brother = hashtree[pos-1]
		mask = 1
	} else {
		// node on left
		brother = hashtree[pos+1]
		mask = 0
	}
	branch = append(branch, brother)

	return branch, mask
}

func GetMerkleBranch(hashtree []Uint256, pos int) (Uint256, []Uint256, int) {
	var branch []Uint256
	var sidemaskArray []int
	var index int

	leaf := hashtree[pos]
	branch, sidemask := storeNode(hashtree, branch, pos)
	sidemaskArray = append(sidemaskArray, sidemask)
	parentpos := (pos - 1) / 2
	parentnode := hashtree[parentpos]
	for parentnode != hashtree[0] {
		branch, sidemask = storeNode(hashtree, branch, parentpos)
		sidemaskArray = append(sidemaskArray, sidemask)
		parentpos = (parentpos - 1) / 2
		parentnode = hashtree[parentpos]
	}

	for i := len(sidemaskArray) - 1; i >= 0; i-- {
		index <<= 1
		index |= sidemaskArray[i]
	}

	return leaf, branch, index
}
