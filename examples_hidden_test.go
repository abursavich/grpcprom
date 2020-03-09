package grpcprom_test

import (
	"context"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bepb "github.com/abursavich/grpcprom/testdata/backend"
	fepb "github.com/abursavich/grpcprom/testdata/frontend"
)

var httpAddr, grpcAddr, backendAddr string

type FrontendServer struct {
	BackendClient bepb.BackendClient
}

func (*FrontendServer) Query(context.Context, *fepb.QueryRequest) (*fepb.QueryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Query not implemented")
}

type BackendServer struct{}

func (*BackendServer) Query(context.Context, *bepb.QueryRequest) (*bepb.QueryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Query not implemented")
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
