// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.17.3
// source: api/sys-exporter/SysExporter.proto

package sys_exporter

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ExporterServiceClient is the client API for ExporterService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ExporterServiceClient interface {
	SendStreamSnapshots(ctx context.Context, in *SendStreamSnapshotsRequest, opts ...grpc.CallOption) (ExporterService_SendStreamSnapshotsClient, error)
}

type exporterServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewExporterServiceClient(cc grpc.ClientConnInterface) ExporterServiceClient {
	return &exporterServiceClient{cc}
}

func (c *exporterServiceClient) SendStreamSnapshots(ctx context.Context, in *SendStreamSnapshotsRequest, opts ...grpc.CallOption) (ExporterService_SendStreamSnapshotsClient, error) {
	stream, err := c.cc.NewStream(ctx, &ExporterService_ServiceDesc.Streams[0], "/sys_exporter.ExporterService/SendStreamSnapshots", opts...)
	if err != nil {
		return nil, err
	}
	x := &exporterServiceSendStreamSnapshotsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ExporterService_SendStreamSnapshotsClient interface {
	Recv() (*SendStreamSnapshotsResponse, error)
	grpc.ClientStream
}

type exporterServiceSendStreamSnapshotsClient struct {
	grpc.ClientStream
}

func (x *exporterServiceSendStreamSnapshotsClient) Recv() (*SendStreamSnapshotsResponse, error) {
	m := new(SendStreamSnapshotsResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ExporterServiceServer is the server API for ExporterService service.
// All implementations must embed UnimplementedExporterServiceServer
// for forward compatibility
type ExporterServiceServer interface {
	SendStreamSnapshots(*SendStreamSnapshotsRequest, ExporterService_SendStreamSnapshotsServer) error
	mustEmbedUnimplementedExporterServiceServer()
}

// UnimplementedExporterServiceServer must be embedded to have forward compatible implementations.
type UnimplementedExporterServiceServer struct {
}

func (UnimplementedExporterServiceServer) SendStreamSnapshots(*SendStreamSnapshotsRequest, ExporterService_SendStreamSnapshotsServer) error {
	return status.Errorf(codes.Unimplemented, "method SendStreamSnapshots not implemented")
}
func (UnimplementedExporterServiceServer) mustEmbedUnimplementedExporterServiceServer() {}

// UnsafeExporterServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ExporterServiceServer will
// result in compilation errors.
type UnsafeExporterServiceServer interface {
	mustEmbedUnimplementedExporterServiceServer()
}

func RegisterExporterServiceServer(s grpc.ServiceRegistrar, srv ExporterServiceServer) {
	s.RegisterService(&ExporterService_ServiceDesc, srv)
}

func _ExporterService_SendStreamSnapshots_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SendStreamSnapshotsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ExporterServiceServer).SendStreamSnapshots(m, &exporterServiceSendStreamSnapshotsServer{stream})
}

type ExporterService_SendStreamSnapshotsServer interface {
	Send(*SendStreamSnapshotsResponse) error
	grpc.ServerStream
}

type exporterServiceSendStreamSnapshotsServer struct {
	grpc.ServerStream
}

func (x *exporterServiceSendStreamSnapshotsServer) Send(m *SendStreamSnapshotsResponse) error {
	return x.ServerStream.SendMsg(m)
}

// ExporterService_ServiceDesc is the grpc.ServiceDesc for ExporterService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ExporterService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "sys_exporter.ExporterService",
	HandlerType: (*ExporterServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SendStreamSnapshots",
			Handler:       _ExporterService_SendStreamSnapshots_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/sys-exporter/SysExporter.proto",
}