package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/SelfHold/shard/core/rpc/pb"
)

// apps manages connected apps through the running daemon: pairing
// requests live in its memory.
func (a *app) apps(ctx context.Context, args []string) error {
	c, ctx, closeConn, err := daemon(ctx)
	if err != nil {
		return err
	}
	defer closeConn()
	sub := "ls"
	if len(args) > 0 {
		sub, args = args[0], args[1:]
	}
	one := func() (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("shard apps %s takes one id", sub)
		}
		return args[0], nil
	}
	switch sub {
	case "ls":
		r, err := c.ListApps(ctx, &pb.Empty{})
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		if len(r.Requests) > 0 {
			fmt.Fprintln(w, "WAITING\tAPP\tNAME\tCODE\tFROM\tSCOPES")
			for _, q := range r.Requests {
				from := q.Platform
				if q.Package != "" {
					from += " " + q.Package
				}
				if q.Description != "" {
					from += " (" + q.Description + ")"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", q.Id, q.AppId, q.AppName, q.Code, from, scopes(q.Scopes))
			}
			fmt.Fprintln(w)
			for _, q := range r.Requests {
				for _, a := range []struct{ scope, says string }{
					{"passwords", "wants to read and change all your website passwords"},
					{"ssh_keys", "wants to use your SSH keys"},
					{"servers", "wants to read and change your servers"},
				} {
					if slices.Contains(q.Scopes, a.scope) {
						fmt.Fprintf(w, "%s %s: 'shard apps approve %s %s'\n", q.AppName, a.says, q.Id, a.scope)
					}
				}
			}
		}
		fmt.Fprintln(w, "PAIRED\tAPP\tNAME\tPLATFORM\tSINCE\tSCOPES\tREAD PASSWORDS\tREAD KEYS\tREAD SERVERS")
		at := func(ms int64) string {
			if ms <= 0 {
				return "-"
			}
			return time.UnixMilli(ms).Format("2006-01-02 15:04")
		}
		for _, p := range r.Pairings {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", p.Id, p.AppId, p.AppName, p.Platform,
				at(p.ApprovedUnixMs), scopes(p.Scopes), at(p.SecretsReadUnixMs), at(p.KeysReadUnixMs), at(p.ServersReadUnixMs))
		}
		if len(r.Data) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "DATA\tRECORDS\tBYTES")
			for _, d := range r.Data {
				fmt.Fprintf(w, "%s\t%d\t%d\n", d.AppId, d.Records, d.Bytes)
			}
		}
		return w.Flush()
	case "approve":
		if len(args) < 1 {
			return errors.New("shard apps approve takes an id and the scopes to grant")
		}
		_, err = c.DecidePairing(ctx, &pb.DecidePairingRequest{Id: args[0], Approve: true, Scopes: args[1:]})
		return err
	case "deny":
		id, err := one()
		if err != nil {
			return err
		}
		_, err = c.DecidePairing(ctx, &pb.DecidePairingRequest{Id: id})
		return err
	case "scopes":
		if len(args) < 1 {
			return errors.New("shard apps scopes takes a pairing id and the scopes to keep")
		}
		_, err = c.SetAppScopes(ctx, &pb.SetAppScopesRequest{Id: args[0], Scopes: args[1:]})
		return err
	case "revoke":
		id, err := one()
		if err != nil {
			return err
		}
		_, err = c.RevokeApp(ctx, &pb.IdRequest{Id: id})
		return err
	case "rm-data":
		id, err := one()
		if err != nil {
			return err
		}
		_, err = c.DeleteAppData(ctx, &pb.IdRequest{Id: id})
		return err
	}
	return fmt.Errorf("unknown command 'shard apps %s' (see shard -h)", sub)
}

func scopes(s []string) string {
	if len(s) == 0 {
		return "records"
	}
	return "records," + strings.Join(s, ",")
}

// daemon connects to the running shardd through its endpoint.json.
func daemon(ctx context.Context) (pb.ShardClient, context.Context, func(), error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, nil, nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "selfhold", "shard", "endpoint.json"))
	if err != nil {
		return nil, nil, nil, errors.New("shardd is not running (no endpoint.json): connected apps are managed by the daemon")
	}
	var ep struct{ Address, Token string }
	if err := json.Unmarshal(data, &ep); err != nil {
		return nil, nil, nil, err
	}
	target := ep.Address
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if p, ok := strings.CutPrefix(ep.Address, "unix://"); ok {
		target = "passthrough:///unix"
		opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", p)
		}))
	}
	cc, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, nil, nil, err
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+ep.Token)
	return pb.NewShardClient(cc), ctx, func() { cc.Close() }, nil
}
