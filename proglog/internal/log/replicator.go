package log

import (
	"context"
	"sync"

	api "github.com/jmnelson12/distributed-systems/proglog/api/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

/*
The replicator connects to other servers with the gRPC client, and we need
to configure the client so it can authenticate with the servers.
*/

type Replicator struct {
	DialOptions []grpc.DialOption
	LocalServer api.LogClient

	logger *zap.Logger

	mu      sync.Mutex
	servers map[string]chan struct{}
	closed  bool
	close   chan struct{}
}

/*
Join(name, addr string) method adds the given server address to the list of
servers to replicate and kicks off the add goroutine to run the actual replication
logic.
*/
func (r *Replicator) Join(name, addr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()
	if r.closed {
		return nil
	}
	if _, ok := r.servers[name]; ok {
		// already replicating so skip
		return nil
	}
	r.servers[name] = make(chan struct{})
	go r.replicate(addr, r.servers[name])
	return nil
}

/*
replicate(addr string, leave chan struct{}) creates a client and opens up a stream to consume all logs on the server.
*/
func (r *Replicator) replicate(addr string, leave chan struct{}) {
	cc, err := grpc.Dial(addr, r.DialOptions...)
	if err != nil {
		r.logError(err, "failed to dial", addr)
		return
	}
	defer cc.Close()

	client := api.NewLogClient(cc)

	ctx := context.Background()
	stream, err := client.ConsumeStream(ctx,
		&api.ConsumeRequest{
			Offset: 0,
		},
	)
	if err != nil {
		r.logError(err, "failed to consume", addr)
		return
	}

	records := make(chan *api.Record)
	go func() {
		for {
			recv, err := stream.Recv()
			if err != nil {
				r.logError(err, "failed to receive", addr)
				return
			}
			records <- recv.Record
		}
	}()

	// consumes the logs from the discovered server in a stream and then
	// produces to the local server to save a copy.
	for {
		select {
		case <-r.close:
			return
		case <-leave:
			return
		case record := <-records:
			_, err = r.LocalServer.Produce(ctx,
				&api.ProduceRequest{
					Record: record,
				},
			)
			if err != nil {
				r.logError(err, "failed to produce", addr)
				return
			}
		}
	}
}

/*
Leave(name string) handles the server leaving the cluster by
removing the server from the list of servers to replicate and closes the server’s
associated channel.
*/
func (r *Replicator) Leave(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()
	if _, ok := r.servers[name]; !ok {
		return nil
	}
	close(r.servers[name])
	delete(r.servers, name)
	return nil
}

/*
Close() closes the replicator so it doesn’t replicate new servers that join the
cluster and it stops replicating existing servers by causing the replicate() gorou-
tines to return.
*/
func (r *Replicator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()
	if r.closed {
		return nil
	}
	r.closed = true
	close(r.close)
	return nil
}

func (r *Replicator) init() {
	if r.logger == nil {
		r.logger = zap.L().Named("replicator")
	}
	if r.servers == nil {
		r.servers = make(map[string]chan struct{})
	}
	if r.close == nil {
		r.close = make(chan struct{})
	}
}

/*
With this method, we just log the errors because we have no other use for
them and to keep the code short and simple. If your users need access to
the errors, a technique you can use to expose these errors is to export an
error channel and send the errors into it for your users to receive and
handle.
*/
func (r *Replicator) logError(err error, msg, addr string) {
	r.logger.Error(
		msg,
		zap.String("addr", addr),
		zap.Error(err),
	)
}
