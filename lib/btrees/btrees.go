package btrees

type BTreeNode struct {
	keys     []int        // keys for indexing
	children []*BTreeNode // node's children
	isLeaf   bool         // flag to indicate if leaf
	n        int          // curr number of keys
}

type BTree struct {
	root      *BTreeNode // pointer to the root
	minDegree int        // minimum degree
}

func NewBTree(degree int) *BTree {
	root := &BTreeNode{isLeaf: true}
	return &BTree{root: root, minDegree: degree}
}

// Methods with capital letters are public
// Here we use BTreeNode as the receiver
func (n *BTreeNode) Search(k int) (*BTreeNode, int) {
	i := 0
	for i < n.n && k > n.keys[i] {
		i++
	}

	if i < n.n && k == n.keys[i] {
		return n, i
	}

	if n.isLeaf {
		return nil, -1
	}

	// Recursive call as usual for Trees
	return n.children[i].Search(k)
}

// Methods with capital letters are public
func (t *BTree) Insert(k int) {
	root := t.root
	if root.n == 2*t.minDegree-1 {
		newRoot := &BTreeNode{isLeaf: false}
		newRoot.children = append(newRoot.children, root)
		t.root = newRoot
		t.splitChild(newRoot, 0, root)
		t.insertNonFull(newRoot, k)
	} else {
		t.insertNonFull(root, k)
	}
}

func (t *BTree) insertNonFull(node *BTreeNode, k int) {
	i := node.n - 1
	if !node.isLeaf {
		for i >= 0 && k < node.keys[i] {
			i--
		}
		i++
		if node.children[i].n == 2*t.minDegree-1 {
			t.splitChild(node, i, node.children[i])
			if k > node.keys[i] {
				i++
			}
		}
		t.insertNonFull(node.children[i], k)
	} else {
		for i >= 0 && k < node.keys[i] {
			i--
		}
		node.keys = append(node.keys[:i+1], append([]int{k}, node.keys[i+1:]...)...)
		node.n++
	}
}

func (t *BTree) splitChild(parent *BTreeNode, i int, fullChild *BTreeNode) {
	tDegree := t.minDegree
	newChild := &BTreeNode{isLeaf: fullChild.isLeaf, n: tDegree - 1}
	// Move the second half of fullChild's keys and children to newChild
	newChild.keys = append(newChild.keys, fullChild.keys[tDegree:]...)
	fullChild.keys = fullChild.keys[:tDegree-1]
	if !fullChild.isLeaf {
		newChild.children = append(newChild.children, fullChild.children[tDegree:]...)
		fullChild.children = fullChild.children[:tDegree]
	}
	// Insert newChild into parent's children
	parent.children = append(parent.children[:i+1], append([]*BTreeNode{newChild}, parent.children[i+1:]...)...)
	parent.keys = append(parent.keys[:i], append([]int{fullChild.keys[tDegree-1]}, parent.keys[i:]...)...)
	fullChild.keys = fullChild.keys[:tDegree-1]
	parent.n++
}
