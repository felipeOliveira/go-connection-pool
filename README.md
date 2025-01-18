# Go Concurrent Programming Challenge: Connection Pooling

Implement connection pooling: Instead of simply selecting a connection from the pool array, implement a more robust connection pooling mechanism. This could involve:
- A queue to manage available connections.
- A mechanism to track idle connections and potentially close them after a timeout.
- Error handling: Add error handling for semaphore acquisition and database operations.
- Metrics: Collect metrics such as the number of active connections, the number of waiting requests, and the average query execution time.
- Context cancellation: Implement support for context cancellation to gracefully handle requests that are canceled or timed out.

## This challenge encourages you to:

- Apply the semaphore pattern to control access to a shared resource (database connections).
- Implement a basic connection pool for resource management.
- Handle potential errors and gracefully handle cancellations.
- Gather performance metrics to understand the behavior of your concurrent system.