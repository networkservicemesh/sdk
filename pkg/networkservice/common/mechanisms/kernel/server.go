package kernel

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/sdk/pkg/tools/inodes"
	"strconv"
)

type mechanismsServer struct {
	mechanisms map[string]networkservice.NetworkServiceServer // key is Mechanism.Type
}

func (m mechanismsServer) Request(ctx context.Context, req *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn := req.Connection
	inode := uint64(0)
	inode, err := inodes.GetInode("/proc/self/net/ns")
	if err != nil {
		return nil, err
	}
	conn.Mechanism.Parameters[common.NetNSInodeKey] = strconv.FormatUint(inode, 10)
	return conn, nil
}

func (m mechanismsServer) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	panic("implement me")
}

func NewServer() networkservice.NetworkServiceServer {
	return mechanismsServer{}
}
