package design

var NotFound = Type("NotFound", func() {
	Description("NotFound is the type returned when attempting to show or delete a hot or a port that does not exist.")
	Field(1, "message", String, "Message of error", func() {
		Meta("struct:error:name")
		Example("host 1 not found")
	})
	Field(2, "id", String, "ID of missing object")
	Required("message", "id")
})

var Criteria = Type("Criteria", func() {
	Description("Criteria describes a set of criteria used to pick an object. All criteria are optional, at least one must be provided.")
	Field(1, "name", String, "Name of object to pick.", func() {
		Example("Machine XYZ")
	})
	Field(2, "projectId", String, "Project ID of object to pick.", func() {
		Example("aabf45d4452ff56")
	})
	Field(3, "projectName", String, "Project name of object to pick.", func() {
		Example("project-X")
	})
	// go on...
})