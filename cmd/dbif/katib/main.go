package main

import (
	"context"
	crand "crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/protobuf/jsonpb"
	dbif "github.com/kubeflow/katib/pkg/db"
	"google.golang.org/grpc"
)

const (
	port = "0.0.0.0:6789"
)

const (
	dbDriver     = "mysql"
	dbNameTmpl   = "root:%s@tcp(vizier-db:3306)/vizier?timeout=5s"
	mysqlTimeFmt = "2006-01-02 15:04:05.999999"

	connectInterval = 5 * time.Second
	connectTimeout  = 60 * time.Second
)

var rs1Letters = []rune("abcdefghijklmnopqrstuvwxyz")

type dbserver struct {
	db *sql.DB
}

func getDbName() string {
	dbPass := os.Getenv("MYSQL_ROOT_PASSWORD")
	if dbPass == "" {
		log.Printf("WARN: Env var MYSQL_ROOT_PASSWORD is empty. Falling back to \"test\".")

		// For backward compatibility, e.g. in case that all but vizier-core
		// is older ones so we do not have Secret nor upgraded vizier-db.
		dbPass = "test"
	}

	return fmt.Sprintf(dbNameTmpl, dbPass)
}

func CreateNewDBServer() *dbserver {
	db, err := openSQLConn(dbDriver, getDbName(), connectInterval, connectTimeout)
	if err != nil {
		return nil, fmt.Errorf("DB open failed: %v", err)
	}
	dbWithConn, err := NewWithSQLConn(db)
	if err != nil {
		return nil, fmt.Errorf("DB open failed: %v", err)
	}
	return &dbserver{db: dbWithConn}, nil
}

func openSQLConn(driverName string, dataSourceName string, interval time.Duration,
	timeout time.Duration) (*sql.DB, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeoutC := time.After(timeout)
	for {
		select {
		case <-ticker.C:
			if db, err := sql.Open(driverName, dataSourceName); err == nil {
				if err = db.Ping(); err == nil {
					return db, nil
				}
			}
		case <-timeoutC:
			return nil, fmt.Errorf("Timeout waiting for DB conn successfully opened.")
		}
	}
}

func NewWithSQLConn(db *sql.DB) (*sql.DB, error) {
	d := new(dbserver)
	d.db = db
	seed, err := crand.Int(crand.Reader, big.NewInt(1<<63-1))
	if err != nil {
		return nil, fmt.Errorf("RNG initialization failed: %v", err)
	}
	// We can do the following instead, but it creates a locking issue
	//d.rng = rand.New(rand.NewSource(seed.Int64()))
	rand.Seed(seed.Int64())
	return d.db, nil
}

// SayHello implements helloworld.GreeterServer
func (s *dbserver) SayHello(ctx context.Context, in *dbif.HelloRequest) (*dbif.HelloReply, error) {
	log.Printf("Received: %v", in.Name)
	return &dbif.HelloReply{Message: "Hello " + in.Name}, nil
}

func (s *dbserver) CreateStudy(ctx context.Context, in *dbif.CreateStudyRequest) (*dbif.CreateStudyReply, error) {
	sc := in.StudyConfig
	if sc.JobId != "" {
		var temporaryId string
		err := s.db.QueryRow("SELECT id FROM studies WHERE job_id = ?", sc.JobId).Scan(&temporaryId)

		if err == nil {
			return &dbif.CreateStudyReply{}, err
		}
	}

	var nasConfig string
	var configs string
	var err error
	if sc.NasConfig != nil {
		nasConfig, err = (&jsonpb.Marshaler{}).MarshalToString(sc.NasConfig)
		if err != nil {
			log.Fatalf("Error marshaling nasConfig: %v", err)
		}
	}

	if sc.ParameterConfigs != nil {
		configs, err = (&jsonpb.Marshaler{}).MarshalToString(sc.ParameterConfigs)
		if err != nil {
			log.Fatalf("Error marshaling configs: %v", err)
		}
	}

	tags := make([]string, len(sc.Tags))
	for i, elem := range sc.Tags {
		tags[i], err = (&jsonpb.Marshaler{}).MarshalToString(elem)
		if err != nil {
			log.Printf("Error marshalling %v: %v", elem, err)
			continue
		}
	}

	var isin bool = false
	for _, m := range sc.Metrics {
		if m == sc.ObjectiveValueName {
			isin = true
		}
	}

	if !isin {
		sc.Metrics = append(sc.Metrics, sc.ObjectiveValueName)
	}

	var studyID string

	i := 3
	for true {
		studyID = generateRandid()
		_, err := s.db.Exec(
			"INSERT INTO studies VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			studyID,
			sc.Name,
			sc.Owner,
			sc.OptimizationType,
			sc.OptimizationGoal,
			configs,
			strings.Join(tags, ",\n"),
			sc.ObjectiveValueName,
			strings.Join(sc.Metrics, ",\n"),
			nasConfig,
			sc.JobId,
			sc.JobType,
		)
		if err == nil {
			break
		} else if isDBDuplicateError(err) {
			i--
			if i > 0 {
				continue
			}
		}
		return &dbif.CreateStudyReply{}, err
	}
	return &dbif.CreateStudyReply{StudyId: studyID}, nil
}

func isDBDuplicateError(err error) bool {
	errmsg := strings.ToLower(err.Error())
	if strings.Contains(errmsg, "unique") || strings.Contains(errmsg, "duplicate") {
		return true
	}
	return false
}

func generateRandid() string {
	// UUID isn't quite handy in the Go world
	id := make([]byte, 8)
	_, err := rand.Read(id)
	if err != nil {
		log.Printf("Error reading random: %v", err)
		return ""
	}
	return string(rs1Letters[rand.Intn(len(rs1Letters))]) + fmt.Sprintf("%016x", id)[1:]
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	dbif.RegisterDBIFServer(s, CreateNewDBServer())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
