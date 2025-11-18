package data

import "fmt"

const PAGE_SIZE int = 50

// Typically represents a stream of data from a paginated query response.
type IterablePaginatedData[T any] struct {
	// Gets data at the given page.
	GetPage func(*string) ([]T, *string, error)
	currentPage []T
	currentPositionInPage int
	currentLastID *string
}
// Provide the next value if it exists, otherwise check for more data before returning nil.
func (i *IterablePaginatedData[T]) Next() (*T, error) {
    // Lazy initialization
    if i.currentLastID == nil && len(i.currentPage) == 0 {
        err := i.goToNextPage()
        if err != nil {
            return nil, err
        }
    }
	for {
        // Within page
        if i.currentPositionInPage < len(i.currentPage) {
            value := &i.currentPage[i.currentPositionInPage]
            i.currentPositionInPage++
            return value, nil
        }

        // End of page
        err := i.goToNextPage()
        if err != nil {
            return nil, err
        }

        // No more data
        if len(i.currentPage) == 0 {
            return nil, nil
        }
    }
}
// Get next page and update state for next next page
func (i *IterablePaginatedData[T]) goToNextPage() error {
    i.currentPositionInPage = 0
    nextPage, newLastID, err := i.GetPage(i.currentLastID)
    if err != nil {
        return fmt.Errorf("error getting page %v: %w", i.currentLastID, err)
    }
    i.currentPage = nextPage
    i.currentLastID = newLastID
    return nil
}