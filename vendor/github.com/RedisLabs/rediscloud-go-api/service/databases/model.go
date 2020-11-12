package databases

import (
	"time"

	"github.com/RedisLabs/rediscloud-go-api/internal"
)

type taskResponse struct {
	ID *string `json:"taskId,omitempty"`
}

func (o taskResponse) String() string {
	return internal.ToString(o)
}

type CreateDatabase struct {
	DryRun                              *bool                        `json:"dryRun,omitempty"`
	Name                                *string                      `json:"name,omitempty"`
	Protocol                            *string                      `json:"protocol,omitempty"`
	MemoryLimitInGB                     *float64                     `json:"memoryLimitInGb,omitempty"`
	SupportOSSClusterAPI                *bool                        `json:"supportOSSClusterApi,omitempty"`
	UseExternalEndpointForOSSClusterAPI *bool                        `json:"useExternalEndpointForOSSClusterApi,omitempty"`
	DataPersistence                     *string                      `json:"dataPersistence,omitempty"`
	DataEvictionPolicy                  *string                      `json:"dataEvictionPolicy,omitempty"`
	Replication                         *bool                        `json:"replication,omitempty"`
	ThroughputMeasurement               *CreateThroughputMeasurement `json:"throughputMeasurement,omitempty"`
	AverageItemSizeInBytes              *int                         `json:"averageItemSizeInBytes,omitempty"`
	ReplicaOf                           []*string                    `json:"replicaOf,omitempty"`
	PeriodicBackupPath                  *string                      `json:"periodicBackupPath,omitempty"`
	SourceIP                            []*string                    `json:"sourceIp,omitempty"`
	ClientSSLCertificate                *string                      `json:"clientSslCertificate,omitempty"`
	Password                            *string                      `json:"password,omitempty"`
	Alerts                              []*CreateAlert               `json:"alerts,omitempty"`
	Modules                             []*CreateModule              `json:"modules,omitempty"`
}

func (o CreateDatabase) String() string {
	return internal.ToString(o)
}

type CreateThroughputMeasurement struct {
	By    *string `json:"by,omitempty"`
	Value *int    `json:"value,omitempty"`
}

func (o CreateThroughputMeasurement) String() string {
	return internal.ToString(o)
}

type CreateAlert struct {
	Name  *string `json:"name,omitempty"`
	Value *int    `json:"value,omitempty"`
}

func (o CreateAlert) String() string {
	return internal.ToString(o)
}

type CreateModule struct {
	Name       *string            `json:"name,omitempty"`
}

func (o CreateModule) String() string {
	return internal.ToString(o)
}

type Database struct {
	ID                     *int        `json:"databaseId,omitempty"`
	Name                   *string     `json:"name,omitempty"`
	Protocol               *string     `json:"protocol,omitempty"`
	Provider               *string     `json:"provider,omitempty"`
	Region                 *string     `json:"region,omitempty"`
	Status                 *string     `json:"status,omitempty"`
	MemoryLimitInGB        *float64    `json:"memoryLimitInGb,omitempty"`
	MemoryUsedInMB         *float64    `json:"memoryUsedInMb,omitempty"`
	SupportOSSClusterAPI   *bool       `json:"supportOSSClusterApi,omitempty"`
	DataPersistence        *string     `json:"dataPersistence,omitempty"`
	Replication            *bool       `json:"replication,omitempty"`
	DataEvictionPolicy     *string     `json:"dataEvictionPolicy,omitempty"`
	ThroughputMeasurement  *Throughput `json:"throughputMeasurement,omitempty"`
	ReplicaOf              []*string   `json:"replicaOf,omitempty"`
	Clustering             *Clustering `json:"clustering,omitempty"`
	Security               *Security   `json:"security,omitempty"`
	Modules                []*Module   `json:"modules,omitempty"`
	Alerts                 []*Alert    `json:"alerts,omitempty"`
	ActivatedOn            *time.Time  `json:"activatedOn,omitempty"`
	LastModified           *time.Time  `json:"lastModified,omitempty"`
	MemoryStorage          *string     `json:"memoryStorage,omitempty"`
	PrivateEndpoint        *string     `json:"privateEndpoint,omitempty"`
	PublicEndpoint         *string     `json:"publicEndpoint,omitempty"`
	RedisVersionCompliance *string     `json:"redisVersionCompliance,omitempty"`
}

func (o Database) String() string {
	return internal.ToString(o)
}

type Clustering struct {
	NumberOfShards *int `json:"numberOfShards,omitempty"`
	// TODO RegexRules interface{} `json:"regexRules,omitempty"`
	// TODO HashingPolicy interface{} `json:"hashingPolicy,omitempty"`
}

func (o Clustering) String() string {
	return internal.ToString(o)
}

type Security struct {
	SSLClientAuthentication *bool     `json:"sslClientAuthentication,omitempty"`
	SourceIPs               []*string `json:"sourceIps,omitempty"`
	Password                *string   `json:"password,omitempty"`
}

func (o Security) String() string {
	return internal.ToString(o)
}

type Module struct {
	Name *string `json:"name,omitempty"`
}

func (o Module) String() string {
	return internal.ToString(o)
}

type Throughput struct {
	By    *string `json:"by,omitempty"`
	Value *int    `json:"value,omitempty"`
}

func (o Throughput) String() string {
	return internal.ToString(o)
}

type Alert struct {
	Name  *string `json:"name,omitempty"`
	Value *int    `json:"value,omitempty"`
}

func (o Alert) String() string {
	return internal.ToString(o)
}

type UpdateDatabase struct {
	DryRun                              *bool                        `json:"dryRun,omitempty"`
	Name                                *string                      `json:"name,omitempty"`
	MemoryLimitInGB                     *float64                     `json:"memoryLimitInGb,omitempty"`
	SupportOSSClusterAPI                *bool                        `json:"supportOSSClusterApi,omitempty"`
	UseExternalEndpointForOSSClusterAPI *bool                        `json:"useExternalEndpointForOSSClusterApi,omitempty"`
	DataEvictionPolicy                  *string                      `json:"dataEvictionPolicy,omitempty"`
	Replication                         *bool                        `json:"replication,omitempty"`
	ThroughputMeasurement               *UpdateThroughputMeasurement `json:"throughputMeasurement,omitempty"`
	RegexRules                          []*string                    `json:"regexRules,omitempty"`
	DataPersistence                     *string                      `json:"dataPersistence,omitempty"`
	ReplicaOf                           []*string                    `json:"replicaOf,omitempty"`
	PeriodicBackupPath                  *string                      `json:"periodicBackupPath,omitempty"`
	SourceIP                            []*string                    `json:"sourceIp,omitempty"`
	ClientSSLCertificate                *string                      `json:"clientSslCertificate,omitempty"`
	Password                            *string                      `json:"password,omitempty"`
	Alerts                              []*UpdateAlert               `json:"alerts,omitempty"`
}

func (o UpdateDatabase) String() string {
	return internal.ToString(o)
}

type UpdateThroughputMeasurement struct {
	By    *string `json:"by,omitempty"`
	Value *int    `json:"value,omitempty"`
}

func (o UpdateThroughputMeasurement) String() string {
	return internal.ToString(o)
}

type UpdateAlert struct {
	Name  *string `json:"name,omitempty"`
	Value *int    `json:"value,omitempty"`
}

func (o UpdateAlert) String() string {
	return internal.ToString(o)
}

type Import struct {
	SourceType    *string   `json:"sourceType,omitempty"`
	ImportFromURI []*string `json:"importFromUri,omitempty"`
}

func (o Import) String() string {
	return internal.ToString(o)
}

type listDatabaseResponse struct {
	Subscription []*listDbSubscription `json:"subscription,omitempty"`
}

func (o listDatabaseResponse) String() string {
	return internal.ToString(o)
}

type listDbSubscription struct {
	ID        *int        `json:"subscriptionId,omitempty"`
	Databases []*Database `json:"databases,omitempty"`
}

func (o listDbSubscription) String() string {
	return internal.ToString(o)
}

const (
	// Active value of the `Status` field in `Database`
	StatusActive = "active"
	// Draft value of the `Status` field in `Database`
	StatusDraft = "draft"
	// Pending value of the `Status` field in `Database`
	StatusPending = "pending"
	// RCP active change draft value of the `Status` field in `Database`
	StatusRCPActiveChangeDraft = "rcp-active-change-draft"
	// Active change draft value of the `Status` field in `Database`
	StatusActiveChangeDraft = "active-change-draft"
	// Active change pending value of the `Status` field in `Database`
	StatusActiveChangePending = "active-change-pending"
	// Error value of the `Status` field in `Database`
	StatusError = "error"
)

func MemoryStorageValues() []string {
	return []string{
		"ram",
		"ram-and-flash",
	}
}

func ProtocolValues() []string {
	return []string{
		"redis",
		"memcached",
	}
}

func DataPersistenceValues() []string {
	return []string{
		"none",
		"aof-every-1-second",
		"aof-every-write",
		"snapshot-every-1-hour",
		"snapshot-every-6-hours",
		"snapshot-every-12-hours",
	}
}

func DataEvictionPolicyValues() []string {
	return []string{
		"allkeys-lru",
		"allkeys-lfu",
		"allkeys-random",
		"volatile-lru",
		"volatile-lfu",
		"volatile-random",
		"volatile-ttl",
		"noeviction",
	}
}

func SourceTypeValues() []string {
	return []string{
		"http",
		"redis",
		"ftp",
		"aws-s3",
		"azure-blob-storage",
		"google-blob-storage",
	}
}
