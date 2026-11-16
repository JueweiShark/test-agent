package wuwen

type VALUE interface {
    struct{} | interface{}
}
type LinkedHashSet[K comparable] struct {
    orderedList []K
    elementSet  map[K]struct{}
}

// NewLinkedHashSet 创建一个新的 LinkedHashSet
func NewLinkedHashSet[K comparable]() *LinkedHashSet[K] {
    return &LinkedHashSet[K]{
        orderedList: make([]K, 0),
        elementSet:  make(map[K]struct{}),
    }
}

// Add 向集合中添加元素
func (set *LinkedHashSet[K]) Add(elements ...K) {
    for _, element := range elements {
        if _, exists := set.elementSet[element]; !exists {
            set.elementSet[element] = struct{}{}
            set.orderedList = append(set.orderedList, element)
        }
    }

}
func (set *LinkedHashSet[K]) AddAll(elements []K) {
    for _, element := range elements {
        if _, exists := set.elementSet[element]; !exists {
            set.elementSet[element] = struct{}{}
            set.orderedList = append(set.orderedList, element)
        }
    }

}
func (set *LinkedHashSet[K]) Size() int64 {
    if set == nil {
        return 0
    }
    return int64(len(set.orderedList))
}

// GetOrderedElements 获取有序的元素列表
func (set *LinkedHashSet[K]) GetOrderedElements() []K {
    return set.orderedList
}

// Clear 获取有序的元素列表
func (set *LinkedHashSet[K]) Clear() {
    set.orderedList = make([]K, 0)
    set.elementSet = make(map[K]struct{})
}
