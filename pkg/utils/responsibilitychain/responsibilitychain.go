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
	nexter := res.Nexter()
	for nexter != nil {
		if err := nexter.Execute(); err != nil {
			return err
		}
		nexter = nexter.Nexter()
	}

	return nil
}
