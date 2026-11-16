package wuwen

import (
    "testing"

    "github.com/go-kratos/kratos/v2/log"
)

func TestLinkedHashSet_Add(t *testing.T) {
    set := NewLinkedHashSet[string]()
    set.Add("1")
    set.Add("2")
    set.Add("3", "4")
    set.AddAll([]string{"3", "5"})
    elements := set.GetOrderedElements()
    log.Info(elements)
}
