package sql_test

import (
	"context"
	"testing"

	"github.com/robbert229/jaeger-postgresql/internal/sql"
	"github.com/robbert229/jaeger-postgresql/internal/sqltest"

	"github.com/stretchr/testify/require"
)

func TestGetOperations(t *testing.T) {
	ctx := context.Background()
	conn, cleanup := sqltest.Harness(t)
	q := sql.New(conn)

	t.Run("should return nothing when no operations exist", func(t *testing.T) {
		require.Nil(t, cleanup())

		err := q.UpsertService(ctx, "service-1")
		require.Nil(t, err)

		operations, err := q.GetOperations(ctx, "service-1")
		require.Nil(t, err)

		require.Empty(t, operations)
	})

	t.Run("should not return operations from another service", func(t *testing.T) {
		require.Nil(t, cleanup())

		err := q.UpsertService(ctx, "service-1")
		require.Nil(t, err)

		serviceID, err := q.GetServiceID(ctx, "service-1")
		require.Nil(t, err)

		err = q.UpsertOperation(ctx, sql.UpsertOperationParams{
			Name:      "Something",
			ServiceID: serviceID,
			Kind:      sql.SpankindClient,
		})
		require.Nil(t, err)

		operations, err := q.GetOperations(ctx, "service-2")
		require.Nil(t, err)

		require.Len(t, operations, 0)
	})

	t.Run("should return something when an operation exists", func(t *testing.T) {
		require.Nil(t, cleanup())

		err := q.UpsertService(ctx, "service-1")
		require.Nil(t, err)

		serviceID, err := q.GetServiceID(ctx, "service-1")
		require.Nil(t, err)

		err = q.UpsertOperation(ctx, sql.UpsertOperationParams{
			Name:      "Something",
			ServiceID: serviceID,
			Kind:      sql.SpankindClient,
		})
		require.Nil(t, err)

		operations, err := q.GetOperations(ctx, "service-1")
		require.Nil(t, err)

		require.Equal(t, []sql.GetOperationsRow{{Name: "Something", Kind: sql.SpankindClient}}, operations)
	})
}

func TestGetServices(t *testing.T) {
	ctx := context.Background()
	conn, cleanup := sqltest.Harness(t)
	q := sql.New(conn)

	t.Run("should return nothing when no services exist", func(t *testing.T) {
		require.Nil(t, cleanup())

		services, err := q.GetServices(ctx)
		require.Nil(t, err)

		require.Empty(t, services)
	})

	t.Run("should return something when an services exists", func(t *testing.T) {
		require.Nil(t, cleanup())

		err := q.UpsertService(ctx, "Something")
		require.Nil(t, err)

		serviceID, err := q.GetServiceID(ctx, "Something")
		require.Nil(t, err)

		require.NotNil(t, serviceID)

		services, err := q.GetServices(ctx)
		require.Nil(t, err)

		require.Equal(t, []string{"Something"}, services)
	})
}
