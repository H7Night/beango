package utils

import (
	"os"
	"path/filepath"
)

type FileNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"` // 用于前端点击时获取完整路径
	Type     string     `json:"type"` // "file" 或 "folder"
	Children []FileNode `json:"children,omitempty"`
}

func BuildFileTree(root string) ([]FileNode, error) {
	var walk func(string) (FileNode, error)
	walk = func(path string) (FileNode, error) {
		info, err := os.Stat(path)
		if err != nil {
			return FileNode{}, err
		}
		node := FileNode{
			Name: info.Name(),
			Path: path,
		}
		if info.IsDir() {
			node.Type = "folder"
			entries, err := os.ReadDir(path)
			if err != nil {
				return node, err
			}
			for _, e := range entries {
				child, err := walk(filepath.Join(path, e.Name()))
				if err == nil {
					node.Children = append(node.Children, child)
				}
			}
		} else {
			node.Type = "file"
		}
		return node, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var nodes []FileNode
	for _, e := range entries {
		child, err := walk(filepath.Join(root, e.Name()))
		if err == nil {
			nodes = append(nodes, child)
		}
	}
	return nodes, nil
}
