package helper

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable](values ...T) *Set[T] {
	s := Set[T]{map[T]struct{}{}}
	s.Add(values...)
	return &s
}

func (s *Set[T]) Has(values ...T) (exists bool) {
	for _, value := range values {
		if _, exists = s.m[value]; !exists {
			return false
		}
	}
	return exists
}

func (s *Set[T]) Add(values ...T) {
	for _, value := range values {
		s.m[value] = struct{}{}
	}
}

func (s *Set[T]) Remove(values ...T) {
	for _, v := range values {
		delete(s.m, v)
	}
}

func (s *Set[T]) Len() int {
	return len(s.m)
}

func (s *Set[T]) Values() (values []T) {
	for value := range s.m {
		values = append(values, value)
	}
	return values
}
