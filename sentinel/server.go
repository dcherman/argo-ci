package sentinel

import (
	"github.com/palantir/go-baseapp/baseapp"
	"github.com/palantir/go-githubapp/githubapp"

	"github.com/rs/zerolog"
)

type sentinelServer struct {
	
}

func NewServer() *sentinelServer {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	server, err := baseapp.NewServer(
		config.Server,
		baseapp.DefaultParams(logger, "argoci.")...,
	)
}

func (s *sentinelServer) Handle(w http.ResponseWriter, r *http.Request) {

}