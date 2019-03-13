package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"

	api_pb "github.com/kubeflow/katib/pkg/api"
	health_pb "github.com/kubeflow/katib/pkg/api/health"
	dbif "github.com/kubeflow/katib/pkg/db"
	"github.com/kubeflow/katib/pkg/manager/modelstore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

/**
To plug in a new data store, implement and run the service and change the
dbIfaddress to point to the new service.
**/

const (
	dbIfaddress = "dbif-mysql:6789"
)
const (
	port = "0.0.0.0:6789"
)

type server struct {
	msIf modelstore.ModelStore
}

var dbIf dbif.DBIFClient

func (s *server) CreateStudy(ctx context.Context, in *dbif.CreateStudyRequest) (*dbif.CreateStudyReply, error) {
	return dbIf.CreateStudy(ctx, in)
}

func (s *server) GetStudy(ctx context.Context, in *dbif.GetStudyRequest) (*dbif.GetStudyReply, error) {
	return dbIf.GetStudy(ctx, in)
}

func (s *server) GetStudyList(ctx context.Context, in *dbif.GetStudyListRequest) (*dbif.GetStudyListReply, error) {
	return dbIf.GetStudyList(ctx, in)
}

func (s *server) DeleteStudy(ctx context.Context, in *dbif.DeleteStudyRequest) (*dbif.DeleteStudyReply, error) {
	return dbIf.DeleteStudy(ctx, in)
}

func (s *server) CreateTrial(ctx context.Context, in *dbif.CreateTrialRequest) (*dbif.CreateTrialReply, error) {
	return dbIf.CreateTrial(ctx, in)
}

func (s *server) GetTrials(ctx context.Context, in *dbif.GetTrialsRequest) (*dbif.GetTrialsReply, error) {
	return dbIf.GetTrials(ctx, in)
}

func (s *server) GetTrial(ctx context.Context, in *dbif.GetTrialRequest) (*dbif.GetTrialReply, error) {
	return dbIf.GetTrial(ctx, in)
}

func (s *server) GetSuggestions(ctx context.Context, in *api_pb.GetSuggestionsRequest) (*api_pb.GetSuggestionsReply, error) {
	if in.SuggestionAlgorithm == "" {
		return &api_pb.GetSuggestionsReply{Trials: []*dbif.Trial{}}, errors.New("No suggest algorithm specified")
	}
	conn, err := grpc.Dial("vizier-suggestion-"+in.SuggestionAlgorithm+":6789", grpc.WithInsecure())
	if err != nil {
		return &api_pb.GetSuggestionsReply{Trials: []*dbif.Trial{}}, err
	}

	defer conn.Close()
	c := api_pb.NewSuggestionClient(conn)
	r, err := c.GetSuggestions(ctx, in)
	if err != nil {
		return &api_pb.GetSuggestionsReply{Trials: []*dbif.Trial{}}, err
	}
	return r, nil
}

func (s *server) RegisterWorker(ctx context.Context, in *dbif.RegisterWorkerRequest) (*dbif.RegisterWorkerReply, error) {
	return dbIf.RegisterWorker(ctx, in)
}

func (s *server) GetWorkers(ctx context.Context, in *dbif.GetWorkersRequest) (*dbif.GetWorkersReply, error) {
	return dbIf.GetWorkers(ctx, in)
}

func (s *server) GetShouldStopWorkers(ctx context.Context, in *api_pb.GetShouldStopWorkersRequest) (*api_pb.GetShouldStopWorkersReply, error) {
	if in.EarlyStoppingAlgorithm == "" {
		return &api_pb.GetShouldStopWorkersReply{}, errors.New("No EarlyStopping Algorithm specified")
	}
	conn, err := grpc.Dial("vizier-earlystopping-"+in.EarlyStoppingAlgorithm+":6789", grpc.WithInsecure())
	if err != nil {
		return &api_pb.GetShouldStopWorkersReply{}, err
	}
	defer conn.Close()
	c := api_pb.NewEarlyStoppingClient(conn)
	return c.GetShouldStopWorkers(context.Background(), in)
}

func (s *server) GetMetrics(ctx context.Context, in *dbif.GetMetricsRequest) (*dbif.GetMetricsReply, error) {
	return dbIf.GetMetrics(ctx, in)
}

func (s *server) ReportMetricsLogs(ctx context.Context, in *dbif.ReportMetricsLogsRequest) (*dbif.ReportMetricsLogsReply, error) {
	return dbIf.ReportMetricsLogs(ctx, in)
}

func (s *server) UpdateWorkerState(ctx context.Context, in *dbif.UpdateWorkerStateRequest) (*dbif.UpdateWorkerStateReply, error) {
	return dbIf.UpdateWorkerState(ctx, in)
}

func (s *server) GetWorkerFullInfo(ctx context.Context, in *dbif.GetWorkerFullInfoRequest) (*dbif.GetWorkerFullInfoReply, error) {
	return dbIf.GetWorkerFullInfo(ctx, in)
}

func (s *server) SetSuggestionParameters(ctx context.Context, in *dbif.SetSuggestionParametersRequest) (*dbif.SetSuggestionParametersReply, error) {
	return dbIf.SetSuggestionParameters(ctx, in)
}

func (s *server) SetEarlyStoppingParameters(ctx context.Context, in *dbif.SetEarlyStoppingParametersRequest) (*dbif.SetEarlyStoppingParametersReply, error) {
	return dbIf.SetEarlyStoppingParameters(ctx, in)
}

func (s *server) GetSuggestionParameters(ctx context.Context, in *dbif.GetSuggestionParametersRequest) (*dbif.GetSuggestionParametersReply, error) {
	return dbIf.GetSuggestionParameters(ctx, in)
}

func (s *server) GetSuggestionParameterList(ctx context.Context, in *dbif.GetSuggestionParameterListRequest) (*dbif.GetSuggestionParameterListReply, error) {
	return dbIf.GetSuggestionParameterList(ctx, in)
}

func (s *server) GetEarlyStoppingParameters(ctx context.Context, in *dbif.GetEarlyStoppingParametersRequest) (*dbif.GetEarlyStoppingParametersReply, error) {
	return dbIf.GetEarlyStoppingParameters(ctx, in)
}

func (s *server) GetEarlyStoppingParameterList(ctx context.Context, in *dbif.GetEarlyStoppingParameterListRequest) (*dbif.GetEarlyStoppingParameterListReply, error) {
	return dbIf.GetEarlyStoppingParameterList(ctx, in)
}

func (s *server) SaveStudy(ctx context.Context, in *api_pb.SaveStudyRequest) (*api_pb.SaveStudyReply, error) {
	var err error
	if s.msIf != nil {
		err = s.msIf.SaveStudy(in)
	}
	return &api_pb.SaveStudyReply{}, err
}

func (s *server) SaveModel(ctx context.Context, in *api_pb.SaveModelRequest) (*api_pb.SaveModelReply, error) {
	if s.msIf != nil {
		err := s.msIf.SaveModel(in)
		if err != nil {
			log.Printf("Save Model failed %v", err)
			return &api_pb.SaveModelReply{}, err
		}
	}
	return &api_pb.SaveModelReply{}, nil
}

func (s *server) GetSavedStudies(ctx context.Context, in *api_pb.GetSavedStudiesRequest) (*api_pb.GetSavedStudiesReply, error) {
	ret := []*dbif.StudyOverview{}
	var err error
	if s.msIf != nil {
		ret, err = s.msIf.GetSavedStudies()
	}
	return &api_pb.GetSavedStudiesReply{Studies: ret}, err
}

func (s *server) GetSavedModels(ctx context.Context, in *api_pb.GetSavedModelsRequest) (*api_pb.GetSavedModelsReply, error) {
	ret := []*api_pb.ModelInfo{}
	var err error
	if s.msIf != nil {
		ret, err = s.msIf.GetSavedModels(in)
	}
	return &api_pb.GetSavedModelsReply{Models: ret}, err
}

func (s *server) GetSavedModel(ctx context.Context, in *api_pb.GetSavedModelRequest) (*api_pb.GetSavedModelReply, error) {
	var ret *api_pb.ModelInfo = nil
	var err error
	if s.msIf != nil {
		ret, err = s.msIf.GetSavedModel(in)
	}
	return &api_pb.GetSavedModelReply{Model: ret}, err
}

func (s *server) Check(ctx context.Context, in *health_pb.HealthCheckRequest) (*health_pb.HealthCheckResponse, error) {
	resp := health_pb.HealthCheckResponse{
		Status: health_pb.HealthCheckResponse_SERVING,
	}

	// We only accept optional service name only if it's set to suggested format.
	if in != nil && in.Service != "" && in.Service != "grpc.health.v1.Health" {
		resp.Status = health_pb.HealthCheckResponse_UNKNOWN
		return &resp, fmt.Errorf("grpc.health.v1.Health can only be accepted if you specify service name.")
	}

	//Check if connection to vizier-db is okay since otherwise manager could not serve most of its methods.
	_, err := dbIf.SelectOne(ctx, &dbif.SelectOneRequest{})
	if err != nil {
		resp.Status = health_pb.HealthCheckResponse_NOT_SERVING
		return &resp, fmt.Errorf("Failed to execute `SELECT 1` probe: %v", err)
	}
	return &resp, nil
}

func main() {
	flag.Parse()
	var err error
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	conn, err := grpc.Dial(dbIfaddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not connect to DBIF service: %v", err)
	}
	defer conn.Close()
	dbIf = dbif.NewDBIFClient(conn)
	size := 1<<31 - 1
	log.Printf("Start Katib manager: %s", port)
	s := grpc.NewServer(grpc.MaxRecvMsgSize(size), grpc.MaxSendMsgSize(size))
	api_pb.RegisterManagerServer(s, &server{})
	health_pb.RegisterHealthServer(s, &server{})
	reflection.Register(s)
	if err = s.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
