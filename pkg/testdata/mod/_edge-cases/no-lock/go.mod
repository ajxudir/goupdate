module github.com/example/no-lock

go 1.21

// Edge case: go.mod without go.sum (LockMissing status)

require (
	github.com/gin-gonic/gin v1.9.0
	github.com/spf13/cobra v1.7.0
)
