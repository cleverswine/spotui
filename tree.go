package main

import (
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// Node represents a node in a tui tree
type Node struct {
	Name         string
	Label        string
	ID           string
	Level        int
	Meta         map[string]interface{}
	ExpandFunc   func(n *Node) ([]*Node, error)
	KeyPressFunc func(n *Node, k string)
}

func setColor(n *Node, tn *tview.TreeNode) {
	if n.Meta != nil {
		if color, ok := n.Meta["color"]; ok {
			tn.SetColor(color.(tcell.Color))
		}
	}
}

func treeKeyBindings(tree *tview.TreeView) func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		if key.Key() == tcell.KeyRune {
			selected := tree.GetCurrentNode().GetReference().(*Node)
			k := string(key.Rune())
			if selected.KeyPressFunc != nil {
				// execute key press on selected node if a func is provided
				selected.KeyPressFunc(selected, k)
				setColor(selected, tree.GetCurrentNode())
			} else if selected.Level == 1 {
				// search top-level nodes
				k := strings.ToUpper(k)
				logger.Println("searching for items starting with " + k)
				children := tree.GetRoot().GetChildren()
				for _, child := range children {
					n := child.GetReference().(*Node)
					if strings.HasPrefix(n.Name, k) {
						logger.Println("found " + n.Label)
						tree.SetCurrentNode(child)
						return nil
					}
				}
			}
			return nil
		}
		switch key.Key() {
		case tcell.KeyLeft:
			// collapse node
			tree.GetCurrentNode().SetExpanded(false)
			return nil
		case tcell.KeyRight:
			// expand node
			selected := tree.GetCurrentNode()
			if selected == nil {
				return nil
			}
			if len(selected.GetChildren()) > 0 {
				selected.SetExpanded(true)
				return nil
			}
			node := selected.GetReference().(*Node)
			if node.ExpandFunc == nil {
				return nil
			}
			children, err := node.ExpandFunc(node)
			if err != nil {
				logger.Println(err)
				return nil
			}
			for _, child := range children {
				childNode := tview.NewTreeNode(child.Label).SetReference(child).SetSelectable(true)
				setColor(child, childNode)
				selected.AddChild(childNode)
			}
			selected.SetExpanded(true)
			return nil
		case tcell.KeyEsc:
			// collapse all nodes
			sel := tree.GetCurrentNode()
			children := tree.GetRoot().GetChildren()
			for _, child := range children {
				child.CollapseAll()
			}
			// this doesn't work if sel is not top level node
			tree.SetCurrentNode(sel)
			return nil
		}
		return key
	}
}
