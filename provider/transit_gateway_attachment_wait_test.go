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
		wantState  string
		wantError  string
	}{
		{
			name:       "unrelated failed attachment does not match and does not stop the wait",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(790), Status: redis.String("failed")},
			wantMatch:  false,
			wantState:  "",
			wantError:  "",
		},
		{
			name:       "available target attachment matches",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("available")},
			wantMatch:  true,
			wantState:  "available",
			wantError:  "",
		},
		{
			name:       "provisioning target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("provisioning")},
			wantMatch:  false,
			wantState:  "provisioning",
			wantError:  "",
		},
		{
			name:       "initiating-request target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("initiating-request")},
			wantMatch:  false,
			wantState:  "initiating-request",
			wantError:  "",
		},
		{
			name:       "pending target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("pending")},
			wantMatch:  false,
			wantState:  "pending",
			wantError:  "",
		},
		{
			name:       "pendingAcceptance target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("pendingAcceptance")},
			wantMatch:  false,
			wantState:  "pendingAcceptance",
			wantError:  "",
		},
		{
			name:       "modifying target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("modifying")},
			wantMatch:  false,
			wantState:  "modifying",
			wantError:  "",
		},
		{
			name:       "deleting target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("deleting")},
			wantMatch:  false,
			wantState:  "deleting",
			wantError:  "",
		},
		{
			name:       "rejecting target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("rejecting")},
			wantMatch:  false,
			wantState:  "rejecting",
			wantError:  "",
		},
		{
			name:       "failing target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("failing")},
			wantMatch:  false,
			wantState:  "failing",
			wantError:  "",
		},
		{
			name:       "rollingBack target attachment keeps waiting",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("rollingBack")},
			wantMatch:  false,
			wantState:  "rollingBack",
			wantError:  "",
		},
		{
			name:       "failed target attachment stops the wait",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("failed")},
			wantMatch:  false,
			wantState:  "failed",
			wantError:  "Transit Gateway attachment reached terminal state: failed",
		},
		{
			name:       "rejected target attachment stops the wait",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("rejected")},
			wantMatch:  false,
			wantState:  "rejected",
			wantError:  "Transit Gateway attachment reached terminal state: rejected",
		},
		{
			name:       "deleted target attachment stops the wait",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("deleted")},
			wantMatch:  false,
			wantState:  "deleted",
			wantError:  "Transit Gateway attachment reached terminal state: deleted",
		},
		{
			name:       "unknown target attachment status stops the wait",
			attachment: &attachments.TransitGatewayAttachment{Id: redis.Int(789), Status: redis.String("unknown")},
			wantMatch:  false,
			wantState:  "unknown",
			wantError:  "Transit Gateway attachment has unexpected state: unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matcher := transitGatewayAttachmentAvailableMatcher(789, "Transit Gateway")

			matched, state, err := matcher(tc.attachment)

			require.Equal(t, tc.wantMatch, matched)
			require.Equal(t, tc.wantState, state)
			if tc.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.wantError)
		})
	}
}
