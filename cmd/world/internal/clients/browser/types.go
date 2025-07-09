package browser

// ClientInterface defines the contract for browser operations.
type ClientInterface interface {
	OpenURL(url string) error
}

var _ ClientInterface = (*Client)(nil)

// Client implements browser operations.
type Client struct{}

// NewClient creates a new browser client.
func NewClient() ClientInterface {
	return &Client{}
}
