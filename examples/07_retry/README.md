# Example 07: Retry and Error Recovery

This example demonstrates the retry mechanism for handling transient failures.

## Features Demonstrated

- **Retry Logic**: Automatic retry with configurable attempts and delays
- **Fallback Handling**: Execute alternative agent if all retries fail
- **Error Resilience**: Graceful degradation when services are unreliable

## Workflow Structure

```yaml
- name: fetch-data
  uses: prompts/unreliable.md
  retry:
    attempts: 3      # Try up to 3 times
    delay: 2         # Wait 2 seconds between attempts
    yield: prompts/fallback.md  # Use this if all attempts fail
  output: data
```

## Retry Parameters

| Parameter  | Type   | Description                                     |
| ---------- | ------ | ----------------------------------------------- |
| `attempts` | int    | Number of retry attempts (default: 1, no retry) |
| `delay`    | int    | Seconds to wait between attempts                |
| `yield`    | string | Path to fallback agent if all retries fail      |

## Execution Flow

1. **First attempt**: Execute `unreliable.md`
2. **On failure**: Wait 2 seconds, try again
3. **Retry 1**: Execute `unreliable.md` again
4. **On failure**: Wait 2 seconds, try again
5. **Retry 2**: Execute `unreliable.md` again
6. **All failed**: Execute `fallback.md` instead
7. **Continue**: Next step receives output (from success or fallback)

## Use Cases

### When to Use Retry

- **Transient network failures**: API rate limits, timeouts
- **Service instability**: Temporary outages, load spikes  
- **Non-deterministic operations**: Operations that may succeed on retry
- **External dependencies**: When calling unreliable external services

### When to Use Yield (Fallback)

- **Graceful degradation**: Provide cached/default response
- **User experience**: Avoid complete failure, show partial results
- **Alternative strategies**: Try different approach if primary fails
- **Error context**: Generate meaningful error messages for users

## Example Scenarios

### API with Rate Limiting
```yaml
retry:
  attempts: 5
  delay: 10  # Exponential backoff could be added
```

### Best-effort with cached fallback
```yaml
retry:
  attempts: 2
  delay: 1
  yield: prompts/use-cache.md
```

### Critical operation with no fallback
```yaml
retry:
  attempts: 10
  delay: 5
  # No yield - let error propagate if all fail
```

## Running the Example

```bash
iq run examples/07_retry/run.yml "San Francisco"
```

If the unreliable agent succeeds (or after retries), you'll see weather data.
If all retries fail, you'll see the fallback response.

## Notes

- Retry applies to **agent execution errors**, not validation errors
- The `yield` agent receives the **same input** as the original agent
- The `output` field stores the result from either success or fallback
- Subsequent steps can't tell if retry or fallback was used
