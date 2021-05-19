package grpc

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/pellepedro/sample-multicast/protos"
	"google.golang.org/grpc"
)

type GrpcClient struct {
	inboundChannel   chan interface{}
	outbundChannel   chan interface{}
	completedCh      chan bool
	clientConnection *grpc.ClientConn
}

func NewGrpcClient(inCh chan interface{}, outCh chan interface{}) (*GrpcClient, chan bool) {
	errorChannel := make(chan bool, 1)
	return &GrpcClient{
		inboundChannel: inCh,
		outbundChannel: outCh,
		completedCh:    errorChannel,
	}, errorChannel
}

func (client *GrpcClient) Connect(server string, port int) error {

	opts := grpc.WithInsecure()
	serverUrl := fmt.Sprintf("%s:%d", server, port)
	cc, err := grpc.Dial(serverUrl, opts)
	if err != nil {
		return err
	}
	fmt.Println("Storing grpc clinet connection")
	client.clientConnection = cc
	return nil
}

func (client *GrpcClient) SubscribeToLinkState(clientId string) error {
	if client.clientConnection == nil {
		return fmt.Errorf("Client not connected to grpc server")
	}

	fmt.Println("Client calls SubscribeToLinkState ...")
	c := protos.NewHopbyHopAdaptiveOptimizerClient(client.clientConnection)

	req := &protos.SubscribeRequest{
		ClientId: clientId,
	}
	subscriptionStream, err := c.SubscribeToLinkStatus(context.Background(), req)
	if err != nil {
		fmt.Printf("Failed to SubscribeToLinkStatus [%s]\n", err.Error())
		return err
	}

	go func() {
		for {
			fmt.Println("Client reading subscription")
			msg, err := subscriptionStream.Recv()
			if err == io.EOF {
				// Server Closed the stream
				fmt.Println("Server closed Stream")
				client.completedCh <- true
				return
			} else if err != nil {
				log.Fatalf("error while reading stream: %v", err)
			}
			client.inboundChannel <- msg
		}
	}()
	return nil
}

func (client *GrpcClient) Close() {
	client.clientConnection.Close()
}
