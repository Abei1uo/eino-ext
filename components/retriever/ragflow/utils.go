package ragflow

func ptrOf[T any](v T) *T {
	return &v
}

func dereferenceOrZero[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

func copyPtr[T any](ptr *T) *T {
	if ptr == nil {
		return nil
	}
	val := *ptr
	return &val
}
