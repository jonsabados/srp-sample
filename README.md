# Do One Thing Well

This repo is a contrived example for discussing how adhering to the [single responsibility principle](https://en.wikipedia.org/wiki/Single-responsibility_principle) leads to more maintainable code. It contains two implementations of a caching database construct, with [one implementation](#big-ball-of-mud-version) containing a single construct responsible for both data access and caching, and [another](#single-responsibility-version) where the distinct concerns are handled by separate constructs.

## Big Ball of Mud Version

Within the [bbom](bbom) package there are three files:

1) [creature.go](bbom/creature.go): This contains the definition of the `Creature` entity as well as a `CreatureLookupResult`
2) [caching_creature_repo.go](bbom/caching_creature_repo.go): This construct handles data access and caching
3) [caching_creature_repo_test.go](bbom/caching_creature_repo_test.go): Unit tests for the repo construct

In this example a single construct handles both the responsibility of accessing data, and caching it. In this model it is impossible to bypass the cache, and should the need arise additional functionality would need to be built and consumers would need to be explicitly aware of this functionality. Testing is also more complex, and some aspects, such as the cache being hit or not, require jumping through hoops or in some cases are [simply not testable](bbom/caching_creature_repo_test.go#L166).

## Single Responsibility Version

This version contains six files:

1) [creature.go](srp/creature.go): This contains the definition of the `Creature` entity as well as a `CreatureLookupResult`
   * No effective difference from the bbom version, however the unexported timestamp field now is hidden away inside the component that handles caching
2) [creature_repo.go](srp/creature_repo.go): This contains construct for accessing data
3) [creature_repo_test.go](srp/creature_repo_test.go): Tests for the data access construct
4) [caching_creature_repo.go](srp/caching_creature_repo_test.go): A caching facade that can wrap a CreatureRepo to add caching in an API compatible manner
5) [caching_creature_repo_tests.go](srp/caching_creature_repo_test.go): Tests for the caching construct
6) [mock_raw_creature_repo_test.go](srp/mock_raw_creature_repo_test.go): A generated mock (created via `make mocks`) used by the caching construct tests

In this version caching is considered its own responsibility, even though it could be argued to be part of data access. With this model consumers likely would not be aware of the caching & areas where caching is appropriate would likely be addressed during dependency injection phases with wiring code making the decisions of what components receive a caching version of the repo, or the raw repo itself. This added flexibility does come at a cost though, as it may not be immediately clear to callers of `GetCreature` that caching may be in the mix. Effectively developing code in this model does require leaning into the idea of writing to interfaces and embracing the idea that individual components do not, and should not, have a full picture of the system as a whole.

## Running the examples
The constructs within this repository can be exercised via unit tests. As these constructs do perform database operations a PostgreSQL database is needed to run the tests so the makefile provides an easy way to get this going. First you will need to be able to run docker containers ([Docker Destkop](https://www.docker.com/products/docker-desktop/) if your on a mac), as well as having [migrate](https://github.com/golang-migrate/migrate) installed. With those pre-requesets simply run the following commands from the root of this project:

```bash
make postgres-docker-start
make migrate
```

You can then run the unit tests via any method you like, there is a recipe available in the Makefile which can be invoked via `make test`.

The docker container may be removed by executing `make postgres-docker-rm`