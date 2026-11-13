package provider

import (
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/stretchr/testify/require"
)

func TestTransitGatewayAttachmentAvailableMatcher(t *testing.T) {
	tests := []struct {
		name       string
		attachment *attachments.TransitGatewayAttachment
		wantMatch  bool
		wantError  string
	}{
		{
			name:       "unrelated failed attachment does not match and does not stop the wait",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(790), Status: redis.String("failed")},
			wantMatch:  false,
			wantError:  "",
		},
		{
			name:       "available target attachment matches",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("available")},
			wantMatch:  true,
			wantError:  "",
		},
		{
			name:       "provisioning target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("provisioning")},
			wantMatch:  false,
			wantError:  "",
		},
		{
			name:       "initiating-request target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("initiating-request")},
			wantMatch:  false,
			wantError:  "",
		},
		{
			name:       "pending target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("pending")},
			wantMatch:  false,
			wantError:  "",
		},
		{
			name:       "pendingAcceptance target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("pendingAcceptance")},
			wantMatch:  false,
			wantError:  "",
		},
		{
			name:       "modifying target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("modifying")},
			wantMatch:  false,
			wantError:  "",
		},
		{
			name:       "deleting target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("deleting")},
			wantMatch:  false,
			wantError:  "",
		},
		{
			name:       "rejecting target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("rejecting")},
			wantMatch:  false,
			wantError:  "",
		},
		{
			name:       "failing target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("failing")},
			wantMatch:  false,
			wantError:  "",
		},
		{
			name:       "rollingBack target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("rollingBack")},
			wantMatch:  false,
			wantError:  "",
		},
		{
			name:       "failed target attachment stops the wait",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("failed")},
			wantMatch:  false,
			wantError:  "Transit Gateway attachment reached terminal state: failed",
		},
		{
			name:       "rejected target attachment stops the wait",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("rejected")},
			wantMatch:  false,
			wantError:  "Transit Gateway attachment reached terminal state: rejected",
		},
		{
			name:       "deleted target attachment stops the wait",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("deleted")},
			wantMatch:  false,
			wantError:  "Transit Gateway attachment reached terminal state: deleted",
		},
		{
			name:       "unknown target attachment status stops the wait",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("unknown")},
			wantMatch:  false,
			wantError:  "Transit Gateway attachment has unexpected state: unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matcher := transitGatewayAttachmentAvailableMatcher(789, "Transit Gateway")

			matched, err := matcher(tc.attachment)

			require.Equal(t, tc.wantMatch, matched)
			if tc.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.wantError)
		})
	}
}
