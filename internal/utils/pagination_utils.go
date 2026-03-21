package utils

import (
	"net/http"
	"strconv"
	"time"
)

type PaginationParams struct {
	Limit  int
	Offset int
}

func ReadPaginationParams(r *http.Request) PaginationParams {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	return PaginationParams{Limit: limit, Offset: offset}
}

type QueryParams struct {
	Limit     int
	Offset    int
	StartDate *time.Time
	EndDate   *time.Time
}

func ReadQueryParams(r *http.Request) QueryParams {
	p := ReadPaginationParams(r)
	startDate := ParseDateParam(r, "start_date")
	endDate := ParseDateParam(r, "end_date")
	if endDate != nil {
		// Push end_date to 23:59:59 IST so same-day queries return the full day.
		// ParseDateParam already subtracted 5h30m, so adding 23h59m59s here
		// is equivalent to setting 23:59:59 IST then converting to UTC.
		t := endDate.Add(24*time.Hour - time.Second)
		endDate = &t
	}
	return QueryParams{
		Limit:     p.Limit,
		Offset:    p.Offset,
		StartDate: startDate,
		EndDate:   endDate,
	}
}
