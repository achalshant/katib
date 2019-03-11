package main

import (
	"context"
	crand "crypto/rand"
	"database/sql"
	"errors"
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

type GetWorkerLogOpts struct {
	Name       string
	SinceTime  *time.Time
	Descending bool
	Limit      int32
	Objective  bool
}

type WorkerLog struct {
	Time  time.Time
	Name  string
	Value string
}

/**
HELPER FUNCTIONS
**/

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
		log.Fatalf("DB open failed: %v", err)
		return nil
	}
	dbWithConn, err := NewWithSQLConn(db)
	if err != nil {
		log.Fatalf("DB open failed: %v", err)
		return nil
	}
	log.Printf("DB connection opened successfully")
	return dbWithConn
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

func DBInit(db *sql.DB) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS studies
		(id CHAR(16) PRIMARY KEY,
		name VARCHAR(255),
		owner VARCHAR(255),
		optimization_type TINYINT,
		optimization_goal DOUBLE,
		parameter_configs TEXT,
		tags TEXT,
		objective_value_name VARCHAR(255),
		metrics TEXT,
		nasconfig TEXT,
		job_id TEXT,
		job_type TEXT)`)

	if err != nil {
		log.Fatalf("Error creating studies table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS study_permissions
		(study_id CHAR(16) NOT NULL,
		access_permission VARCHAR(255),
		PRIMARY KEY (study_id, access_permission),
		FOREIGN KEY(study_id) REFERENCES studies(id) ON DELETE CASCADE)`)
	if err != nil {
		log.Fatalf("Error creating study_permissions table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS trials
		(id CHAR(16) PRIMARY KEY,
		study_id CHAR(16),
		parameters TEXT,
		objective_value VARCHAR(255),
		tags TEXT,
		time DATETIME(6),
		FOREIGN KEY(study_id) REFERENCES studies(id) ON DELETE CASCADE)`)
	if err != nil {
		log.Fatalf("Error creating trials table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS workers
		(id CHAR(16) PRIMARY KEY,
		study_id CHAR(16),
		trial_id CHAR(16),
		type VARCHAR(255),
		status TINYINT,
		template_path TEXT,
		tags TEXT,
		FOREIGN KEY(study_id) REFERENCES studies(id) ON DELETE CASCADE,
		FOREIGN KEY(trial_id) REFERENCES trials(id) ON DELETE CASCADE)`)
	if err != nil {
		log.Fatalf("Error creating workers table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS worker_metrics
		(worker_id CHAR(16) NOT NULL,
		id INT AUTO_INCREMENT PRIMARY KEY,
		time DATETIME(6),
		name VARCHAR(255),
		value TEXT,
		is_objective TINYINT,
		FOREIGN KEY (worker_id) REFERENCES workers(id) ON DELETE CASCADE)`)
	if err != nil {
		log.Fatalf("Error creating worker_metrics table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS worker_lastlogs
		(worker_id CHAR(16) PRIMARY KEY,
		time DATETIME(6),
		FOREIGN KEY (worker_id) REFERENCES workers(id) ON DELETE CASCADE)`)
	if err != nil {
		log.Fatalf("Error creating worker_lastlogs table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS suggestion_param
		(id CHAR(16) PRIMARY KEY,
		suggestion_algo TEXT,
		study_id CHAR(16),
		parameters TEXT,
		FOREIGN KEY(study_id) REFERENCES studies(id) ON DELETE CASCADE)`)
	if err != nil {
		log.Fatalf("Error creating suggestion_param table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS earlystop_param
		(id CHAR(16) PRIMARY KEY,
		earlystop_argo TEXT,
		study_id CHAR(16),
		parameters TEXT,
		FOREIGN KEY(study_id) REFERENCES studies(id) ON DELETE CASCADE)`)
	if err != nil {
		log.Fatalf("Error creating earlystop_param table: %v", err)
	}

}

func NewWithSQLConn(db *sql.DB) (*dbserver, error) {
	d := new(dbserver)
	d.db = db
	seed, err := crand.Int(crand.Reader, big.NewInt(1<<63-1))
	if err != nil {
		return nil, fmt.Errorf("RNG initialization failed: %v", err)
	}
	// We can do the following instead, but it creates a locking issue
	//d.rng = rand.New(rand.NewSource(seed.Int64()))
	DBInit(d.db)
	log.Printf("DB initialized successfully!")
	rand.Seed(seed.Int64())
	return d, nil
}

func getStudy(db *sql.DB, StudyID string) (*dbif.StudyConfig, error) {
	row := db.QueryRow("SELECT * FROM studies WHERE id = ?", StudyID)
	study := new(dbif.StudyConfig)
	var dummyID, nasConfig, parameters, tags, metrics string
	err := row.Scan(&dummyID,
		&study.Name,
		&study.Owner,
		&study.OptimizationType,
		&study.OptimizationGoal,
		&parameters,
		&tags,
		&study.ObjectiveValueName,
		&metrics,
		&nasConfig,
		&study.JobId,
		&study.JobType,
	)
	if err != nil {
		return nil, err
	}
	if parameters != "" {
		study.ParameterConfigs = new(dbif.StudyConfig_ParameterConfigs)
		err = jsonpb.UnmarshalString(parameters, study.ParameterConfigs)
		if err != nil {
			return nil, err
		}
	}
	if nasConfig != "" {
		study.NasConfig = new(dbif.NasConfig)
		err = jsonpb.UnmarshalString(nasConfig, study.NasConfig)
		if err != nil {
			log.Printf("Failed to unmarshal NasConfig")
			return nil, err
		}
	}

	var tagsArray []string
	if len(tags) > 0 {
		tagsArray = strings.Split(tags, ",\n")
	}
	study.Tags = make([]*dbif.Tag, len(tagsArray))
	for i, j := range tagsArray {
		tag := new(dbif.Tag)
		err = jsonpb.UnmarshalString(j, tag)
		if err != nil {
			log.Printf("err unmarshal %s", j)
			return nil, err
		}
		study.Tags[i] = tag
	}
	study.Metrics = strings.Split(metrics, ",\n")
	return study, nil
}

func getStudyList(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT id FROM studies")
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var result []string
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			log.Printf("err scanning studies.id: %v", err)
			continue
		}
		result = append(result, id)
	}
	return result, nil
}

func deleteStudy(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM studies WHERE id = ?", id)
	return err
}

func marshalTrial(trial *dbif.Trial) ([]string, []string, error) {
	var err, lastErr error

	params := make([]string, len(trial.ParameterSet))
	for i, elem := range trial.ParameterSet {
		params[i], err = (&jsonpb.Marshaler{}).MarshalToString(elem)
		if err != nil {
			log.Printf("Error marshalling trial.ParameterSet %v: %v",
				elem, err)
			lastErr = err
		}
	}
	tags := make([]string, len(trial.Tags))
	for i := range tags {
		tags[i], err = (&jsonpb.Marshaler{}).MarshalToString(trial.Tags[i])
		if err != nil {
			log.Printf("Error marshalling trial.Tags %v: %v",
				trial.Tags[i], err)
			lastErr = err
		}
	}
	return params, tags, lastErr
}

func createTrial(db *sql.DB, trial *dbif.Trial) error {
	params, tags, lastErr := marshalTrial(trial)

	var trialID string
	i := 3
	for true {
		trialID = generateRandid()
		createTimeString := time.Now().UTC().Format(mysqlTimeFmt)
		_, err := db.Exec("INSERT INTO trials VALUES (?, ?, ?, ?, ?, ?)",
			trialID, trial.StudyId, strings.Join(params, ",\n"),
			trial.ObjectiveValue, strings.Join(tags, ",\n"), createTimeString)
		if err == nil {
			trial.TrialId = trialID
			break
		} else if isDBDuplicateError(err) {
			i--
			if i > 0 {
				continue
			}
		}
		return err
	}
	return lastErr
}

func getTrial(db *sql.DB, id string) (*dbif.Trial, error) {
	trials, err := getTrials(db, id, "")
	if err != nil {
		return nil, err
	}

	if len(trials) > 1 {
		return nil, errors.New("multiple trials found")
	} else if len(trials) == 0 {
		return nil, errors.New("trials not found")
	}

	return trials[0], nil
}

func getTrialList(db *sql.DB, id string) ([]*dbif.Trial, error) {
	trials, err := getTrials(db, "", id)
	return trials, err
}

func getTrials(db *sql.DB, trialID string, studyID string) ([]*dbif.Trial, error) {
	var rows *sql.Rows
	var err error

	if trialID != "" {
		rows, err = db.Query("SELECT * FROM trials WHERE id = ?", trialID)
	} else if studyID != "" {
		rows, err = db.Query("SELECT * FROM trials WHERE study_id = ?", studyID)
	} else {
		return nil, errors.New("trial_id or study_id must be set")
	}

	if err != nil {
		return nil, err
	}

	var result []*dbif.Trial
	for rows.Next() {
		trial := new(dbif.Trial)

		var parameters, tags string
		var createTimeStr string
		err := rows.Scan(&trial.TrialId,
			&trial.StudyId,
			&parameters,
			&trial.ObjectiveValue,
			&tags,
			&createTimeStr,
		)
		if err != nil {
			return nil, err
		}
		params := strings.Split(parameters, ",\n")
		p := make([]*dbif.Parameter, len(params))
		for i, pstr := range params {
			if pstr == "" {
				continue
			}
			p[i] = &dbif.Parameter{}
			err := jsonpb.UnmarshalString(pstr, p[i])
			if err != nil {
				return nil, err
			}
		}
		trial.ParameterSet = p
		taglist := strings.Split(tags, ",\n")
		t := make([]*dbif.Tag, len(taglist))
		for i, tstr := range taglist {
			t[i] = &dbif.Tag{}
			if tstr == "" {
				continue
			}
			err := jsonpb.UnmarshalString(tstr, t[i])
			if err != nil {
				return nil, err
			}
		}
		trial.Tags = t
		result = append(result, trial)
	}

	return result, nil
}

func createWorker(db *sql.DB, worker *dbif.Worker) (string, error) {
	// Users should not overwrite worker.id
	var err, lastErr error
	tags := make([]string, len(worker.Tags))
	for i := range tags {
		tags[i], err = (&jsonpb.Marshaler{}).MarshalToString(worker.Tags[i])
		if err != nil {
			log.Printf("Error marshalling worker.Tags %v: %v",
				worker.Tags[i], err)
			lastErr = err
		}
	}
	var workerID string
	i := 3
	for true {
		workerID = generateRandid()
		_, err = db.Exec("INSERT INTO workers VALUES (?, ?, ?, ?, ?, ?, ?)",
			workerID, worker.StudyId, worker.TrialId, worker.Type,
			dbif.State_PENDING, worker.TemplatePath, strings.Join(tags, ",\n"))
		if err == nil {
			worker.WorkerId = workerID
			break
		} else if isDBDuplicateError(err) {
			i--
			if i > 0 {
				continue
			}
		}
		return "", err
	}
	return worker.WorkerId, lastErr

}

func getWorkers(db *sql.DB, workerID string, trialID string, studyID string) ([]*dbif.Worker, error) {
	var rows *sql.Rows
	var err error

	if workerID != "" {
		rows, err = db.Query("SELECT * FROM workers WHERE id = ?", workerID)
	} else if trialID != "" {
		rows, err = db.Query("SELECT * FROM workers WHERE trial_id = ?", trialID)
	} else if studyID != "" {
		rows, err = db.Query("SELECT * FROM workers WHERE study_id = ?", studyID)
	} else {
		return nil, errors.New("worker_id, trial_id or study_id must be set")
	}

	if err != nil {
		return nil, err
	}

	var result []*dbif.Worker
	for rows.Next() {
		worker := new(dbif.Worker)

		var tags string
		err := rows.Scan(
			&worker.WorkerId,
			&worker.StudyId,
			&worker.TrialId,
			&worker.Type,
			&worker.Status,
			&worker.TemplatePath,
			&tags,
		)
		if err != nil {
			return nil, err
		}

		taglist := strings.Split(tags, ",\n")
		t := make([]*dbif.Tag, len(taglist))
		for i, tstr := range taglist {
			t[i] = &dbif.Tag{}
			if tstr == "" {
				continue
			}
			err := jsonpb.UnmarshalString(tstr, t[i])
			if err != nil {
				return nil, err
			}
		}
		worker.Tags = t
		result = append(result, worker)
	}
	return result, nil
}

func getWorkerList(db *sql.DB, sid string, tid string) ([]*dbif.Worker, error) {
	workers, err := getWorkers(db, "", tid, sid)
	return workers, err
}

func getWorker(db *sql.DB, id string) (*dbif.Worker, error) {
	workers, err := getWorkers(db, id, "", "")
	if err != nil {
		return nil, err
	}
	if len(workers) > 1 {
		return nil, errors.New("multiple workers found")
	} else if len(workers) == 0 {
		return nil, errors.New("worker not found")
	}

	return workers[0], nil

}

func getWorkerLogs(db *sql.DB, id string, opts *GetWorkerLogOpts) ([]*WorkerLog, error) {
	qstr := ""
	qfield := []interface{}{id}
	order := ""
	if opts != nil {
		if opts.SinceTime != nil {
			qstr += " AND time >= ?"
			qfield = append(qfield, opts.SinceTime)
		}
		if opts.Name != "" {
			qstr += " AND name = ?"
			qfield = append(qfield, opts.Name)
		}
		if opts.Objective {
			qstr += " AND is_objective = 1"
		}
		if opts.Descending {
			order = " DESC"
		}
		if opts.Limit > 0 {
			order += fmt.Sprintf(" LIMIT %d", opts.Limit)
		}
	}

	rows, err := db.Query("SELECT time, name, value FROM worker_metrics WHERE worker_id = ?"+
		qstr+" ORDER BY time"+order, qfield...)
	if err != nil {
		return nil, err
	}

	var result []*WorkerLog
	for rows.Next() {
		log1 := new(WorkerLog)
		var timeStr string
		err := rows.Scan(&timeStr, &((*log1).Name), &((*log1).Value))
		if err != nil {
			log.Printf("Error scanning log: %v", err)
			continue
		}
		log1.Time, err = time.Parse(mysqlTimeFmt, timeStr)
		if err != nil {
			log.Printf("Error parsing time %s: %v", timeStr, err)
			continue
		}
		result = append(result, log1)
	}
	return result, nil
}

func getWorkerLastlogs(db *sql.DB, id string) (time.Time, []*WorkerLog, error) {
	var timeStr string
	var timeVal time.Time
	var err error

	// Use LEFT JOIN to ensure a result even if there's no matching
	// in worker_metrics.
	rows, err := db.Query(
		`SELECT worker_lastlogs.time, name, value FROM worker_lastlogs
                 LEFT JOIN worker_metrics
                 ON (worker_lastlogs.worker_id = worker_metrics.worker_id AND worker_lastlogs.time = worker_metrics.time)
                 WHERE worker_lastlogs.worker_id = ?`, id)
	if err != nil {
		return timeVal, nil, err
	}

	var result []*WorkerLog
	for rows.Next() {
		log1 := new(WorkerLog)
		var thisTime string
		var name, value sql.NullString

		err := rows.Scan(&thisTime, &name, &value)
		if err != nil {
			log.Printf("Error scanning log: %v", err)
			continue
		}
		if timeStr == "" {
			timeStr = thisTime
			timeVal, err = time.Parse(mysqlTimeFmt, timeStr)
			if err != nil {
				log.Printf("Error parsing time %s: %v", timeStr, err)
				return timeVal, nil, err
			}
		} else if timeStr != thisTime {
			log.Printf("Unexpected query result %s != %s",
				timeStr, thisTime)
		}
		log1.Time = timeVal
		if !name.Valid {
			continue
		}
		(*log1).Name = name.String
		(*log1).Value = value.String
		result = append(result, log1)
	}
	return timeVal, result, nil
}

func storeWorkerLog(db *sql.DB, workerID string, time string, metricsName string, metricsValue string, objectiveValueName string) error {
	isObjective := 0
	if metricsName == objectiveValueName {
		isObjective = 1
	}
	_, err := db.Exec("INSERT INTO worker_metrics (worker_id, time, name, value, is_objective) VALUES (?, ?, ?, ?, ?)",
		workerID, time, metricsName, metricsValue, isObjective)
	if err != nil {
		return err
	}
	return nil
}

func storeWorkerLogs(db *sql.DB, workerID string, logs []*dbif.MetricsLog) error {
	var lasterr error

	dbT, lastLogs, err := getWorkerLastlogs(db, workerID)
	if err != nil {
		log.Printf("Error getting last log timestamp: %v", err)
	}

	row := db.QueryRow("SELECT objective_value_name FROM workers "+
		"JOIN (studies) ON (workers.study_id = studies.id) WHERE "+
		"workers.id = ?", workerID)
	var objectiveValueName string
	err = row.Scan(&objectiveValueName)
	if err != nil {
		log.Printf("Cannot get objective_value_name or metrics: %v", err)
		return err
	}

	// Store logs when
	//   1. a log is newer than dbT, or,
	//   2. a log is not yet in the DB when the timestamps are equal
	var formattedTime string
	var lastTime time.Time
	for _, mlog := range logs {
		metricsName := mlog.Name
	logLoop:
		for _, mv := range mlog.Values {
			t, err := time.Parse(time.RFC3339Nano, mv.Time)
			if err != nil {
				log.Printf("Error parsing time %s: %v", mv.Time, err)
				lasterr = err
				continue
			}
			if t.Before(dbT) {
				// dbT is from mysql and has microsec precision.
				// This code assumes nanosec fractions are rounded down.
				continue
			}
			// use UTC as mysql DATETIME lacks timezone
			formattedTime = t.UTC().Format(mysqlTimeFmt)
			if !dbT.IsZero() {
				// Parse again to get rounding effect, otherwise
				// the next comparison will be almost always false.
				reparsed_time, err := time.Parse(mysqlTimeFmt, formattedTime)
				if err != nil {
					log.Printf("Error parsing time %s: %v", formattedTime, err)
					lasterr = err
					continue
				}
				if reparsed_time == dbT {
					for _, l := range lastLogs {
						if l.Name == metricsName && l.Value == mv.Value {
							continue logLoop
						}
					}
				}
			}
			err = storeWorkerLog(db, workerID,
				formattedTime,
				metricsName, mv.Value,
				objectiveValueName)
			if err != nil {
				log.Printf("Error storing log %s: %v", mv.Value, err)
				lasterr = err
			} else if t.After(lastTime) {
				lastTime = t
			}
		}
	}
	if lasterr != nil {
		// If lastlog were updated, logs that couldn't be saved
		// would be lost.
		return lasterr
	}
	if !lastTime.IsZero() {
		formattedTime = lastTime.UTC().Format(mysqlTimeFmt)
		_, err = db.Exec("REPLACE INTO worker_lastlogs VALUES (?, ?)",
			workerID, formattedTime)
	}
	return err
}

func getEarlyStopParam(db *sql.DB, paramID string) ([]*dbif.EarlyStoppingParameter, error) {
	var params string
	row := db.QueryRow("SELECT parameters FROM earlystopping_param WHERE id = ?", paramID)
	err := row.Scan(&params)
	if err != nil {
		return nil, err
	}
	var pArray []string
	if len(params) > 0 {
		pArray = strings.Split(params, ",\n")
	} else {
		return nil, nil
	}
	ret := make([]*dbif.EarlyStoppingParameter, len(pArray))
	for i, j := range pArray {
		p := new(dbif.EarlyStoppingParameter)
		err = jsonpb.UnmarshalString(j, p)
		if err != nil {
			log.Printf("err unmarshal %s", j)
			return nil, err
		}
		ret[i] = p
	}
	return ret, nil
}

func getEarlyStopParamList(db *sql.DB, studyID string) ([]*dbif.EarlyStoppingParameterSet, error) {
	var rows *sql.Rows
	var err error
	rows, err = db.Query("SELECT id, earlystop_algo, parameters FROM earlystopping_param WHERE study_id = ?", studyID)
	if err != nil {
		return nil, err
	}
	var result []*dbif.EarlyStoppingParameterSet
	for rows.Next() {
		var id string
		var algorithm string
		var params string
		err := rows.Scan(&id, &algorithm, &params)
		if err != nil {
			return nil, err
		}
		var pArray []string
		if len(params) > 0 {
			pArray = strings.Split(params, ",\n")
		} else {
			return nil, nil
		}
		esparams := make([]*dbif.EarlyStoppingParameter, len(pArray))
		for i, j := range pArray {
			p := new(dbif.EarlyStoppingParameter)
			err = jsonpb.UnmarshalString(j, p)
			if err != nil {
				log.Printf("err unmarshal %s", j)
				return nil, err
			}
			esparams[i] = p
		}
		result = append(result, &dbif.EarlyStoppingParameterSet{
			ParamId:                 id,
			EarlyStoppingAlgorithm:  algorithm,
			EarlyStoppingParameters: esparams,
		})
	}
	return result, nil
}

func setEarlyStopParam(db *sql.DB, algorithm string, studyID string, params []*dbif.EarlyStoppingParameter) (string, error) {
	ps := make([]string, len(params))
	var err error
	for i, elem := range params {
		ps[i], err = (&jsonpb.Marshaler{}).MarshalToString(elem)
		if err != nil {
			log.Printf("Error marshalling %v: %v", elem, err)
			return "", err
		}
	}
	var paramID string
	for true {
		paramID = generateRandid()
		_, err = db.Exec("INSERT INTO earlystopping_param VALUES (?,?, ?, ?)",
			paramID, algorithm, studyID, strings.Join(ps, ",\n"))
		if err == nil {
			break
		} else if !isDBDuplicateError(err) {
			return "", err
		}
	}
	return paramID, nil
}

func updateEarlyStopParam(db *sql.DB, paramID string, params []*dbif.EarlyStoppingParameter) error {
	ps := make([]string, len(params))
	var err error
	for i, elem := range params {
		ps[i], err = (&jsonpb.Marshaler{}).MarshalToString(elem)
		if err != nil {
			log.Printf("Error marshalling %v: %v", elem, err)
			return err
		}
	}
	_, err = db.Exec("UPDATE earlystopping_param SET parameters = ? WHERE id = ?",
		strings.Join(ps, ",\n"), paramID)
	return err
}

func getSuggestionParam(db *sql.DB, paramID string) ([]*dbif.SuggestionParameter, error) {
	var params string
	row := db.QueryRow("SELECT parameters FROM suggestion_param WHERE id = ?", paramID)
	err := row.Scan(&params)
	if err != nil {
		return nil, err
	}
	var pArray []string
	if len(params) > 0 {
		pArray = strings.Split(params, ",\n")
	} else {
		return nil, nil
	}
	ret := make([]*dbif.SuggestionParameter, len(pArray))
	for i, j := range pArray {
		p := new(dbif.SuggestionParameter)
		err = jsonpb.UnmarshalString(j, p)
		if err != nil {
			log.Printf("err unmarshal %s", j)
			return nil, err
		}
		ret[i] = p
	}
	return ret, nil
}

func setSuggestionParam(db *sql.DB, algorithm string, studyID string, params []*dbif.SuggestionParameter) (string, error) {
	var err error
	ps := make([]string, len(params))
	for i, elem := range params {
		ps[i], err = (&jsonpb.Marshaler{}).MarshalToString(elem)
		if err != nil {
			log.Printf("Error marshalling %v: %v", elem, err)
			return "", err
		}
	}
	var paramID string
	for true {
		paramID = generateRandid()
		_, err = db.Exec("INSERT INTO suggestion_param VALUES (?, ?, ?, ?)",
			paramID, algorithm, studyID, strings.Join(ps, ",\n"))
		if err == nil {
			break
		} else if !isDBDuplicateError(err) {
			return "", err
		}
	}
	return paramID, err
}

func updateSuggestionParam(db *sql.DB, paramID string, params []*dbif.SuggestionParameter) error {
	var err error
	ps := make([]string, len(params))
	for i, elem := range params {
		ps[i], err = (&jsonpb.Marshaler{}).MarshalToString(elem)
		if err != nil {
			log.Printf("Error marshalling %v: %v", elem, err)
			return err
		}
	}
	_, err = db.Exec("UPDATE suggestion_param SET parameters = ? WHERE id = ?",
		strings.Join(ps, ",\n"), paramID)
	return err
}

func getSuggestionParamList(db *sql.DB, studyID string) ([]*dbif.SuggestionParameterSet, error) {
	var rows *sql.Rows
	var err error
	rows, err = db.Query("SELECT id, suggestion_algo, parameters FROM suggestion_param WHERE study_id = ?", studyID)
	if err != nil {
		return nil, err
	}
	var result []*dbif.SuggestionParameterSet
	for rows.Next() {
		var id string
		var algorithm string
		var params string
		err := rows.Scan(&id, &algorithm, &params)
		if err != nil {
			return nil, err
		}
		var pArray []string
		if len(params) > 0 {
			pArray = strings.Split(params, ",\n")
		} else {
			return nil, nil
		}
		suggestparams := make([]*dbif.SuggestionParameter, len(pArray))
		for i, j := range pArray {
			p := new(dbif.SuggestionParameter)
			err = jsonpb.UnmarshalString(j, p)
			if err != nil {
				log.Printf("err unmarshal %s", j)
				return nil, err
			}
			suggestparams[i] = p
		}
		result = append(result, &dbif.SuggestionParameterSet{
			ParamId:              id,
			SuggestionAlgorithm:  algorithm,
			SuggestionParameters: suggestparams,
		})
	}
	return result, nil
}

func getStudyMetrics(db *sql.DB, id string) ([]string, error) {
	row := db.QueryRow("SELECT metrics FROM studies WHERE id = ?", id)
	var metrics string
	err := row.Scan(&metrics)
	if err != nil {
		return nil, err
	}
	retMetrics := strings.Split(metrics, ",\n")
	return retMetrics, nil
}

func getWorkerFullInfo(db *sql.DB, studyId string, trialId string, workerId string, OnlyLatestLog bool) (*dbif.GetWorkerFullInfoReply, error) {
	ret := &dbif.GetWorkerFullInfoReply{}
	var err error
	ws := []*dbif.Worker{}

	if workerId != "" {
		w, err := getWorker(db, workerId)
		ws = append(ws, w)
		if err != nil {
			return ret, err
		}
	} else {
		ws, err = getWorkerList(db, studyId, trialId)
		if err != nil {
			return ret, err
		}
	}
	ts, err := getTrialList(db, studyId)
	if err != nil {
		return ret, err
	}
	// Actually no need to get full config now
	metrics, err := getStudyMetrics(db, studyId)
	if err != nil {
		return ret, err
	}

	plist := make(map[string][]*dbif.Parameter)
	for _, t := range ts {
		plist[t.TrialId] = t.ParameterSet
	}

	wfilist := make([]*dbif.WorkerFullInfo, len(ws))
	var qstr, id string
	if OnlyLatestLog {
		qstr = `
		SELECT 
			WM.worker_id, WM.time, WM.name, WM.value 
		FROM (
			SELECT 
				Master.worker_id, Master.time,  Master.name,  Master.value
			FROM (
				SELECT 
					worker_id, name, 
					MAX(id) AS MaxID
				FROM 
					worker_metrics 
				GROUP BY 
					worker_id, name
				) AS LATEST
				JOIN worker_metrics AS Master
				ON Master.id = LATEST.MaxID
		) AS WM 
		JOIN workers AS WS 
		ON WM.worker_id = WS.id 
		AND`
	} else {
		qstr = `
		SELECT 
			WM.worker_id, WM.time, WM.name, WM.value 
		FROM 
			worker_metrics AS WM 
		JOIN workers AS WS 
		ON WM.worker_id = WS.id 
		AND`
	}
	if workerId != "" {
		if OnlyLatestLog {
			qstr = `
			SELECT 
			WM.worker_id, WM.time, WM.name, WM.value 
			FROM (
				SELECT 
					Master.worker_id, Master.time,  Master.name,  Master.value
				FROM (
					SELECT 
						worker_id, name, 
						MAX(id) AS MaxID
					FROM 
						worker_metrics 
					GROUP BY 
						worker_id, name
				) AS LATEST
				JOIN worker_metrics AS Master
				ON Master.id = LATEST.MaxID
				AND Master.worker_id = ?
			) AS WM`
		} else {
			qstr = "SELECT worker_id, time, name, value FROM worker_metrics WHERE worker_id = ?"
		}
		id = workerId
	} else if trialId != "" {
		qstr += " WS.trial_id = ? "
		id = trialId
	} else if studyId != "" {
		qstr += " WS.study_id = ? "
		id = studyId
	}
	rows, err := db.Query(qstr+" ORDER BY time", id)
	if err != nil {
		log.Printf("SQL query: %v", err)
		return ret, err
	}
	metricslist := make(map[string]map[string][]*dbif.MetricsValueTime, len(ws))
	for rows.Next() {
		var name, value, timeStr, wid string
		err := rows.Scan(&wid, &timeStr, &name, &value)
		if err != nil {
			log.Printf("Error scanning log: %v", err)
			continue
		}
		ptime, err := time.Parse(mysqlTimeFmt, timeStr)
		if err != nil {
			log.Printf("Error parsing time %s: %v", timeStr, err)
			continue
		}
		if _, ok := metricslist[wid]; ok {
			metricslist[wid][name] = append(metricslist[wid][name], &dbif.MetricsValueTime{
				Value: value,
				Time:  ptime.UTC().Format(time.RFC3339Nano),
			})
		} else {
			metricslist[wid] = make(map[string][]*dbif.MetricsValueTime, len(metrics))
			metricslist[wid][name] = append(metricslist[wid][name], &dbif.MetricsValueTime{
				Value: value,
				Time:  ptime.UTC().Format(time.RFC3339Nano),
			})
		}
	}
	for i, w := range ws {
		wfilist[i] = &dbif.WorkerFullInfo{
			Worker:       w,
			ParameterSet: plist[w.TrialId],
		}
		for _, m := range metrics {
			if v, ok := metricslist[w.WorkerId][m]; ok {
				wfilist[i].MetricsLogs = append(wfilist[i].MetricsLogs, &dbif.MetricsLog{
					Name:   m,
					Values: v,
				},
				)
			}
		}
	}
	ret.WorkerFullInfos = wfilist
	return ret, nil
}

func updateWorker(db *sql.DB, id string, newstatus dbif.State) error {
	_, err := db.Exec("UPDATE workers SET status = ? WHERE id = ?", newstatus, id)
	return err
}

/**
INTERFACE FUNCTIONS
**/

func (s *dbserver) CreateStudy(ctx context.Context, in *dbif.CreateStudyRequest) (*dbif.CreateStudyReply, error) {
	if in == nil || in.StudyConfig == nil {
		return &dbif.CreateStudyReply{}, errors.New("StudyConfig is missing.")
	}
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

func (s *dbserver) GetStudy(ctx context.Context, in *dbif.GetStudyRequest) (*dbif.GetStudyReply, error) {
	row := s.db.QueryRow("SELECT * FROM studies WHERE id = ?", in.StudyId)
	study := new(dbif.StudyConfig)
	var dummyID, nasConfig, parameters, tags, metrics string
	err := row.Scan(&dummyID,
		&study.Name,
		&study.Owner,
		&study.OptimizationType,
		&study.OptimizationGoal,
		&parameters,
		&tags,
		&study.ObjectiveValueName,
		&metrics,
		&nasConfig,
		&study.JobId,
		&study.JobType,
	)
	if err != nil {
		log.Fatalf("Error in fetching")
		return &dbif.GetStudyReply{}, err
	}
	if parameters != "" {
		study.ParameterConfigs = new(dbif.StudyConfig_ParameterConfigs)
		err = jsonpb.UnmarshalString(parameters, study.ParameterConfigs)
		if err != nil {
			log.Fatalf("Error in unmarshaling json")
			return &dbif.GetStudyReply{}, err
		}
	}
	if nasConfig != "" {
		study.NasConfig = new(dbif.NasConfig)
		err = jsonpb.UnmarshalString(nasConfig, study.NasConfig)
		if err != nil {
			log.Printf("Failed to unmarshal NasConfig")
			return &dbif.GetStudyReply{}, err
		}
	}

	var tagsArray []string
	if len(tags) > 0 {
		tagsArray = strings.Split(tags, ",\n")
	}
	study.Tags = make([]*dbif.Tag, len(tagsArray))
	for i, j := range tagsArray {
		tag := new(dbif.Tag)
		err = jsonpb.UnmarshalString(j, tag)
		if err != nil {
			log.Printf("err unmarshal %s", j)
			return &dbif.GetStudyReply{}, err
		}
		study.Tags[i] = tag
	}
	study.Metrics = strings.Split(metrics, ",\n")
	return &dbif.GetStudyReply{StudyConfig: study}, nil
}

func (s *dbserver) GetStudyList(ctx context.Context, in *dbif.GetStudyListRequest) (*dbif.GetStudyListReply, error) {
	sl, err := getStudyList(s.db)
	if err != nil {
		return &dbif.GetStudyListReply{}, err
	}
	result := make([]*dbif.StudyOverview, len(sl))
	for i, id := range sl {
		sc, err := getStudy(s.db, id)
		if err != nil {
			return &dbif.GetStudyListReply{}, err
		}
		result[i] = &dbif.StudyOverview{
			Name:  sc.Name,
			Owner: sc.Owner,
			Id:    id,
		}
	}
	return &dbif.GetStudyListReply{StudyOverviews: result}, err
}

func (s *dbserver) DeleteStudy(ctx context.Context, in *dbif.DeleteStudyRequest) (*dbif.DeleteStudyReply, error) {
	if in == nil || in.StudyId == "" {
		return &dbif.DeleteStudyReply{}, errors.New("StudyId is missing.")
	}
	err := deleteStudy(s.db, in.StudyId)
	if err != nil {
		return &dbif.DeleteStudyReply{}, err
	}
	return &dbif.DeleteStudyReply{StudyId: in.StudyId}, nil
}

func (s *dbserver) CreateTrial(ctx context.Context, in *dbif.CreateTrialRequest) (*dbif.CreateTrialReply, error) {
	err := createTrial(s.db, in.Trial)
	return &dbif.CreateTrialReply{TrialId: in.Trial.TrialId}, err
}

func (s *dbserver) GetTrials(ctx context.Context, in *dbif.GetTrialsRequest) (*dbif.GetTrialsReply, error) {
	tl, err := getTrialList(s.db, in.StudyId)
	return &dbif.GetTrialsReply{Trials: tl}, err
}

func (s *dbserver) GetTrial(ctx context.Context, in *dbif.GetTrialRequest) (*dbif.GetTrialReply, error) {
	t, err := getTrial(s.db, in.TrialId)
	return &dbif.GetTrialReply{Trial: t}, err
}

func (s *dbserver) RegisterWorker(ctx context.Context, in *dbif.RegisterWorkerRequest) (*dbif.RegisterWorkerReply, error) {
	wid, err := createWorker(s.db, in.Worker)
	return &dbif.RegisterWorkerReply{WorkerId: wid}, err
}

func (s *dbserver) GetWorkers(ctx context.Context, in *dbif.GetWorkersRequest) (*dbif.GetWorkersReply, error) {
	var ws []*dbif.Worker
	var err error
	if in.WorkerId == "" {
		ws, err = getWorkerList(s.db, in.StudyId, in.TrialId)
	} else {
		var w *dbif.Worker
		w, err = getWorker(s.db, in.WorkerId)
		ws = append(ws, w)
	}
	return &dbif.GetWorkersReply{Workers: ws}, err
}

func (s *dbserver) GetMetrics(ctx context.Context, in *dbif.GetMetricsRequest) (*dbif.GetMetricsReply, error) {
	var mNames []string
	if in.StudyId == "" {
		return &dbif.GetMetricsReply{}, errors.New("StudyId should be set")
	}
	sc, err := getStudy(s.db, in.StudyId)
	if err != nil {
		return &dbif.GetMetricsReply{}, err
	}
	if len(in.MetricsNames) > 0 {
		mNames = in.MetricsNames
	} else {
		mNames = sc.Metrics
	}
	if err != nil {
		return &dbif.GetMetricsReply{}, err
	}
	mls := make([]*dbif.MetricsLogSet, len(in.WorkerIds))
	for i, w := range in.WorkerIds {
		wr, err := s.GetWorkers(ctx, &dbif.GetWorkersRequest{
			StudyId:  in.StudyId,
			WorkerId: w,
		})
		if err != nil {
			return &dbif.GetMetricsReply{}, err
		}
		mls[i] = &dbif.MetricsLogSet{
			WorkerId:     w,
			MetricsLogs:  make([]*dbif.MetricsLog, len(mNames)),
			WorkerStatus: wr.Workers[0].Status,
		}
		for j, m := range mNames {
			ls, err := getWorkerLogs(s.db, w, &GetWorkerLogOpts{Name: m})
			if err != nil {
				return &dbif.GetMetricsReply{}, err
			}
			mls[i].MetricsLogs[j] = &dbif.MetricsLog{
				Name:   m,
				Values: make([]*dbif.MetricsValueTime, len(ls)),
			}
			for k, l := range ls {
				mls[i].MetricsLogs[j].Values[k] = &dbif.MetricsValueTime{
					Value: l.Value,
					Time:  l.Time.UTC().Format(time.RFC3339Nano),
				}
			}
		}
	}
	return &dbif.GetMetricsReply{MetricsLogSets: mls}, nil
}

func (s *dbserver) ReportMetricsLogs(ctx context.Context, in *dbif.ReportMetricsLogsRequest) (*dbif.ReportMetricsLogsReply, error) {
	for _, mls := range in.MetricsLogSets {
		err := storeWorkerLogs(s.db, mls.WorkerId, mls.MetricsLogs)
		if err != nil {
			return &dbif.ReportMetricsLogsReply{}, err
		}
	}
	return &dbif.ReportMetricsLogsReply{}, nil
}

func (s *dbserver) GetEarlyStoppingParameters(ctx context.Context, in *dbif.GetEarlyStoppingParametersRequest) (*dbif.GetEarlyStoppingParametersReply, error) {
	ps, err := getEarlyStopParam(s.db, in.ParamId)
	return &dbif.GetEarlyStoppingParametersReply{EarlyStoppingParameters: ps}, err
}

func (s *dbserver) GetEarlyStoppingParameterList(ctx context.Context, in *dbif.GetEarlyStoppingParameterListRequest) (*dbif.GetEarlyStoppingParameterListReply, error) {
	pss, err := getEarlyStopParamList(s.db, in.StudyId)
	return &dbif.GetEarlyStoppingParameterListReply{EarlyStoppingParameterSets: pss}, err
}

func (s *dbserver) SetSuggestionParameters(ctx context.Context, in *dbif.SetSuggestionParametersRequest) (*dbif.SetSuggestionParametersReply, error) {
	var err error
	var id string
	if in.ParamId == "" {
		id, err = setSuggestionParam(s.db, in.SuggestionAlgorithm, in.StudyId, in.SuggestionParameters)
	} else {
		id = in.ParamId
		err = updateSuggestionParam(s.db, in.ParamId, in.SuggestionParameters)
	}
	return &dbif.SetSuggestionParametersReply{ParamId: id}, err
}

func (s *dbserver) SetEarlyStoppingParameters(ctx context.Context, in *dbif.SetEarlyStoppingParametersRequest) (*dbif.SetEarlyStoppingParametersReply, error) {
	var err error
	var id string
	if in.ParamId == "" {
		id, err = setEarlyStopParam(s.db, in.EarlyStoppingAlgorithm, in.StudyId, in.EarlyStoppingParameters)
	} else {
		id = in.ParamId
		err = updateEarlyStopParam(s.db, in.ParamId, in.EarlyStoppingParameters)
	}
	return &dbif.SetEarlyStoppingParametersReply{ParamId: id}, err
}

func (s *dbserver) GetSuggestionParameters(ctx context.Context, in *dbif.GetSuggestionParametersRequest) (*dbif.GetSuggestionParametersReply, error) {
	ps, err := getSuggestionParam(s.db, in.ParamId)
	return &dbif.GetSuggestionParametersReply{SuggestionParameters: ps}, err
}

func (s *dbserver) GetSuggestionParameterList(ctx context.Context, in *dbif.GetSuggestionParameterListRequest) (*dbif.GetSuggestionParameterListReply, error) {
	pss, err := getSuggestionParamList(s.db, in.StudyId)
	return &dbif.GetSuggestionParameterListReply{SuggestionParameterSets: pss}, err
}

func (s *dbserver) UpdateWorkerState(ctx context.Context, in *dbif.UpdateWorkerStateRequest) (*dbif.UpdateWorkerStateReply, error) {
	err := updateWorker(s.db, in.WorkerId, in.Status)
	return &dbif.UpdateWorkerStateReply{}, err
}

func (s *dbserver) GetWorkerFullInfo(ctx context.Context, in *dbif.GetWorkerFullInfoRequest) (*dbif.GetWorkerFullInfoReply, error) {
	return getWorkerFullInfo(s.db, in.StudyId, in.TrialId, in.WorkerId, in.OnlyLatestLog)
}

func (s *dbserver) SelectOne(ctx context.Context, in *dbif.SelectOneRequest) (*dbif.SelectOneReply, error) {
	_, err := s.db.Exec(`SELECT 1`)
	if err != nil {
		return &dbif.SelectOneReply{}, fmt.Errorf("Error `SELECT 1` probing: %v", err)
	}
	return &dbif.SelectOneReply{}, nil
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
