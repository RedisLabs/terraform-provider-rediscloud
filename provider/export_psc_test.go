package provider

// This file is a test-only bridge that re-exports unexported helpers and adds
// exported accessor methods on private ID types so the external test package
// (provider_test) can exercise them without polluting the public API. Because
// the file has the _test.go suffix it is compiled only by `go test` and never
// shipped in the production binary; because it declares `package provider` it
// has access to the package's unexported identifiers.

var (
	ToPscEndpointAccepterId             = toPscEndpointAccepterId
	ToPscEndpointActiveActiveAccepterId = toPscEndpointActiveActiveAccepterId
	FindPrivateServiceConnectEndpoints  = findPrivateServiceConnectEndpoints
)

func (p *privateServiceConnectEndpointAccepterId) SubscriptionId() int { return p.subscriptionId }
func (p *privateServiceConnectEndpointAccepterId) PscServiceId() int   { return p.pscServiceId }
func (p *privateServiceConnectEndpointAccepterId) EndpointId() int     { return p.endpointId }

func (p *privateServiceConnectActiveActiveEndpointAccepterId) SubscriptionId() int {
	return p.subscriptionId
}
func (p *privateServiceConnectActiveActiveEndpointAccepterId) RegionId() int { return p.regionId }
func (p *privateServiceConnectActiveActiveEndpointAccepterId) PscServiceId() int {
	return p.pscServiceId
}
func (p *privateServiceConnectActiveActiveEndpointAccepterId) EndpointId() int { return p.endpointId }
