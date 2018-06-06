package agent

import (
	pb "github.com/go-ignite/ignite-agent/protos"

	"google.golang.org/grpc"
)

type Client struct {
	pb.AgentServiceClient
	Address string
}

func NewClient(address string) *Client {
	return &Client{Address: address}
}

func (client *Client) Dial() error {
	conn, err := grpc.Dial(client.Address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	client.AgentServiceClient = pb.NewAgentServiceClient(conn)
	return nil
}

func Dial(address string) (*Client, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client := &Client{
		Address:            address,
		AgentServiceClient: pb.NewAgentServiceClient(conn),
	}
	return client, nil
}