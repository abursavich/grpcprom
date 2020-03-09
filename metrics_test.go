package grpcprom

import (
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "google.golang.org/grpc/test/grpc_testing"
)

func TestMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	serverMetrics := NewServerMetrics(
		LatencySeconds(NoBuckets()),
		RecvBytes(Disable()),
		SentBytes(Disable()),
	)
	check(t, registry.Register(serverMetrics))
	clientMetrics := NewClientMetrics(
		LatencySeconds(NoBuckets()),
		RecvBytes(Disable()),
		SentBytes(Disable()),
	)
	check(t, registry.Register(clientMetrics))

	httpListener, err := net.Listen("tcp", ":0")
	check(t, err)
	defer httpListener.Close()

	httpMux := http.NewServeMux()
	httpMux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	go http.Serve(httpListener, httpMux)

	grpcListener, err := net.Listen("tcp", ":0")
	check(t, err)
	defer grpcListener.Close()

	srv := grpc.NewServer(
		grpc.StatsHandler(serverMetrics.StatsHandler()),
		grpc.StreamInterceptor(serverMetrics.StreamInterceptor()),
		grpc.UnaryInterceptor(serverMetrics.UnaryInterceptor()),
	)
	pb.RegisterTestServiceServer(srv, &testServiceServer{})
	serverMetrics.Init(srv, codes.OK)

	conn, err := grpc.Dial(
		grpcListener.Addr().String(),
		grpc.WithStatsHandler(clientMetrics.StatsHandler()),
		grpc.WithStreamInterceptor(clientMetrics.StreamInterceptor()),
		grpc.WithUnaryInterceptor(clientMetrics.UnaryInterceptor()),
		grpc.WithInsecure(),
	)
	check(t, err)
	client := pb.NewTestServiceClient(conn)

	dummySrv := grpc.NewServer()
	pb.RegisterTestServiceServer(dummySrv, &pb.UnimplementedTestServiceServer{})
	clientMetrics.Init(dummySrv, codes.OK)

	go srv.Serve(grpcListener)

	ctx := context.Background()
	client.UnaryCall(ctx, &pb.SimpleRequest{Payload: genPayload(1024)})
	client.EmptyCall(ctx, &pb.Empty{}) // Unimplemented

	streamingOutput, err := client.StreamingOutputCall(ctx, &pb.StreamingOutputCallRequest{
		ResponseParameters: []*pb.ResponseParameters{
			{Size: 256},
			{Size: 512},
			{Size: 1024},
		},
		Payload: genPayload(2048),
	})
	check(t, err)
	for {
		if _, err := streamingOutput.Recv(); err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			break
		}
	}

	resp, err := http.Get("http://" + httpListener.Addr().String() + "/metrics")
	check(t, err)
	buf, err := ioutil.ReadAll(resp.Body)
	check(t, err)
	t.Logf("%s\n%s", resp.Status, buf)
}

func check(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Fatal(err)
	}
}

type testServiceServer struct {
	pb.UnimplementedTestServiceServer
}

func (*testServiceServer) UnaryCall(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	return &pb.SimpleResponse{Payload: req.Payload}, nil
}

func (*testServiceServer) StreamingOutputCall(req *pb.StreamingOutputCallRequest, srv pb.TestService_StreamingOutputCallServer) error {
	for _, params := range req.ResponseParameters {
		srv.Send(&pb.StreamingOutputCallResponse{
			Payload: genPayload(int(params.Size)),
		})
	}
	return nil
}

func genPayload(size int) *pb.Payload {
	if size < 0 {
		size = 32
	}
	body := make([]byte, size)
	rand.Read(body)
	return &pb.Payload{Body: body}
}
