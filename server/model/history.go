package model

import (
	"context"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type HistoryParams struct {
	Department string
	Start      time.Time `form:"start" binding:"required" time_format:"20060102T150405Z0700"`
	End        time.Time `form:"end" binding:"required" time_format:"20060102T150405Z0700"`
	Limit      int       `form:"limit" binding:"omitempty"`
	Comment    string
}

type HistoryResult struct {
	Counts Counts
	Calls  []Call
}

type Counts struct {
	Total    uint64 `json:"total"`    // Total calls count
	In       uint64 `json:"incoming"` // Incoming calls count
	Out      uint64 `json:"outgoing"` // Outgoing calls count
	Missed   uint64 `json:"missed"`   // Incoming missed calls count
	Noanswer uint64 `json:"noanswer"` // Outgoing missed calls count
}

type Call struct {
	// Call start in UTC ('20211005T063228Z')
	Start time.Time
	// Index number in result set
	ID string
	// Call uuid
	UID string
	// Call type:
	// * in - incoming answered
	// * out - outgoing answered
	// * missed - incoming missed
	// * noanswer - outgoing missed
	CallType string
	// Client phone number
	Client uint64
	// Employee phone number
	EmployeeNumber uint64
	// Employee name
	EmployeeName string
	// Waiting for answer, [sec]
	Waiting uint32
	// Call duration (0 if missed), [sec]
	Duration uint32
}

func GetHistory(params *HistoryParams, conn clickhouse.Conn, tmpl *template.Template) (calls []Call, counts Counts, err error) {
	var query strings.Builder
	err = tmpl.Execute(&query, params)
	if err != nil {
		return calls, counts, err
	}
	args := []interface{}{
		params.Department,
		params.Start,
		params.End,
		params.Limit,
		params.Comment,
	}
	rows, err := conn.Query(context.Background(), query.String(), args...)
	if err != nil {
		return calls, counts, err
	}
	defer rows.Close()

	lastID := 0
	for rows.Next() {
		call := Call{}
		values := []interface{}{
			&counts.Total, &counts.In, &counts.Out, &counts.Missed, &counts.Noanswer, &call.Start, &call.UID,
			&call.CallType, &call.Client, &call.EmployeeNumber, &call.EmployeeName, &call.Waiting, &call.Duration,
		}
		err := rows.Scan(values...)
		if err != nil {
			return calls, counts, err
		}
		call.ID = strconv.Itoa(lastID)
		lastID++
		calls = append(calls, call)
	}
	if len(calls) > 0 {
		calls = calls[:len(calls)-1]
	}
	return calls, counts, nil
}
