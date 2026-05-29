package semantic

type Generator interface {
	Run() error
	ReadChan() <-chan interface{}
}
