package provider_test

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/psc"
)

type privateServiceConnectEndpointAccepterTestId struct {
	subscriptionId int
	pscServiceId   int
	endpointId     int
}

func toPscEndpointAccepterId(id string) (*privateServiceConnectEndpointAccepterTestId, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid id: %s", id)
	}

	subId, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	pscId, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	endpointId, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, err
	}

	return &privateServiceConnectEndpointAccepterTestId{
		subscriptionId: subId,
		pscServiceId:   pscId,
		endpointId:     endpointId,
	}, nil
}

type privateServiceConnectActiveActiveEndpointAccepterTestId struct {
	subscriptionId int
	regionId       int
	pscServiceId   int
	endpointId     int
}

func toPscEndpointActiveActiveAccepterId(id string) (*privateServiceConnectActiveActiveEndpointAccepterTestId, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid id: %s", id)
	}

	subId, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	regionId, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	pscId, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, err
	}

	endpointId, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, err
	}

	return &privateServiceConnectActiveActiveEndpointAccepterTestId{
		subscriptionId: subId,
		regionId:       regionId,
		pscServiceId:   pscId,
		endpointId:     endpointId,
	}, nil
}

func findPrivateServiceConnectEndpoints(id int, endpoints []*psc.PrivateServiceConnectEndpoint) *psc.PrivateServiceConnectEndpoint {
	for _, endpoint := range endpoints {
		if redis.IntValue(endpoint.ID) == id {
			return endpoint
		}
	}
	return nil
}
