package semantic

type Generator interface {
	Read() <-chan interface{}
}
