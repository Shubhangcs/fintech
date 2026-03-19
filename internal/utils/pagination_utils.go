package utils

import (
	"net/http"
	"strconv"
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
