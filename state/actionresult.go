// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"fmt"
	"strings"

	"github.com/juju/names"
	"gopkg.in/mgo.v2/txn"
)

// ActionStatus represents the possible end states for an action.
type ActionStatus string

const (
	// ActionFailed signifies that the action did not complete successfully.
	ActionFailed ActionStatus = "fail"

	// ActionCompleted indicates that the action ran to completion as intended.
	ActionCompleted ActionStatus = "complete"
)

const actionResultMarker string = "_ar_"

type actionResultDoc struct {
	// DocId is the key for this document.  The format of the id encodes
	// the id of the Action that was used to produce this ActionResult.
	// The format is: <action id> + actionResultMarker + <generated sequence>
	// The <action id> encodes the EnvUUID.
	DocId string `bson:"_id"`

	// EnvUUID is the environment identifier.
	EnvUUID string `bson:"env-uuid"`

	// Receiver is the Name of the Unit or any other ActionReceiver for
	// which this Action is queued.
	Receiver string `bson:"receiver"`

	// Sequence is the unique identifier for this instance of this Action,
	// and is encoded in the DocId too.
	Sequence int `bson:"sequence"`

	// Name identifies the action that was run.
	Name string `bson:"name"`

	// Parameters describes the parameters passed in for the action
	// when it was run.
	Parameters map[string]interface{} `bson:"parameters"`

	// Status represents the end state of the Action; ActionFailed for an
	// action that was removed prematurely, or that failed, and
	// ActionCompleted for an action that successfully completed.
	Status ActionStatus `bson:"status"`

	// Message captures any error returned by the action.
	Message string `bson:"message"`

	// Results are the structured results from the action.
	Results map[string]interface{} `bson:"results"`
}

// ActionResult represents an instruction to do some "action" and is
// expected to match an action definition in a charm.
type ActionResult struct {
	st  *State
	doc actionResultDoc
}

// Id returns the local id of the ActionResult
func (a *ActionResult) Id() string {
	return a.st.localID(a.doc.DocId)
}

// Receiver  returns the Name of the ActionReceiver for which this action
// is enqueued.  Usually this is a Unit Name()
func (a *ActionResult) Receiver() string {
	return a.doc.Receiver
}

// Sequence returns the unique suffix of the ActionResult _id
func (a *ActionResult) Sequence() int {
	return a.doc.Sequence
}

// Name returns the name of the Action.
func (a *ActionResult) Name() string {
	return a.doc.Name
}

// Parameters will contain a structure representing arguments or parameters
// that were passed to the action.
func (a *ActionResult) Parameters() map[string]interface{} {
	return a.doc.Parameters
}

// Status returns the final state of the action.
func (a *ActionResult) Status() ActionStatus {
	return a.doc.Status
}

// Results returns the structured output of the action and any error.
func (a *ActionResult) Results() (map[string]interface{}, string) {
	return a.doc.Results, a.doc.Message
}

// Tag implements the Entity interface and returns a names.Tag that
// is a names.ActionResultTag
func (a *ActionResult) Tag() names.Tag {
	return a.ActionResultTag()
}

// ActionResultTag returns an ActionResultTag constructed from this
// actionResult's Prefix and Sequence
func (a *ActionResult) ActionResultTag() names.ActionResultTag {
	return names.NewActionResultTag(a.Id())
}

// newActionResult builds an ActionResult from the supplied state and
// actionResultDoc.
func newActionResult(st *State, adoc actionResultDoc) *ActionResult {
	return &ActionResult{
		st:  st,
		doc: adoc,
	}
}

// newActionResultDoc converts an Action into an actionResultDoc given
// the finalStatus and the output of the action, and any error.
func newActionResultDoc(a *Action, finalStatus ActionStatus, results map[string]interface{}, message string) actionResultDoc {
	actionId := a.Id()
	id, ok := convertActionIdToActionResultId(actionId)
	if !ok {
		panic(fmt.Sprintf("cannot convert actionId to actionResultId: %v", actionId))
	}
	return actionResultDoc{
		DocId:      a.st.docID(id),
		EnvUUID:    a.doc.EnvUUID,
		Receiver:   a.doc.Receiver,
		Sequence:   a.doc.Sequence,
		Name:       a.doc.Name,
		Parameters: a.doc.Parameters,
		Status:     finalStatus,
		Results:    results,
		Message:    message,
	}
}

// convertActionIdToActionResultId builds an actionResultId from an actionId.
func convertActionIdToActionResultId(actionId string) (string, bool) {
	parts := strings.Split(actionId, actionMarker)
	if len(parts) != 2 {
		return "", false
	}
	actionResultId := strings.Join(parts, actionResultMarker)
	return actionResultId, true
}

// actionResultPrefix returns a string prefix for matching action results for
// the given ActionReceiver.
func actionResultPrefix(ar ActionReceiver) string {
	return ar.Name() + actionResultMarker
}

// addActionResultOp builds the txn.Op used to add an actionresult.
func addActionResultOp(st *State, doc *actionResultDoc) txn.Op {
	return txn.Op{
		C:      actionresultsC,
		Id:     doc.DocId,
		Assert: txn.DocMissing,
		Insert: doc,
	}
}
