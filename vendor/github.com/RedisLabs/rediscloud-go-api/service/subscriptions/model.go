package subscriptions

import (
	"github.com/RedisLabs/rediscloud-go-api/internal"
)

type CreateSubscription struct {
	Name                        *string                `json:"name,omitempty"`
	DryRun                      *bool                  `json:"dryRun,omitempty"`
	PaymentMethodID             *int                   `json:"paymentMethodId,omitempty"`
	MemoryStorage               *string                `json:"memoryStorage,omitempty"`
	PersistentStorageEncryption *bool                  `json:"persistentStorageEncryption,omitempty"`
	CloudProviders              []*CreateCloudProvider `json:"cloudProviders,omitempty"`
	Databases                   []*CreateDatabase      `json:"databases,omitempty"`
}

func (o CreateSubscription) String() string {
	return internal.ToString(o)
}

type CreateCloudProvider struct {
	Provider       *string         `json:"provider,omitempty"`
	CloudAccountID *int            `json:"cloudAccountId,omitempty"`
	Regions        []*CreateRegion `json:"regions,omitempty"`
}

func (o CreateCloudProvider) String() string {
	return internal.ToString(o)
}

type CreateRegion struct {
	Region                     *string           `json:"region,omitempty"`
	MultipleAvailabilityZones  *bool             `json:"multipleAvailabilityZones,omitempty"`
	PreferredAvailabilityZones []*string         `json:"preferredAvailabilityZones,omitempty"`
	Networking                 *CreateNetworking `json:"networking,omitempty"`
}

func (o CreateRegion) String() string {
	return internal.ToString(o)
}

type CreateNetworking struct {
	DeploymentCIDR *string `json:"deploymentCIDR,omitempty"`
	VPCId          *string `json:"vpcId,omitempty"`
}

func (o CreateNetworking) String() string {
	return internal.ToString(o)
}

type CreateDatabase struct {
	Name                   *string           `json:"name,omitempty"`
	Protocol               *string           `json:"protocol,omitempty"`
	MemoryLimitInGB        *float64          `json:"memoryLimitInGb,omitempty"`
	SupportOSSClusterAPI   *bool             `json:"supportOSSClusterApi,omitempty"`
	DataPersistence        *string           `json:"dataPersistence,omitempty"`
	Replication            *bool             `json:"replication,omitempty"`
	ThroughputMeasurement  *CreateThroughput `json:"throughputMeasurement,omitempty"`
	Modules                []*CreateModules  `json:"modules,omitempty"`
	Quantity               *int              `json:"quantity,omitempty"`
	AverageItemSizeInBytes *int              `json:"averageItemSizeInBytes,omitempty"`
}

func (o CreateDatabase) String() string {
	return internal.ToString(o)
}

type CreateThroughput struct {
	By    *string `json:"by,omitempty"`
	Value *int    `json:"value,omitempty"`
}

func (o CreateThroughput) String() string {
	return internal.ToString(o)
}

type CreateModules struct {
	Name       *string            `json:"name,omitempty"`
	Parameters map[string]*string `json:"parameters,omitempty"`
}

func (o CreateModules) String() string {
	return internal.ToString(o)
}

type UpdateSubscription struct {
	Name            *string `json:"name,omitempty"`
	PaymentMethodID *int    `json:"paymentMethodId,omitempty"`
}

func (o UpdateSubscription) String() string {
	return internal.ToString(o)
}

type Subscription struct {
	ID                *int           `json:"id,omitempty"`
	Name              *string        `json:"name,omitempty"`
	Status            *string        `json:"status,omitempty"`
	PaymentMethodID   *int           `json:"paymentMethodId,omitempty"`
	MemoryStorage     *string        `json:"memoryStorage,omitempty"`
	StorageEncryption *bool          `json:"storageEncryption,omitempty"`
	NumberOfDatabases *int           `json:"numberOfDatabases"`
	CloudDetails      []*CloudDetail `json:"cloudDetails,omitempty"`
}

func (o Subscription) String() string {
	return internal.ToString(o)
}

type CloudDetail struct {
	Provider       *string   `json:"provider,omitempty"`
	CloudAccountID *int      `json:"cloudAccountId,omitempty"`
	TotalSizeInGB  *float64  `json:"totalSizeInGb,omitempty"`
	Regions        []*Region `json:"regions,omitempty"`
}

func (o CloudDetail) String() string {
	return internal.ToString(o)
}

type Region struct {
	Region                     *string       `json:"region,omitempty"`
	Networking                 []*Networking `json:"networking,omitempty"`
	PreferredAvailabilityZones []*string     `json:"preferredAvailabilityZones,omitempty"`
	MultipleAvailabilityZones  *bool         `json:"multipleAvailabilityZones,omitempty"`
}

func (o Region) String() string {
	return internal.ToString(o)
}

type Networking struct {
	DeploymentCIDR *string `json:"deploymentCIDR,omitempty"`
	VPCId          *string `json:"vpcId,omitempty"`
	SubnetID       *string `json:"subnetId,omitempty"`
}

func (o Networking) String() string {
	return internal.ToString(o)
}

type CIDRAllowlist struct {
	CIDRIPs          []*string   `json:"cidr_ips,omitempty"`
	SecurityGroupIDs []*string   `json:"security_group_ids,omitempty"`
	Errors           interface{} `json:"errors,omitempty"` // TODO the structure of this is undocumented
}

func (o CIDRAllowlist) String() string {
	return internal.ToString(o)
}

type UpdateCIDRAllowlist struct {
	CIDRIPs          []*string `json:"cidrIps,omitempty"`
	SecurityGroupIDs []*string `json:"securityGroupIds,omitempty"`
}

func (o UpdateCIDRAllowlist) String() string {
	return internal.ToString(o)
}

type CreateVPCPeering struct {
	Region       *string `json:"region,omitempty"`
	AWSAccountID *string `json:"awsAccountId,omitempty"`
	VPCId        *string `json:"vpcId,omitempty"`
	VPCCidr      *string `json:"vpcCidr,omitempty"`
}

func (o CreateVPCPeering) String() string {
	return internal.ToString(o)
}

type VPCPeering struct {
	ID     *int    `json:"id,omitempty"`
	Status *string `json:"status,omitempty"`
}

func (o VPCPeering) String() string {
	return internal.ToString(o)
}

type listSubscriptionResponse struct {
	Subscriptions []*Subscription `json:"subscriptions"`
}

type taskResponse struct {
	ID *string `json:"taskId,omitempty"`
}

func (o taskResponse) String() string {
	return internal.ToString(o)
}

const (
	// Active value of the `Status` field in `Subscription`
	SubscriptionStatusActive = "active"
	// Pending value of the `Status` field in `Subscription`
	SubscriptionStatusPending = "pending"
	// Error value of the `Status` field in `Subscription`
	SubscriptionStatusError = "error"
	// Deleting value of the `Status` field in `Subscription`
	SubscriptionStatusDeleting = "deleting"

	// Active value of the `Status` field in `VPCPeering`
	VPCPeeringStatusActive = "active"
	// Inactive value of the `Status` field in `VPCPeering`
	VPCPeeringStatusInactive = "inactive"
	// Pending acceptance value of the `Status` field in `VPCPeering`
	VPCPeeringStatusPendingAcceptance = "pending-acceptance"
	// Failed value of the `Status` field in `VPCPeering`
	VPCPeeringStatusFailed = "failed"
)
