package workflow

import (
	"context"
	"fmt"

	"github.com/rockship/cosmo-agents-go/pkg/workflow/event"
	"github.com/rockship/cosmo-agents-go/pkg/workflow/node"
)

// DAG represents a Directed Acyclic Graph for workflow execution.
type DAG struct {
	eventHandler event.Handler
	nodes        []string
	edges        []Edge
	state        *DAGState
}

// Edge represents a connection between two nodes.
type Edge struct {
	From string
	To   string
}

// DAGState holds the execution state of the DAG.
type DAGState struct {
	Order      []string
	AdjList    map[string][]string
	NodeStates map[string]*NodeState
}

// NodeState represents the execution state of a node.
type NodeState struct {
	OutputDeps map[string]interface{}
	Output     interface{}
}

// NewDAG creates a new DAG.
func NewDAG(eventHandler event.Handler, nodes []string, edges []Edge) *DAG {
	return &DAG{
		eventHandler: eventHandler,
		nodes:        nodes,
		edges:        edges,
	}
}

// Compile arranges the execution order based on nodes and edges.
func (d *DAG) Compile(ctx context.Context) (*DAGState, error) {
	// Initialize in-degree map and adjacency list
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)
	nodeStates := make(map[string]*NodeState)

	for _, nodeName := range d.nodes {
		inDegree[nodeName] = 0
		nodeStates[nodeName] = &NodeState{
			OutputDeps: make(map[string]interface{}),
			Output:     nil,
		}
	}

	// Build adjacency list and calculate in-degrees
	for _, edge := range d.edges {
		adjList[edge.From] = append(adjList[edge.From], edge.To)
		inDegree[edge.To]++
		nodeStates[edge.To].OutputDeps[edge.From] = nil
	}

	// Topological sort using Kahn's algorithm
	var order []string
	remainingNodes := make([]string, len(d.nodes))
	copy(remainingNodes, d.nodes)

	for len(remainingNodes) > 0 {
		found := false

		for i, nodeName := range remainingNodes {
			if inDegree[nodeName] == 0 {
				found = true
				order = append(order, nodeName)

				// Remove node from remaining
				remainingNodes = append(remainingNodes[:i], remainingNodes[i+1:]...)

				// Decrease in-degree of dependent nodes
				for _, dependent := range adjList[nodeName] {
					inDegree[dependent]--
				}

				break
			}
		}

		if !found {
			return nil, fmt.Errorf("cycle detected in workflow - cycles are not allowed")
		}
	}

	d.state = &DAGState{
		Order:      order,
		AdjList:    adjList,
		NodeStates: nodeStates,
	}

	return d.state, nil
}

// GetState returns the current DAG state.
func (d *DAG) GetState() *DAGState {
	return d.state
}

// Execute runs the DAG in topological order.
func (d *DAG) Execute(ctx context.Context, nodes map[string]node.Node) error {
	if d.state == nil {
		return fmt.Errorf("DAG not compiled - call Compile() first")
	}

	for _, nodeName := range d.state.Order {
		n, exists := nodes[nodeName]
		if !exists {
			return fmt.Errorf("node %s not found in nodes map", nodeName)
		}

		// Prepare input from dependencies
		nodeState := d.state.NodeStates[nodeName]
		var input interface{}

		if len(nodeState.OutputDeps) > 0 {
			// Use output from dependencies as input
			input = nodeState.OutputDeps
		}

		// Execute node
		output, err := n.Invoke(ctx, input)
		if err != nil {
			return fmt.Errorf("node %s execution failed: %w", nodeName, err)
		}

		// Store output
		nodeState.Output = output

		// Propagate output to dependent nodes
		for _, dependent := range d.state.AdjList[nodeName] {
			d.state.NodeStates[dependent].OutputDeps[nodeName] = output
		}
	}

	return nil
}
