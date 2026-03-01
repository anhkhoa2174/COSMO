package node

import (
	"context"
	"fmt"
	"reflect"
)

// Node represents a workflow node that processes input and produces output.
type Node interface {
	// GetName returns the name of the node.
	GetName() string

	// Invoke executes the node with the given input.
	Invoke(ctx context.Context, input interface{}) (interface{}, error)

	// InputType returns the type of input this node accepts.
	InputType() reflect.Type

	// OutputType returns the type of output this node produces.
	OutputType() reflect.Type

	// Validate validates the node configuration.
	Validate() error
}

// BaseNode provides common functionality for all nodes.
type BaseNode struct {
	Name       string
	inputType  reflect.Type
	outputType reflect.Type
}

// NewBaseNode creates a new base node.
func NewBaseNode(name string, inputType, outputType reflect.Type) *BaseNode {
	return &BaseNode{
		Name:       name,
		inputType:  inputType,
		outputType: outputType,
	}
}

// GetName returns the name of the node.
func (n *BaseNode) GetName() string {
	if n.Name != "" {
		return n.Name
	}
	return "BaseNode"
}

// InputType returns the type of input this node accepts.
func (n *BaseNode) InputType() reflect.Type {
	return n.inputType
}

// OutputType returns the type of output this node produces.
func (n *BaseNode) OutputType() reflect.Type {
	return n.outputType
}

// Validate validates the node configuration.
func (n *BaseNode) Validate() error {
	if n.Name == "" {
		return fmt.Errorf("node name is required")
	}
	return nil
}

// FuncNode wraps a function as a workflow node.
type FuncNode struct {
	*BaseNode
	fn func(ctx context.Context, input interface{}) (interface{}, error)
}

// NewFuncNode creates a new function-based node.
func NewFuncNode(name string, fn func(ctx context.Context, input interface{}) (interface{}, error)) *FuncNode {
	return &FuncNode{
		BaseNode: NewBaseNode(name, nil, nil),
		fn:       fn,
	}
}

// Invoke executes the function node.
func (n *FuncNode) Invoke(ctx context.Context, input interface{}) (interface{}, error) {
	return n.fn(ctx, input)
}

// NodeMetadata contains metadata about a node.
type NodeMetadata struct {
	Description string
	Tags        []string
	Properties  map[string]interface{}
}

// NodeSpec represents a node with its metadata.
type NodeSpec struct {
	Node     Node
	Metadata *NodeMetadata
}

// NewNodeSpec creates a new node specification.
func NewNodeSpec(node Node, metadata *NodeMetadata) *NodeSpec {
	return &NodeSpec{
		Node:     node,
		Metadata: metadata,
	}
}
