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
	if limit == 0 {
		limit = 10
	}

	return &Pagination{
		Page:   page,
		Limit:  max(min(limit, 80), 1),
		Offset: (page - 1) * limit,
		Total:  -1, // assert
	}
}

// apply as query filter string
func (p *Pagination) Query() string {
	return fmt.Sprintf("OFFSET %d LIMIT %d ", p.Offset, p.Limit)
}

// calculate last page and the in-between pages
func (p *Pagination) WithPages() *Pagination {
	if p.Total < 0 {
		panic("count and set total rows first")
	}

	p.Last = int(math.Ceil(float64(p.Total) / float64(p.Limit)))

	for i := p.Page - 2; i < p.Page+3; i++ {
		if i > 0 && i <= p.Last {
			p.Between = append(p.Between, i)
		}
	}

	return p
}
