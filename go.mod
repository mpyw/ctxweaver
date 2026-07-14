module github.com/mpyw/ctxweaver

go 1.25.0

require (
	github.com/dave/dst v0.27.4
	github.com/google/go-cmp v0.7.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
	golang.org/x/term v0.44.0
	golang.org/x/tools v0.47.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

// Retract unstable versions
retract v0.1.0 // Unstable: significant API and behavior changes expected
