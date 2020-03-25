package integration

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// DeleteIntegration deletes a integration
func DeleteIntegration(db gorp.SqlExecutor, integration sdk.ProjectIntegration) error {
	pp := dbProjectIntegration{ProjectIntegration: integration}
	if _, err := db.Delete(&pp); err != nil {
		return sdk.WrapError(err, "Cannot remove integration")
	}
	return nil
}

func load(db gorp.SqlExecutor, query gorpmapping.Query, clearPassword bool) (sdk.ProjectIntegration, error) {
	var pp dbProjectIntegration
	found, err := gorpmapping.Get(context.Background(), db, query, &pp, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return sdk.ProjectIntegration{}, err
	}
	if !found {
		return sdk.ProjectIntegration{}, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(pp, pp.Signature)
	if err != nil {
		return sdk.ProjectIntegration{}, err
	}
	if !isValid {
		log.Error(context.Background(), "integration.LoadModelByName> model  %d data corrupted", pp.ID)
		return sdk.ProjectIntegration{}, sdk.WithStack(sdk.ErrNotFound)
	}

	imodel, err := LoadModel(db, pp.IntegrationModelID, clearPassword)
	if err != nil {
		return sdk.ProjectIntegration{}, err
	}
	pp.Model = imodel

	if !clearPassword {
		pp.Blur()
	}
	return pp.ProjectIntegration, nil
}

// LoadProjectIntegrationByName Load a integration by project key and its name
func LoadProjectIntegrationByName(db gorp.SqlExecutor, key string, name string, clearPassword bool) (sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery(`
		SELECT project_integration.*
		FROM project_integration
		JOIN project ON project.id = project_integration.project_id
		WHERE project.projectkey = $1 AND project_integration.name = $2`).Args(key, name)

	pp, err := load(db, query, clearPassword)
	return pp, sdk.WithStack(err)
}

// LoadProjectIntegrationByID returns integration, selecting by its id
func LoadProjectIntegrationByID(db gorp.SqlExecutor, id int64, clearPassword bool) (*sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery("SELECT * from project_integration WHERE id = $1").Args(id)
	pp, err := load(db, query, clearPassword)
	return &pp, sdk.WithStack(err)
}

func loadAll(db gorp.SqlExecutor, query gorpmapping.Query, clearPassword bool) ([]sdk.ProjectIntegration, error) {
	var pp []dbProjectIntegration
	if err := gorpmapping.GetAll(context.Background(), db, query, &pp, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, err
	}

	var integrations = make([]sdk.ProjectIntegration, len(pp))
	for i, p := range pp {
		isValid, err := gorpmapping.CheckSignature(p, p.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(context.Background(), "integration.loadAll> model %d data corrupted", p.ID)
			continue
		}

		imodel, err := LoadModel(db, p.IntegrationModelID, clearPassword)
		if err != nil {
			return nil, err
		}
		p.Model = imodel
		integrations[i] = p.ProjectIntegration

		if !clearPassword {
			integrations[i].Blur()
		}
	}
	return integrations, nil
}

// LoadIntegrationsByProjectID load integration integrations by project id
func LoadIntegrationsByProjectID(db gorp.SqlExecutor, id int64, clearPassword bool) ([]sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery("SELECT * from project_integration WHERE project_id = $1").Args(id)
	integrations, err := loadAll(db, query, clearPassword)
	return integrations, sdk.WithStack(err)
}

// InsertIntegration inserts a integration
func InsertIntegration(db gorp.SqlExecutor, pp *sdk.ProjectIntegration) error {
	ppDb := dbProjectIntegration{ProjectIntegration: *pp}
	if err := gorpmapping.InsertAndSign(context.Background(), db, &ppDb); err != nil {
		return sdk.WrapError(err, "Cannot insert integration")
	}
	*pp = ppDb.ProjectIntegration
	return nil
}

// UpdateIntegration Update a integration
func UpdateIntegration(db gorp.SqlExecutor, pp sdk.ProjectIntegration) error {
	ppDb := dbProjectIntegration{ProjectIntegration: pp}
	if err := gorpmapping.UpdateAndSign(context.Background(), db, &ppDb); err != nil {
		return sdk.WrapError(err, "Cannot update integration")
	}
	return nil
}

// LoadIntegrationsByWorkflowID load integration integrations by Workflow id
func LoadIntegrationsByWorkflowID(db gorp.SqlExecutor, id int64, clearPassword bool) ([]sdk.ProjectIntegration, error) {
	query := gorpmapping.NewQuery(`SELECT project_integration.*
	FROM project_integration
		JOIN workflow_project_integration ON workflow_project_integration.project_integration_id = project_integration.id
	WHERE workflow_project_integration.workflow_id = $1`).Args(id)
	integrations, err := loadAll(db, query, clearPassword)
	return integrations, sdk.WithStack(err)
}

// AddOnWorkflow link a project integration on a workflow
func AddOnWorkflow(db gorp.SqlExecutor, workflowID int64, projectIntegrationID int64) error {
	query := "INSERT INTO workflow_project_integration (workflow_id, project_integration_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"
	if _, err := db.Exec(query, workflowID, projectIntegrationID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// RemoveFromWorkflow remove a project integration on a workflow
func RemoveFromWorkflow(db gorp.SqlExecutor, workflowID int64, projectIntegrationID int64) error {
	query := "DELETE FROM workflow_project_integration WHERE workflow_id = $1 AND project_integration_id = $2"
	if _, err := db.Exec(query, workflowID, projectIntegrationID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// DeleteFromWorkflow remove a project integration on a workflow
func DeleteFromWorkflow(db gorp.SqlExecutor, workflowID int64) error {
	query := "DELETE FROM workflow_project_integration WHERE workflow_id = $1"
	if _, err := db.Exec(query, workflowID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
