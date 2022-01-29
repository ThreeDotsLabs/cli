package files

type RealPather interface {
	RealPath(name string) (path string, err error)
}
