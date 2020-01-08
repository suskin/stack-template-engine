# Stack template engine

## Developing

### Testing

There are some rudimentary integration tests.

To start the controller locally, run:

```
make run
```

Then, in another window, run:

```
make integration-test
```

It should create a config map named `mycustomname-{{ engine }}`. So, for
example, `mycustomname-helm2` for the helm 2 integration test.

To clean up, run:

```
make clean-integration-test
```

The source files for the integration tests are in `test/`.

To debug the integration test, inspect the logs for any jobs or pods
which were run by the controller. Also take a look at the controller's
logs.
