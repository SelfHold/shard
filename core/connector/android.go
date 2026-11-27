package connector

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/SelfHold/shard/core/connector/pb"
)

// Call serves one unary method for the Android bound service: req is the
// serialized request message, the result a serialized pb.CallReply. The
// caller is what the service read from Binder (package, certificate).
func (s *Server) Call(ctx context.Context, c Caller, method string, req []byte) []byte {
	ctx = WithCaller(ctx, c)
	var (
		out proto.Message
		err error
	)
	switch method {
	case "Hello":
		out, err = unary(ctx, req, &pb.HelloRequest{}, s.Hello)
	case "RequestPairing":
		out, err = unary(ctx, req, &pb.PairingRequest{}, s.RequestPairing)
	case "AwaitPairing":
		out, err = unary(ctx, req, &pb.AwaitPairingRequest{}, s.AwaitPairing)
	case "Get":
		out, err = unary(ctx, req, &pb.GetRequest{}, s.Get)
	case "List":
		out, err = unary(ctx, req, &pb.ListRequest{}, s.List)
	case "Put":
		out, err = unary(ctx, req, &pb.PutRequest{}, s.Put)
	case "Delete":
		out, err = unary(ctx, req, &pb.DeleteRequest{}, s.Delete)
	case "AppKey":
		out, err = unary(ctx, req, &pb.AppKeyRequest{}, s.AppKey)
	case "Unpair":
		out, err = unary(ctx, req, &pb.UnpairRequest{}, s.Unpair)
	case "ListLogins":
		out, err = unary(ctx, req, &pb.ListLoginsRequest{}, s.ListLogins)
	case "GetLogins":
		out, err = unary(ctx, req, &pb.GetLoginsRequest{}, s.GetLogins)
	case "PutLogin":
		out, err = unary(ctx, req, &pb.PutLoginRequest{}, s.PutLogin)
	case "DeleteLogin":
		out, err = unary(ctx, req, &pb.DeleteLoginRequest{}, s.DeleteLogin)
	case "TouchLogin":
		out, err = unary(ctx, req, &pb.TouchLoginRequest{}, s.TouchLogin)
	case "ListChanges":
		out, err = unary(ctx, req, &pb.ListChangesRequest{}, s.ListChanges)
	case "ListSshKeys":
		out, err = unary(ctx, req, &pb.ListSshKeysRequest{}, s.ListSshKeys)
	case "GetSshKeys":
		out, err = unary(ctx, req, &pb.GetSshKeysRequest{}, s.GetSshKeys)
	case "ListServers":
		out, err = unary(ctx, req, &pb.ListServersRequest{}, s.ListServers)
	case "GetServers":
		out, err = unary(ctx, req, &pb.GetServersRequest{}, s.GetServers)
	case "PutServer":
		out, err = unary(ctx, req, &pb.PutServerRequest{}, s.PutServer)
	case "DeleteServer":
		out, err = unary(ctx, req, &pb.DeleteServerRequest{}, s.DeleteServer)
	case "ListServerChanges":
		out, err = unary(ctx, req, &pb.ListChangesRequest{}, s.ListServerChanges)
	default:
		err = status.Errorf(codes.Unimplemented, "unknown method %q", method)
	}
	return reply(out, err)
}

func unary[Req proto.Message, Rep proto.Message](ctx context.Context, data []byte, req Req, h func(context.Context, Req) (Rep, error)) (proto.Message, error) {
	if err := proto.Unmarshal(data, req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "bad request: %v", err)
	}
	return h(ctx, req)
}

func reply(m proto.Message, err error) []byte {
	r := &pb.CallReply{}
	if err != nil {
		st := status.Convert(err)
		r.Code, r.Message = int32(st.Code()), st.Message()
	} else if r.Payload, err = proto.Marshal(m); err != nil {
		r.Code, r.Message = int32(codes.Internal), err.Error()
	}
	out, _ := proto.Marshal(r)
	return out
}

// WatchFunc is Watch for the Android bound service: send gets serialized
// pb.Change messages until ctx ends. The returned bytes are a pb.CallReply
// with the reason the stream ended.
func (s *Server) WatchFunc(ctx context.Context, c Caller, send func([]byte) error) []byte {
	err := s.watch(WithCaller(ctx, c), func(ch *pb.Change) error {
		b, err := proto.Marshal(ch)
		if err != nil {
			return err
		}
		return send(b)
	})
	return reply(&pb.Change{}, err)
}
