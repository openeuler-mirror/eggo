package responsibilitychain

type Responsibility interface {
	Execute() error
	SetNexter(Responsibility)
	Nexter() Responsibility
}

func RunChainOfResponsibility(res Responsibility) error {
	if res == nil {
		return nil
	}
	if err := res.Execute(); err != nil {
		return err
	}
	return res.Nexter().Execute()
}
