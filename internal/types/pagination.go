package types

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
)

type Pagination struct {
	Limit  int
	Offset int
	Total  int // TODO: use cache OR reuse it from frontend in the path param

	Page    int
	Between []int
	Last    int
}

func NewPagination(u url.Values) *Pagination {
	page, _ := strconv.Atoi(u.Get("page"))
	page = max(page, 1)

	limit, _ := strconv.Atoi(u.Get("limit"))
	limit = max(min(limit, 80), 10)

	return &Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

// apply as query filter string
func (p *Pagination) Query() string {
	return fmt.Sprintf("OFFSET %d LIMIT %d ", p.Offset, p.Limit)
}

// calculate last page and the in-between pages
func (p *Pagination) ParsePages(count int) {
	ceil := math.Ceil(float64(count) / float64(p.Limit))
	p.Last = int(ceil)
	p.Total = count

	for i := p.Page - 2; i < p.Page+3; i++ {
		if i > 0 && i <= p.Last {
			p.Between = append(p.Between, i)
		}
	}
}
