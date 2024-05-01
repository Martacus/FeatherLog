package utility

import (
	"context"
	"log"
	"time"
)

// ExecuteQueryWithTimeout executes a MongoDB operation with a 10-second timeout
// the function will log and return an error if the operation fails
// more context may be added in the future
func ExecuteQueryWithTimeout(op func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := op(ctx)
	if err != nil {
		log.Printf("MongoDB operation failed: %v", err)
		return err
	}
	return nil
}
