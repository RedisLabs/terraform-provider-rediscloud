package cloud_accounts

import (
	"fmt"

	"github.com/RedisLabs/rediscloud-go-api/internal"
)

type CreateCloudAccount struct {
	AccessKeyID     *string `json:"accessKeyId,omitempty"`
	AccessSecretKey *string `json:"accessSecretKey,omitempty"`
	ConsoleUsername *string `json:"consoleUsername,omitempty"`
	ConsolePassword *string `json:"consolePassword,omitempty"`
	Name            *string `json:"name,omitempty"`
	Provider        *string `json:"provider,omitempty"`
	SignInLoginURL  *string `json:"signInLoginUrl,omitempty"`
}

func (o CreateCloudAccount) String() string {
	return internal.ToString(o)
}

type UpdateCloudAccount struct {
	AccessKeyID     *string `json:"accessKeyId,omitempty"`
	AccessSecretKey *string `json:"accessSecretKey,omitempty"`
	ConsoleUsername *string `json:"consoleUsername,omitempty"`
	ConsolePassword *string `json:"consolePassword,omitempty"`
	Name            *string `json:"name,omitempty"`
	SignInLoginURL  *string `json:"signInLoginUrl,omitempty"`
}

func (o UpdateCloudAccount) String() string {
	return internal.ToString(o)
}

type NotFound struct {
	id int
}

func (f *NotFound) Error() string {
	return fmt.Sprintf("cloud account %d not found", f.id)
}

type listCloudAccounts struct {
	CloudAccounts []*CloudAccount `json:"cloudAccounts"`
}

type taskResponse struct {
	ID *string `json:"taskId,omitempty"`
}

func (o taskResponse) String() string {
	return internal.ToString(o)
}

type CloudAccount struct {
	ID          *int    `json:"id"`
	Name        *string `json:"name,omitempty"`
	Provider    *string `json:"provider,omitempty"`
	Status      *string `json:"status,omitempty"`
	AccessKeyID *string `json:"accessKeyId,omitempty"`
}

func (o CloudAccount) String() string {
	return internal.ToString(o)
}

const (
	// Active value of the `Status` field in `CloudAccount`
	StatusActive = "active"
	// Draft value of the `Status` field in `CloudAccount`
	StatusDraft = "draft"
	// Change draft value of the `Status` field in `CloudAccount`
	StatusChangeDraft = "change-draft"
	// Error value of the `Status` field in `CloudAccount`
	StatusError = "error"
)

func ProviderValues() []string {
	return []string{
		"AWS",
		"GCP",
	}
}
