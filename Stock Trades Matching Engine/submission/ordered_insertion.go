package submission

import(
    "sort"
)

func insertSorted(orders []*order, newOrder *order, less func(*order, *order) bool) []*order {
    idx := sort.Search(len(orders), func(idx int) bool {
        return !less(orders[idx], newOrder)
    })
    orders = append(orders, nil)
    copy(orders[idx+1:], orders[idx:])
    orders[idx] = newOrder
    return orders
}
