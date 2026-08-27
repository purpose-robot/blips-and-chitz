package health

import "context"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Check(ctx context.Context) (Status, error) {
	return Status{Message: "available"}, nil
}
