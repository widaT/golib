package server

import (
	"strings"
)

type Tree struct {
	prefix string
	routers []*Tree
	runnable interface{}
}

func NewTree() *Tree {
	return &Tree{}
}

func (t *Tree) AddTree(prefix string, tree *Tree) {
	t.addTree(splitPath(prefix), tree)
}

func (t *Tree) addTree(segments []string, tree *Tree) {
	if len(segments) == 0 {
		panic("prefix should has path")
	}
	seg := segments[0]

	if len(segments) == 1 {
		tree.prefix = seg
		t.routers = append(t.routers, tree)
		return
	}

	subTree := NewTree()
	subTree.prefix = seg
	t.routers = append(t.routers, subTree)
	subTree.addTree(segments[1:], tree)
}

func (t *Tree) AddRouter(pattern string, runnable interface{}) {
	t.addseg(splitPath(pattern), runnable)
}

func (t *Tree) addseg(segments []string, route interface{}) {
	if len(segments) == 0 {
		t.runnable = route
	} else {
		seg := segments[0]
		var subTree *Tree
		for _, sub := range t.routers {
			if sub.prefix == seg {
				subTree = sub
				break
			}
		}
		if subTree == nil {
			subTree = NewTree()
			subTree.prefix = seg
			t.routers = append(t.routers, subTree)
		}
		subTree.addseg(segments[1:], route)
	}
}

func (t *Tree) Match(pattern string) (runnable interface{}) {
	if len(pattern) == 0 || pattern[0] != '/' {
		return nil
	}
	return t.match(pattern[1:], pattern)
}

func (t *Tree) match(treePattern string, pattern string) (runnable interface{}) {
	if len(pattern) > 0 {
		i := 0
		for ; i < len(pattern) && pattern[i] == '/'; i++ {
		}
		pattern = pattern[i:]
	}
	var seg string
	i, l := 0, len(pattern)
	for ; i < l && pattern[i] != '/'; i++ {
	}
	if i == 0 {
		return t.runnable
	} else {
		seg = pattern[:i]
		pattern = pattern[i:]
	}
	for _, subTree := range t.routers {
		if subTree.prefix == seg {
			if len(pattern) != 0 && pattern[0] == '/' {
				treePattern = pattern[1:]
			} else {
				treePattern = pattern
			}
			runnable = subTree.match(treePattern, pattern)
			if runnable != nil {
				break
			}
		}
	}
	return runnable
}

func splitPath(key string) []string {
	key = strings.Trim(key, "/ ")
	if key == "" {
		return []string{}
	}
	return strings.Split(key, "/")
}