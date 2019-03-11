package modelstore

import (
	"github.com/kubeflow/katib/pkg/api"
	"github.com/kubeflow/katib/pkg/db"
)

type ModelStore interface {
	SaveStudy(*api.SaveStudyRequest) error
	SaveModel(*api.SaveModelRequest) error
	GetSavedStudies() ([]*db.StudyOverview, error)
	GetSavedModels(*api.GetSavedModelsRequest) ([]*api.ModelInfo, error)
	GetSavedModel(*api.GetSavedModelRequest) (*api.ModelInfo, error)
}
