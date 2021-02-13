package design

import (
	. "goa.design/goa/v3/dsl"
)

var _ = API("brokerd", func() {
	Title("OpenStack Event Broker")
	Description("Service for collecting and analysing OpenStack notifications and dispatching virtual machine related events")
	Version("1.0")
	TermsOfService("Copyright Andrea Funtò 2021 - Licensed under the EUPL v1.2")
	Contact(func() {
		Name("Andrea Funtò")
		Email("dihedron dot dev at the mail by Google dot com")
		URL("https://dihedron.org")
	})
	License(func() { // License
		Name("European Union Public License (EUPL) v1.2")
		URL("https://joinup.ec.europa.eu/sites/default/files/custom-page/attachment/2020-03/EUPL-1.2%20EN.txt")
	})
	Docs(func() { // Documentation links
		Description("OpenStack Event Boker documentation")
		URL("http://github.com/dihedron/brokerd/README.md")
	})
	Server("brokerd", func() {
		Description("brokerd hosts the vms, ports and swagger services.")
		Services("hosts", "ports", "swagger")
		Host("localhost", func() {
			Description("localhost")
			URI("http://localhost:11000")
			URI("grpc://localhost:10000")
		})
		Host("development", func() {
			Description("development host")
			URI("http://localhost:11000")
			URI("grpc://localhost:10000")
		})
		Host("production", func() {
			Description("production host")
			URI("http://localhost:11000")
			URI("grpc://localhost:10000")
		})
	})
	HTTP(func() {
		Path("/api") // prefix to HTTP path of all requests
	})
})

/*

var Port = ResultType("application/vnd.brokerd.port", func() {
	Description("A Port describes a network port retrieved by the ports service.")

	TypeName("VirtualMachine")

	Attributes(func() {
		Field(1, "id", String, "ID is the unique id of the virtual machine.", func() {
			MaxLength(64)
			Example("123abc")
		})
		Field(2, "name", String, "Name is the name of the virtual machine.", func() {
			MaxLength(64)
			Example("my-machine")
		})
		Field(3, "fqdn", String, "FQDN is the fully-qualified host name of the virtual machine.", func() {
			MaxLength(255)
			Example("my-machine.my-domain.example.com")
		})
		Field(4, "tenantId", String, "TenantID is the ID of the tenant owning this VM", func() {
			MaxLength(64)
			Example("abc123")
		})
		Field(5, "tenantName", String, "TenantName is the name of the tenant owning this VM", func() {
			MaxLength(64)
			Pattern(`^([^\-]+)\-([^\-]+)$`) // "<account>-<tenant>"
			Example("tenant-XYZ")
		})
		Field(6, "ram", UInt8)
		Field(7, "vcpus", UInt8)
		Field(8, "description", String, func() {
			MaxLength(256)
		})
		Field(9, "ports", ArrayOf(Port))
	})

	View("default", func() {
		Attribute("id")
		Attribute("tenantId")
		Attribute("fqdn")
		Attribute("ports", func() {
			View("default")
		})
	})

	View("extended", func() {
		Attribute("id")
		Attribute("name")
		Attribute("tenantId")
		Attribute("tenantName")
		Attribute("ports", func() {
			View("extended")
		})
	})

	Required("id", "tenantId", "fqdn")
})
*/
/*

var _ = Service("vms", func() {
	Title("virtual machines CRUD service")
	Description("The vms service performs operations on virtual machines.")

	Method("list", func() {
		Payload(func() {
			Field(1, "a", Int, "Left operand")
			Field(2, "b", Int, "Right operand")
			Required("a", "b")
		})

		Result(Int)

		HTTP(func() {
			GET("/add/{a}/{b}")
		})

		GRPC(func() {
		})
	})

	Files("/openapi.json", "./gen/http/openapi.json")
})
*/
