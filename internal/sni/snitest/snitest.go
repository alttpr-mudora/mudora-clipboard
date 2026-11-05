// Package snitest is an in-process fake SNI server for tests, so the client and the poll loop can be tested
// without an emulator. It serves 128 KB of fake WRAM at the FX Pak Pro address 0xF50000.
package snitest

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/alttpr-mudora/mudora-clipboard/internal/sni"
	pb "github.com/alttpr-mudora/mudora-clipboard/internal/sni/pb"
)

// WRAMBase is the FX Pak Pro address of WRAM $7E:0000.
const WRAMBase = 0xF50000

// Fake is a fake SNI. The zero value reports no devices and all-zero memory.
type Fake struct {
	pb.UnimplementedDevicesServer
	pb.UnimplementedDeviceMemoryServer

	// Devices are the device URIs to report.
	Devices []string

	mu    sync.Mutex
	wram  [0x20000]byte
	reads int
}

// Put writes bytes into the fake WRAM at an FX Pak Pro address; tests use it to play the game.
func (f *Fake) Put(addr int, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	copy(f.wram[addr-WRAMBase:], data)
}

// Reads returns how many MultiRead requests have been served.
func (f *Fake) Reads() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads
}

// ListDevices implements the SNI Devices service.
func (f *Fake) ListDevices(context.Context, *pb.DevicesRequest) (*pb.DevicesResponse, error) {
	rsp := &pb.DevicesResponse{}
	for _, u := range f.Devices {
		rsp.Devices = append(rsp.Devices, &pb.DevicesResponse_Device{Uri: u, DisplayName: "fake " + u, Kind: "fake"})
	}
	return rsp, nil
}

// MultiRead implements the SNI DeviceMemory service.
func (f *Fake) MultiRead(_ context.Context, req *pb.MultiReadMemoryRequest) (*pb.MultiReadMemoryResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	rsp := &pb.MultiReadMemoryResponse{Uri: req.Uri}
	for _, r := range req.Requests {
		if r.RequestAddressSpace != pb.AddressSpace_FxPakPro {
			return nil, fmt.Errorf("unexpected address space %v", r.RequestAddressSpace)
		}
		lo := int(r.RequestAddress) - WRAMBase
		hi := lo + int(r.Size)
		if lo < 0 || hi > len(f.wram) {
			return nil, fmt.Errorf("read %06X+%d is outside the fake WRAM", r.RequestAddress, r.Size)
		}
		rsp.Responses = append(rsp.Responses, &pb.ReadMemoryResponse{
			RequestAddress:      r.RequestAddress,
			RequestAddressSpace: r.RequestAddressSpace,
			Data:                append([]byte(nil), f.wram[lo:hi]...),
		})
	}
	return rsp, nil
}

// Start serves f on an in-memory listener and returns a client connected to it. Both are cleaned up with the test.
func Start(t testing.TB, f *Fake) *sni.Client {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	pb.RegisterDevicesServer(srv, f)
	pb.RegisterDeviceMemoryServer(srv, f)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	c, err := sni.Dial("passthrough:///bufnet", grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}
