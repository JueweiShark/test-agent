package wuwen

import "strings"

type Opt struct {
    NoRepeat bool
}
type SliceOpt func(o *Opt)

func WithNoRepeat() SliceOpt {
    return func(o *Opt) {
        o.NoRepeat = true
    }
}
func parseOpt(o []SliceOpt) *Opt {
    opt := &Opt{}
    for _, o := range o {
        o(opt)
    }
    return opt
}

func SplitSlice[T any](input []T, chunkSize int) [][]T {
    var result [][]T
    for i := 0; i < len(input); i += chunkSize {
        end := i + chunkSize
        if end > len(input) {
            end = len(input)
        }
        result = append(result, input[i:end])
    }

    return result
}

func SliceMapper[T any, W any](slice []T, mapper func(T) W) []W {

    ws := make([]W, 0)
    for _, item := range slice {
        ws = append(ws, mapper(item))
    }

    return ws
}
func SliceMapperFilter[T any, W any](slice []T, mapper func(T) (bool, W)) []W {

    ws := make([]W, 0)
    for _, item := range slice {
        if filter, w := mapper(item); filter {
            ws = append(ws, w)
        }
    }
    return ws
}
func SliceMapperFilterWithNonRepeat[T any, W comparable](slice []T, mapper func(T) (bool, W)) []W {
    set := NewLinkedHashSet[W]()
    for _, item := range slice {
        if filter, w := mapper(item); filter {
            set.Add(w)
        }
    }
    return set.GetOrderedElements()
}

func SliceMapper2Map[T any, K comparable, V any](slice []T, mapper func(T) (K, V)) map[K]V {

    out_map := make(map[K]V)
    for _, item := range slice {
        k, v := mapper(item)
        out_map[k] = v

    }
    return out_map
}
func SliceMapper2MapWithFilter[T any, K comparable, V any](slice []T, mapper func(T) (bool, K, V)) map[K]V {

    out_map := make(map[K]V)
    for _, item := range slice {
        if b, k, v := mapper(item); b {
            out_map[k] = v
        }
    }
    return out_map
}

func SliceMapper2MapValueMapper[T any, K comparable, V any](slice []T, mapper func(T) (K, V), mergeFunction func(V, V) V) map[K]V {
    out_map := make(map[K]V)
    for _, item := range slice {
        k, v := mapper(item)
        _, exists := out_map[k]
        if exists {
            mapValue := mergeFunction(out_map[k], v)
            out_map[k] = mapValue
        } else {
            out_map[k] = v
        }
    }
    return out_map
}
func SliceMapper2MapWithGroup[T any, K comparable, V any](slice []T, mapper func(T) (K, V)) map[K][]V {

    outMap := make(map[K][]V)
    for _, item := range slice {
        k, v := mapper(item)
        if v2, exists := outMap[k]; exists {
            outMap[k] = append(v2, v)
        } else {
            vNew := make([]V, 0)
            outMap[k] = append(vNew, v)
        }
    }
    return outMap
}
func MapContainsKey[T comparable, V VALUE](maps map[T]V, key T) bool {
    _, exists := maps[key]
    return exists
}
func ToCamelCase(str string) string {
    // 将字符串分割为单词
    words := strings.Split(str, "_")

    // 将单词转换为首字母大写的驼峰格式
    for i := 0; i < len(words); i++ {
        words[i] = strings.Title(words[i])
    }
    // 将单词连接起来
    camelCaseStr := strings.Join(words, "")
    return camelCaseStr
}

func RemoveIfContains[k comparable](source []k, filters []k) []k {
    filterMap := make(map[k]struct{})
    for _, id := range filters {
        filterMap[id] = struct{}{}
    }

    var result []k
    for _, id := range source {
        if _, exist := filterMap[id]; !exist {
            result = append(result, id)
        }
    }
    return result
}

func MaxInt64(a, b int64) int64 {
    if a > b {
        return a
    } else {
        return b
    }
}

func MinInt64(a, b int64) int64 {
    if a < b {
        return a
    } else {
        return b
    }
}
