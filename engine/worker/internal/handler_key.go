package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func keyInstallHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		keyName := vars["key"]

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			sdkerr := sdk.NewError(sdk.ErrWrongRequest, err).(sdk.Error)
			writeJSON(w, sdkerr, sdkerr.Status)
			return
		}

		defer r.Body.Close() // nolint

		var mapBody = make(map[string]string)
		if len(body) > 0 {
			if err := json.Unmarshal(body, &mapBody); err != nil {
				sdkerr := sdk.Error{
					Status:  sdk.ErrWrongRequest.Status,
					Message: err.Error()}
				writeJSON(w, sdkerr, sdkerr.Status)
				return
			}
		}

		var key *sdk.Variable

		if wk.currentJob.secrets == nil {
			err := sdk.Error{
				Message: "Cannot find any keys for your job",
				Status:  http.StatusBadRequest,
			}
			log.Error(ctx, "%v", err)
			writeJSON(w, err, err.Status)
			return
		}

		for _, k := range wk.currentJob.secrets {
			if k.Name == ("cds.key." + keyName + ".priv") {
				key = &k
				break
			}
		}

		if key == nil {
			err := sdk.Error{
				Message: fmt.Sprintf("Key %s not found", keyName),
				Status:  http.StatusNotFound,
			}
			log.Error(ctx, "%v", err)
			writeJSON(w, err, err.Status)
			return
		}

		filename := mapBody["file"]
		response, err := keyInstall(wk, filename, key)
		if err != nil {
			log.Error(ctx, "Unable to install key %s: %v", key.Name, err)
			if sdkerr, ok := err.(*sdk.Error); ok {
				writeJSON(w, sdkerr, sdkerr.Status)
			} else {
				err := sdk.Error{
					Message: err.Error(),
					Status:  sdk.ErrUnknownError.Status,
				}
				writeJSON(w, err, err.Status)
			}
			return
		}
		log.Debug("key %s installed to %s", key.Name, response.PKey)
		writeJSON(w, response, 200)
	}
}

func keyInstall(wk workerruntime.Runtime, filename string, key *sdk.Variable) (*workerruntime.KeyResponse, error) {
	if filename == "" {
		return wk.InstallKey(*key)
	}

	log.Debug("worker.keyInstall> installing key %s to %s", key.Name, filename)

	if !sdk.PathIsAbs(filename) {
		return nil, fmt.Errorf("unsupported relative path")
	}

	log.Debug("worker.keyInstall> destination: %s", filename)
	return wk.InstallKeyTo(*key, filename)
}
