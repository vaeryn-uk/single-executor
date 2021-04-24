# Single executor

To compile everything,
```
docker-compose run --rm dev make
```

To bring up a test system,

```
docker-compose -f docker-compose.system.yaml up
```

This will bring up:
* A `chain`, which emulates some kind of external blockchain (as a single node for simplicity).
* One or more `node`s, which will each individually periodically write signatures to the chain.

To see the signatures written to the chain, our `chain` service will respond with JSON over
HTTP, `localhost:8080`.
