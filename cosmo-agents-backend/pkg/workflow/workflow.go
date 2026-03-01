package workflow

import (
	"context"
	"fmt"

	"github.com/rockship/cosmo-agents-go/pkg/workflow/event"
	"github.com/rockship/cosmo-agents-go/pkg/workflow/node"
)

const (
	// START is the workflow entry point marker.
	START = "__start__"
	// END is the workflow exit point marker.
	END = "__end__"
)

// Workflow represents a workflow with nodes and edges.
type Workflow struct {
	nodes        map[string]*node.NodeSpec
	edges        []Edge
	compiled     bool
	eventHandler event.Handler
	dag          *DAG
}

// NewWorkflow creates a new workflow.
func NewWorkflow(eventHandler event.Handler) *Workflow {
	if eventHandler == nil {
		eventHandler = event.NewMemoryHandler()
	}

	return &Workflow{
		nodes:        make(map[string]*node.NodeSpec),
		edges:        make([]Edge, 0),
		compiled:     false,
		eventHandler: eventHandler,
	}
}

// AddNode adds a node to the workflow.
func (w *Workflow) AddNode(name string, n node.Node, metadata *node.NodeMetadata) error {
	if _, exists := w.nodes[name]; exists {
		return fmt.Errorf("node '%s' already present", name)
	}

	w.nodes[name] = node.NewNodeSpec(n, metadata)
	return nil
}

// AddEdge adds an edge between two nodes.
func (w *Workflow) AddEdge(from, to string) error {
	w.edges = append(w.edges, Edge{From: from, To: to})
	return nil
}

// SetEntryPoint sets the workflow entry point.
func (w *Workflow) SetEntryPoint(nodeName string) error {
	return w.AddEdge(START, nodeName)
}

// SetFinishPoint sets the workflow exit point.
func (w *Workflow) SetFinishPoint(nodeName string) error {
	return w.AddEdge(nodeName, END)
}

// Compile validates the workflow structure and prepares it for execution.
func (w *Workflow) Compile(ctx context.Context) (*DAGState, error) {
	// Validate edges
	for _, edge := range w.edges {
		if edge.From != START {
			if _, exists := w.nodes[edge.From]; !exists {
				return nil, fmt.Errorf("source node '%s' not found", edge.From)
			}
		}

		if edge.To != END {
			if _, exists := w.nodes[edge.To]; !exists {
				return nil, fmt.Errorf("destination node '%s' not found", edge.To)
			}
		}
	}

	// Validate all nodes
	for name, spec := range w.nodes {
		if err := spec.Node.Validate(); err != nil {
			return nil, fmt.Errorf("node '%s' validation failed: %w", name, err)
		}
	}

	// Build node list (exclude START and END markers)
	nodeNames := make([]string, 0, len(w.nodes))
	for name := range w.nodes {
		nodeNames = append(nodeNames, name)
	}

	// Filter edges (remove START and END markers)
	filteredEdges := make([]Edge, 0)
	for _, edge := range w.edges {
		if edge.From != START && edge.To != END {
			filteredEdges = append(filteredEdges, edge)
		}
	}

	// Create and compile DAG
	w.dag = NewDAG(w.eventHandler, nodeNames, filteredEdges)
	state, err := w.dag.Compile(ctx)
	if err != nil {
		return nil, err
	}

	w.compiled = true
	return state, nil
}

// Invoke executes the workflow.
func (w *Workflow) Invoke(ctx context.Context) error {
	if !w.compiled {
		return fmt.Errorf("workflow not compiled - call Compile() first")
	}

	// Build node map
	nodes := make(map[string]node.Node)
	for name, spec := range w.nodes {
		nodes[name] = spec.Node
	}

	// Execute DAG
	return w.dag.Execute(ctx, nodes)
}

// InvokeNode runs a single node by name.
func (w *Workflow) InvokeNode(ctx context.Context, nodeName string, input interface{}) (interface{}, error) {
	spec, exists := w.nodes[nodeName]
	if !exists {
		return nil, fmt.Errorf("node '%s' not found", nodeName)
	}

	return spec.Node.Invoke(ctx, input)
}

// GetNodes returns all nodes in the workflow.
func (w *Workflow) GetNodes() map[string]*node.NodeSpec {
	return w.nodes
}

// GetEdges returns all edges in the workflow.
func (w *Workflow) GetEdges() []Edge {
	return w.edges
}

// IsCompiled returns whether the workflow has been compiled.
func (w *Workflow) IsCompiled() bool {
	return w.compiled
}

// GetState returns the current workflow state.
func (w *Workflow) GetState() *DAGState {
	if w.dag != nil {
		return w.dag.GetState()
	}
	return nil
}
