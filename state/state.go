package state

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/go-ignite/ignite-agent/protos"
	"github.com/go-ignite/ignite/agent"
	"github.com/go-ignite/ignite/db"

	"github.com/sirupsen/logrus"
)

var (
	l  *Loader
	no sync.Once
)

type Loader struct {
	nodeMutex sync.RWMutex
	nodeMap   map[int64]*NodeStatus
	*logrus.Logger
}

type NodeStatus struct {
	*agent.Client
	available bool
}

func NewNodeStatus(client *agent.Client) *NodeStatus {
	return &NodeStatus{
		Client: client,
	}
}

func (ns *NodeStatus) Heartbeat() error {
	if ns.Client.AgentServiceClient == nil {
		if err := ns.Client.Dial(); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	streamClient, err := ns.Client.NodeHeartbeat(ctx, &pb.GeneralRequest{})
	if err != nil {
		return err
	}
	for {
		_, err := streamClient.Recv()
		if err != nil {
			ns.Client.AgentServiceClient = nil
			ns.available = false
			return err
		} else {
			ns.available = true
		}
	}
	return nil
}

func GetLoader() *Loader {
	no.Do(func() {
		l = &Loader{
			nodeMap: map[int64]*NodeStatus{},
		}
	})
	return l
}

func (loader *Loader) Load() error {
	loader.nodeMutex.Lock()
	defer loader.nodeMutex.Unlock()

	nodes, err := db.GetAllNodes()
	if err != nil {
		return fmt.Errorf("db.GetAllNodes error: %v", err)
	}
	for _, node := range nodes {
		client := agent.NewClient(node.Address)
		ns := NewNodeStatus(client)
		go loader.WatchNode(ns)
		loader.nodeMap[node.Id] = ns
	}
	return nil
}

func (loader *Loader) WatchNode(ns *NodeStatus) {
	for {
		if err := ns.Heartbeat(); err != nil {
			loader.WithError(err).Error()
			time.Sleep(5 * time.Second)
		}
	}
}

func (loader *Loader) GetNode(id int64) *NodeStatus {
	loader.nodeMutex.RLock()
	defer loader.nodeMutex.RUnlock()

	return loader.nodeMap[id]
}

func (loader *Loader) DelNode(id int64) {
	loader.nodeMutex.Lock()
	defer loader.nodeMutex.Unlock()

	delete(loader.nodeMap, id)
}

func (loader *Loader) AddNode(id int64, ns *NodeStatus) {
	loader.nodeMutex.Lock()
	defer loader.nodeMutex.Unlock()

	go loader.WatchNode(ns)
	loader.nodeMap[id] = ns
}