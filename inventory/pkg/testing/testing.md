# Testing

The testing package offers utility functions that help running experiments involving
the presence of Inventory and a local database. Note, this pkg expects the DB running
as container, look at [pkg/util/util.go](/pkg/util/util.go) for the env variables expected
for the local deployment. This pkg can be used by resource managers and other projects
that rely on Inventory to perform their duties in a test/unit test setting.

## API Documentation

At the time of writing 2 functions are exported:

- StartTestingEnvironment
- StopTestingEnvironment

The aforementioned functions are used to start and stop the testing environment. In particular,
the `start` function init the OPA agent, prepare the bufconn listener for the `in-memory`
communication, bootstrap Inventory server, API client and RM client. The `stop` function cleans
the environment and should be used at the end of the tests.

See the Go doc of the package for detailed function descriptions. Here is the
general workflow:

```go

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(wd))

    // Note that policy bundle and the migrations dir are created at runtime using `embed`
    migrationsDir := projectRoot + "/_workspace"
    policyPath := projectRoot + "/_workspace"
    // Certificate path will trigger the loading of the certificate if not empty
    certificatePath := ""

	inv_testing.StartTestingEnvironment(policyPath, certificatePath, migrationsDir)
	run := m.Run() // run all the tests defined in the pkg
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}
```

Note that this package exports additionally the symbol `BufconnLis` which can be used
to create additionally local clients connected to the local server started by this package.
See the tests in [pkg/client](/pkg/client) for further examples.
