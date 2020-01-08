# Stack template engine

## Developing

### Testing

There are some rudimentary integration tests.

First, build the stack:

```
kubectl crossplane stack build
```

This is the most convenient way to get the contents of the stack into
the cluster.

Next, start the controller locally:

```
make run
```

Then, in another window, run the integration test to build all the
helpers and create test objects:

```
make integration-test
```

It should create a config map named `mycustomname-{{ engine }}`. So, for
example, `mycustomname-helm2` for the helm 2 integration test.

To clean up, run:

```
make clean-integration-test
```

The source files for the integration tests are in `test/` and in the
`Makefile`.

To debug the integration test, inspect the logs for any jobs or pods
which were run by the controller. Also take a look at the controller's
logs.
