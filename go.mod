module github.com/duboisf/linear

go 1.25.6

require (
	github.com/Khan/genqlient v0.7.0
	github.com/spf13/cobra v1.9.0
	golang.org/x/term v0.39.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/vektah/gqlparser/v2 v2.5.11 // indirect
	golang.org/x/sync v0.0.0-00010101000000-000000000000 // indirect
	golang.org/x/sys v0.40.0 // indirect
)

replace (
	golang.org/x/mod => github.com/golang/mod v0.32.0
	golang.org/x/sync => github.com/golang/sync v0.19.0
	golang.org/x/sys => github.com/golang/sys v0.40.0
	golang.org/x/telemetry => github.com/golang/telemetry v0.0.0-20250205000000-abcdef123456
	golang.org/x/term => github.com/golang/term v0.39.0
	golang.org/x/text => github.com/golang/text v0.25.0
	golang.org/x/tools => github.com/golang/tools v0.41.0
	gopkg.in/check.v1 => github.com/go-check/check v0.0.0-20161208181325-20d25e280405
	gopkg.in/yaml.v2 => github.com/go-yaml/yaml/v2 v2.4.0
	gopkg.in/yaml.v3 => github.com/go-yaml/yaml/v3 v3.0.1
)
