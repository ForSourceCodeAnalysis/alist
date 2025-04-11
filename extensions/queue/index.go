package queue

import (
	"context"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var client *asynq.Client
var server *asynq.Server
var mux *asynq.ServeMux

func Init() {

	redisOpt := asynq.RedisClientOpt{Addr: conf.Conf.Redis.Addr,
		Password: conf.Conf.Redis.Password,
		DB:       conf.Conf.Redis.DB}

	client = asynq.NewClient(redisOpt)

	server = asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)
	mux = asynq.NewServeMux()
}

// Start starts the queue server
func Start() {
	go func() {
		if err := server.Run(mux); err != nil {
			logrus.Fatal(errors.WithStack(err))
		}
	}()
}

// GetClient returns the client
func GetClient() *asynq.Client {
	return client
}

// RegisterHandler registers a handler for a task
func RegisterHandler(typename string, Handler func(ctx context.Context, task *asynq.Task) error) {
	mux.HandleFunc(typename, Handler)
}
