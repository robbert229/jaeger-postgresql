package sql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

func getFindTraceIDs(limit int) string {
	return fmt.Sprintf(`-- name: FindTraceIDs :many
SELECT DISTINCT spans.trace_id
FROM spans
    INNER JOIN operations ON (operations.id = spans.operation_id)
    INNER JOIN services ON (services.id = spans.service_id)
WHERE
    (services.name = $1::VARCHAR OR $2::BOOLEAN = FALSE) AND
    (operations.name = $3::VARCHAR OR $4::BOOLEAN = FALSE) AND
    (start_time >= $5::TIMESTAMPTZ OR $6::BOOLEAN = FALSE) AND
    (start_time < $7::TIMESTAMPTZ OR $8::BOOLEAN = FALSE) AND
    (duration > $9::INTERVAL OR $10::BOOLEAN = FALSE) AND
    (duration < $11::INTERVAL OR $12::BOOLEAN = FALSE)
LIMIT %d
`, limit)
}

type FindTraceIDsParams struct {
	ServiceName            string
	ServiceNameEnable      bool
	OperationName          string
	OperationNameEnable    bool
	StartTimeMinimum       pgtype.Timestamptz
	StartTimeMinimumEnable bool
	StartTimeMaximum       pgtype.Timestamptz
	StartTimeMaximumEnable bool
	DurationMinimum        pgtype.Interval
	DurationMinimumEnable  bool
	DurationMaximum        pgtype.Interval
	DurationMaximumEnable  bool
	NumTraces              int
}

func (q *Queries) FindTraceIDs(ctx context.Context, arg FindTraceIDsParams) ([][]byte, error) {
	rows, err := q.db.Query(ctx, getFindTraceIDs(arg.NumTraces),
		arg.ServiceName,
		arg.ServiceNameEnable,
		arg.OperationName,
		arg.OperationNameEnable,
		arg.StartTimeMinimum,
		arg.StartTimeMinimumEnable,
		arg.StartTimeMaximum,
		arg.StartTimeMaximumEnable,
		arg.DurationMinimum,
		arg.DurationMinimumEnable,
		arg.DurationMaximum,
		arg.DurationMaximumEnable,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var items [][]byte
	for rows.Next() {
		var trace_id []byte
		if err := rows.Scan(&trace_id); err != nil {
			return nil, err
		}
		items = append(items, trace_id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
