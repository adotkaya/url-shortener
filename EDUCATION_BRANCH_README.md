# Education Branch - Learning Comments Added

This branch contains the same codebase with comprehensive educational comments added to help junior developers understand:

## Files with Educational Comments

### 1. **cmd/server/main.go**
- Application startup flow
- Dependency injection pattern
- Connection pooling
- Graceful shutdown
- Middleware chaining
- Goroutines and channels

### 2. **internal/domain/url.go**
- Domain-driven design
- Structs vs classes
- Methods and receivers
- Pointers and nullable fields
- Sentinel errors
- Builder pattern

### 3. **internal/handler/http/middleware.go**
- Middleware pattern
- Function closures
- Panic recovery
- CORS configuration
- Request ID propagation

## How to Use This Branch

1. **Read the code top-to-bottom** - Comments explain concepts as they appear
2. **Compare to .NET** - Many comments include C# equivalents
3. **Follow the learning roadmap** - See `go-backend-learning-guide.md`

## Switching Between Branches

```bash
# View education branch with comments
git checkout education

# Return to clean main branch
git checkout main
```

## Note

The `main` branch contains production-ready code without educational comments. Use the `education` branch for learning purposes only.
