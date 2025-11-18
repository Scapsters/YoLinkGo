package data

import "fmt"

const PAGE_SIZE int = 50

// Typically represents a stream of data from a paginated query response.
type IterablePaginatedData[T any] struct {
	// Gets data at the given page.
	GetPage func(int) ([]T, error)
	currentPage []T
	currentPositionInPage int
	currentPageNumber int
}
// Provide the next value if it exists, otherwise check for more data before returning nil.
func (i *IterablePaginatedData[T]) Next() (*T, error) {
	for {
        // Within page
        if i.currentPositionInPage < len(i.currentPage) {
            value := &i.currentPage[i.currentPositionInPage]
            i.currentPositionInPage++
            return value, nil
        }

        // End of page
        i.currentPageNumber++
        i.currentPositionInPage = 0
        nextPage, err := i.GetPage(i.currentPageNumber)
		if err != nil {
			return nil, fmt.Errorf("error getting page %v: %w", i.currentPageNumber, err)
		}

        // No more data
        if len(nextPage) == 0 {
            return nil, nil
        }

        i.currentPage = nextPage
    }
}