package nocontext

// NoContext has no context parameter - should not be modified
func NoContext(value string) error {
	return nil
}

// UnnamedContext has unnamed context - should not be modified
func UnnamedContext(_ string) error {
	return nil
}

type Service struct{}

// NoContextMethod has no context - should not be modified
func (s *Service) NoContextMethod(id int) error {
	return nil
}
