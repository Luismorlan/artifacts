package deduplicator

import (
	"context"

	"github.com/rnr-capital/newsfeed-backend/protocol"
	"google.golang.org/grpc"
)

type FakeDeduplicatorClient struct {
	protocol.DeduplicatorClient
}

func (FakeDeduplicatorClient) GetSimHash(ctx context.Context, in *protocol.GetSimHashRequest, opts ...grpc.CallOption) (*protocol.GetSimHashResponse, error) {
	return &protocol.GetSimHashResponse{
		// length: 128 bits
		Binary: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	}, nil
}
