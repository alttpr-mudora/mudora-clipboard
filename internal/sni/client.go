// Package sni is a client for a running SNI (https://github.com/alttpo/sni) that reads the Copy Rod tables from a
// connected emulator or flash cart.
package sni

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod"
	pb "github.com/alttpr-mudora/mudora-clipboard/internal/sni/pb"
)

// rpcTimeout bounds every individual SNI call.
const rpcTimeout = 3 * time.Second

// Client talks to a running SNI (https://github.com/alttpo/sni) over gRPC.
type Client struct {
	addr    string
	conn    *grpc.ClientConn
	devices pb.DevicesClient
	mem     pb.DeviceMemoryClient
}

// Dial prepares a connection to SNI. The connection is lazy: nothing is sent until the first call.
// Extra dial options are for tests.
func Dial(addr string, opts ...grpc.DialOption) (*Client, error) {
	opts = append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, opts...)
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return &Client{
		addr:    addr,
		conn:    conn,
		devices: pb.NewDevicesClient(conn),
		mem:     pb.NewDeviceMemoryClient(conn),
	}, nil
}

// Close releases the connection.
func (c *Client) Close() error { return c.conn.Close() }

// describe turns a gRPC error into something a person can act on.
func (c *Client) describe(what string, err error) error {
	if status.Code(err) == codes.Unavailable {
		return fmt.Errorf("cannot reach SNI at %s (is it running?)", c.addr)
	}
	return fmt.Errorf("%s: %w", what, err)
}

// PickDevice returns the first device whose URI contains match (any device if match is empty).
func (c *Client) PickDevice(ctx context.Context, match string) (*pb.DevicesResponse_Device, error) {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	rsp, err := c.devices.ListDevices(ctx, &pb.DevicesRequest{})
	if err != nil {
		return nil, c.describe("list devices", err)
	}
	if len(rsp.Devices) == 0 {
		return nil, errors.New("SNI has no devices (is the emulator running with NWA enabled?)")
	}
	var uris []string
	for _, d := range rsp.Devices {
		if match == "" || strings.Contains(d.Uri, match) {
			return d, nil
		}
		uris = append(uris, d.Uri)
	}
	return nil, fmt.Errorf("no device matches %q (available: %s)", match, strings.Join(uris, ", "))
}

// Read fetches both tables from the device in a single request.
func (c *Client) Read(ctx context.Context, uri string) (copyrod.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	rsp, err := c.mem.MultiRead(ctx, &pb.MultiReadMemoryRequest{
		Uri: uri,
		Requests: []*pb.ReadMemoryRequest{
			{RequestAddress: copyrod.AddrEnemyMap, RequestAddressSpace: pb.AddressSpace_FxPakPro, Size: copyrod.EnemyMapSize},
			{RequestAddress: copyrod.AddrSpriteSeen, RequestAddressSpace: pb.AddressSpace_FxPakPro, Size: copyrod.SpriteSeenSize},
		},
	})
	if err != nil {
		return copyrod.Snapshot{}, c.describe("read memory", err)
	}
	if len(rsp.Responses) != 2 {
		return copyrod.Snapshot{}, fmt.Errorf("read memory: got %d responses, want 2", len(rsp.Responses))
	}
	var s copyrod.Snapshot
	if n := copy(s.EnemyMap[:], rsp.Responses[0].Data); n != copyrod.EnemyMapSize {
		return copyrod.Snapshot{}, fmt.Errorf("read memory: enemy table is %d bytes, want %d", n, copyrod.EnemyMapSize)
	}
	if n := copy(s.Seen[:], rsp.Responses[1].Data); n != copyrod.SpriteSeenSize {
		return copyrod.Snapshot{}, fmt.Errorf("read memory: seen table is %d bytes, want %d", n, copyrod.SpriteSeenSize)
	}
	return s, nil
}
