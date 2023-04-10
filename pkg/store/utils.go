package store

import (
	"fmt"
	"strconv"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

// BuildSlotBoundsFilterClause builds a SQL query clause that filters by slot bounds.
func BuildSlotBoundsFilterClause(query string, slotBounds *types.SlotBounds) string {
	if slotBounds == nil {
		return query
	}
	if slotBounds.StartSlot != nil {
		query = query + ` AND slot >= ` + strconv.FormatUint(uint64(*slotBounds.StartSlot), 10)
	}
	if slotBounds.EndSlot != nil {
		query = query + ` AND slot <= ` + strconv.FormatUint(uint64(*slotBounds.EndSlot), 10)
	}
	return query
}

// BuildCategoryFilterClause builds a SQL query clause that filters by category.
func BuildCategoryFilterClause(query string, filter *types.AnalysisQueryFilter) string {
	if filter == nil {
		return query
	}

	return query + ` AND category ` + filter.Comparator + ` '` + fmt.Sprintf("%d", filter.Category) + `'`
}
