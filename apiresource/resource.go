package apiresource

type Resource[T any] func(T) any

func One[T any](item T, transform Resource[T]) any {
	if transform == nil {
		return item
	}
	return transform(item)
}

func Collection[T any](items []T, transform Resource[T]) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, One(item, transform))
	}
	return result
}

type Paginated struct {
	Data       any `json:"data"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func Page(data any, page, perPage, total int) Paginated {
	totalPages := 0
	if perPage > 0 && total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	return Paginated{
		Data: data,
		Page: page,
		PerPage: perPage,
		Total: total,
		TotalPages: totalPages,
	}
}
