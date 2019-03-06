package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	api_pb "github.com/kubeflow/katib/pkg/api"
	health_pb "github.com/kubeflow/katib/pkg/api/health"
	kdb "github.com/kubeflow/katib/pkg/db"
	"github.com/kubeflow/katib/pkg/manager/modelstore"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = "0.0.0.0:6789"
)

var dbIf = nil

type server struct {
	msIf modelstore.ModelStore
}

func (s *server) CreateStudy(ctx context.Context, in *kdb.CreateStudyRequest) (*kdb.CreateStudyReply, error) {
	if in == nil || in.StudyConfig == nil {
		return &kdb.CreateStudyReply{}, errors.New("StudyConfig is missing.")
	}

	studyID, err := dbIf.CreateStudy(in.StudyConfig)
	if err != nil {
		return &kdb.CreateStudyReply{}, err
	}

	return &kdb.CreateStudyReply{StudyId: studyID}, nil
}

func (s *server) DeleteStudy(ctx context.Context, in *kdb.DeleteStudyRequest) (*kdb.DeleteStudyReply, error) {
	if in == nil || in.StudyId == "" {
		return &kdb.DeleteStudyReply{}, errors.New("StudyId is missing.")
	}
	err := dbIf.DeleteStudy(in.StudyId)
	if err != nil {
		return &kdb.DeleteStudyReply{}, err
	}
	return &kdb.DeleteStudyReply{StudyId: in.StudyId}, nil
}

func (s *server) GetStudy(ctx context.Context, in *kdb.GetStudyRequest) (*kdb.GetStudyReply, error) {

	sc, err := dbIf.GetStudy(in.StudyId)

	if err != nil {
		return &kdb.GetStudyReply{}, err
	}
	return &kdb.GetStudyReply{StudyConfig: sc}, err
}

func (s *server) GetStudyList(ctx context.Context, in *kdb.GetStudyListRequest) (*kdb.GetStudyListReply, error) {
	sl, err := dbIf.GetStudyList()
	if err != nil {
		return &kdb.GetStudyListReply{}, err
	}
	result := make([]*kdb.StudyOverview, len(sl))
	for i, id := range sl {
		sc, err := dbIf.GetStudy(id)
		if err != nil {
			return &kdb.GetStudyListReply{}, err
		}
		result[i] = &kdb.StudyOverview{
			Name:  sc.Name,
			Owner: sc.Owner,
			Id:    id,
		}
	}
	return &kdb.GetStudyListReply{StudyOverviews: result}, err
}

func (s *server) CreateTrial(ctx context.Context, in *kdb.CreateTrialRequest) (*kdb.CreateTrialReply, error) {
	err := dbIf.CreateTrial(in.Trial)
	return &kdb.CreateTrialReply{TrialId: in.Trial.TrialId}, err
}

func (s *server) GetTrials(ctx context.Context, in *kdb.GetTrialsRequest) (*kdb.GetTrialsReply, error) {
	tl, err := dbIf.GetTrialList(in.StudyId)
	return &kdb.GetTrialsReply{Trials: tl}, err
}

func (s *server) GetTrial(ctx context.Context, in *kdb.GetTrialRequest) (*kdb.GetTrialReply, error) {
	t, err := dbIf.GetTrial(in.TrialId)
	return &kdb.GetTrialReply{Trial: t}, err
}

func (s *server) GetSuggestions(ctx context.Context, in *api_pb.GetSuggestionsRequest) (*api_pb.GetSuggestionsReply, error) {
	if in.SuggestionAlgorithm == "" {
		return &api_pb.GetSuggestionsReply{Trials: []*api_pb.Trial{}}, errors.New("No suggest algorithm specified")
	}
	conn, err := grpc.Dial("vizier-suggestion-"+in.SuggestionAlgorithm+":6789", grpc.WithInsecure())
	if err != nil {
		return &api_pb.GetSuggestionsReply{Trials: []*api_pb.Trial{}}, err
	}

	defer conn.Close()
	c := api_pb.NewSuggestionClient(conn)
	r, err := c.GetSuggestions(ctx, in)
	if err != nil {
		return &api_pb.GetSuggestionsReply{Trials: []*api_pb.Trial{}}, err
	}
	return r, nil
}

func (s *server) RegisterWorker(ctx context.Context, in *kdb.RegisterWorkerRequest) (*kdb.RegisterWorkerReply, error) {
	wid, err := dbIf.CreateWorker(in.Worker)
	return &kdb.RegisterWorkerReply{WorkerId: wid}, err
}

func (s *server) GetWorkers(ctx context.Context, in *kdb.GetWorkersRequest) (*kdb.GetWorkersReply, error) {
	var ws []*kdb.Worker
	var err error
	if in.WorkerId == "" {
		ws, err = dbIf.GetWorkerList(in.StudyId, in.TrialId)
	} else {
		var w *kdb.Worker
		w, err = dbIf.GetWorker(in.WorkerId)
		ws = append(ws, w)
	}
	return &kdb.GetWorkersReply{Workers: ws}, err
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

func (s *server) GetMetrics(ctx context.Context, in *kdb.GetMetricsRequest) (*kdb.GetMetricsReply, error) {
	var mNames []string
	if in.StudyId == "" {
		return &kdb.GetMetricsReply{}, errors.New("StudyId should be set")
	}
	sc, err := dbIf.GetStudy(in.StudyId)
	if err != nil {
		return &kdb.GetMetricsReply{}, err
	}
	if len(in.MetricsNames) > 0 {
		mNames = in.MetricsNames
	} else {
		mNames = sc.Metrics
	}
	if err != nil {
		return &kdb.GetMetricsReply{}, err
	}
	mls := make([]*kdb.MetricsLogSet, len(in.WorkerIds))
	for i, w := range in.WorkerIds {
		wr, err := s.GetWorkers(ctx, &kdb.GetWorkersRequest{
			StudyId:  in.StudyId,
			WorkerId: w,
		})
		if err != nil {
			return &kdb.GetMetricsReply{}, err
		}
		mls[i] = &kdb.MetricsLogSet{
			WorkerId:     w,
			MetricsLogs:  make([]*kdb.MetricsLog, len(mNames)),
			WorkerStatus: wr.Workers[0].Status,
		}
		for j, m := range mNames {
			ls, err := dbIf.GetWorkerLogs(w, &kdb.GetWorkerLogOpts{Name: m})
			if err != nil {
				return &kdb.GetMetricsReply{}, err
			}
			mls[i].MetricsLogs[j] = &kdb.MetricsLog{
				Name:   m,
				Values: make([]*kdb.MetricsValueTime, len(ls)),
			}
			for k, l := range ls {
				mls[i].MetricsLogs[j].Values[k] = &kdb.MetricsValueTime{
					Value: l.Value,
					Time:  l.Time.UTC().Format(time.RFC3339Nano),
				}
			}
		}
	}
	return &kdb.GetMetricsReply{MetricsLogSets: mls}, nil
}

func (s *server) ReportMetricsLogs(ctx context.Context, in *kdb.ReportMetricsLogsRequest) (*kdb.ReportMetricsLogsReply, error) {
	for _, mls := range in.MetricsLogSets {
		err := dbIf.StoreWorkerLogs(mls.WorkerId, mls.MetricsLogs)
		if err != nil {
			return &kdb.ReportMetricsLogsReply{}, err
		}

	}
	return &kdb.ReportMetricsLogsReply{}, nil
}

func (s *server) UpdateWorkerState(ctx context.Context, in *kdb.UpdateWorkerStateRequest) (*kdb.UpdateWorkerStateReply, error) {
	err := dbIf.UpdateWorker(in.WorkerId, in.Status)
	return &kdb.UpdateWorkerStateReply{}, err
}

func (s *server) GetWorkerFullInfo(ctx context.Context, in *kdb.GetWorkerFullInfoRequest) (*kdb.GetWorkerFullInfoReply, error) {
	return dbIf.GetWorkerFullInfo(in.StudyId, in.TrialId, in.WorkerId, in.OnlyLatestLog)
}

func (s *server) SetSuggestionParameters(ctx context.Context, in *kdb.SetSuggestionParametersRequest) (*kdb.SetSuggestionParametersReply, error) {
	var err error
	var id string
	if in.ParamId == "" {
		id, err = dbIf.SetSuggestionParam(in.SuggestionAlgorithm, in.StudyId, in.SuggestionParameters)
	} else {
		id = in.ParamId
		err = dbIf.UpdateSuggestionParam(in.ParamId, in.SuggestionParameters)
	}
	return &kdb.SetSuggestionParametersReply{ParamId: id}, err
}

func (s *server) SetEarlyStoppingParameters(ctx context.Context, in *kdb.SetEarlyStoppingParametersRequest) (*kdb.SetEarlyStoppingParametersReply, error) {
	var err error
	var id string
	if in.ParamId == "" {
		id, err = dbIf.SetEarlyStopParam(in.EarlyStoppingAlgorithm, in.StudyId, in.EarlyStoppingParameters)
	} else {
		id = in.ParamId
		err = dbIf.UpdateEarlyStopParam(in.ParamId, in.EarlyStoppingParameters)
	}
	return &kdb.SetEarlyStoppingParametersReply{ParamId: id}, err
}

func (s *server) GetSuggestionParameters(ctx context.Context, in *kdb.GetSuggestionParametersRequest) (*kdb.GetSuggestionParametersReply, error) {
	ps, err := dbIf.GetSuggestionParam(in.ParamId)
	return &kdb.GetSuggestionParametersReply{SuggestionParameters: ps}, err
}

func (s *server) GetSuggestionParameterList(ctx context.Context, in *kdb.GetSuggestionParameterListRequest) (*kdb.GetSuggestionParameterListReply, error) {
	pss, err := dbIf.GetSuggestionParamList(in.StudyId)
	return &kdb.GetSuggestionParameterListReply{SuggestionParameterSets: pss}, err
}

func (s *server) GetEarlyStoppingParameters(ctx context.Context, in *kdb.GetEarlyStoppingParametersRequest) (*kdb.GetEarlyStoppingParametersReply, error) {
	ps, err := dbIf.GetEarlyStopParam(in.ParamId)
	return &kdb.GetEarlyStoppingParametersReply{EarlyStoppingParameters: ps}, err
}

func (s *server) GetEarlyStoppingParameterList(ctx context.Context, in *kdb.GetEarlyStoppingParameterListRequest) (*kdb.GetEarlyStoppingParameterListReply, error) {
	pss, err := dbIf.GetEarlyStopParamList(in.StudyId)
	return &kdb.GetEarlyStoppingParameterListReply{EarlyStoppingParameterSets: pss}, err
}

func (s *server) SaveStudy(ctx context.Context, in *kdb.SaveStudyRequest) (*kdb.SaveStudyReply, error) {
	var err error
	if s.msIf != nil {
		err = s.msIf.SaveStudy(in)
	}
	return &kdb.SaveStudyReply{}, err
}

func (s *server) SaveModel(ctx context.Context, in *kdb.SaveModelRequest) (*kdb.SaveModelReply, error) {
	if s.msIf != nil {
		err := s.msIf.SaveModel(in)
		if err != nil {
			log.Printf("Save Model failed %v", err)
			return &kdb.SaveModelReply{}, err
		}
	}
	return &kdb.SaveModelReply{}, nil
}

func (s *server) GetSavedStudies(ctx context.Context, in *kdb.GetSavedStudiesRequest) (*kdb.GetSavedStudiesReply, error) {
	ret := []*kdb.StudyOverview{}
	var err error
	if s.msIf != nil {
		ret, err = s.msIf.GetSavedStudies()
	}
	return &kdb.GetSavedStudiesReply{Studies: ret}, err
}

func (s *server) GetSavedModels(ctx context.Context, in *kdb.GetSavedModelsRequest) (*kdb.GetSavedModelsReply, error) {
	ret := []*kdb.ModelInfo{}
	var err error
	if s.msIf != nil {
		ret, err = s.msIf.GetSavedModels(in)
	}
	return &kdb.GetSavedModelsReply{Models: ret}, err
}

func (s *server) GetSavedModel(ctx context.Context, in *kdb.GetSavedModelRequest) (*kdb.GetSavedModelReply, error) {
	var ret *kdb.ModelInfo = nil
	var err error
	if s.msIf != nil {
		ret, err = s.msIf.GetSavedModel(in)
	}
	return &kdb.GetSavedModelReply{Model: ret}, err
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

	// Check if connection to vizier-db is okay since otherwise manager could not serve most of its methods.
	err := dbIf.SelectOne()
	if err != nil {
		resp.Status = health_pb.HealthCheckResponse_NOT_SERVING
		return &resp, fmt.Errorf("Failed to execute `SELECT 1` probe: %v", err)
	}

	return &resp, nil
}

func main() {
	flag.Parse()
	var err error
	conn, err := grpc.Dial("katib-dbif:6789", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to open db connection: %v", err)
	}
	defer conn.Close()
	dbIf = kdb.DBIFClient(conn)
	//dbIf, err = kdb.New()

	dbIf.DBInit()
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	size := 1<<31 - 1
	log.Printf("Start Katib manager: %s", port)
	log.Printf("Hello from Achal's own Katib!")
	s := grpc.NewServer(grpc.MaxRecvMsgSize(size), grpc.MaxSendMsgSize(size))
	api_pb.RegisterManagerServer(s, &server{})
	health_pb.RegisterHealthServer(s, &server{})
	reflection.Register(s)
	if err = s.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
