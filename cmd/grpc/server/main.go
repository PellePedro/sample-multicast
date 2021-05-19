package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/pellepedro/sample-multicast/protos"
	"google.golang.org/grpc"
)

const (
	GRPC_PORT = 50051
)

type Server struct{}

func (s *Server) ExecUnary(ctx context.Context, req *protos.UnaryRequest) (*protos.UnaryResponse, error) {
	fmt.Printf("Server Received ExecUnary() with [%s]\n", req.GetMessage())
	return &protos.UnaryResponse{Status: "Server Hello"}, nil
}

// Server Streaming
func (s *Server) SubscribeToLinkStatus(req *protos.SubscribeRequest, stream protos.HopbyHopAdaptiveOptimizer_SubscribeToLinkStatusServer) error {

	ch := make(chan interface{}, 100)
	eventChannel.RegisterListener(req.GetClientId(), ch)
	defer func() {
		eventChannel.UnregisterListener(req.GetClientId())
		close(ch)
	}()

	for {
		select {
		case data := <-ch:
			link, ok := data.(Metric)
			if ok {
				res := protos.LinkMetricsStream{
					Src:     link.src,
					Dst:     link.dst,
					Linkid:  link.linkid,
					Jitter:  int32(link.jitter),
					Latency: int32(link.latency),
				}
				if err := stream.Send(&res); err != nil {
					fmt.Printf("Stream error [%#v]\n", err)
					return nil
				}
				fmt.Printf("Server Streams [%#v]\n", link)
			} else {
				fmt.Println("Failed to decode Metric from event channel")
			}
		}
	}
}

// Client Streaming
func (s *Server) PublishOptimalRoute(stream protos.HopbyHopAdaptiveOptimizer_PublishOptimalRouteServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			fmt.Printf("EOF while reading client stream: %v", err)
			// finished reading the client stream
			return stream.SendAndClose(&protos.OptimalRouteStreamResponse{})
		}
		if err != nil {
			fmt.Printf("Error while reading client stream: %v", err)
		}
		fmt.Printf("Received Streaming Message message [%#v]\n", req)
	}
}

// BiDi Streaming
func (wu *Server) DoBiDiStreaming(stream protos.HopbyHopAdaptiveOptimizer_DoBiDiStreamingServer) error {
	return nil
}

func main() {
	fmt.Printf("Starting GRPC Server")

	var wg sync.WaitGroup
	wg.Add(1)

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", GRPC_PORT))
	if err != nil {
		fmt.Printf("Failed to listen to grpc port: %v", err)
	}

	s := grpc.NewServer()
	protos.RegisterHopbyHopAdaptiveOptimizerServer(s, &Server{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve grpc: %v", err)
	}

	wg.Wait()

	fmt.Printf("Stopping GRPC Server")
}
