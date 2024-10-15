package btrees

import (
	"testing"
)

func TestBTreeInsertSingle(t *testing.T) {
	tree := NewBTree(3)
	tree.Insert(10)
	if tree.root.keys[0] != 10 {
		t.Errorf("Expected root key to be 10, got %d", tree.root.keys[0])
	}
	if tree.root.n != 1 {
		t.Errorf("Expected root n to be 1, got %d", tree.root.n)
	}
}

func TestBTreeInsertMultiple(t *testing.T) {
	tree := NewBTree(3)
	keys := []int{10, 20, 5, 6, 12, 30, 7, 17}
	for _, key := range keys {
		tree.Insert(key)
	}
	if tree.root.keys[0] != 10 {
		t.Errorf("Expected root key to be 10, got %d", tree.root.keys[0])
	}
	if tree.root.n != 1 {
		t.Errorf("Expected root node count to be 1, got %d", tree.root.n)
	}
	if tree.root.children[0].keys[0] != 5 || tree.root.children[0].keys[1] != 6 || tree.root.children[0].keys[2] != 7 {
		t.Errorf("Expected left child keys to be [5, 6, 7], got %v", tree.root.children[0].keys)
	}
	if tree.root.children[1].keys[0] != 12 || tree.root.children[1].keys[1] != 17 || tree.root.children[1].keys[2] != 20 || tree.root.children[1].keys[3] != 30 {
		t.Errorf("Expected right child keys to be [12, 17, 20, 30], got %v", tree.root.children[1].keys)
	}
}
