package design

var Host = Type("Host", func() {
	Description("A Host describes an OpenStack virtual machine.")
	TypeName("Host")

	Field(1, "id", String, "ID of the virtual machine.", func() {
		MaxLength(64)
		Example("123abc")
	})
	Field(2, "name", String, "Name of the virtual machine.", func() {
		MaxLength(64)
		Example("my-machine")
	})
	Field(3, "fqdn", String, "The fully-qualified domain name of the virtual machine.", func() {
		MaxLength(255)
		Example("my-machine.my-domain.example.com")
	})
	Field(4, "projectId", String, "The ID of the OpenStack project owning the virtual machine.", func() {
		MinLength(32)
		MaxLength(32)
		Example("3d4c2c82bd5948f0bcab0cf3a7c9b48c")
	})
	Field(5, "projectName", String, "The name of the OpenStack project owning the virtual machine.", func() {
		MaxLength(64)
		Pattern(`^([^\-]+)\-(.+)$`) // "<account>-<project name>"
		Example("account-my-project-XYZ")
	})
	Field(6, "ram", UInt32, "The amount of RAM, in mebibytes", func() {
		Minimum(1)
		Maximum(524288) // 512 Gibibytes in Mebibytes
	})
	Field(7, "vcpus", UInt8, "The number of vCPUs", func() {
		Minimum(1)
		Maximum(64)
	})
	Field(8, "description", String, func() {
		MaxLength(256)
	})
	Field(9, "ports", ArrayOf(Port), "The network ports (NICs) associated with the virtual machine.", func() {
		MinLength(0)
		MaxLength(1) // can be more?
	})
	// go on like this...

	Required("id", "projectId", "fqdn")
})

var HostInfo = ResultType("application/vnd.brokerd.host", func() {
	Description("HostInfo describes a virtual machine retrieved by the hosts service.")
	Reference(Host)
	TypeName("HostInfo")

	Attributes(func() {
		Field(1, "id")
		Field(2, "name")
		Field(3, "fqdn")
		Field(4, "projectId")
		Field(5, "projectName"))
		Field(6, "ram")
		Field(7, "vcpus")
		Field(8, "description")
		Field(9, "ports")
	})

	View("default", func() {
		Attribute("id")
		Attribute("projectId")
		Attribute("fqdn")
		Attribute("ports", func() {
			View("default")
		})
	})

	View("extended", func() {
		Attribute("id")
		Attribute("name")
		Attribute("projectId")
		Attribute("projectName")
		Attribute("ram")
		Attribute("vcpus")
		Attribute("description")		
		Attribute("ports", func() {
			View("default")
		})
	})

	Required("id", "projectId", "fqdn")
})


// https://github.com/goadesign/examples/blob/master/cellar/design/storage.go