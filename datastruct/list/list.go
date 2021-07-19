package list

type Consumer func(int, interface{}) bool

type List interface {
	Add(val interface{})
	Get(index int) (val interface{})
	Set(index int, val interface{})
	Insert(index int, val interface{})
	Remove(index int) (val interface{})
	RemoveLast() (val interface{})
	RemoveAllByVal(val interface{}) int
	RemoveByVal(val interface{}, count int) int
	ReverseRemoveByVal(val interface{}, count int) int
	Len() int
	ForEach(consumer func(int, interface{}) bool)
	Contains(val interface{}) bool
	Range(start int, stop int) []interface{}
}