package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/nats-io/go-nats-streaming"
	"github.com/tonto/gossip/pkg/agent"
	"github.com/tonto/gossip/pkg/broker"
	"github.com/tonto/gossip/pkg/chat"
	"github.com/tonto/gossip/pkg/ingest"
	"github.com/tonto/gossip/pkg/platform/redis"
	"github.com/tonto/kit/http"
	"github.com/tonto/kit/http/adapter"
)

func main() {
	var (
		admin = flag.String("admin", "admin", "chat administrator username (basic auth)")
		pass  = flag.String("password", "admin", "chat administrator password (basic auth)")
	)

	flag.Parse()

	store, err := redis.NewStore("localhost")
	checkErr(err)

	nats, err := stan.Connect("test-cluster", "test-client", stan.NatsURL(stan.DefaultNatsURL))
	checkErr(err)

	ig := ingest.New(
		nats,
		store,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = ig.Run(ctx)
	checkErr(err)

	logger := log.New(os.Stdout, "chat/ws => ", log.Ldate|log.Ltime|log.Lshortfile)

	srv := http.NewServer(
		http.WithLogger(logger),
		http.WithAdapters(
			adapter.WithRequestLogger(logger),
			adapter.WithCORS(
				adapter.WithCORSAllowOrigins("*"),
				adapter.WithCORSAllowHeaders("Authorization", "Accept", "Accept-Language", "Content-Language", "Content-Type"),
			),
		),
	)

	srv.RegisterServices(
		agent.NewAPI(broker.New(nats), store),
		chat.NewAPI(store, *admin, *pass),
	)

	log.Fatal(srv.Run(8080))
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
