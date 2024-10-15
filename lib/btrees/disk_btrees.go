package btrees

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const HEADER = 4 // 4 bytes for the header

const BTREE_PAGE_SIZE = 4096
const BTREE_MAX_KEY_SIZE = 1000
const BTREE_MAX_VAL_SIZE = 3000

type BPlusNode []byte // Using byte slices to represent nodes to make simpler to dump to disk

type BPlusTree struct {
	// pointer (nonzero page number)
	root uint64
	// callbacks for managing pages
	get func(uint64) BPlusNode // get page from disk
	new func([]byte) uint64    // create new page on disk
	del func(uint64)           // delete page from disk
}

const (
	BNODE_INTERNAL = 1 // Internal node type
	BNODE_LEAF     = 2 // Leaf node type
)

/*
 * B+Tree node layout
 * +------------------------------------+
 * | type | nkeys | pointers   | offsets    | key-values |
 * | 2B   | 2B    | nkeys * 8B | nkeys * 2B |     ...    |
 * +------------------------------------+
 * Key-values layout
 * +---------------------+
 * | klen | vlen | key | value |
 * | 2B   | 2B   | ... |  ...  |
 * +---------------------+
 */

func (node BPlusNode) btype() uint16 {
	return binary.LittleEndian.Uint16(node[0:2])
}

func (node BPlusNode) nkeys() uint16 {
	return binary.LittleEndian.Uint16(node[2:4])
}

func (node BPlusNode) setHeader(btype uint16, nkeys uint16) {
	binary.LittleEndian.PutUint16(node[0:2], btype)
	binary.LittleEndian.PutUint16(node[2:4], nkeys)
}

func (node BPlusNode) getPtr(idx uint16) (uint64, error) {
	if idx >= node.nkeys() {
		return 0, errors.New("index out of range")
	}
	offset := HEADER + 8*idx
	return binary.LittleEndian.Uint64(node[offset : offset+8]), nil
}

func (node BPlusNode) setPtr(idx uint16, val uint64) error {
	if idx >= node.nkeys() {
		return errors.New("index out of range")
	}
	offset := HEADER + 8*idx
	binary.LittleEndian.PutUint64(node[offset:offset+8], val)
	return nil
}

func offsetPos(node BPlusNode, idx uint16) (uint16, error) {
	if idx <= 0 || idx > node.nkeys() {
		return 0, errors.New("index out of range")
	}

	return HEADER + 8*node.nkeys() + 2*(idx-1), nil
}

func (node BPlusNode) getOffset(idx uint16) (uint16, error) {
	if idx == 0 {
		return 0, nil
	}
	offset, err := offsetPos(node, idx)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(node[offset:]), nil
}

func (node BPlusNode) setOffset(idx uint16, offset uint16) {
	pos, err := offsetPos(node, idx)
	if err != nil {
		panic(err)
	}
	binary.LittleEndian.PutUint16(node[pos:], offset)
}

// Key values operations
func (node BPlusNode) kvPos(idx uint16) uint16 {
	offset, err := node.getOffset(idx)
	if err != nil {
		panic(err)
	}
	return HEADER + 8*node.nkeys() + 2*node.nkeys() + offset
}

func (node BPlusNode) getKey(idx uint16) []byte {
	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos : pos+2])
	return node[pos+4:][:klen]
}

func (node BPlusNode) getVal(idx uint16) []byte {
	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos : pos+2])
	vlen := binary.LittleEndian.Uint16(node[pos+2 : pos+4])
	return node[pos+4+klen:][:vlen]
}

// Node size in bytes
func (node BPlusNode) nbytes() uint16 {
	return node.kvPos(node.nkeys())
}

// Key value lookups on nodes
// TODO: Implement binary search
func nodeLookupLE(node BPlusNode, key []byte) uint16 {
	nkeys := node.nkeys()
	found := uint16(0)

	for i := uint16(1); i < nkeys; i++ {
		cmp := bytes.Compare(node.getKey(i), key)
		if cmp <= 0 {
			found = i
		}
		if cmp >= 0 {
			break
		}
	}
	return found
}

// Insert new key to leaf node
// using copy-on-write strategy
func leafInsert(new BPlusNode, old BPlusNode, idx uint16, key []byte, val []byte) {
	// Let's set the new node as a leaf node
	// and increment the number of keys
	// it is going to hold the new key
	new.setHeader(BNODE_LEAF, old.nkeys()+1)
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, 0, key, val)
	nodeAppendRange(new, old, idx+1, idx, old.nkeys()-idx)
}

// Update value in leaf node
func leafUpdate(new BPlusNode, old BPlusNode, idx uint16, key []byte, val []byte) {
	// Let's set the new node as a leaf node
	// and keep the number of keys
	new.setHeader(BNODE_LEAF, old.nkeys())
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, 0, key, val)
	nodeAppendRange(new, old, idx+1, idx+1, old.nkeys()-idx-1)
}

// Append KV value in new node
func nodeAppendKV(new BPlusNode, idx uint16, ptr uint64, key []byte, val []byte) {
	err := new.setPtr(idx, ptr)
	if err != nil {
		panic(err)
	}
	pos := new.kvPos(idx)
	binary.LittleEndian.PutUint16(new[pos:pos+2], uint16(len(key)))
	binary.LittleEndian.PutUint16(new[pos+2:pos+4], uint16(len(val)))
	copy(new[pos+4:], key)
	copy(new[pos+4+uint16(len(key)):], val)
	// Update the offset
	old_offset, err := new.getOffset(idx)
	if err != nil {
		panic(err)
	}
	new.setOffset(idx+1, old_offset+4+uint16((len(key)+len(val))))
}

// Append range of pointers and offsets
func nodeAppendRange(new BPlusNode, old BPlusNode, dstNew uint16, srcOld uint16, n uint16) {
	if n == 0 {
		return
	}
	// Copy pointers
	copy(new[HEADER+8*dstNew:], old[HEADER+8*srcOld:][:8*n])
	// Copy offsets
	copy(new[HEADER+8*new.nkeys()+2*dstNew:], old[HEADER+8*old.nkeys()+2*srcOld:][:2*n])
}

func nodeReplaceKidNode(tree *BPlusTree, new BPlusNode, old BPlusNode, idx uint16, kids ...BPlusNode) {
	inc := uint16(len(kids))
	new.setHeader(BNODE_INTERNAL, old.nkeys()+inc-1)
	nodeAppendRange(new, old, 0, 0, idx)
	for i, node := range kids {
		nodeAppendKV(new, idx+uint16(i), tree.new(node), node.getKey(0), nil)
	}
	nodeAppendRange(new, old, idx+inc, idx+1, old.nkeys()-idx-1)
}

// Split node in two so the 2nd node always fit on page
func nodeSplit2(left BPlusNode, right BPlusNode, old BPlusNode) {
	nkeys := old.nkeys()
	mid := nkeys / 2
	// Copy left node
	copy(left, old)
	left.setHeader(old.btype(), mid)
	// Copy right node
	copy(right, old[mid:])
	right.setHeader(old.btype(), nkeys-mid)
}

// Split node if it is too big. The results are 1-3 nodes
func nodeSplit3(old BPlusNode) (uint16, [3]BPlusNode) {
	if old.nbytes() <= BTREE_PAGE_SIZE {
		old := old[:BTREE_PAGE_SIZE]
		return 1, [3]BPlusNode{old}
	}

	left := BPlusNode(make([]byte, 2*BTREE_PAGE_SIZE))
	right := BPlusNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(left, right, old)
	if left.nbytes() <= BTREE_PAGE_SIZE {
		left := left[:BTREE_PAGE_SIZE]
		return 2, [3]BPlusNode{left, right}
	}

	leftleft := BPlusNode(make([]byte, BTREE_PAGE_SIZE))
	middle := BPlusNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(leftleft, middle, left)
	if leftleft.nbytes() > BTREE_PAGE_SIZE {
		panic("left node too big!")
	}
	return 3, [3]BPlusNode{leftleft, middle, right}
}

// Insert key-value in kid node
func nodeInsert(tree *BPlusTree, new BPlusNode, old BPlusNode, idx uint16, key []byte, val []byte) {
	kptr, err := old.getPtr(idx)
	if err != nil {
		panic(err)
	}
	// recursively insert the key-value in the kid node
	knode := treeInsert(tree, tree.get(kptr), key, val)
	// split the result
	nsplit, split := nodeSplit3(knode)
	// deallocate the kid node
	tree.del(kptr)
	// update kid links
	nodeReplaceKidNode(tree, new, old, idx, split[:nsplit]...)
}

// Using previous functions to insert a key-value in the Tree
// We must remember that the tree is a B+Tree, so the keys are only in the leaf nodes
// inserting the key-value might result in a split
// the caller is responsible for deallocating the input node
// and splitting and allocating new nodes
func treeInsert(tree *BPlusTree, node BPlusNode, key []byte, val []byte) BPlusNode {
	// Result node can be bigger than one page
	// so we need to split it in 1-3 nodes
	new := BPlusNode(make([]byte, 2*BTREE_PAGE_SIZE))

	// Find idx for insertion
	idx := nodeLookupLE(node, key)
	// If it is a leaf node
	// insert the key-value
	// otherwise, insert the key-value in the leaf node

	switch node.btype() {
	case BNODE_LEAF:
		if bytes.Equal(key, node.getKey(idx)) {
			// In this case we are updating the value
			leafUpdate(new, node, idx, key, val)
		} else {
			// In this case we are inserting a new key
			// right after the key at idx
			leafInsert(new, node, idx+1, key, val)
		}
	case BNODE_INTERNAL:
		// Internal node, insert it to a kid node
		nodeInsert(tree, new, node, idx, key, val)
	default:
		panic("unknown node type")
	}
	return new
}

func init() {
	node1max := HEADER + 8 + 2 + 4 + BTREE_MAX_KEY_SIZE + BTREE_MAX_VAL_SIZE
	if node1max > BTREE_PAGE_SIZE {
		panic("Node size exceeds page size")
	}
}
