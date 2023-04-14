# Testing Strategy and Developer Guideline

Intent of this document is to introduce you (the developer) to the following:
* Libraries that are used to write tests.
* Best practices to write tests that are correct, stable, fast and maintainable.
* How to run tests.

For any new contributions **tests are a strict requirement**.
`Boy Scouts Rule` is followed: If you touch a code for which either no tests exist or coverage is insufficient then it is expected that you will add relevant tests.

## Tools Used for Writing Tests

These are the following tools that were used to write tests. It is preferred not to introduce any additional tools / test frameworks for writing tests:

### Gomega

We use gomega as our matcher or assertion library. Refer to Gomega's [official documentation](https://onsi.github.io/gomega/) for details regarding its installation and application in tests.

### `Testing` Package from Standard Library

We use the `Testing` package provided by the standard library in golang for writing all our tests. Refer to its [official documentation](https://pkg.go.dev/testing) to learn how to write tests using `Testing` package. You can also refer to [this](https://go.dev/doc/tutorial/add-a-test) example.

## Writing Tests

### Common for All Kinds
- For naming the individual tests (`TestXxx` and `testXxx` methods) and helper methods, make sure that the name describes the implementation of the method. For eg: `TestGetBaseAddressWithTLSEnabled` tests the behaviour of the `GetBaseAddress` method when TLS is enabled.
- Maintain proper logging in tests. Use `t.Log()` method to add appropriate messages wherever necessary to describe the flow of the test. See [this](../../internal/brclient/brclient_test.go) for examples.

### Table-driven tests
We need a tabular structure in the following case:

- **When we have the same code path and multiple possible values to check**:- In this case we have the arguments and expectations in a struct. We iterate through the slice of all such structs, passing the arguments to appropriate methods and checking if the expectation is met. See [this](../../internal/brclient/brclient_test.go) for examples.

## Run Tests
To run unit tests, use the following Makefile target
```shell
make test
```
To view coverage after running the tests, run :
```shell
go tool cover -html=cover.out
```