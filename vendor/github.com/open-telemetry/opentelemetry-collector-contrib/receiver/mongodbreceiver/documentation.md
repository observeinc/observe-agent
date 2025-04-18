[comment]: <> (Code generated by mdatagen. DO NOT EDIT.)

# mongodb

## Default Metrics

The following metrics are emitted by default. Each of them can be disabled by applying the following configuration:

```yaml
metrics:
  <metric_name>:
    enabled: false
```

### mongodb.cache.operations

The number of cache operations of the instance.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {operations} | Sum | Int | Cumulative | true |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| type | The result of a cache request. | Str: ``hit``, ``miss`` |

### mongodb.collection.count

The number of collections.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {collections} | Sum | Int | Cumulative | false |

### mongodb.connection.count

The number of connections.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {connections} | Sum | Int | Cumulative | false |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| type | The status of the connection. | Str: ``active``, ``available``, ``current`` |

### mongodb.cursor.count

The number of open cursors maintained for clients.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {cursors} | Sum | Int | Cumulative | false |

### mongodb.cursor.timeout.count

The number of cursors that have timed out.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {cursors} | Sum | Int | Cumulative | false |

### mongodb.data.size

The size of the collection. Data compression does not affect this value.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| By | Sum | Int | Cumulative | false |

### mongodb.database.count

The number of existing databases.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {databases} | Sum | Int | Cumulative | false |

### mongodb.document.operation.count

The number of document operations executed.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {documents} | Sum | Int | Cumulative | false |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| operation | The MongoDB operation being counted. | Str: ``insert``, ``query``, ``update``, ``delete``, ``getmore``, ``command`` |

### mongodb.extent.count

The number of extents.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {extents} | Sum | Int | Cumulative | false |

### mongodb.global_lock.time

The time the global lock has been held.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| ms | Sum | Int | Cumulative | true |

### mongodb.index.access.count

The number of times an index has been accessed.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {accesses} | Sum | Int | Cumulative | false |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| collection | The name of a collection. | Any Str |

### mongodb.index.count

The number of indexes.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {indexes} | Sum | Int | Cumulative | false |

### mongodb.index.size

Sum of the space allocated to all indexes in the database, including free index space.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| By | Sum | Int | Cumulative | false |

### mongodb.memory.usage

The amount of memory used.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| By | Sum | Int | Cumulative | false |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| type | The type of memory used. | Str: ``resident``, ``virtual`` |

### mongodb.network.io.receive

The number of bytes received.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| By | Sum | Int | Cumulative | false |

### mongodb.network.io.transmit

The number of by transmitted.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| By | Sum | Int | Cumulative | false |

### mongodb.network.request.count

The number of requests received by the server.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {requests} | Sum | Int | Cumulative | false |

### mongodb.object.count

The number of objects.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {objects} | Sum | Int | Cumulative | false |

### mongodb.operation.count

The number of operations executed.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {operations} | Sum | Int | Cumulative | true |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| operation | The MongoDB operation being counted. | Str: ``insert``, ``query``, ``update``, ``delete``, ``getmore``, ``command`` |

### mongodb.operation.time

The total time spent performing operations.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| ms | Sum | Int | Cumulative | true |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| operation | The MongoDB operation being counted. | Str: ``insert``, ``query``, ``update``, ``delete``, ``getmore``, ``command`` |

### mongodb.session.count

The total number of active sessions.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {sessions} | Sum | Int | Cumulative | false |

### mongodb.storage.size

The total amount of storage allocated to this collection.

If collection data is compressed it reflects the compressed size.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| By | Sum | Int | Cumulative | true |

## Optional Metrics

The following metrics are not emitted by default. Each of them can be enabled by applying the following configuration:

```yaml
metrics:
  <metric_name>:
    enabled: true
```

### mongodb.health

The health status of the server.

A value of '1' indicates healthy. A value of '0' indicates unhealthy.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| 1 | Gauge | Int |

### mongodb.lock.acquire.count

Number of times the lock was acquired in the specified mode.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {count} | Sum | Int | Cumulative | true |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| lock_type | The Resource over which the Lock controls access | Str: ``parallel_batch_write_mode``, ``replication_state_transition``, ``global``, ``database``, ``collection``, ``mutex``, ``metadata``, ``oplog`` |
| lock_mode | The mode of Lock which denotes the degree of access | Str: ``shared``, ``exclusive``, ``intent_shared``, ``intent_exclusive`` |

### mongodb.lock.acquire.time

Cumulative wait time for the lock acquisitions.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| microseconds | Sum | Int | Cumulative | true |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| lock_type | The Resource over which the Lock controls access | Str: ``parallel_batch_write_mode``, ``replication_state_transition``, ``global``, ``database``, ``collection``, ``mutex``, ``metadata``, ``oplog`` |
| lock_mode | The mode of Lock which denotes the degree of access | Str: ``shared``, ``exclusive``, ``intent_shared``, ``intent_exclusive`` |

### mongodb.lock.acquire.wait_count

Number of times the lock acquisitions encountered waits because the locks were held in a conflicting mode.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {count} | Sum | Int | Cumulative | true |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| lock_type | The Resource over which the Lock controls access | Str: ``parallel_batch_write_mode``, ``replication_state_transition``, ``global``, ``database``, ``collection``, ``mutex``, ``metadata``, ``oplog`` |
| lock_mode | The mode of Lock which denotes the degree of access | Str: ``shared``, ``exclusive``, ``intent_shared``, ``intent_exclusive`` |

### mongodb.lock.deadlock.count

Number of times the lock acquisitions encountered deadlocks.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {count} | Sum | Int | Cumulative | true |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| lock_type | The Resource over which the Lock controls access | Str: ``parallel_batch_write_mode``, ``replication_state_transition``, ``global``, ``database``, ``collection``, ``mutex``, ``metadata``, ``oplog`` |
| lock_mode | The mode of Lock which denotes the degree of access | Str: ``shared``, ``exclusive``, ``intent_shared``, ``intent_exclusive`` |

### mongodb.operation.latency.time

The latency of operations.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| us | Gauge | Int |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| operation | The MongoDB operation with regards to latency | Str: ``read``, ``write``, ``command`` |

### mongodb.operation.repl.count

The number of replicated operations executed.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {operations} | Sum | Int | Cumulative | true |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| operation | The MongoDB operation being counted. | Str: ``insert``, ``query``, ``update``, ``delete``, ``getmore``, ``command`` |

### mongodb.repl_commands_per_sec

The number of replicated commands executed per second.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {command}/s | Gauge | Double |

### mongodb.repl_deletes_per_sec

The number of replicated deletes executed per second.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {delete}/s | Gauge | Double |

### mongodb.repl_getmores_per_sec

The number of replicated getmores executed per second.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {getmore}/s | Gauge | Double |

### mongodb.repl_inserts_per_sec

The number of replicated insertions executed per second.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {insert}/s | Gauge | Double |

### mongodb.repl_queries_per_sec

The number of replicated queries executed per second.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {query}/s | Gauge | Double |

### mongodb.repl_updates_per_sec

The number of replicated updates executed per second.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {update}/s | Gauge | Double |

### mongodb.uptime

The amount of time that the server has been running.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| ms | Sum | Int | Cumulative | true |

## Resource Attributes

| Name | Description | Values | Enabled |
| ---- | ----------- | ------ | ------- |
| database | The name of a database. | Any Str | true |
| server.address | The address of the MongoDB host. | Any Str | true |
| server.port | The port of the MongoDB host. | Any Int | false |
