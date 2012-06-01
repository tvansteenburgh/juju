package state

import (
	. "launchpad.net/gocheck"
	"launchpad.net/goyaml"
	"launchpad.net/gozk/zookeeper"
)

type TopologySuite struct {
	zkServer   *zookeeper.Server
	zkTestRoot string
	zkTestPort int
	zkAddr     string
	zkConn     *zookeeper.Conn
	t          *topology
}

var _ = Suite(&TopologySuite{})
var TestingZkAddr string

func (s *TopologySuite) SetUpSuite(c *C) {
	st, err := Initialize(&Info{
		Addrs: []string{TestingZkAddr},
	})
	c.Assert(err, IsNil)
	s.zkConn = ZkConn(st)
}

func (s *TopologySuite) TearDownSuite(c *C) {
	err := zkRemoveTree(s.zkConn, "/")
	c.Assert(err, IsNil)
	s.zkConn.Close()
}

func (s *TopologySuite) SetUpTest(c *C) {
	var err error
	s.t, err = readTopology(s.zkConn)
	c.Assert(err, IsNil)
}

func (s *TopologySuite) TearDownTest(c *C) {
	// Clear out the topology node.
	err := zkRemoveTree(s.zkConn, "/topology")
	c.Assert(err, IsNil)
}

func (s *TopologySuite) TestAddMachine(c *C) {
	// Check that adding machines works correctly.
	err := s.t.AddMachine("m-0")
	c.Assert(err, IsNil)
	err = s.t.AddMachine("m-1")
	c.Assert(err, IsNil)
	keys := s.t.MachineKeys()
	c.Assert(keys, DeepEquals, []string{"m-0", "m-1"})
}

func (s *TopologySuite) TestAddDuplicatedMachine(c *C) {
	// Check that adding a duplicated machine by key fails.
	err := s.t.AddMachine("m-0")
	c.Assert(err, IsNil)
	err = s.t.AddMachine("m-0")
	c.Assert(err, ErrorMatches, `attempted to add duplicated machine "m-0"`)
}

func (s *TopologySuite) TestRemoveMachine(c *C) {
	// Check that removing machines works correctly.
	err := s.t.AddMachine("m-0")
	c.Assert(err, IsNil)
	err = s.t.AddMachine("m-1")
	c.Assert(err, IsNil)
	// Add non-assigned unit. This tests that the logic of
	// checking for assigned units works correctly too.
	err = s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-0")

	c.Assert(err, IsNil)

	err = s.t.RemoveMachine("m-0")
	c.Assert(err, IsNil)

	found := s.t.HasMachine("m-0")
	c.Assert(found, Equals, false)
	found = s.t.HasMachine("m-1")
	c.Assert(found, Equals, true)
}

func (s *TopologySuite) TestRemoveNonExistentMachine(c *C) {
	// Check that the removing of a non-existent machine fails.
	err := s.t.RemoveMachine("m-0")
	c.Assert(err, ErrorMatches, `machine with key "m-0" not found`)
}

func (s *TopologySuite) TestRemoveMachineWithAssignedUnits(c *C) {
	// Check that a machine can't be removed when it has assigned units.
	err := s.t.AddMachine("m-0")
	c.Assert(err, IsNil)
	err = s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-0")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-1")
	c.Assert(err, IsNil)
	err = s.t.AssignUnitToMachine("unit-0-1", "m-0")
	c.Assert(err, IsNil)
	err = s.t.RemoveMachine("m-0")
	c.Assert(err, ErrorMatches, `can't remove machine "m-0" while units ared assigned`)
}

func (s *TopologySuite) TestMachineHasUnits(c *C) {
	// Check various ways a machine might or might not be assigned
	// to a unit.
	err := s.t.AddMachine("m-0")
	c.Assert(err, IsNil)
	err = s.t.AddMachine("m-1")
	c.Assert(err, IsNil)
	err = s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-0")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-1")
	c.Assert(err, IsNil)
	err = s.t.AssignUnitToMachine("unit-0-1", "m-0")
	c.Assert(err, IsNil)
	ok, err := s.t.MachineHasUnits("m-0")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	ok, err = s.t.MachineHasUnits("m-1")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, false)
	ok, err = s.t.MachineHasUnits("m-99")
	c.Assert(err, ErrorMatches, `machine with key "m-99" not found`)
}

func (s *TopologySuite) TestHasMachine(c *C) {
	// Check that the test for a machine works correctly.
	found := s.t.HasMachine("m-0")
	c.Assert(found, Equals, false)
	err := s.t.AddMachine("m-0")
	c.Assert(err, IsNil)
	found = s.t.HasMachine("m-0")
	c.Assert(found, Equals, true)
	found = s.t.HasMachine("m-1")
	c.Assert(found, Equals, false)
}

func (s *TopologySuite) TestMachineKeys(c *C) {
	// Check that the retrieval of all services keys works correctly.
	keys := s.t.MachineKeys()
	c.Assert(keys, DeepEquals, []string{})
	err := s.t.AddMachine("m-0")
	c.Assert(err, IsNil)
	err = s.t.AddMachine("m-1")
	c.Assert(err, IsNil)
	keys = s.t.MachineKeys()
	c.Assert(keys, DeepEquals, []string{"m-0", "m-1"})
}

func (s *TopologySuite) TestAddService(c *C) {
	// Check that adding services works correctly.
	c.Assert(s.t.HasService("service-0"), Equals, false)
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddService("service-1", "mysql")
	c.Assert(err, IsNil)
	c.Assert(s.t.HasService("service-0"), Equals, true)
	c.Assert(s.t.HasService("service-1"), Equals, true)
}

func (s *TopologySuite) TestAddDuplicatedService(c *C) {
	// Check that adding a duplicated service by key or name fails.
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddService("service-0", "mysql")
	c.Assert(err, ErrorMatches, `attempted to add duplicated service "service-0"`)
	err = s.t.AddService("service-1", "wordpress")
	c.Assert(err, ErrorMatches, `service name "wordpress" already in use`)
}

func (s *TopologySuite) TestHasService(c *C) {
	// Check that the test for a service works correctly.
	found := s.t.HasService("service-0")
	c.Assert(found, Equals, false)
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	found = s.t.HasService("service-0")
	c.Assert(found, Equals, true)
	found = s.t.HasService("service-1")
	c.Assert(found, Equals, false)
}

func (s *TopologySuite) TestServiceKey(c *C) {
	// Check that the key retrieval for a service name works correctly.
	key, err := s.t.ServiceKey("wordpress")
	c.Assert(err, ErrorMatches, `service with name "wordpress" not found`)
	err = s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	key, err = s.t.ServiceKey("wordpress")
	c.Assert(err, IsNil)
	c.Assert(key, Equals, "service-0")
}

func (s *TopologySuite) TestServiceKeys(c *C) {
	// Check that the retrieval of all services keys works correctly.
	keys := s.t.ServiceKeys()
	c.Assert(keys, DeepEquals, []string{})
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddService("service-1", "mysql")
	c.Assert(err, IsNil)
	keys = s.t.ServiceKeys()
	c.Assert(keys, DeepEquals, []string{"service-0", "service-1"})
}

func (s *TopologySuite) TestServiceName(c *C) {
	// Check that the name retrieval for a service name works correctly.
	name, err := s.t.ServiceName("service-0")
	c.Assert(err, ErrorMatches, `service with key "service-0" not found`)
	err = s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	name, err = s.t.ServiceName("service-0")
	c.Assert(err, IsNil)
	c.Assert(name, Equals, "wordpress")
}

func (s *TopologySuite) TestRemoveService(c *C) {
	// Check that the removing of a service works correctly.
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddService("service-1", "mysql")
	c.Assert(err, IsNil)
	err = s.t.RemoveService("service-0")
	c.Assert(err, IsNil)
	c.Assert(s.t.HasService("service-0"), Equals, false)
	c.Assert(s.t.HasService("service-1"), Equals, true)
}

func (s *TopologySuite) TestRemoveNonExistentService(c *C) {
	// Check that the removing of a non-existent service fails.
	err := s.t.RemoveService("service-99")
	c.Assert(err, ErrorMatches, `service with key "service-99" not found`)
}

func (s *TopologySuite) TestAddUnit(c *C) {
	// Check that the adding of a unit works correctly.
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddService("service-1", "mysql")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-05")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-12")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-1-07")
	c.Assert(err, IsNil)
	keys, err := s.t.UnitKeys("service-0")
	c.Assert(err, IsNil)
	c.Assert(keys, DeepEquals, []string{"unit-0-05", "unit-0-12"})
	keys, err = s.t.UnitKeys("service-1")
	c.Assert(err, IsNil)
	c.Assert(keys, DeepEquals, []string{"unit-1-07"})
}

func (s *TopologySuite) TestGlobalUniqueUnitNames(c *C) {
	// Check that even if the underlying service is destroyed
	// and a new one with the same name is created we'll never
	// get a duplicate unit name.
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-0")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-1")
	c.Assert(err, IsNil)
	err = s.t.RemoveService("service-0")
	c.Assert(err, IsNil)
	err = s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-1")
	c.Assert(err, IsNil)
	name, err := s.t.UnitName("unit-0-1")
	c.Assert(err, IsNil)
	c.Assert(name, Equals, "wordpress/1")
}

func (s *TopologySuite) TestAddDuplicatedUnit(c *C) {
	// Check that it's not possible to add a unit twice.
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-0")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-0")
	c.Assert(err, ErrorMatches, `unit "unit-0-0" already in use`)
}

func (s *TopologySuite) TestAddUnitToNonExistingService(c *C) {
	// Check that the adding of a unit to a non-existing services
	// fails correctly.
	err := s.t.AddUnit("unit-0-0")
	c.Assert(err, ErrorMatches, `service with key "service-0" not found`)
}

func (s *TopologySuite) TestUnitKeys(c *C) {
	// Check if registered units from a service are returned correctly.
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddService("service-1", "mysql")
	c.Assert(err, IsNil)
	units, err := s.t.UnitKeys("service-0")
	c.Assert(err, IsNil)
	c.Assert(units, HasLen, 0)
	err = s.t.AddUnit("unit-0-0")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-1")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-1-2")
	c.Assert(err, IsNil)
	units, err = s.t.UnitKeys("service-0")
	c.Assert(err, IsNil)
	c.Assert(units, DeepEquals, []string{"unit-0-0", "unit-0-1"})
	units, err = s.t.UnitKeys("service-1")
	c.Assert(err, IsNil)
	c.Assert(units, DeepEquals, []string{"unit-1-2"})
}

func (s *TopologySuite) TestUnitKeysWithNonExistingService(c *C) {
	// Check if the retrieving of unit keys from a non-existing
	// service fails correctly.
	_, err := s.t.UnitKeys("service-0")
	c.Assert(err, ErrorMatches, `service with key "service-0" not found`)
}

func (s *TopologySuite) TestHasUnit(c *C) {
	// Check that the test for a unit in a service works correctly.
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	found := s.t.HasUnit("unit-0-0")
	c.Assert(found, Equals, false)
	err = s.t.AddUnit("unit-0-0")
	c.Assert(err, IsNil)
	found = s.t.HasUnit("unit-0-0")
	c.Assert(found, Equals, true)
	found = s.t.HasUnit("unit-0-1")
	c.Assert(found, Equals, false)
}

func (s *TopologySuite) TestUnitName(c *C) {
	// Check that the human readable names are returned correctly.
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddService("service-1", "mysql")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-0")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-1")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-1-2")
	c.Assert(err, IsNil)
	name, err := s.t.UnitName("unit-0-0")
	c.Assert(err, IsNil)
	c.Assert(name, Equals, "wordpress/0")
	name, err = s.t.UnitName("unit-0-1")
	c.Assert(err, IsNil)
	c.Assert(name, Equals, "wordpress/1")
	name, err = s.t.UnitName("unit-1-2")
	c.Assert(err, IsNil)
	c.Assert(name, Equals, "mysql/2")
}

func (s *TopologySuite) TestUnitNameWithNonExistingServiceOrUnit(c *C) {
	// Check if the retrieval of unit names fails if the service
	// or the unit doesn't exist.
	_, err := s.t.UnitName("unit-0-1")
	c.Assert(err, ErrorMatches, `service with key "service-0" not found`)
	err = s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	_, err = s.t.UnitName("unit-0-1")
	c.Assert(err, ErrorMatches, `unit with key "unit-0-1" not found`)
	err = s.t.AddUnit("unit-0-0")
	c.Assert(err, IsNil)
	_, err = s.t.UnitName("unit-0-1")
	c.Assert(err, ErrorMatches, `unit with key "unit-0-1" not found`)
}

func (s *TopologySuite) TestRemoveUnit(c *C) {
	// Check that the removing of a unit works correctly.
	err := s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-0")
	c.Assert(err, IsNil)
	err = s.t.AddUnit("unit-0-1")
	c.Assert(err, IsNil)
	err = s.t.RemoveUnit("unit-0-0")
	c.Assert(err, IsNil)
	found := s.t.HasUnit("unit-0-0")
	c.Assert(found, Equals, false)
	found = s.t.HasUnit("unit-0-1")
	c.Assert(found, Equals, true)
}

func (s *TopologySuite) TestRemoveNonExistingUnit(c *C) {
	// Check that the removing of non-existing units fails.
	err := s.t.RemoveUnit("unit-0-0")
	c.Assert(err, ErrorMatches, `service with key "service-0" not found`)
	err = s.t.AddService("service-0", "wordpress")
	c.Assert(err, IsNil)
	err = s.t.RemoveUnit("unit-0-0")
	c.Assert(err, ErrorMatches, `unit with key "unit-0-0" not found`)
}

func (s *TopologySuite) TestRelation(c *C) {
	// Check that the retrieving of relations works correctly.
	relation, err := s.t.Relation("r-1")
	c.Assert(relation, IsNil)
	c.Assert(err, ErrorMatches, `relation "r-1" does not exist`)
	s.t.AddService("service-p", "riak")
	s.t.AddRelation("r-1", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services:  map[RelationRole]*topoRelationService{RolePeer: &topoRelationService{"service-p", "cache"}},
	})
	relation, err = s.t.Relation("r-1")
	c.Assert(err, IsNil)
	c.Assert(relation, NotNil)
	c.Assert(relation.Services[RolePeer].Service, Equals, "service-p")
}

func (s *TopologySuite) TestAddRelation(c *C) {
	// Check that adding a relation works and can only be done once and with 
	// valid services.
	relation, err := s.t.Relation("r-1")
	c.Assert(relation, IsNil)
	c.Assert(err, ErrorMatches, `relation "r-1" does not exist`)
	s.t.AddService("service-p", "mysql")
	s.t.AddService("service-r", "wordpress")
	err = s.t.AddRelation("r-1", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RoleProvider: &topoRelationService{"service-p", "db"},
			RoleRequirer: &topoRelationService{"service-r", "db"},
		},
	})
	c.Assert(err, IsNil)
	relation, err = s.t.Relation("r-1")
	c.Assert(err, IsNil)
	c.Assert(relation, NotNil)
	c.Assert(relation.Services[RoleProvider].Service, Equals, "service-p")
	c.Assert(relation.Services[RoleRequirer].Service, Equals, "service-r")

	err = s.t.AddRelation("r-2", &topoRelation{
		Interface: "",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RoleProvider: &topoRelationService{"service-p", "db"},
			RoleRequirer: &topoRelationService{"service-r", "db"},
		},
	})
	c.Assert(err, ErrorMatches, `relation interface is empty`)

	err = s.t.AddRelation("r-3", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services:  map[RelationRole]*topoRelationService{},
	})
	c.Assert(err, ErrorMatches, `relation has no services`)

	err = s.t.AddRelation("r-4", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RoleProvider: &topoRelationService{"service-p", "db"},
		},
	})
	c.Assert(err, ErrorMatches, `relation has provider but no requirer`)

	err = s.t.AddRelation("r-5", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RoleProvider: &topoRelationService{"service-p", "db"},
			RolePeer:     &topoRelationService{"service-r", "db"},
		},
	})
	c.Assert(err, ErrorMatches, `relation has provider but no requirer`)

	err = s.t.AddRelation("r-6", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RoleProvider: &topoRelationService{"service-p", "db"},
			RoleRequirer: &topoRelationService{"service-r", "db"},
			RolePeer:     &topoRelationService{"service-r", "db"},
		},
	})
	c.Assert(err, ErrorMatches, `relation with mixed peer, provider, and requirer roles`)

	err = s.t.AddRelation("r-7", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RoleProvider: &topoRelationService{"service-p", "db"},
			RoleRequirer: &topoRelationService{"illegal", "db"},
		},
	})
	c.Assert(err, ErrorMatches, `service with key "illegal" not found`)

	err = s.t.AddRelation("r-1", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RoleProvider: &topoRelationService{"service-p", "db"},
			RoleRequirer: &topoRelationService{"service-r", "db"},
		},
	})
	c.Assert(err, ErrorMatches, `relation key "r-1" already in use`)
}

func (s *TopologySuite) TestRelationKeys(c *C) {
	// Check that fetching the relation keys works.
	keys := s.t.RelationKeys()
	c.Assert(keys, DeepEquals, []string{})

	s.t.AddService("service-p", "riak")
	s.t.AddRelation("r-1", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RolePeer: &topoRelationService{"service-p", "cache"},
		},
	})
	keys = s.t.RelationKeys()
	c.Assert(keys, DeepEquals, []string{"r-1"})

	s.t.AddRelation("r-2", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RolePeer: &topoRelationService{"service-p", "cache"},
		},
	})
	keys = s.t.RelationKeys()
	c.Assert(keys, DeepEquals, []string{"r-1", "r-2"})
}

func (s *TopologySuite) TestRelationsForService(c *C) {
	// Check that fetching the relations for a service works.
	s.t.AddService("service-p", "riak")
	relations, err := s.t.RelationsForService("service-p")
	c.Assert(err, IsNil)
	c.Assert(relations, HasLen, 0)

	s.t.AddRelation("r-0", &topoRelation{
		Interface: "ifce0",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RolePeer: &topoRelationService{"service-p", "cache"},
		},
	})
	s.t.AddRelation("r-1", &topoRelation{
		Interface: "ifce1",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RolePeer: &topoRelationService{"service-p", "cache"},
		},
	})
	relations, err = s.t.RelationsForService("service-p")
	c.Assert(err, IsNil)
	c.Assert(relations, HasLen, 2)
	c.Assert(relations["r-0"].Interface, Equals, "ifce0")
	c.Assert(relations["r-1"].Interface, Equals, "ifce1")

	s.t.RemoveRelation("r-0")
	relations, err = s.t.RelationsForService("service-p")
	c.Assert(err, IsNil)
	c.Assert(relations, HasLen, 1)
	c.Assert(relations["r-1"].Interface, Equals, "ifce1")
}

func (s *TopologySuite) TestRemoveRelation(c *C) {
	// Check that removing of a relation works.
	s.t.AddService("service-r", "wordpress")
	s.t.AddService("service-p", "mysql")

	err := s.t.AddRelation("r-1", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RoleProvider: &topoRelationService{"service-p", "db"},
			RoleRequirer: &topoRelationService{"service-r", "db"},
		},
	})
	c.Assert(err, IsNil)

	relation, err := s.t.Relation("r-1")
	c.Assert(err, IsNil)
	c.Assert(relation, NotNil)
	c.Assert(relation.Services[RoleProvider].Service, Equals, "service-p")
	c.Assert(relation.Services[RoleRequirer].Service, Equals, "service-r")

	s.t.RemoveRelation("r-1")

	relation, err = s.t.Relation("r-1")
	c.Assert(relation, IsNil)
	c.Assert(err, ErrorMatches, `relation "r-1" does not exist`)
}

func (s *TopologySuite) TestRemoveServiceWithRelations(c *C) {
	// Check that the removing of a service with
	// associated relations leads to an error.
	s.t.AddService("service-p", "riak")
	s.t.AddRelation("r-1", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RolePeer: &topoRelationService{"service-p", "cache"},
		},
	})

	err := s.t.RemoveService("service-p")
	c.Assert(err, ErrorMatches, `cannot remove service "service-p" with active relations`)
}

func (s *TopologySuite) TestRelationKeyEndpoints(c *C) {
	mysqlep1 := RelationEndpoint{"mysql", "ifce1", "db", RoleProvider, ScopeGlobal}
	blogep1 := RelationEndpoint{"wordpress", "ifce1", "db", RoleRequirer, ScopeGlobal}
	mysqlep2 := RelationEndpoint{"mysql", "ifce2", "db", RoleProvider, ScopeGlobal}
	blogep2 := RelationEndpoint{"wordpress", "ifce2", "db", RoleRequirer, ScopeGlobal}
	mysqlep3 := RelationEndpoint{"mysql", "ifce3", "db", RoleProvider, ScopeGlobal}
	blogep3 := RelationEndpoint{"wordpress", "ifce3", "db", RoleRequirer, ScopeGlobal}
	s.t.AddService("service-r", "wordpress")
	s.t.AddService("service-p", "mysql")
	s.t.AddRelation("r-0", &topoRelation{
		Interface: "ifce1",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RoleProvider: &topoRelationService{"service-p", "db"},
			RoleRequirer: &topoRelationService{"service-r", "db"},
		},
	})
	s.t.AddRelation("r-1", &topoRelation{
		Interface: "ifce2",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RoleProvider: &topoRelationService{"service-p", "db"},
			RoleRequirer: &topoRelationService{"service-r", "db"},
		},
	})

	// Valid relations.
	key, err := s.t.RelationKey(mysqlep1, blogep1)
	c.Assert(err, IsNil)
	c.Assert(key, Equals, "r-0")
	key, err = s.t.RelationKey(blogep1, mysqlep1)
	c.Assert(err, IsNil)
	c.Assert(key, Equals, "r-0")
	key, err = s.t.RelationKey(mysqlep2, blogep2)
	c.Assert(err, IsNil)
	c.Assert(key, Equals, "r-1")
	key, err = s.t.RelationKey(blogep2, mysqlep2)
	c.Assert(err, IsNil)
	c.Assert(key, Equals, "r-1")

	// Endpoints without relation.
	_, err = s.t.RelationKey(mysqlep3, blogep3)
	c.Assert(err, ErrorMatches, `state: no relation between "mysql:db" and "wordpress:db"`)

	// Mix of endpoints of two relations.
	_, err = s.t.RelationKey(mysqlep1, blogep2)
	c.Assert(err, ErrorMatches, `state: no relation between "mysql:db" and "wordpress:db"`)

	// Illegal number of endpoints.
	_, err = s.t.RelationKey()
	c.Assert(err, ErrorMatches, `state: illegal number of relation endpoints provided`)
	_, err = s.t.RelationKey(mysqlep1, mysqlep2, blogep1)
	c.Assert(err, ErrorMatches, `state: illegal number of relation endpoints provided`)
}

func (s *TopologySuite) TestRelationKeyIllegalEndpoints(c *C) {
	mysqlep1 := RelationEndpoint{"mysql", "ifce", "db", RoleProvider, ScopeGlobal}
	blogep1 := RelationEndpoint{"wordpress", "ifce", "db", RoleRequirer, ScopeGlobal}
	mysqlep2 := RelationEndpoint{"illegal-mysql", "ifce", "db", RoleProvider, ScopeGlobal}
	blogep2 := RelationEndpoint{"illegal-wordpress", "ifce", "db", RoleRequirer, ScopeGlobal}
	riakep3 := RelationEndpoint{"riak", "ifce", "ring", RolePeer, ScopeGlobal}
	s.t.AddService("service-r", "wordpress")
	s.t.AddService("service-p1", "mysql")
	s.t.AddService("service-p2", "riak")
	s.t.AddRelation("r-0", &topoRelation{
		Interface: "ifce1",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RoleProvider: &topoRelationService{"service-p", "db"},
			RoleRequirer: &topoRelationService{"service-r", "db"},
		},
	})

	key, err := s.t.RelationKey(mysqlep1, blogep2)
	c.Assert(key, Equals, "")
	c.Assert(err, ErrorMatches, `state: no relation between "mysql:db" and "illegal-wordpress:db"`)
	key, err = s.t.RelationKey(mysqlep2, blogep1)
	c.Assert(key, Equals, "")
	c.Assert(err, ErrorMatches, `state: no relation between "illegal-mysql:db" and "wordpress:db"`)
	key, err = s.t.RelationKey(mysqlep1, riakep3)
	c.Assert(key, Equals, "")
	c.Assert(err, ErrorMatches, `state: no relation between "mysql:db" and "riak:ring"`)
}

func (s *TopologySuite) TestPeerRelationKeyEndpoints(c *C) {
	riakep1 := RelationEndpoint{"riak", "ifce1", "ring", RolePeer, ScopeGlobal}
	riakep2 := RelationEndpoint{"riak", "ifce2", "ring", RolePeer, ScopeGlobal}
	riakep3 := RelationEndpoint{"riak", "ifce3", "ring", RolePeer, ScopeGlobal}
	s.t.AddService("service-p", "ring")
	s.t.AddRelation("r-0", &topoRelation{
		Interface: "ifce1",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RolePeer: &topoRelationService{"service-p", "ring"},
		},
	})
	s.t.AddRelation("r-1", &topoRelation{
		Interface: "ifce2",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RolePeer: &topoRelationService{"service-p", "ring"},
		},
	})

	// Valid relations.
	key, err := s.t.RelationKey(riakep1)
	c.Assert(err, IsNil)
	c.Assert(key, Equals, "r-0")
	key, err = s.t.RelationKey(riakep2)
	c.Assert(err, IsNil)
	c.Assert(key, Equals, "r-1")

	// Endpoint without relation.
	key, err = s.t.RelationKey(riakep3)
	c.Assert(err, ErrorMatches, `state: no peer relation for "riak:ring"`)
}

func (s *TopologySuite) TestPeerRelationKeyIllegalEndpoints(c *C) {
	riakep1 := RelationEndpoint{"riak", "ifce", "illegal-ring", RolePeer, ScopeGlobal}
	s.t.AddService("service-p", "riak")
	s.t.AddRelation("r-0", &topoRelation{
		Interface: "ifce",
		Scope:     ScopeGlobal,
		Services: map[RelationRole]*topoRelationService{
			RolePeer: &topoRelationService{"service-p", "ring"},
		},
	})

	key, err := s.t.RelationKey(riakep1)
	c.Assert(key, Equals, "")
	c.Assert(err, ErrorMatches, `state: no peer relation for "riak:illegal-ring"`)
}

type ConfigNodeSuite struct {
	zkServer   *zookeeper.Server
	zkTestRoot string
	zkTestPort int
	zkAddr     string
	zkConn     *zookeeper.Conn
	path       string
}

var _ = Suite(&ConfigNodeSuite{})

func (s *ConfigNodeSuite) SetUpSuite(c *C) {
	st, err := Initialize(&Info{
		Addrs: []string{TestingZkAddr},
	})
	c.Assert(err, IsNil)
	s.zkConn = ZkConn(st)
	s.path = "/config"
}

func (s *ConfigNodeSuite) TearDownSuite(c *C) {
	err := zkRemoveTree(s.zkConn, "/")
	c.Assert(err, IsNil)
	s.zkConn.Close()
}

func (s *ConfigNodeSuite) TearDownTest(c *C) {
	// Delete the config node path.
	err := zkRemoveTree(s.zkConn, s.path)
	c.Assert(err, IsNil)
}

func (s *ConfigNodeSuite) TestCreateEmptyConfigNode(c *C) {
	// Check that creating an empty node works correctly.
	node, err := createConfigNode(s.zkConn, s.path, nil)
	c.Assert(err, IsNil)
	c.Assert(node.Keys(), DeepEquals, []string{})
}

func (s *ConfigNodeSuite) TestReadWithoutWrite(c *C) {
	// Check reading without writing.
	node, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	c.Assert(node, Not(IsNil))
}

func (s *ConfigNodeSuite) TestSetWithoutWrite(c *C) {
	// Check that config values can be set.
	_, err := s.zkConn.Create(s.path, "", 0, zkPermAll)
	c.Assert(err, IsNil)
	node, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	options := map[string]interface{}{"alpha": "beta", "one": 1}
	node.Update(options)
	c.Assert(node.Map(), DeepEquals, options)
	// Node data has to be empty.
	yaml, _, err := s.zkConn.Get("/config")
	c.Assert(err, IsNil)
	c.Assert(yaml, Equals, "")
}

func (s *ConfigNodeSuite) TestSetWithWrite(c *C) {
	// Check that write updates the local and the ZooKeeper state.
	node, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	options := map[string]interface{}{"alpha": "beta", "one": 1}
	node.Update(options)
	changes, err := node.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemAdded, "alpha", nil, "beta"},
		{ItemAdded, "one", nil, 1},
	})
	// Check local state.
	c.Assert(node.Map(), DeepEquals, options)
	// Check ZooKeeper state.
	yaml, _, err := s.zkConn.Get(s.path)
	c.Assert(err, IsNil)
	zkData := make(map[string]interface{})
	err = goyaml.Unmarshal([]byte(yaml), zkData)
	c.Assert(zkData, DeepEquals, options)
}

func (s *ConfigNodeSuite) TestConflictOnSet(c *C) {
	// Check version conflict errors.
	nodeOne, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	nodeTwo, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)

	optionsOld := map[string]interface{}{"alpha": "beta", "one": 1}
	nodeOne.Update(optionsOld)
	nodeOne.Write()

	nodeTwo.Update(optionsOld)
	changes, err := nodeTwo.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemAdded, "alpha", nil, "beta"},
		{ItemAdded, "one", nil, 1},
	})

	// First test node one.
	c.Assert(nodeOne.Map(), DeepEquals, optionsOld)

	// Write on node one.
	optionsNew := map[string]interface{}{"alpha": "gamma", "one": "two"}
	nodeOne.Update(optionsNew)
	changes, err = nodeOne.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemModified, "alpha", "beta", "gamma"},
		{ItemModified, "one", 1, "two"},
	})

	// Verify that node one reports as expected.
	c.Assert(nodeOne.Map(), DeepEquals, optionsNew)

	// Verify that node two has still the old data.
	c.Assert(nodeTwo.Map(), DeepEquals, optionsOld)

	// Now issue a Set/Write from node two. This will
	// merge the data deleting 'one' and updating
	// other values.
	optionsMerge := map[string]interface{}{"alpha": "cappa", "new": "next"}
	nodeTwo.Update(optionsMerge)
	nodeTwo.Delete("one")

	expected := map[string]interface{}{"alpha": "cappa", "new": "next"}
	changes, err = nodeTwo.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemModified, "alpha", "beta", "cappa"},
		{ItemAdded, "new", nil, "next"},
		{ItemDeleted, "one", 1, nil},
	})
	c.Assert(expected, DeepEquals, nodeTwo.Map())

	// But node one still reflects the former data.
	c.Assert(nodeOne.Map(), DeepEquals, optionsNew)
}

func (s *ConfigNodeSuite) TestSetItem(c *C) {
	// Check that Set works as expected.
	node, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	options := map[string]interface{}{"alpha": "beta", "one": 1}
	node.Set("alpha", "beta")
	node.Set("one", 1)
	changes, err := node.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemAdded, "alpha", nil, "beta"},
		{ItemAdded, "one", nil, 1},
	})
	// Check local state.
	c.Assert(node.Map(), DeepEquals, options)
	// Check ZooKeeper state.
	yaml, _, err := s.zkConn.Get(s.path)
	c.Assert(err, IsNil)
	zkData := make(map[string]interface{})
	err = goyaml.Unmarshal([]byte(yaml), zkData)
	c.Assert(zkData, DeepEquals, options)
}

func (s *ConfigNodeSuite) TestMultipleReads(c *C) {
	// Check that reads without writes always resets the data.
	nodeOne, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	nodeOne.Update(map[string]interface{}{"alpha": "beta", "foo": "bar"})
	value, ok := nodeOne.Get("alpha")
	c.Assert(ok, Equals, true)
	c.Assert(value, Equals, "beta")
	value, ok = nodeOne.Get("foo")
	c.Assert(ok, Equals, true)
	c.Assert(value, Equals, "bar")
	value, ok = nodeOne.Get("baz")
	c.Assert(ok, Equals, false)

	// A read resets the data to the empty state.
	err = nodeOne.Read()
	c.Assert(err, IsNil)
	c.Assert(nodeOne.Map(), DeepEquals, map[string]interface{}{})
	nodeOne.Update(map[string]interface{}{"alpha": "beta", "foo": "bar"})
	changes, err := nodeOne.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemAdded, "alpha", nil, "beta"},
		{ItemAdded, "foo", nil, "bar"},
	})

	// A write retains the newly set values.
	value, ok = nodeOne.Get("alpha")
	c.Assert(ok, Equals, true)
	c.Assert(value, Equals, "beta")
	value, ok = nodeOne.Get("foo")
	c.Assert(ok, Equals, true)
	c.Assert(value, Equals, "bar")

	// Now get another state instance and change ZooKeeper state.
	nodeTwo, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	nodeTwo.Update(map[string]interface{}{"foo": "different"})
	changes, err = nodeTwo.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemModified, "foo", "bar", "different"},
	})

	// This should pull in the new state into node one.
	err = nodeOne.Read()
	c.Assert(err, IsNil)
	value, ok = nodeOne.Get("alpha")
	c.Assert(ok, Equals, true)
	c.Assert(value, Equals, "beta")
	value, ok = nodeOne.Get("foo")
	c.Assert(ok, Equals, true)
	c.Assert(value, Equals, "different")
}

func (s *ConfigNodeSuite) TestDeleteEmptiesState(c *C) {
	// Check that delete creates an empty state.
	node, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	node.Set("a", "foo")
	changes, err := node.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemAdded, "a", nil, "foo"},
	})
	node.Delete("a")
	changes, err = node.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemDeleted, "a", "foo", nil},
	})
	c.Assert(node.Map(), DeepEquals, map[string]interface{}{})
}

func (s *ConfigNodeSuite) TestReadResync(c *C) {
	// Check that read pulls the data into the node.
	nodeOne, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	nodeOne.Set("a", "foo")
	changes, err := nodeOne.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemAdded, "a", nil, "foo"},
	})
	nodeTwo, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	nodeTwo.Delete("a")
	changes, err = nodeTwo.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemDeleted, "a", "foo", nil},
	})
	nodeTwo.Set("a", "bar")
	changes, err = nodeTwo.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemAdded, "a", nil, "bar"},
	})
	// Read of node one should pick up the new value.
	err = nodeOne.Read()
	c.Assert(err, IsNil)
	value, ok := nodeOne.Get("a")
	c.Assert(ok, Equals, true)
	c.Assert(value, Equals, "bar")
}

func (s *ConfigNodeSuite) TestMultipleWrites(c *C) {
	// Check that multiple writes only do the right changes.
	node, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	node.Update(map[string]interface{}{"foo": "bar", "this": "that"})
	changes, err := node.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemAdded, "foo", nil, "bar"},
		{ItemAdded, "this", nil, "that"},
	})
	node.Delete("this")
	node.Set("another", "value")
	changes, err = node.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemAdded, "another", nil, "value"},
		{ItemDeleted, "this", "that", nil},
	})

	expected := map[string]interface{}{"foo": "bar", "another": "value"}
	c.Assert(expected, DeepEquals, node.Map())

	changes, err = node.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{})

	err = node.Read()
	c.Assert(err, IsNil)
	c.Assert(expected, DeepEquals, node.Map())

	changes, err = node.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{})
}

func (s *ConfigNodeSuite) TestWriteTwice(c *C) {
	// Check the correct writing into a node by two config nodes.
	nodeOne, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	nodeOne.Set("a", "foo")
	changes, err := nodeOne.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemAdded, "a", nil, "foo"},
	})

	nodeTwo, err := readConfigNode(s.zkConn, s.path)
	c.Assert(err, IsNil)
	nodeTwo.Set("a", "bar")
	changes, err = nodeTwo.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{
		{ItemModified, "a", "foo", "bar"},
	})

	// Shouldn't write again. Changes were already
	// flushed and acted upon by other parties.
	changes, err = nodeOne.Write()
	c.Assert(err, IsNil)
	c.Assert(changes, DeepEquals, []ItemChange{})

	err = nodeOne.Read()
	c.Assert(err, IsNil)
	c.Assert(nodeOne, DeepEquals, nodeTwo)
}
