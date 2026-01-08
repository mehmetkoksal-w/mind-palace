package analysis

import (
	"encoding/json"
	"fmt"
	"sync"
)

// MockLSPClient is a mock implementation of LSP client for testing
type MockLSPClient struct {
	responses map[int64]chan json.RawMessage
	ready     bool
	closed    bool

	// Mock data
	DocumentSymbolsFunc func(uri string, content string) ([]LSPDocumentSymbol, error)
	InitializeFunc      func() error
	ShutdownFunc        func() error
	CloseFunc           func() error
}

// NewMockLSPClient creates a new mock LSP client with default behavior
func NewMockLSPClient() *MockLSPClient {
	mock := &MockLSPClient{
		responses: make(map[int64]chan json.RawMessage),
		ready:     true,
	}

	// Default implementation returns empty symbols
	mock.DocumentSymbolsFunc = func(_ string, _ string) ([]LSPDocumentSymbol, error) {
		return []LSPDocumentSymbol{}, nil
	}

	mock.InitializeFunc = func() error {
		return nil
	}

	mock.ShutdownFunc = func() error {
		return nil
	}

	mock.CloseFunc = func() error {
		mock.closed = true
		return nil
	}

	return mock
}

// DocumentSymbols returns document symbols (delegates to mock function)
func (m *MockLSPClient) DocumentSymbols(uri, content string) ([]LSPDocumentSymbol, error) {
	if m.closed {
		return nil, fmt.Errorf("client closed")
	}
	return m.DocumentSymbolsFunc(uri, content)
}

// Initialize initializes the LSP connection (delegates to mock function)
func (m *MockLSPClient) Initialize() error {
	return m.InitializeFunc()
}

// Shutdown shuts down the LSP server (delegates to mock function)
func (m *MockLSPClient) Shutdown() error {
	return m.ShutdownFunc()
}

// Close closes the LSP client (delegates to mock function)
func (m *MockLSPClient) Close() error {
	return m.CloseFunc()
}

// MockLSPClientFactory creates mock LSP clients for testing
type MockLSPClientFactory struct {
	mu      sync.Mutex
	clients map[string]*MockLSPClient
}

// NewMockLSPClientFactory creates a new mock client factory
func NewMockLSPClientFactory() *MockLSPClientFactory {
	return &MockLSPClientFactory{
		clients: make(map[string]*MockLSPClient),
	}
}

// CreateClient creates a mock client for a specific configuration
func (f *MockLSPClientFactory) CreateClient(config LSPClientConfig) (*MockLSPClient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := config.ServerCmd + ":" + config.RootPath
	if client, exists := f.clients[key]; exists {
		return client, nil
	}

	client := NewMockLSPClient()
	f.clients[key] = client
	return client, nil
}

// GetClient retrieves a mock client by key
func (f *MockLSPClientFactory) GetClient(serverCmd, rootPath string) *MockLSPClient {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := serverCmd + ":" + rootPath
	return f.clients[key]
}

// WithDocumentSymbols configures the mock to return specific symbols
func (m *MockLSPClient) WithDocumentSymbols(symbols []LSPDocumentSymbol) *MockLSPClient {
	m.DocumentSymbolsFunc = func(_ string, _ string) ([]LSPDocumentSymbol, error) {
		return symbols, nil
	}
	return m
}

// WithError configures the mock to return an error
func (m *MockLSPClient) WithError(err error) *MockLSPClient {
	m.DocumentSymbolsFunc = func(_ string, _ string) ([]LSPDocumentSymbol, error) {
		return nil, err
	}
	return m
}

// WithCustomHandler configures the mock with a custom handler function
func (m *MockLSPClient) WithCustomHandler(handler func(uri string, content string) ([]LSPDocumentSymbol, error)) *MockLSPClient {
	m.DocumentSymbolsFunc = handler
	return m
}

// CreateTestSymbols creates a set of test symbols for mocking
func CreateTestSymbols() []LSPDocumentSymbol {
	return []LSPDocumentSymbol{
		{
			Name: "Person",
			Kind: LSPSymbolKindClass,
			Range: LSPRange{
				Start: LSPPosition{Line: 7, Character: 0},
				End:   LSPPosition{Line: 10, Character: 1},
			},
			SelectionRange: LSPRange{
				Start: LSPPosition{Line: 7, Character: 5},
				End:   LSPPosition{Line: 7, Character: 11},
			},
			Children: []LSPDocumentSymbol{
				{
					Name: "Name",
					Kind: LSPSymbolKindField,
					Range: LSPRange{
						Start: LSPPosition{Line: 8, Character: 1},
						End:   LSPPosition{Line: 8, Character: 13},
					},
					SelectionRange: LSPRange{
						Start: LSPPosition{Line: 8, Character: 1},
						End:   LSPPosition{Line: 8, Character: 5},
					},
				},
				{
					Name: "Age",
					Kind: LSPSymbolKindField,
					Range: LSPRange{
						Start: LSPPosition{Line: 9, Character: 1},
						End:   LSPPosition{Line: 9, Character: 8},
					},
					SelectionRange: LSPRange{
						Start: LSPPosition{Line: 9, Character: 1},
						End:   LSPPosition{Line: 9, Character: 4},
					},
				},
			},
		},
		{
			Name: "NewPerson",
			Kind: LSPSymbolKindFunction,
			Range: LSPRange{
				Start: LSPPosition{Line: 13, Character: 0},
				End:   LSPPosition{Line: 18, Character: 1},
			},
			SelectionRange: LSPRange{
				Start: LSPPosition{Line: 13, Character: 5},
				End:   LSPPosition{Line: 13, Character: 14},
			},
		},
		{
			Name: "Greet",
			Kind: LSPSymbolKindMethod,
			Range: LSPRange{
				Start: LSPPosition{Line: 21, Character: 0},
				End:   LSPPosition{Line: 23, Character: 1},
			},
			SelectionRange: LSPRange{
				Start: LSPPosition{Line: 21, Character: 16},
				End:   LSPPosition{Line: 21, Character: 21},
			},
			Detail: "(p *Person)",
		},
		{
			Name: "IsAdult",
			Kind: LSPSymbolKindMethod,
			Range: LSPRange{
				Start: LSPPosition{Line: 26, Character: 0},
				End:   LSPPosition{Line: 28, Character: 1},
			},
			SelectionRange: LSPRange{
				Start: LSPPosition{Line: 26, Character: 16},
				End:   LSPPosition{Line: 26, Character: 23},
			},
			Detail: "(p *Person)",
		},
		{
			Name: "MaxAge",
			Kind: LSPSymbolKindConstant,
			Range: LSPRange{
				Start: LSPPosition{Line: 30, Character: 0},
				End:   LSPPosition{Line: 30, Character: 18},
			},
			SelectionRange: LSPRange{
				Start: LSPPosition{Line: 30, Character: 6},
				End:   LSPPosition{Line: 30, Character: 12},
			},
		},
	}
}

// LSPClientInterface defines the interface that both real and mock clients implement
type LSPClientInterface interface {
	DocumentSymbols(uri string, content string) ([]LSPDocumentSymbol, error)
	Initialize() error
	Shutdown() error
	Close() error
}

// Ensure MockLSPClient implements LSPClientInterface
var _ LSPClientInterface = (*MockLSPClient)(nil)

// Ensure LSPClient implements LSPClientInterface (if we add the methods)
// var _ LSPClientInterface = (*LSPClient)(nil)
