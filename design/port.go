package design

var Port = Type("Port", func() {
	Description("A POrt describes an OpenStack network port.")
	TypeName("Port")

	Field(1, "id", String, "ID of the port.", func() {
		MaxLength(64)
		Example("123abc")
	})
	Field(2, "name", String, "Name of the port.", func() {
		MaxLength(64)
		Example("my-machine")
	})
	Field(3, "projectId", String, "The ID of the OpenStack project owning the port.", func() {
		MinLength(32)
		MaxLength(32)
		Example("3d4c2c82bd5948f0bcab0cf3a7c9b48c")
	})
	Field(4, "projectName", String, "The name of the OpenStack project owning the port.", func() {
		MaxLength(64)
		Pattern(`^([^\-]+)\-(.+)$`) // "<account>-<project name>"
		Example("account-my-project-XYZ")
	})
	Field(5, "ipaddr", String, "The IP address assigned to the port.", func() {
		MaxLength(15)
		Example("192.168.1.35")
	})
	Field(6, "macaddr", String, "The AMC address assigned to the port.", func() {
		MaxLength(17)
		Example("00:0a:95:9d:68:16")
	})
	Field(7, "tags", ArrayOf(String), "The tags assigned to the port.", func() {
		Minimum(0)
		Maximum(1024)
	})
	Field(8, "description", String, "The description of the port.", func() {
		MaxLength(256)
	})
	Field(9, "host", Host, "The virtual machine on which the port is mounted.", func() {
	})
	// go on like this...

	Required("id", "projectId", "macaddr")
})

var PortInfo = ResultType("application/vnd.brokerd.port", func() {
	Description("PortInfo describes a network port retrieved by the port service.")
	Reference(Port)
	TypeName("PortInfo")

	Attributes(func() {
		Field(1, "id")
		Field(2, "name")
		Field(4, "projectId")
		Field(5, "projectName"))
		Field(6, "ipaddr")
		Field(7, "macaddr")
		Field(8, "tags")
		Field(9, "description")
		Field(10, "hostId", String, "The ID of the virtual machine mounting the port.", func() {
			MinLength(0)
			MaxLength(32)
		})
	})

	View("default", func() {
		Attribute("id")
		Attribute("projectId")
		Attribute("ipaddr")
		Attribute("macaddr")
		Attribute("tags")
		Attribute("hostId")
	})

	View("extended", func() {
		Attribute("id")
		Attribute("projectId")
		Attribute("projectName")
		Attribute("ipaddr")
		Attribute("macaddr")
		Attribute("tags")
		Attribute("description")
		Attribute("hostId")
	})

	Required("id", "projectId", "macaddr")
})
