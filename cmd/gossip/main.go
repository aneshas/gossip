package main

import (
	"flag"
	"log"
	"os"

	"github.com/nats-io/go-nats-streaming"
	"github.com/tonto/gossip/pkg/agent"
	"github.com/tonto/gossip/pkg/broker"
	"github.com/tonto/gossip/pkg/chat"
	"github.com/tonto/gossip/pkg/ingest"
	"github.com/tonto/gossip/pkg/platform/nats"
	"github.com/tonto/gossip/pkg/platform/redis"
	"github.com/tonto/kit/http"
	"github.com/tonto/kit/http/adapter"
)

func main() {
	var (
		admin = flag.String("admin", "admin", "chat administrator username (basic auth)")
		pass  = flag.String("password", "admin", "chat administrator password (basic auth)")

		clusterID = flag.String("nats-cluster-id", "test-cluster", "nats streaming cluster id")
		clientID  = flag.String("nats-client-id", "test-client", "nats streaming client id")
		natsURL   = flag.String("nats-url", "nats://nats_stream:4222", "nats streaming url")

		redisHost = flag.String("redis-host", "redis", "redis host url")
	)

	flag.Parse()

	store, err := redis.NewStore(*redisHost)
	checkErr(err)

	nt, err := stan.Connect(*clusterID, *clientID, stan.NatsURL(*natsURL))
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
		agent.NewAPI(
			broker.New(
				nt,
				ingest.New(
					nats.New(nt),
					store,
				),
			),
			store,
		),
		chat.NewAPI(store, *admin, *pass),
	)

	log.Fatal(srv.Run(8080))
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
