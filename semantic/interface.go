package semantic

type Generator interface {
	Run() error
	ReadChan() <-chan SemanticClaim
}

type Validator interface {
	ValidateSemantic(claim SemanticClaim) (bool, error)
}
