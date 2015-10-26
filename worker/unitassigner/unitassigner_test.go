// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package unitassigner

import (
	"errors"

	"github.com/juju/names"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/api/watcher"
	"github.com/juju/juju/apiserver/params"
)

var _ = gc.Suite(testsuite{})

type testsuite struct{}

func (testsuite) TestSetup(c *gc.C) {
	f := &fakeAPI{}
	ua := unitAssigner{api: f}
	_, err := ua.SetUp()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(f.calledWatch, jc.IsTrue)

	f.err = errors.New("boo")
	_, err = ua.SetUp()
	c.Assert(err, gc.Equals, f.err)
}

func (testsuite) TestHandle(c *gc.C) {
	f := &fakeAPI{}
	ua := unitAssigner{api: f}
	ids := []string{"foo/0", "bar/0"}
	err := ua.Handle(ids)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(f.assignTags, gc.DeepEquals, []names.UnitTag{
		names.NewUnitTag("foo/0"),
		names.NewUnitTag("bar/0"),
	})

	f.err = errors.New("boo")
	err = ua.Handle(ids)
	c.Assert(err, gc.Equals, f.err)
}

type fakeAPI struct {
	calledWatch bool
	assignTags  []names.UnitTag
	err         error
	status      params.SetStatus
}

func (f *fakeAPI) AssignUnits(tags []names.UnitTag) ([]error, error) {
	f.assignTags = tags
	return nil, f.err
}

func (f *fakeAPI) WatchUnitAssignments() (watcher.StringsWatcher, error) {
	f.calledWatch = true
	return nil, f.err
}

func (f *fakeAPI) SetAgentStatus(args params.SetStatus) error {
	f.status = args
	return f.err
}
